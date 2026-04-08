package main

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type deletedLine struct {
	afterLine int
	content   string
	oldNumber int
}

// collectHunkChanges extracts added line numbers and deleted line info from diff hunks.
func collectHunkChanges(hunks []DiffHunk) (map[int]bool, []deletedLine) {
	addedLines := make(map[int]bool)
	var deletedLines []deletedLine

	for _, hunk := range hunks {
		newLineNum := hunk.NewStart
		oldLineNum := hunk.OldStart

		for _, line := range hunk.Lines {
			if len(line) == 0 {
				continue
			}

			lineContent := ""
			if len(line) > 1 {
				lineContent = line[1:]
			}

			switch line[0] {
			case '+':
				if !strings.HasPrefix(line, "+++") {
					addedLines[newLineNum] = true
					newLineNum++
				}
			case '-':
				if !strings.HasPrefix(line, "---") {
					deletedLines = append(deletedLines, deletedLine{
						afterLine: newLineNum - 1,
						content:   lineContent,
						oldNumber: oldLineNum,
					})
					oldLineNum++
				}
			default:
				newLineNum++
				oldLineNum++
			}
		}
	}
	return addedLines, deletedLines
}

// buildAnnotatedLines creates annotated lines from file content and diff hunks
func buildAnnotatedLines(content string, hunks []DiffHunk) []AnnotatedLine {
	lines := strings.Split(content, "\n")
	annotated := make([]AnnotatedLine, 0, len(lines)*2)

	addedLines, deleted := collectHunkChanges(hunks)

	deletedIdx := 0
	for i, line := range lines {
		lineNum := i + 1

		for deletedIdx < len(deleted) && deleted[deletedIdx].afterLine < lineNum {
			annotated = append(annotated, AnnotatedLine{
				Number:    0,
				Content:   deleted[deletedIdx].content,
				Change:    LineChangeDeleted,
				OldNumber: deleted[deletedIdx].oldNumber,
			})
			deletedIdx++
		}

		change := LineChangeNone
		if addedLines[lineNum] {
			change = LineChangeAdded
		}
		annotated = append(annotated, AnnotatedLine{
			Number:  lineNum,
			Content: line,
			Change:  change,
		})
	}

	for deletedIdx < len(deleted) {
		annotated = append(annotated, AnnotatedLine{
			Number:    0,
			Content:   deleted[deletedIdx].content,
			Change:    LineChangeDeleted,
			OldNumber: deleted[deletedIdx].oldNumber,
		})
		deletedIdx++
	}

	return annotated
}

// parseDiffHeaderLine updates stats based on diff header metadata lines.
// Returns true if the line was a recognized header line.
func parseDiffHeaderLine(line string, stats *DiffStats, filesSet map[string]bool) bool {
	if strings.HasPrefix(line, "diff --git") {
		filesSet[line] = true
		return true
	}
	if strings.HasPrefix(line, "rename from ") {
		stats.IsRename = true
		stats.OldPath = strings.TrimPrefix(line, "rename from ")
		return true
	}
	if strings.HasPrefix(line, "rename to ") {
		return true
	}
	if strings.HasPrefix(line, "new file mode") {
		stats.IsNewFile = true
		return true
	}
	if strings.HasPrefix(line, "deleted file mode") {
		stats.IsDeletedFile = true
		return true
	}
	return false
}

// computeEnhancedStats fills in derived metrics (NetLines, HunkCount, etc.) from parsed hunks.
func computeEnhancedStats(stats *DiffStats, hunks []DiffHunk) {
	stats.NetLines = stats.Additions - stats.Deletions
	stats.HunkCount = len(hunks)
	for _, h := range hunks {
		if n := countHunkChangedLines(h); n > stats.LargestHunk {
			stats.LargestHunk = n
		}
	}
	if total := stats.Additions + stats.Deletions; total > 0 && stats.HunkCount > 0 {
		stats.Density = float64(stats.HunkCount) / float64(total)
	}
}

var hunkHeaderRegex = regexp.MustCompile(`^@@\s*-(\d+)(?:,(\d+))?\s*\+(\d+)(?:,(\d+))?\s*@@(.*)$`)

