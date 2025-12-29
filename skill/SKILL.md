---
name: gh-issue-sync
description: "Manage GitHub issues locally as Markdown files. Use for triaging, searching, editing, and creating issues without leaving your editor or terminal."
---

# gh-issue-sync Skill

This skill teaches you to use `gh-issue-sync` for managing GitHub issues as local Markdown files.

## Setup

Initialize in a git repository:
```bash
gh-issue-sync init
```

Or specify the repo explicitly:
```bash
gh-issue-sync init --owner acme --repo roadmap
```

## Syncing Issues

Pull all open issues from GitHub:
```bash
gh-issue-sync pull
```

Pull all issues including closed:
```bash
gh-issue-sync pull --all
```

Pull specific issues by number:
```bash
gh-issue-sync pull 42 123
```

Pull issues with specific labels:
```bash
gh-issue-sync pull --label bug --label urgent
```

Push local changes to GitHub:
```bash
gh-issue-sync push
```

Preview what would be pushed:
```bash
gh-issue-sync push --dry-run
```

## File Structure

Issues are stored in `.issues/`:
```
.issues/
├── open/           # Open issues
│   ├── 1-fix-login-bug.md
│   └── 2-add-search-feature.md
├── closed/         # Closed issues
│   └── 3-update-docs.md
└── .sync/          # Internal sync state (don't edit)
```

## Issue File Format

Each issue is a Markdown file with YAML frontmatter:

```markdown
---
number: 42
title: Fix login bug
labels:
    - bug
    - priority:high
assignees:
    - alice
milestone: v1.0
state: open
---

Issue body goes here.

Use standard Markdown formatting.
```

### Frontmatter Fields

| Field | Type | Description |
|-------|------|-------------|
| `number` | string | Issue number (read-only for pushed issues) |
| `title` | string | Issue title (required) |
| `labels` | list | Labels as strings |
| `assignees` | list | GitHub usernames |
| `milestone` | string | Milestone name |
| `state` | string | `open` or `closed` |
| `state_reason` | string | `completed` or `not_planned` (for closed) |
| `parent` | string | Parent issue reference (e.g., `42` or `owner/repo#42`) |
| `blocked_by` | list | Issues blocking this one |
| `blocks` | list | Issues this one blocks |

## Creating Issues

Create a new issue with a title:
```bash
gh-issue-sync new "Add dark mode support"
```

Create with labels:
```bash
gh-issue-sync new "Fix bug" --label bug --label urgent
```

Create interactively in editor:
```bash
gh-issue-sync new --edit
```

New issues get temporary `T` prefixed IDs (e.g., `T1a2b3c`) until pushed.

## Editing Issues

Open an issue in your editor:
```bash
gh-issue-sync edit 42
```

Or edit the file directly - it's just Markdown.

## State Changes

Close an issue:
```bash
gh-issue-sync close 42
```

Close with a reason:
```bash
gh-issue-sync close 42 --reason completed
gh-issue-sync close 42 --reason not_planned
```

Reopen an issue:
```bash
gh-issue-sync reopen 42
```

## Checking Status

Show local changes not yet pushed:
```bash
gh-issue-sync status
```

Show diff for a specific issue:
```bash
gh-issue-sync diff 42
```

Diff against remote (re-fetches from GitHub):
```bash
gh-issue-sync diff 42 --remote
```

## Searching and Triaging

Issues are plain text files, so use standard tools:

Find issues mentioning "authentication":
```bash
grep -r "authentication" .issues/open/
```

List all high-priority bugs:
```bash
grep -l "priority:high" .issues/open/*.md | xargs grep -l "bug"
```

Find issues assigned to someone:
```bash
grep -r "assignees:" .issues/open/ -A 1 | grep "alice"
```

Count open issues by label:
```bash
grep -h "^    - " .issues/open/*.md | sort | uniq -c | sort -rn
```

## Workflow Tips

### Triage Workflow
1. `gh-issue-sync pull` - Get latest issues
2. Review files in `.issues/open/`
3. Add labels, assignees, milestones as needed
4. `gh-issue-sync push` - Sync changes to GitHub

### Creating Related Issues
Reference other issues in the body using `#123` syntax. For local issues not yet pushed, use `#T...` references - they'll be updated automatically when pushed.

### Bulk Operations
Since issues are files, use shell tools:
```bash
# Add a label to all issues mentioning "database"
for f in $(grep -l "database" .issues/open/*.md); do
    # Edit the file to add label
done
```

### Conflict Handling
If GitHub has newer changes than your local copy:
- Pull skips that issue (shows as conflict)
- Use `gh-issue-sync pull --force` to overwrite local changes
- Or manually resolve and push

## Quick Reference

| Command | Description |
|---------|-------------|
| `init` | Initialize issue sync in current repo |
| `pull` | Fetch issues from GitHub |
| `push` | Push local changes to GitHub |
| `status` | Show local modifications |
| `new` | Create a new local issue |
| `edit` | Open issue in editor |
| `close` | Mark issue as closed |
| `reopen` | Reopen a closed issue |
| `diff` | Show changes vs original/remote |
