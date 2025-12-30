package search

import (
	"testing"

	"github.com/mitsuhiko/gh-issue-sync/internal/issue"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  Query
	}{
		{
			name:  "empty query",
			query: "",
			want:  Query{SortField: "created", SortAsc: false},
		},
		{
			name:  "free text only",
			query: "error message",
			want:  Query{Text: "error message", SortField: "created", SortAsc: false},
		},
		{
			name:  "is:open",
			query: "is:open",
			want:  Query{State: "open", SortField: "created", SortAsc: false},
		},
		{
			name:  "is:closed",
			query: "is:closed",
			want:  Query{State: "closed", SortField: "created", SortAsc: false},
		},
		{
			name:  "label filter",
			query: "label:bug",
			want:  Query{Labels: []string{"bug"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "multiple labels",
			query: "label:bug label:urgent",
			want:  Query{Labels: []string{"bug", "urgent"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "no:assignee",
			query: "no:assignee",
			want:  Query{NoAssignee: true, SortField: "created", SortAsc: false},
		},
		{
			name:  "no:label",
			query: "no:label",
			want:  Query{NoLabel: true, SortField: "created", SortAsc: false},
		},
		{
			name:  "assignee filter",
			query: "assignee:alice",
			want:  Query{Assignees: []string{"alice"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "author filter",
			query: "author:bob",
			want:  Query{Authors: []string{"bob"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "milestone filter",
			query: "milestone:v1.0",
			want:  Query{Milestones: []string{"v1.0"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "sort created-asc",
			query: "sort:created-asc",
			want:  Query{SortField: "created", SortAsc: true},
		},
		{
			name:  "sort created-desc",
			query: "sort:created-desc",
			want:  Query{SortField: "created", SortAsc: false},
		},
		{
			name:  "sort updated-asc",
			query: "sort:updated-asc",
			want:  Query{SortField: "updated", SortAsc: true},
		},
		{
			name:  "complex query",
			query: "error no:assignee sort:created-asc",
			want:  Query{Text: "error", NoAssignee: true, SortField: "created", SortAsc: true},
		},
		{
			name:  "quoted value",
			query: `label:"help wanted"`,
			want:  Query{Labels: []string{"help wanted"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "type filter",
			query: "type:Bug",
			want:  Query{Types: []string{"Bug"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "project filter",
			query: "project:roadmap",
			want:  Query{Projects: []string{"roadmap"}, SortField: "created", SortAsc: false},
		},
		{
			name:  "mentions filter",
			query: "mentions:alice",
			want:  Query{Mentions: []string{"alice"}, SortField: "created", SortAsc: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.query)
			if got.Text != tt.want.Text {
				t.Errorf("Text = %q, want %q", got.Text, tt.want.Text)
			}
			if got.State != tt.want.State {
				t.Errorf("State = %q, want %q", got.State, tt.want.State)
			}
			if !slicesEqual(got.Labels, tt.want.Labels) {
				t.Errorf("Labels = %v, want %v", got.Labels, tt.want.Labels)
			}
			if got.NoLabel != tt.want.NoLabel {
				t.Errorf("NoLabel = %v, want %v", got.NoLabel, tt.want.NoLabel)
			}
			if !slicesEqual(got.Assignees, tt.want.Assignees) {
				t.Errorf("Assignees = %v, want %v", got.Assignees, tt.want.Assignees)
			}
			if got.NoAssignee != tt.want.NoAssignee {
				t.Errorf("NoAssignee = %v, want %v", got.NoAssignee, tt.want.NoAssignee)
			}
			if !slicesEqual(got.Authors, tt.want.Authors) {
				t.Errorf("Authors = %v, want %v", got.Authors, tt.want.Authors)
			}
			if !slicesEqual(got.Milestones, tt.want.Milestones) {
				t.Errorf("Milestones = %v, want %v", got.Milestones, tt.want.Milestones)
			}
			if !slicesEqual(got.Types, tt.want.Types) {
				t.Errorf("Types = %v, want %v", got.Types, tt.want.Types)
			}
			if !slicesEqual(got.Projects, tt.want.Projects) {
				t.Errorf("Projects = %v, want %v", got.Projects, tt.want.Projects)
			}
			if !slicesEqual(got.Mentions, tt.want.Mentions) {
				t.Errorf("Mentions = %v, want %v", got.Mentions, tt.want.Mentions)
			}
			if got.SortField != tt.want.SortField {
				t.Errorf("SortField = %q, want %q", got.SortField, tt.want.SortField)
			}
			if got.SortAsc != tt.want.SortAsc {
				t.Errorf("SortAsc = %v, want %v", got.SortAsc, tt.want.SortAsc)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name  string
		query string
		issue IssueData
		want  bool
	}{
		{
			name:  "empty query matches all",
			query: "",
			issue: IssueData{Title: "Test", State: "open"},
			want:  true,
		},
		{
			name:  "text search in title",
			query: "error",
			issue: IssueData{Title: "Error handling", State: "open"},
			want:  true,
		},
		{
			name:  "text search case insensitive",
			query: "ERROR",
			issue: IssueData{Title: "error handling", State: "open"},
			want:  true,
		},
		{
			name:  "text search in body",
			query: "fix",
			issue: IssueData{Title: "Bug", Body: "Need to fix this", State: "open"},
			want:  true,
		},
		{
			name:  "text search no match",
			query: "missing",
			issue: IssueData{Title: "Bug", Body: "Something", State: "open"},
			want:  false,
		},
		{
			name:  "state filter match",
			query: "is:open",
			issue: IssueData{Title: "Test", State: "open"},
			want:  true,
		},
		{
			name:  "state filter no match",
			query: "is:closed",
			issue: IssueData{Title: "Test", State: "open"},
			want:  false,
		},
		{
			name:  "label filter match",
			query: "label:bug",
			issue: IssueData{Title: "Test", State: "open", Labels: []string{"bug", "urgent"}},
			want:  true,
		},
		{
			name:  "label filter case insensitive",
			query: "label:BUG",
			issue: IssueData{Title: "Test", State: "open", Labels: []string{"bug"}},
			want:  true,
		},
		{
			name:  "label filter no match",
			query: "label:bug",
			issue: IssueData{Title: "Test", State: "open", Labels: []string{"feature"}},
			want:  false,
		},
		{
			name:  "multiple labels all must match",
			query: "label:bug label:urgent",
			issue: IssueData{Title: "Test", State: "open", Labels: []string{"bug"}},
			want:  false,
		},
		{
			name:  "multiple labels all match",
			query: "label:bug label:urgent",
			issue: IssueData{Title: "Test", State: "open", Labels: []string{"bug", "urgent"}},
			want:  true,
		},
		{
			name:  "no:assignee with no assignees",
			query: "no:assignee",
			issue: IssueData{Title: "Test", State: "open", Assignees: nil},
			want:  true,
		},
		{
			name:  "no:assignee with assignees",
			query: "no:assignee",
			issue: IssueData{Title: "Test", State: "open", Assignees: []string{"alice"}},
			want:  false,
		},
		{
			name:  "no:label with no labels",
			query: "no:label",
			issue: IssueData{Title: "Test", State: "open", Labels: nil},
			want:  true,
		},
		{
			name:  "no:label with labels",
			query: "no:label",
			issue: IssueData{Title: "Test", State: "open", Labels: []string{"bug"}},
			want:  false,
		},
		{
			name:  "assignee filter match",
			query: "assignee:alice",
			issue: IssueData{Title: "Test", State: "open", Assignees: []string{"alice", "bob"}},
			want:  true,
		},
		{
			name:  "assignee filter no match",
			query: "assignee:charlie",
			issue: IssueData{Title: "Test", State: "open", Assignees: []string{"alice"}},
			want:  false,
		},
		{
			name:  "author filter match",
			query: "author:alice",
			issue: IssueData{Title: "Test", State: "open", Author: "alice"},
			want:  true,
		},
		{
			name:  "author filter case insensitive",
			query: "author:ALICE",
			issue: IssueData{Title: "Test", State: "open", Author: "alice"},
			want:  true,
		},
		{
			name:  "milestone filter match",
			query: "milestone:v1.0",
			issue: IssueData{Title: "Test", State: "open", Milestone: "v1.0"},
			want:  true,
		},
		{
			name:  "no:milestone with no milestone",
			query: "no:milestone",
			issue: IssueData{Title: "Test", State: "open", Milestone: ""},
			want:  true,
		},
		{
			name:  "no:milestone with milestone",
			query: "no:milestone",
			issue: IssueData{Title: "Test", State: "open", Milestone: "v1.0"},
			want:  false,
		},
		{
			name:  "mentions filter match",
			query: "mentions:alice",
			issue: IssueData{Title: "Test", State: "open", Body: "cc @alice for review"},
			want:  true,
		},
		{
			name:  "mentions filter no match",
			query: "mentions:bob",
			issue: IssueData{Title: "Test", State: "open", Body: "cc @alice for review"},
			want:  false,
		},
		{
			name:  "type filter match",
			query: "type:Bug",
			issue: IssueData{Title: "Test", State: "open", IssueType: "Bug"},
			want:  true,
		},
		{
			name:  "project filter match",
			query: "project:roadmap",
			issue: IssueData{Title: "Test", State: "open", Projects: []string{"roadmap"}},
			want:  true,
		},
		{
			name:  "complex query match",
			query: "error is:open label:bug no:assignee",
			issue: IssueData{Title: "Error handling", State: "open", Labels: []string{"bug"}, Assignees: nil},
			want:  true,
		},
		{
			name:  "complex query partial no match",
			query: "error is:open label:bug no:assignee",
			issue: IssueData{Title: "Error handling", State: "open", Labels: []string{"bug"}, Assignees: []string{"alice"}},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := Parse(tt.query)
			got := q.Match(tt.issue)
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSort(t *testing.T) {
	ts1 := int64(1000)
	ts2 := int64(2000)
	ts3 := int64(3000)

	issues := []IssueData{
		{Number: issue.IssueNumber("2"), Title: "Second", CreatedAt: &ts2},
		{Number: issue.IssueNumber("1"), Title: "First", CreatedAt: &ts1},
		{Number: issue.IssueNumber("3"), Title: "Third", CreatedAt: &ts3},
		{Number: issue.IssueNumber("T1"), Title: "Local", CreatedAt: nil},
	}

	t.Run("sort created-desc (default)", func(t *testing.T) {
		q := Parse("")
		sorted := make([]IssueData, len(issues))
		copy(sorted, issues)
		q.Sort(sorted)

		// Should be: 3 (newest), 2, 1, T1 (local/nil at end)
		if sorted[0].Number != "3" || sorted[1].Number != "2" || sorted[2].Number != "1" || sorted[3].Number != "T1" {
			t.Errorf("unexpected order: %v %v %v %v", sorted[0].Number, sorted[1].Number, sorted[2].Number, sorted[3].Number)
		}
	})

	t.Run("sort created-asc", func(t *testing.T) {
		q := Parse("sort:created-asc")
		sorted := make([]IssueData, len(issues))
		copy(sorted, issues)
		q.Sort(sorted)

		// Should be: T1 (nil first when asc), 1, 2, 3
		// Wait, nil should still go after synced issues
		if sorted[0].Number != "1" || sorted[1].Number != "2" || sorted[2].Number != "3" || sorted[3].Number != "T1" {
			t.Errorf("unexpected order: %v %v %v %v", sorted[0].Number, sorted[1].Number, sorted[2].Number, sorted[3].Number)
		}
	})

	t.Run("sort updated-desc", func(t *testing.T) {
		// Create issues with different created/updated times
		u1 := int64(3000) // issue 1 updated most recently
		u2 := int64(1000)
		u3 := int64(2000)
		issuesWithUpdated := []IssueData{
			{Number: issue.IssueNumber("2"), Title: "Second", CreatedAt: &ts2, UpdatedAt: &u2},
			{Number: issue.IssueNumber("1"), Title: "First", CreatedAt: &ts1, UpdatedAt: &u1},
			{Number: issue.IssueNumber("3"), Title: "Third", CreatedAt: &ts3, UpdatedAt: &u3},
			{Number: issue.IssueNumber("T1"), Title: "Local", CreatedAt: nil, UpdatedAt: nil},
		}

		q := Parse("sort:updated-desc")
		sorted := make([]IssueData, len(issuesWithUpdated))
		copy(sorted, issuesWithUpdated)
		q.Sort(sorted)

		// Should be: 1 (updated 3000), 3 (updated 2000), 2 (updated 1000), T1 (nil at end)
		if sorted[0].Number != "1" || sorted[1].Number != "3" || sorted[2].Number != "2" || sorted[3].Number != "T1" {
			t.Errorf("unexpected order: %v %v %v %v", sorted[0].Number, sorted[1].Number, sorted[2].Number, sorted[3].Number)
		}
	})
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
