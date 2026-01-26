package docschema

import (
	"errors"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	problemsHeading = regexp.MustCompile(`^##\s+(\d{4}-\d{2}-\d{2})\b`)
	progressRow     = regexp.MustCompile(`^\|\s*(\d{4}-\d{2}-\d{2})\s*\|`)
)

// ResetConfig defines how to clean a known-structure document.
type ResetConfig struct {
	DocType        DocType
	MaxAgeDays     int  // Remove entries older than this
	KeepMinEntries int  // Always keep at least this many entries
	PreviewMode    bool // If true, return what would be removed without changing
}

// ResetResult contains the result of a reset operation.
type ResetResult struct {
	RemovedCount   int
	KeptCount      int
	RemovedEntries []string // Preview of what was/would be removed
	NewContent     string   // The cleaned content (or preview)
}

// ResetDocument cleans old entries from a known-structure document.
func ResetDocument(filePath string, config ResetConfig) (*ResetResult, error) {
	return resetDocumentWithClock(filePath, config, time.Now())
}

func resetDocumentWithClock(filePath string, config ResetConfig, now time.Time) (*ResetResult, error) {
	if config.DocType != DocTypeProblems && config.DocType != DocTypeProgress {
		return nil, errors.New("reset is only supported for problems and progress documents")
	}
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(contentBytes)
	trailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(content, "\n")

	var result *ResetResult
	switch config.DocType {
	case DocTypeProblems:
		result, err = resetProblems(lines, config, now, trailingNewline)
	case DocTypeProgress:
		result, err = resetProgress(lines, config, now, trailingNewline)
	default:
		return nil, errors.New("unsupported doc type")
	}
	if err != nil {
		return nil, err
	}

	if config.PreviewMode {
		return result, nil
	}
	if result.NewContent != content {
		if writeErr := os.WriteFile(filePath, []byte(result.NewContent), 0o644); writeErr != nil {
			return nil, writeErr
		}
	}
	return result, nil
}

func resetProblems(lines []string, config ResetConfig, now time.Time, trailingNewline bool) (*ResetResult, error) {
	preamble, entries := parseProblemEntries(lines)
	if len(entries) == 0 {
		return &ResetResult{KeptCount: 0, NewContent: finalizeContent(lines, trailingNewline)}, nil
	}

	keep, removed := selectEntries(entries, config.MaxAgeDays, config.KeepMinEntries, now)

	var newLines []string
	newLines = append(newLines, preamble...)

	var removedEntries []string
	keptCount := 0
	for i, entry := range entries {
		if keep[i] {
			keptCount++
			newLines = append(newLines, lines[entry.start:entry.end]...)
			continue
		}
		if removed[i] {
			removedEntries = append(removedEntries, strings.TrimSpace(lines[entry.start]))
		}
	}

	newContent := finalizeContent(newLines, trailingNewline)
	return &ResetResult{
		RemovedCount:   len(removedEntries),
		KeptCount:      keptCount,
		RemovedEntries: removedEntries,
		NewContent:     newContent,
	}, nil
}

func resetProgress(lines []string, config ResetConfig, now time.Time, trailingNewline bool) (*ResetResult, error) {
	preamble, entries := parseProgressEntries(lines)
	if len(entries) == 0 {
		return &ResetResult{KeptCount: 0, NewContent: finalizeContent(lines, trailingNewline)}, nil
	}

	keep, removed := selectEntries(entries, config.MaxAgeDays, config.KeepMinEntries, now)

	var newLines []string
	newLines = append(newLines, preamble...)

	var removedEntries []string
	keptCount := 0
	for i, entry := range entries {
		line := lines[entry.start]
		if keep[i] {
			keptCount++
			newLines = append(newLines, line)
			continue
		}
		if removed[i] {
			removedEntries = append(removedEntries, strings.TrimSpace(line))
		}
	}

	newContent := finalizeContent(newLines, trailingNewline)
	return &ResetResult{
		RemovedCount:   len(removedEntries),
		KeptCount:      keptCount,
		RemovedEntries: removedEntries,
		NewContent:     newContent,
	}, nil
}

func parseProblemEntries(lines []string) ([]string, []docEntry) {
	var entries []docEntry
	preambleEnd := len(lines)

	for i, line := range lines {
		clean := strings.TrimSuffix(line, "\r")
		match := problemsHeading.FindStringSubmatch(clean)
		if match == nil {
			continue
		}
		if len(entries) == 0 {
			preambleEnd = i
		} else {
			entries[len(entries)-1].end = i
		}
		entry := docEntry{start: i}
		if parsed, err := time.Parse("2006-01-02", match[1]); err == nil {
			entry.date = parsed
			entry.dateValid = true
		}
		entries = append(entries, entry)
	}
	if len(entries) > 0 {
		entries[len(entries)-1].end = len(lines)
	}

	preamble := lines
	if preambleEnd < len(lines) {
		preamble = lines[:preambleEnd]
	}
	return preamble, entries
}

func parseProgressEntries(lines []string) ([]string, []docEntry) {
	var entries []docEntry
	preambleEnd := len(lines)

	for i, line := range lines {
		clean := strings.TrimSuffix(line, "\r")
		match := progressRow.FindStringSubmatch(clean)
		if match == nil {
			continue
		}
		if len(entries) == 0 {
			preambleEnd = i
		}
		entry := docEntry{start: i, end: i + 1}
		if parsed, err := time.Parse("2006-01-02", match[1]); err == nil {
			entry.date = parsed
			entry.dateValid = true
		}
		entries = append(entries, entry)
	}

	preamble := lines
	if preambleEnd < len(lines) {
		preamble = lines[:preambleEnd]
	}
	return preamble, entries
}

type docEntry struct {
	start     int
	end       int
	date      time.Time
	dateValid bool
}

func selectEntries(entries []docEntry, maxAgeDays int, keepMin int, now time.Time) ([]bool, []bool) {
	keep := make([]bool, len(entries))
	removed := make([]bool, len(entries))
	if len(entries) == 0 {
		return keep, removed
	}
	if maxAgeDays <= 0 {
		for i := range entries {
			keep[i] = true
		}
		return keep, removed
	}

	cutoff := now.AddDate(0, 0, -maxAgeDays)
	for i, entry := range entries {
		if !entry.dateValid {
			keep[i] = true
			continue
		}
		if entry.date.Before(cutoff) {
			removed[i] = true
			continue
		}
		keep[i] = true
	}

	keepCount := countTrue(keep)
	if keepMin > 0 && keepCount < keepMin {
		candidates := make([]candidateEntry, 0, len(entries))
		for i, entry := range entries {
			if removed[i] && entry.dateValid {
				candidates = append(candidates, candidateEntry{index: i, date: entry.date})
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].date.After(candidates[j].date)
		})
		for _, candidate := range candidates {
			if keepCount >= keepMin {
				break
			}
			keep[candidate.index] = true
			removed[candidate.index] = false
			keepCount++
		}
	}

	return keep, removed
}

type candidateEntry struct {
	index int
	date  time.Time
}

func countTrue(values []bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func finalizeContent(lines []string, trailingNewline bool) string {
	content := strings.Join(lines, "\n")
	if trailingNewline && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content
}
