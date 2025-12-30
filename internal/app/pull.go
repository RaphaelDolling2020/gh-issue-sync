package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mitsuhiko/gh-issue-sync/internal/config"
	"github.com/mitsuhiko/gh-issue-sync/internal/ghcli"
	"github.com/mitsuhiko/gh-issue-sync/internal/issue"
	"github.com/mitsuhiko/gh-issue-sync/internal/lock"
	"github.com/mitsuhiko/gh-issue-sync/internal/paths"
)

func (a *App) Pull(ctx context.Context, opts PullOptions, args []string) error {
	p := paths.New(a.Root)
	cfg, err := loadConfig(p.ConfigPath)
	if err != nil {
		return err
	}

	// Acquire lock
	lck, err := lock.Acquire(p.SyncDir, lock.DefaultTimeout)
	if err != nil {
		return err
	}
	defer lck.Release()

	client := ghcli.NewClient(a.Runner, repoSlug(cfg))
	t := a.Theme

	localIssues, err := loadLocalIssues(p)
	if err != nil {
		return err
	}

	var remoteIssues []issue.Issue
	var labelColors map[string]string

	if len(args) > 0 {
		// Fetch specific issues by number
		labelColors = a.fetchLabelColors(ctx, client)

		for _, arg := range args {
			number := strings.TrimSpace(arg)
			if number == "" {
				continue
			}
			remote, err := client.GetIssue(ctx, number)
			if err != nil {
				return err
			}
			remoteIssues = append(remoteIssues, remote)
		}
		// Enrich with relationships
		if err := client.EnrichWithRelationshipsBatch(ctx, remoteIssues); err != nil {
			fmt.Fprintf(a.Err, "%s fetching relationships: %v\n", t.WarningText("Warning:"), err)
		}
	} else {
		state := "open"
		if opts.All {
			state = "all"
		}

		// Collect issue numbers we need to fetch for closed issues
		var toFetch []string
		if !opts.All {
			// We don't know remote issue numbers yet, so we'll collect all local non-local issues
			// and filter after we get the open issues
			for _, local := range localIssues {
				if !local.Issue.Number.IsLocal() {
					toFetch = append(toFetch, local.Issue.Number.String())
				}
			}
		}

		// Run both queries in parallel
		type listResult struct {
			result ghcli.ListIssuesResult
			err    error
		}
		type batchResult struct {
			issues map[string]issue.Issue
			err    error
		}

		listCh := make(chan listResult, 1)
		batchCh := make(chan batchResult, 1)

		go func() {
			r, e := client.ListIssuesWithRelationships(ctx, state, opts.Label)
			listCh <- listResult{r, e}
		}()

		go func() {
			if len(toFetch) > 0 {
				r, e := client.GetIssuesBatch(ctx, toFetch)
				batchCh <- batchResult{r, e}
			} else {
				batchCh <- batchResult{nil, nil}
			}
		}()

		listRes := <-listCh
		if listRes.err != nil {
			return listRes.err
		}
		remoteIssues = listRes.result.Issues
		labelColors = listRes.result.LabelColors

		batchRes := <-batchCh
		if batchRes.err == nil && len(batchRes.issues) > 0 {
			// Filter out issues we already have from the open list
			fetched := make(map[string]struct{}, len(remoteIssues))
			for _, ri := range remoteIssues {
				fetched[ri.Number.String()] = struct{}{}
			}
			for num, iss := range batchRes.issues {
				if _, ok := fetched[num]; !ok {
					remoteIssues = append(remoteIssues, iss)
				}
			}
		}
	}

	localIssues, err = loadLocalIssues(p)
	if err != nil {
		return err
	}
	localByNumber := map[string]IssueFile{}
	for _, item := range localIssues {
		localByNumber[item.Issue.Number.String()] = item
	}

	var conflicts []string
	unchanged := 0
	for _, remote := range remoteIssues {
		remote.State = strings.ToLower(remote.State)
		remote.SyncedAt = ptrTime(a.Now().UTC())

		local, hasLocal := localByNumber[remote.Number.String()]
		original, hasOriginal := readOriginalIssue(p, remote.Number.String())
		localChanged := false
		if hasLocal {
			if !hasOriginal {
				localChanged = true
			} else {
				localChanged = !issue.EqualIgnoringSyncedAt(local.Issue, original)
			}
		}

		if hasLocal && localChanged && !opts.Force {
			conflicts = append(conflicts, remote.Number.String())
			continue
		}

		targetDir := p.OpenDir
		if remote.State == "closed" {
			targetDir = p.ClosedDir
		}
		newPath := issue.PathFor(targetDir, remote.Number, remote.Title)
		contentChanged := !hasLocal || !issue.EqualIgnoringSyncedAt(local.Issue, remote)
		pathChanged := hasLocal && local.Path != newPath
		if hasOriginal && !contentChanged && !pathChanged {
			unchanged++
			continue
		}

		if hasLocal && local.Path != newPath {
			if err := os.Rename(local.Path, newPath); err != nil {
				return err
			}
		}
		if err := issue.WriteFile(newPath, remote); err != nil {
			return err
		}
		if err := writeOriginalIssue(p, remote); err != nil {
			return err
		}
		if !hasLocal {
			fmt.Fprintln(a.Out, t.FormatIssueHeader("A", remote.Number.String(), remote.Title))
			continue
		}
		lines := a.formatChangeLines(local.Issue, remote, labelColors)
		if len(lines) == 0 && pathChanged {
			lines = append(lines, t.FormatChange("file", fmt.Sprintf("%q", relPath(a.Root, local.Path)), fmt.Sprintf("%q", relPath(a.Root, newPath))))
		}
		fmt.Fprintln(a.Out, t.FormatIssueHeader("U", remote.Number.String(), remote.Title))
		for _, line := range lines {
			fmt.Fprintln(a.Out, line)
		}
	}

	if len(args) == 0 {
		now := a.Now().UTC()
		cfg.Sync.LastFullPull = &now
		if err := config.Save(p.ConfigPath, cfg); err != nil {
			return err
		}

		// Save labels to cache
		if len(labelColors) > 0 {
			labels := make([]LabelEntry, 0, len(labelColors))
			for name, color := range labelColors {
				labels = append(labels, LabelEntry{Name: name, Color: color})
			}
			// Sort for consistent output
			sort.Slice(labels, func(i, j int) bool {
				return strings.ToLower(labels[i].Name) < strings.ToLower(labels[j].Name)
			})
			cache := LabelCache{Labels: labels, SyncedAt: now}
			if err := saveLabelCache(p, cache); err != nil {
				fmt.Fprintf(a.Err, "%s saving label cache: %v\n", t.WarningText("Warning:"), err)
			}
		}

		// Fetch and save milestones to cache
		milestones, err := client.ListMilestones(ctx)
		if err != nil {
			fmt.Fprintf(a.Err, "%s fetching milestones: %v\n", t.WarningText("Warning:"), err)
		} else {
			entries := make([]MilestoneEntry, 0, len(milestones))
			for _, m := range milestones {
				entries = append(entries, MilestoneEntry{
					Title:       m.Title,
					Description: m.Description,
					DueOn:       m.DueOn,
					State:       m.State,
				})
			}
			// Sort for consistent output
			sort.Slice(entries, func(i, j int) bool {
				return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
			})
			msCache := MilestoneCache{Milestones: entries, SyncedAt: now}
			if err := saveMilestoneCache(p, msCache); err != nil {
				fmt.Fprintf(a.Err, "%s saving milestone cache: %v\n", t.WarningText("Warning:"), err)
			}
		}

		// Fetch and save issue types to cache (org repos only)
		issueTypes, err := client.ListIssueTypes(ctx)
		if err != nil {
			fmt.Fprintf(a.Err, "%s fetching issue types: %v\n", t.WarningText("Warning:"), err)
		} else if len(issueTypes) > 0 {
			entries := make([]IssueTypeEntry, 0, len(issueTypes))
			for _, it := range issueTypes {
				entries = append(entries, IssueTypeEntry{
					ID:          it.ID,
					Name:        it.Name,
					Description: it.Description,
				})
			}
			// Sort for consistent output
			sort.Slice(entries, func(i, j int) bool {
				return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
			})
			itCache := IssueTypeCache{IssueTypes: entries, SyncedAt: now}
			if err := saveIssueTypeCache(p, itCache); err != nil {
				fmt.Fprintf(a.Err, "%s saving issue type cache: %v\n", t.WarningText("Warning:"), err)
			}
		}

		// Fetch and save projects to cache (requires read:project scope)
		projects, err := client.ListProjects(ctx)
		if err != nil {
			// Don't warn - scope might not be available
		} else if len(projects) > 0 {
			entries := make([]ProjectEntry, 0, len(projects))
			for _, proj := range projects {
				entries = append(entries, ProjectEntry{
					ID:    proj.ID,
					Title: proj.Title,
				})
			}
			// Sort for consistent output
			sort.Slice(entries, func(i, j int) bool {
				return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
			})
			projCache := ProjectCache{Projects: entries, SyncedAt: now}
			if err := saveProjectCache(p, projCache); err != nil {
				fmt.Fprintf(a.Err, "%s saving project cache: %v\n", t.WarningText("Warning:"), err)
			}
		}
	}

	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		fmt.Fprintf(a.Err, "%s %s\n", t.WarningText("Conflicts (local changes, skipped):"), strings.Join(conflicts, ", "))
	}
	if unchanged > 0 {
		noun := "issues"
		if unchanged == 1 {
			noun = "issue"
		}
		fmt.Fprintf(a.Out, "%s\n", t.MutedText(fmt.Sprintf("Nothing to pull: %d %s up to date", unchanged, noun)))
	}

	// Restore locally deleted issues (originals exist but no local file)
	if len(args) == 0 {
		if err := a.restoreDeletedIssues(ctx, p, client, labelColors); err != nil {
			return err
		}
	}

	return nil
}

