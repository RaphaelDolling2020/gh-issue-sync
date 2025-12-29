# gh-issue-sync

Sync GitHub issues to local Markdown files for offline editing, batch updates,
and seamless integration with coding agents.

The idea here is that you can bring in the state of a bunch of GitHub issues
locally so that you can refine them with an agent until you're satisfied and
sync back up the changes.  It can also be useful if you are working with an
internet for one reason or another and you want to have your issues readable
locally.

## Overview

`gh-issue-sync` mirrors GitHub issues into a local `.issues/` directory as
Markdown files with YAML front matter.  Edit issues in your favorite editor,
create new issues locally, and push changes back to GitHub when ready.

**Key features:**

- **Bidirectional sync** – Pull issues from GitHub, push local changes back
- **Local issue creation** – Create issues locally with temporary IDs (T1, T2), promote to real GitHub issues on push
- **Conflict detection** – Three-way merge logic prevents accidental overwrites
- **Folder-based state** – Move files between `open/` and `closed/` to change issue state
- **Leverages `gh` CLI** – Uses your existing GitHub CLI authentication

## Installation

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated (`gh auth login`)

### Install with Go

```bash
go install github.com/mitsuhiko/gh-issue-sync/cmd/gh-issue-sync@latest
```

This installs the binary to `$GOBIN` (or `$GOPATH/bin`). Make sure it's in your `PATH`.

### Build from source

```bash
git clone https://github.com/mitsuhiko/gh-issue-sync.git
cd gh-issue-sync
go build -o gh-issue-sync ./cmd/gh-issue-sync
```

Optionally move the binary to your PATH:

```bash
mv gh-issue-sync ~/.local/bin/
# or
sudo mv gh-issue-sync /usr/local/bin/
```

## Quickstart

```bash
# Navigate to your project
cd my-project

# Initialize issue sync (auto-detects repo from git remote)
gh-issue-sync init

# Pull all open issues from GitHub
gh-issue-sync pull

# View your local issues
ls .issues/open/

# Edit an issue
$EDITOR .issues/open/123-fix-login-bug.md
gh-issue-sync edit 123

# Push your changes
gh-issue-sync push
```

## Creating Local Issues

Since the issue numbers come from github you can use temporary issue numbers
until then.  `T42` or `TABC` are all valid temporary issue IDs.  But they need
to start with a "T" so that we know they are temporary ones.  After synching
they are given new numbers and all references are updated.

## Sync Behavior

- New issues are created in `open/` or `closed/` based on their state
- Existing issues are updated if unchanged locally
- Local changes are preserved (conflicts are reported, not overwritten)
- Original versions are stored in `.sync/originals/` for conflict detection
- Local-only issues (T1, T2, etc.) are created on GitHub
- After creation, files are renamed with real issue numbers
- References like `#T1` in other issues are automatically updated

### Check Status

See what's changed locally:

```bash
gh-issue-sync status
```

### Create New Issues

Create issues locally before pushing to GitHub:

```bash
# Create with a title
gh-issue-sync new "My new feature idea"

# Create and open in editor
gh-issue-sync new "Fix login bug" --edit

# Create with labels
gh-issue-sync new "Critical bug" --label bug --label urgent

# Create with just the editor (no title required)
gh-issue-sync new --edit
```

Local issues get temporary IDs like `T1`, `T2`. When pushed, they become real
GitHub issues and files are renamed automatically.

### Close and Reopen Issues

```bash
# Close an issue (marks for closing on next push)
gh-issue-sync close 123

# Close with a reason
gh-issue-sync close 123 --reason not_planned

# Reopen a closed issue
gh-issue-sync reopen 456
```

Alternatively, move files manually:
- Move from `open/` to `closed/` to close
- Move from `closed/` to `open/` to reopen

## Issue File Format

Each issue is a Markdown file with YAML front matter:

```markdown
---
number: 123
title: Fix login bug on mobile Safari
labels:
  - bug
  - ios
assignees:
  - alice
  - bob
milestone: v2.0
state: open
state_reason:
synced_at: 2025-12-29T17:00:00Z
---

The login button doesn't respond to taps on iOS Safari 17.x.

## Steps to Reproduce

1. Open the app on iOS Safari
2. Tap the login button
3. Nothing happens

## Expected Behavior

Should open the login modal.
```

### Front Matter Fields

| Field | Type | Description | Editable |
|-------|------|-------------|----------|
| `number` | int/string | Issue number or local ID (T1, T2) | No (managed) |
| `title` | string | Issue title | Yes |
| `labels` | string[] | Label names | Yes |
| `assignees` | string[] | GitHub usernames | Yes |
| `milestone` | string | Milestone name | Yes |
| `state` | string | `open` or `closed` | Via folder |
| `state_reason` | string | `completed` or `not_planned` | Yes |
| `parent` | int | Parent issue number | Yes |
| `blocked_by` | int[] | Blocking issue numbers | Yes |
| `blocks` | int[] | Issues this blocks | Yes |
| `synced_at` | datetime | Last sync time | No (managed) |

### File Naming

Files are named `{number}-{slug}.md` where slug is derived from the title:
- `123-fix-login-bug.md`
- `T1-new-feature.md`

The slug is for readability only—the tool identifies issues by the number prefix.

## Conflict Handling

`gh-issue-sync` uses three-way comparison to detect conflicts:

| Local | Original | Remote | Action |
|-------|----------|--------|--------|
| Same | Same | Same | No action |
| Changed | Same | Same | Push local changes |
| Same | Same | Changed | Pull remote changes |
| Changed | Same | Changed | **Conflict** – skip with warning |

When a conflict occurs:
- On **pull**: Local changes are preserved, remote update is skipped
- On **push**: Remote changes are detected, local push is skipped
- Use `--force` on pull to overwrite local changes

## Agent Skill

`gh-issue-sync` includes a skill file for coding agents. Install it with:

```bash
gh-issue-sync write-skill
```

This writes a `SKILL.md` to `~/.codex/skills/gh-issue-sync/` that teaches agents
how to use the tool for triaging, searching, and managing issues.

To install to a custom location:

```bash
gh-issue-sync write-skill --output /path/to/skills/
```

## License

MIT
