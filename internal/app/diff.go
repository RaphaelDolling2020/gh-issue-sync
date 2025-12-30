package app

import (
	"fmt"
	"strings"
)

func (a *App) printUnifiedDiff(oldText, newText, oldLabel, newLabel string) {
	t := a.Theme

	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	// Simple line-by-line diff using LCS
	ops := computeDiff(oldLines, newLines)

	fmt.Fprintf(a.Out, "%s\n", t.MutedText(fmt.Sprintf("--- %s", oldLabel)))
	fmt.Fprintf(a.Out, "%s\n", t.MutedText(fmt.Sprintf("+++ %s", newLabel)))

	for _, op := range ops {
		switch op.Type {
		case diffEqual:
			fmt.Fprintf(a.Out, " %s\n", op.Text)
		case diffDelete:
			fmt.Fprintf(a.Out, "%s%s\n", t.Fg(t.Removed, "-"), t.Fg(t.OldValue, op.Text))
		case diffInsert:
			fmt.Fprintf(a.Out, "%s%s\n", t.Fg(t.Added, "+"), t.Fg(t.NewValue, op.Text))
		}
	}
}

type diffOpType int

const (
	diffEqual diffOpType = iota
	diffDelete
	diffInsert
)

type diffOp struct {
	Type diffOpType
	Text string
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.TrimSuffix(text, "\n")
	return strings.Split(text, "\n")
}

// computeDiff computes a simple line-based diff using the LCS algorithm
func computeDiff(oldLines, newLines []string) []diffOp {
	// Build LCS table
	m, n := len(oldLines), len(newLines)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else if lcs[i-1][j] >= lcs[i][j-1] {
				lcs[i][j] = lcs[i-1][j]
			} else {
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	// Backtrack to build diff
	var ops []diffOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, diffOp{Type: diffEqual, Text: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			ops = append(ops, diffOp{Type: diffInsert, Text: newLines[j-1]})
			j--
		} else {
			ops = append(ops, diffOp{Type: diffDelete, Text: oldLines[i-1]})
			i--
		}
	}

	// Reverse to get correct order
	for left, right := 0, len(ops)-1; left < right; left, right = left+1, right-1 {
		ops[left], ops[right] = ops[right], ops[left]
	}

	return ops
}
