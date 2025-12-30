package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mitsuhiko/gh-issue-sync/internal/issue"
	"github.com/mitsuhiko/gh-issue-sync/internal/paths"
)

// commentFilePattern matches comment files like "42.comment.md" or "42-slug.comment.md"
var commentFilePattern = regexp.MustCompile(`^(\d+|T[a-zA-Z0-9]+)(?:-[^.]+)?\.comment\.md$`)

// PendingComment represents a pending comment for an issue
type PendingComment struct {
	IssueNumber issue.IssueNumber
	Body        string
	Path        string
}

// findPendingComment looks for a pending comment file for the given issue number in the given directory.
// It checks for both "NUMBER.comment.md" (preferred) and "NUMBER-*.comment.md" patterns.
func findPendingComment(dir string, number issue.IssueNumber) (PendingComment, bool) {
	numStr := number.String()

	// First try the preferred format: NUMBER.comment.md
	preferredPath := filepath.Join(dir, numStr+".comment.md")
	if content, err := os.ReadFile(preferredPath); err == nil {
		return PendingComment{
			IssueNumber: number,
			Body:        strings.TrimSpace(string(content)),
			Path:        preferredPath,
		}, true
	}

	// Fall back to glob pattern: NUMBER-*.comment.md
	pattern := filepath.Join(dir, numStr+"-*.comment.md")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return PendingComment{}, false
	}

	// Use the first match
	content, err := os.ReadFile(matches[0])
	if err != nil {
		return PendingComment{}, false
	}

	return PendingComment{
		IssueNumber: number,
		Body:        strings.TrimSpace(string(content)),
		Path:        matches[0],
	}, true
}

// findPendingCommentForIssue finds a pending comment for an issue, checking both open and closed directories.
func findPendingCommentForIssue(p paths.Paths, number issue.IssueNumber, state string) (PendingComment, bool) {
	// Check the directory matching the issue's state first
	dir := p.OpenDir
	if state == "closed" {
		dir = p.ClosedDir
	}

	if comment, found := findPendingComment(dir, number); found {
		return comment, true
	}

	// Also check the other directory in case state changed
	otherDir := p.ClosedDir
	if state == "closed" {
		otherDir = p.OpenDir
	}

	return findPendingComment(otherDir, number)
}

// loadAllPendingComments scans both open and closed directories for pending comment files.
func loadAllPendingComments(p paths.Paths) map[string]PendingComment {
	comments := make(map[string]PendingComment)

	for _, dir := range []string{p.OpenDir, p.ClosedDir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasSuffix(name, ".comment.md") {
				continue
			}

			matches := commentFilePattern.FindStringSubmatch(name)
			if matches == nil {
				continue
			}

			number := issue.IssueNumber(matches[1])
			path := filepath.Join(dir, name)

			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			body := strings.TrimSpace(string(content))
			if body == "" {
				continue
			}

			// Don't overwrite if we already found one (first one wins)
			if _, exists := comments[number.String()]; !exists {
				comments[number.String()] = PendingComment{
					IssueNumber: number,
					Body:        body,
					Path:        path,
				}
			}
		}
	}

	return comments
}

// deletePendingComment removes the pending comment file.
func deletePendingComment(comment PendingComment) error {
	return os.Remove(comment.Path)
}