// restoreDeletedIssues finds issues that have originals but no local file and restores them
func (a *App) restoreDeletedIssues(ctx context.Context, p paths.Paths, client *ghcli.Client, labelColors map[string]string) error {
	t := a.Theme

	// List all originals
	entries, err := os.ReadDir(p.OriginalsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	// Build set of local issue numbers
	localIssues, err := loadLocalIssues(p)
	if err != nil {
		return err
	}
	localNumbers := make(map[string]struct{}, len(localIssues))
	for _, item := range localIssues {
		localNumbers[item.Issue.Number.String()] = struct{}{}
	}

	// Find orphaned originals (original exists but no local file)
	var orphaned []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		number := strings.TrimSuffix(entry.Name(), ".md")
		// Skip local issues (T-prefixed)
		if strings.HasPrefix(number, "T") {
			continue
		}
		if _, exists := localNumbers[number]; !exists {
			orphaned = append(orphaned, number)
		}
	}

	if len(orphaned) == 0 {
		return nil
	}

	// Fetch and restore orphaned issues from GitHub
	for _, number := range orphaned {
		remote, err := client.GetIssue(ctx, number)
		if err != nil {
			fmt.Fprintf(a.Err, "%s restoring #%s: %v\n", t.WarningText("Warning:"), number, err)
			continue
		}
		if err := client.EnrichWithRelationships(ctx, &remote); err != nil {
			fmt.Fprintf(a.Err, "%s fetching relationships for #%s: %v\n", t.WarningText("Warning:"), number, err)
		}

		remote.State = strings.ToLower(remote.State)
		remote.SyncedAt = ptrTime(a.Now().UTC())

		targetDir := p.OpenDir
		if remote.State == "closed" {
			targetDir = p.ClosedDir
		}
		newPath := issue.PathFor(targetDir, remote.Number, remote.Title)

		if err := issue.WriteFile(newPath, remote); err != nil {
			return err
		}
		if err := writeOriginalIssue(p, remote); err != nil {
			return err
		}

		fmt.Fprintln(a.Out, t.FormatIssueHeader("R", remote.Number.String(), remote.Title))
	}

	return nil
}

// fetchLabelColors fetches label colors from GitHub, returning a map of name -> hex color.
// Errors are silently ignored (we'll just use default colors).
func (a *App) fetchLabelColors(ctx context.Context, client *ghcli.Client) map[string]string {
	colors := make(map[string]string)
	labels, err := client.ListLabels(ctx)
	if err != nil {
		return colors
	}
	for _, l := range labels {
		colors[strings.ToLower(l.Name)] = l.Color
	}
	return colors
}
