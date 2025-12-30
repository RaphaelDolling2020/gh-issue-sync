package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitsuhiko/gh-issue-sync/internal/issue"
	"github.com/mitsuhiko/gh-issue-sync/internal/paths"
)

type IssueFile struct {
	Issue issue.Issue
	Path  string
	State string
}

// LabelCache stores the synced labels from GitHub
type LabelCache struct {
	Labels   []LabelEntry `json:"labels"`
	SyncedAt time.Time    `json:"synced_at"`
}

// LabelEntry represents a single label with its color
type LabelEntry struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// MilestoneCache stores the synced milestones from GitHub
type MilestoneCache struct {
	Milestones []MilestoneEntry `json:"milestones"`
	SyncedAt   time.Time        `json:"synced_at"`
}

// MilestoneEntry represents a single milestone
type MilestoneEntry struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	DueOn       *string `json:"due_on,omitempty"`
	State       string  `json:"state"`
}

// IssueTypeCache stores the synced issue types from GitHub
type IssueTypeCache struct {
	IssueTypes []IssueTypeEntry `json:"issue_types"`
	SyncedAt   time.Time        `json:"synced_at"`
}

// IssueTypeEntry represents a single issue type
type IssueTypeEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ProjectCache stores the synced projects from GitHub
type ProjectCache struct {
	Projects []ProjectEntry `json:"projects"`
	SyncedAt time.Time      `json:"synced_at"`
}

// ProjectEntry represents a single project
type ProjectEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ParseError represents an error parsing a specific issue file
type ParseError struct {
	Path string
	Err  error
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}

// LoadResult contains loaded issues and any parse errors encountered
type LoadResult struct {
	Issues []IssueFile
	Errors []ParseError
}

func loadLocalIssues(p paths.Paths) ([]IssueFile, error) {
	result := loadLocalIssuesWithErrors(p)
	if len(result.Errors) > 0 {
		return nil, result.Errors[0]
	}
	return result.Issues, nil
}

func loadLocalIssuesWithErrors(p paths.Paths) LoadResult {
	result := LoadResult{}
	for _, dir := range []struct {
		Path  string
		State string
	}{{p.OpenDir, "open"}, {p.ClosedDir, "closed"}} {
		entries, err := os.ReadDir(dir.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			// Directory read errors are fatal
			result.Errors = append(result.Errors, ParseError{Path: dir.Path, Err: err})
			return result
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if filepath.Ext(entry.Name()) != ".md" {
				continue
			}
			path := filepath.Join(dir.Path, entry.Name())
			relPath := filepath.Join(filepath.Base(filepath.Dir(dir.Path)), filepath.Base(dir.Path), entry.Name())
			parsed, err := issue.ParseFile(path)
			if err != nil {
				result.Errors = append(result.Errors, ParseError{Path: relPath, Err: err})
				continue
			}
			parsed.State = dir.State
			result.Issues = append(result.Issues, IssueFile{Issue: parsed, Path: path, State: dir.State})
		}
	}
	return result
}

func findIssueByNumber(p paths.Paths, number string) (IssueFile, error) {
	issues, err := loadLocalIssues(p)
	if err != nil {
		return IssueFile{}, err
	}
	for _, item := range issues {
		if item.Issue.Number.String() == number {
			return item, nil
		}
	}
	return IssueFile{}, fmt.Errorf("issue %s not found", number)
}

// findIssueByRef finds an issue by number, local ID (T...), or file path
func findIssueByRef(root string, p paths.Paths, ref string) (IssueFile, error) {
	ref = strings.TrimSpace(ref)

	// Check if it's a file path
	if strings.HasSuffix(ref, ".md") || strings.Contains(ref, string(os.PathSeparator)) {
		path := ref
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		parsed, err := issue.ParseFile(path)
		if err != nil {
			return IssueFile{}, fmt.Errorf("failed to parse %s: %w", ref, err)
		}
		// Determine state from path
		state := "open"
		if strings.Contains(path, string(os.PathSeparator)+"closed"+string(os.PathSeparator)) {
			state = "closed"
		}
		parsed.State = state
		return IssueFile{Issue: parsed, Path: path, State: state}, nil
	}

	// Otherwise look up by number
	return findIssueByNumber(p, ref)
}

