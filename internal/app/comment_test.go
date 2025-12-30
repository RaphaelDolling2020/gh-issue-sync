package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitsuhiko/gh-issue-sync/internal/issue"
	"github.com/mitsuhiko/gh-issue-sync/internal/paths"
)

func TestFindPendingComment(t *testing.T) {
	dir := t.TempDir()

	// Create a comment file
	commentPath := filepath.Join(dir, "42.comment.md")
	if err := os.WriteFile(commentPath, []byte("Test comment body"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find the comment
	comment, found := findPendingComment(dir, issue.IssueNumber("42"))
	if !found {
		t.Fatal("expected to find pending comment")
	}
	if comment.Body != "Test comment body" {
		t.Errorf("expected body 'Test comment body', got %q", comment.Body)
	}
	if comment.IssueNumber != "42" {
		t.Errorf("expected issue number '42', got %q", comment.IssueNumber)
	}
	if comment.Path != commentPath {
		t.Errorf("expected path %q, got %q", commentPath, comment.Path)
	}

	// Should not find comment for different issue
	_, found = findPendingComment(dir, issue.IssueNumber("99"))
	if found {
		t.Error("should not find comment for issue 99")
	}
}

func TestFindPendingCommentWithSlug(t *testing.T) {
	dir := t.TempDir()

	// Create a comment file with slug (fallback format)
	commentPath := filepath.Join(dir, "42-some-title.comment.md")
	if err := os.WriteFile(commentPath, []byte("Comment with slug"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find the comment
	comment, found := findPendingComment(dir, issue.IssueNumber("42"))
	if !found {
		t.Fatal("expected to find pending comment")
	}
	if comment.Body != "Comment with slug" {
		t.Errorf("expected body 'Comment with slug', got %q", comment.Body)
	}
}

func TestFindPendingCommentPreferred(t *testing.T) {
	dir := t.TempDir()

	// Create both formats - preferred should win
	preferredPath := filepath.Join(dir, "42.comment.md")
	if err := os.WriteFile(preferredPath, []byte("Preferred comment"), 0644); err != nil {
		t.Fatal(err)
	}
	slugPath := filepath.Join(dir, "42-some-title.comment.md")
	if err := os.WriteFile(slugPath, []byte("Slug comment"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find the preferred format
	comment, found := findPendingComment(dir, issue.IssueNumber("42"))
	if !found {
		t.Fatal("expected to find pending comment")
	}
	if comment.Body != "Preferred comment" {
		t.Errorf("expected body 'Preferred comment', got %q", comment.Body)
	}
}

func TestLoadAllPendingComments(t *testing.T) {
	dir := t.TempDir()

	// Create paths structure
	p := paths.New(dir)
	if err := p.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	// Create comment files in open and closed dirs
	if err := os.WriteFile(filepath.Join(p.OpenDir, "1.comment.md"), []byte("Open comment"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.ClosedDir, "2.comment.md"), []byte("Closed comment"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load all comments
	comments := loadAllPendingComments(p)
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}

	if c, ok := comments["1"]; !ok {
		t.Error("expected comment for issue 1")
	} else if c.Body != "Open comment" {
		t.Errorf("expected body 'Open comment', got %q", c.Body)
	}

	if c, ok := comments["2"]; !ok {
		t.Error("expected comment for issue 2")
	} else if c.Body != "Closed comment" {
		t.Errorf("expected body 'Closed comment', got %q", c.Body)
	}
}

func TestLoadAllPendingCommentsSkipsEmpty(t *testing.T) {
	dir := t.TempDir()

	// Create paths structure
	p := paths.New(dir)
	if err := p.EnsureLayout(); err != nil {
		t.Fatal(err)
	}

	// Create empty comment file
	if err := os.WriteFile(filepath.Join(p.OpenDir, "1.comment.md"), []byte("   \n\n  "), 0644); err != nil {
		t.Fatal(err)
	}

	// Load all comments
	comments := loadAllPendingComments(p)
	if len(comments) != 0 {
		t.Errorf("expected 0 comments (empty should be skipped), got %d", len(comments))
	}
}

func TestDeletePendingComment(t *testing.T) {
	dir := t.TempDir()

	// Create a comment file
	commentPath := filepath.Join(dir, "42.comment.md")
	if err := os.WriteFile(commentPath, []byte("Test comment"), 0644); err != nil {
		t.Fatal(err)
	}

	comment := PendingComment{
		IssueNumber: "42",
		Body:        "Test comment",
		Path:        commentPath,
	}

	// Delete the comment
	if err := deletePendingComment(comment); err != nil {
		t.Fatalf("failed to delete comment: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(commentPath); !os.IsNotExist(err) {
		t.Error("expected comment file to be deleted")
	}
}

func TestLocalIssueCommentNumber(t *testing.T) {
	dir := t.TempDir()

	// Create a comment file for local issue
	commentPath := filepath.Join(dir, "Tabc123.comment.md")
	if err := os.WriteFile(commentPath, []byte("Local issue comment"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find the comment
	comment, found := findPendingComment(dir, issue.IssueNumber("Tabc123"))
	if !found {
		t.Fatal("expected to find pending comment for local issue")
	}
	if comment.Body != "Local issue comment" {
		t.Errorf("expected body 'Local issue comment', got %q", comment.Body)
	}
}
