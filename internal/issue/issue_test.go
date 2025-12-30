package issue

import (
	"strings"
	"testing"
	"time"
)

func TestParseRenderRoundTrip(t *testing.T) {
	syncedAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	// Note: number is derived from filename, not frontmatter
	input := strings.TrimSpace(`---
title: "Test issue"
labels:
  - bug
assignees:
  - alice
milestone: "v1"
state: open
state_reason: null
blocked_by:
  - 2
synced_at: 2025-01-02T03:04:05Z
---
Body line
`) + "\n"

	parsed, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	// Number is empty when using Parse directly (use ParseFile to get number from path)
	if parsed.Number != "" {
		t.Fatalf("expected empty number from Parse, got %s", parsed.Number)
	}
	if parsed.Title != "Test issue" {
		t.Fatalf("expected title, got %q", parsed.Title)
	}
	if parsed.SyncedAt == nil || !parsed.SyncedAt.Equal(syncedAt) {
		t.Fatalf("unexpected synced_at")
	}

	rendered, err := Render(parsed)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	parsedAgain, err := Parse([]byte(rendered))
	if err != nil {
		t.Fatalf("parse rendered failed: %v", err)
	}
	if !EqualIgnoringSyncedAt(parsed, parsedAgain) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestParseFileExtractsNumber(t *testing.T) {
	// Mock file read
	oldReadFile := osReadFile
	defer func() { osReadFile = oldReadFile }()

	osReadFile = func(path string) ([]byte, error) {
		return []byte(`---
title: Test
state: open
---
Body
`), nil
	}

	issue, err := ParseFile("/tmp/.issues/open/42-test-issue.md")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if issue.Number != "42" {
		t.Fatalf("expected number 42, got %s", issue.Number)
	}

	issue, err = ParseFile("/tmp/.issues/open/T5-new-issue.md")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if issue.Number != "T5" {
		t.Fatalf("expected number T5, got %s", issue.Number)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Fix login bug":          "fix-login-bug",
		"  Weird---Title  ":      "weird-title",
		"Symbols & stuff!":       "symbols-stuff",
		"":                       "",
		"Multiple     spaces":    "multiple-spaces",
		"Already-slugified-text": "already-slugified-text",
	}
	for input, expected := range cases {
		if got := Slugify(input); got != expected {
			t.Fatalf("slugify %q => %q, want %q", input, got, expected)
		}
	}
}

func TestInfoSectionRoundTrip(t *testing.T) {
	input := strings.TrimSpace(`---
title: "Test issue with author"
state: open
info:
    author: testuser
---
Body
`) + "\n"

	parsed, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.Author != "testuser" {
		t.Fatalf("expected author 'testuser', got %q", parsed.Author)
	}

	rendered, err := Render(parsed)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(rendered, "info:") {
		t.Fatalf("rendered should contain info section: %s", rendered)
	}
	if !strings.Contains(rendered, "author: testuser") {
		t.Fatalf("rendered should contain author: %s", rendered)
	}

	parsedAgain, err := Parse([]byte(rendered))
	if err != nil {
		t.Fatalf("parse rendered failed: %v", err)
	}
	if parsedAgain.Author != "testuser" {
		t.Fatalf("expected author 'testuser' after round-trip, got %q", parsedAgain.Author)
	}
}

func TestInfoSectionOmittedWhenEmpty(t *testing.T) {
	iss := Issue{
		Title: "No author",
		State: "open",
	}
	rendered, err := Render(iss)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(rendered, "info:") {
		t.Fatalf("rendered should not contain info section when empty: %s", rendered)
	}
}
