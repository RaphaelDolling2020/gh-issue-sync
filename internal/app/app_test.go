package app

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/mitsuhiko/gh-issue-sync/internal/config"
	"github.com/mitsuhiko/gh-issue-sync/internal/ghcli"
	"github.com/mitsuhiko/gh-issue-sync/internal/issue"
	"github.com/mitsuhiko/gh-issue-sync/internal/paths"
)

func TestApplyMapping(t *testing.T) {
	parent := issue.IssueRef("T1")
	item := issue.Issue{
		Number: issue.IssueNumber("T2"),
		Title:  "Test",
		Body:   "Refs #T1 and #T10\n",
		Parent: &parent,
		BlockedBy: []issue.IssueRef{
			"T1",
			"99",
		},
	}
	mapping := map[string]string{"T1": "123"}
	changed := applyMapping(&item, mapping)
	if !changed {
		t.Fatalf("expected mapping to report change")
	}
	if item.Body != "Refs #123 and #T10\n" {
		t.Fatalf("unexpected body: %q", item.Body)
	}
	if item.Parent == nil || item.Parent.String() != "123" {
		t.Fatalf("unexpected parent: %v", item.Parent)
	}
	if got := item.BlockedBy[0].String(); got != "123" {
		t.Fatalf("unexpected blocked_by mapping: %s", got)
	}
}

func TestNewIssueFromEditor(t *testing.T) {
	root := t.TempDir()
	p := paths.New(root)
	if err := p.EnsureLayout(); err != nil {
		t.Fatalf("layout: %v", err)
	}
	if err := config.Save(p.ConfigPath, config.Default("owner", "repo")); err != nil {
		t.Fatalf("config: %v", err)
	}

	previousInteractive := runInteractiveCommand
	runInteractiveCommand = func(ctx context.Context, command string, args ...string) error {
		if len(args) == 0 {
			t.Fatalf("expected editor path")
		}
		payload, err := issue.Render(issue.Issue{
			Number: issue.IssueNumber("T1"),
			Title:  "Edited Title",
			State:  "open",
			Body:   "Hello\n",
		})
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if err := os.WriteFile(args[len(args)-1], []byte(payload), 0o644); err != nil {
			t.Fatalf("write editor payload: %v", err)
		}
		return nil
	}
	t.Cleanup(func() { runInteractiveCommand = previousInteractive })
	t.Setenv("EDITOR", "true")

	application := New(root, ghcli.ExecRunner{}, io.Discard, io.Discard)
	if err := application.NewIssue(context.Background(), "", NewOptions{Edit: true}); err != nil {
		t.Fatalf("new issue: %v", err)
	}

	expectedPath := issue.PathFor(p.OpenDir, issue.IssueNumber("T1"), "Edited Title")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected issue file: %v", err)
	}
	parsed, err := issue.ParseFile(expectedPath)
	if err != nil {
		t.Fatalf("parse issue: %v", err)
	}
	if parsed.Title != "Edited Title" {
		t.Fatalf("unexpected title: %q", parsed.Title)
	}
	cfg, err := config.Load(p.ConfigPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if cfg.Local.NextLocalID != 2 {
		t.Fatalf("expected next local id to be 2, got %d", cfg.Local.NextLocalID)
	}
}