func readOriginalIssue(p paths.Paths, number string) (issue.Issue, bool) {
	path := filepath.Join(p.OriginalsDir, fmt.Sprintf("%s.md", number))
	parsed, err := issue.ParseFile(path)
	if err != nil {
		return issue.Issue{}, false
	}
	return parsed, true
}

func writeOriginalIssue(p paths.Paths, item issue.Issue) error {
	path := filepath.Join(p.OriginalsDir, fmt.Sprintf("%s.md", item.Number))
	return issue.WriteFile(path, item)
}

func loadLabelCache(p paths.Paths) (LabelCache, error) {
	var cache LabelCache
	data, err := os.ReadFile(p.LabelsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	return cache, nil
}

func saveLabelCache(p paths.Paths, cache LabelCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p.LabelsPath, data, 0o644)
}

// labelCacheToColorMap converts a LabelCache to a map of lowercase name -> color for quick lookups.
func labelCacheToColorMap(cache LabelCache) map[string]string {
	m := make(map[string]string, len(cache.Labels))
	for _, l := range cache.Labels {
		m[strings.ToLower(l.Name)] = l.Color
	}
	return m
}

// labelsFromColorMap creates a LabelCache from a color map.
func labelsFromColorMap(colors map[string]string, syncedAt time.Time) LabelCache {
	labels := make([]LabelEntry, 0, len(colors))
	for name, color := range colors {
		labels = append(labels, LabelEntry{Name: name, Color: color})
	}
	sort.Slice(labels, func(i, j int) bool {
		return strings.ToLower(labels[i].Name) < strings.ToLower(labels[j].Name)
	})
	return LabelCache{Labels: labels, SyncedAt: syncedAt}
}

func loadMilestoneCache(p paths.Paths) (MilestoneCache, error) {
	var cache MilestoneCache
	data, err := os.ReadFile(p.MilestonesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	return cache, nil
}

func saveMilestoneCache(p paths.Paths, cache MilestoneCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p.MilestonesPath, data, 0o644)
}

// milestoneNames returns a set of milestone titles (case-insensitive lookup).
func milestoneNames(cache MilestoneCache) map[string]struct{} {
	m := make(map[string]struct{}, len(cache.Milestones))
	for _, ms := range cache.Milestones {
		m[strings.ToLower(ms.Title)] = struct{}{}
	}
	return m
}

func loadIssueTypeCache(p paths.Paths) (IssueTypeCache, error) {
	var cache IssueTypeCache
	data, err := os.ReadFile(p.IssueTypesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	return cache, nil
}

func saveIssueTypeCache(p paths.Paths, cache IssueTypeCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p.IssueTypesPath, data, 0o644)
}

// issueTypeByName returns a map of lowercase name -> IssueTypeEntry for quick lookups.
func issueTypeByName(cache IssueTypeCache) map[string]IssueTypeEntry {
	m := make(map[string]IssueTypeEntry, len(cache.IssueTypes))
	for _, it := range cache.IssueTypes {
		m[strings.ToLower(it.Name)] = it
	}
	return m
}

func loadProjectCache(p paths.Paths) (ProjectCache, error) {
	var cache ProjectCache
	data, err := os.ReadFile(p.ProjectsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cache, nil
		}
		return cache, err
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}
	return cache, nil
}

func saveProjectCache(p paths.Paths, cache ProjectCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p.ProjectsPath, data, 0o644)
}

// projectByTitle returns a map of lowercase title -> ProjectEntry for quick lookups.
func projectByTitle(cache ProjectCache) map[string]ProjectEntry {
	m := make(map[string]ProjectEntry, len(cache.Projects))
	for _, p := range cache.Projects {
		m[strings.ToLower(p.Title)] = p
	}
	return m
}