// parseHunkHeader creates a DiffHunk from regex match groups.
func parseHunkHeader(matches []string) *DiffHunk {
	return &DiffHunk{
		OldStart: atoi(matches[1]),
		OldCount: atoiWithDefault(matches[2], 1),
		NewStart: atoi(matches[3]),
		NewCount: atoiWithDefault(matches[4], 1),
		Header:   strings.TrimSpace(matches[5]),
		Lines:    []string{},
	}
}

// countDiffLineStat increments addition or deletion stats for a single diff line.
func countDiffLineStat(line string, stats *DiffStats) {
	if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
		stats.Additions++
	} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
		stats.Deletions++
	}
}

// ParseDiffOutput parses raw git diff output into structured form
func ParseDiffOutput(raw string) *DiffResponse {
	resp := &DiffResponse{
		Raw:   raw,
		Hunks: []DiffHunk{},
		Stats: DiffStats{},
	}

	if strings.TrimSpace(raw) == "" {
		resp.HasDiff = false
		return resp
	}

	resp.HasDiff = true
	lines := strings.Split(raw, "\n")
	filesSet := make(map[string]bool)

	var currentHunk *DiffHunk

	for _, line := range lines {
		if parseDiffHeaderLine(line, &resp.Stats, filesSet) {
			continue
		}

		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				resp.Hunks = append(resp.Hunks, *currentHunk)
			}
			currentHunk = parseHunkHeader(matches)
			continue
		}

		if currentHunk != nil {
			currentHunk.Lines = append(currentHunk.Lines, line)
		}
		countDiffLineStat(line, &resp.Stats)
	}

	if currentHunk != nil {
		resp.Hunks = append(resp.Hunks, *currentHunk)
	}

	resp.Stats.Files = len(filesSet)
	computeEnhancedStats(&resp.Stats, resp.Hunks)

	return resp
}

// countHunkChangedLines counts added/deleted lines in a single hunk.
func countHunkChangedLines(h DiffHunk) int {
	count := 0
	for _, line := range h.Lines {
		if len(line) > 0 && (line[0] == '+' || line[0] == '-') {
			if !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
				count++
			}
		}
	}
	return count
}

// isCommentLine checks if a trimmed line (without +/- prefix) is a comment
// based on the file extension.
func isCommentLine(trimmedContent string, ext string) bool {
	if trimmedContent == "" {
		return false
	}
	ext = strings.ToLower(ext)
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".java", ".c", ".cpp", ".rs", ".swift":
		return strings.HasPrefix(trimmedContent, "//") ||
			strings.HasPrefix(trimmedContent, "/*") ||
			strings.HasPrefix(trimmedContent, "*/") ||
			strings.HasPrefix(trimmedContent, "*")
	case ".py", ".rb", ".sh", ".bash", ".yaml", ".yml", ".toml":
		return strings.HasPrefix(trimmedContent, "#")
	case ".html", ".xml", ".svg":
		return strings.HasPrefix(trimmedContent, "<!--")
	}
	return false
}

// classifyDiffLine checks if a diff line is a comment addition or deletion.
// Returns +1 for comment addition, -1 for comment deletion, 0 otherwise.
func classifyDiffLine(line, ext string) int {
	if len(line) == 0 {
		return 0
	}
	prefix := line[0]
	if prefix == '+' && !strings.HasPrefix(line, "+++") {
		if isCommentLine(strings.TrimSpace(line[1:]), ext) {
			return 1
		}
	} else if prefix == '-' && !strings.HasPrefix(line, "---") {
		if isCommentLine(strings.TrimSpace(line[1:]), ext) {
			return -1
		}
	}
	return 0
}

// enrichCommentStats counts comment additions/deletions in the parsed diff.
func enrichCommentStats(resp *DiffResponse, path string) {
	if path == "" || resp == nil {
		return
	}
	ext := filepath.Ext(path)
	if ext == "" {
		return
	}
	for _, hunk := range resp.Hunks {
		for _, line := range hunk.Lines {
			switch classifyDiffLine(line, ext) {
			case 1:
				resp.Stats.CommentAdditions++
			case -1:
				resp.Stats.CommentDeletions++
			}
		}
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atoiWithDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
