package doclogs

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"knowledge-observatory/internal/doccontract"
)

var (
	ErrUnsupportedFormat = errors.New("append-log format is not supported")
	ErrTargetNotFound    = errors.New("append-log target heading was not found")
	ErrTitleRequired     = errors.New("title is required")

	datedHeading = regexp.MustCompile(`^#{1,6}\s+(\d{4}-\d{2}-\d{2})\b`)
	tableDateRow = regexp.MustCompile(`^\|\s*(\d{4}-\d{2}-\d{2})\s*\|`)
)

type Entry struct {
	Title  string
	Body   string
	Author string
	Status string
	Fields map[string]string
}

type AppendResult struct {
	FilePath   string
	EntryAdded string
}

type ResetConfig struct {
	MaxAgeDays     int
	KeepMinEntries int
	PreviewMode    bool
}

type ResetResult struct {
	RemovedCount   int
	KeptCount      int
	RemovedEntries []string
	NewContent     string
}

func Append(filePath string, op doccontract.AppendLogOperation, entry Entry) (*AppendResult, error) {
	return appendWithClock(filePath, op, entry, time.Now())
}

func appendWithClock(filePath string, op doccontract.AppendLogOperation, entry Entry, now time.Time) (*AppendResult, error) {
	if strings.TrimSpace(entry.Title) == "" && op.Format == "dated-markdown-section" {
		return nil, ErrTitleRequired
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	text := string(content)
	added, err := formatEntry(op, entry, now)
	if err != nil {
		return nil, err
	}
	updated, err := appendToHeadingRegion(text, op.TargetHeading, op.EmptyMarker, added)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filePath, []byte(updated), 0o644); err != nil {
		return nil, err
	}
	return &AppendResult{FilePath: filePath, EntryAdded: added}, nil
}

func Reset(filePath string, op doccontract.AppendLogOperation, config ResetConfig) (*ResetResult, error) {
	return resetWithClock(filePath, op, config, time.Now())
}

func resetWithClock(filePath string, op doccontract.AppendLogOperation, config ResetConfig, now time.Time) (*ResetResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	text := string(content)
	region, err := findHeadingRegion(strings.Split(text, "\n"), op.TargetHeading)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(text, "\n")
	body := lines[region.bodyStart:region.end]
	keep, keptCount, removedLabels := pruneLogEntries(op, body, config, now)
	if len(removedLabels) == 0 {
		return &ResetResult{KeptCount: keptCount, NewContent: text}, nil
	}
	newBody := keep
	if len(newBody) == 0 && op.EmptyMarker != "" {
		newBody = []string{emptyMarkerLine(op)}
	}
	var newLines []string
	newLines = append(newLines, lines[:region.bodyStart]...)
	newLines = append(newLines, newBody...)
	newLines = append(newLines, lines[region.end:]...)
	newContent := strings.Join(newLines, "\n")
	result := &ResetResult{
		RemovedCount:   len(removedLabels),
		KeptCount:      keptCount,
		RemovedEntries: removedLabels,
		NewContent:     newContent,
	}
	if config.PreviewMode {
		return result, nil
	}
	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		return nil, err
	}
	return result, nil
}

func formatEntry(op doccontract.AppendLogOperation, entry Entry, now time.Time) (string, error) {
	date := now.Format("2006-01-02")
	switch op.Format {
	case "dated-markdown-section":
		var b strings.Builder
		fmt.Fprintf(&b, "### %s - %s\n\n", date, strings.TrimSpace(entry.Title))
		if strings.TrimSpace(entry.Body) != "" {
			b.WriteString(strings.TrimSpace(entry.Body))
			b.WriteString("\n")
		}
		return strings.TrimRight(b.String(), "\n"), nil
	case "markdown-table":
		values := map[string]string{
			"date":   date,
			"author": defaultString(entry.Author, "system"),
			"status": defaultString(entry.Status, "done"),
			"notes":  strings.TrimSpace(entry.Body),
			"title":  strings.TrimSpace(entry.Title),
		}
		for key, value := range entry.Fields {
			values[key] = value
		}
		cells := make([]string, 0, len(op.Fields))
		for _, field := range op.Fields {
			cells = append(cells, strings.ReplaceAll(values[field], "|", "\\|"))
		}
		return "| " + strings.Join(cells, " | ") + " |", nil
	default:
		return "", ErrUnsupportedFormat
	}
}

func appendToHeadingRegion(content, heading, emptyMarker, entry string) (string, error) {
	lines := strings.Split(content, "\n")
	region, err := findHeadingRegion(lines, heading)
	if err != nil {
		return "", err
	}
	body := append([]string{}, lines[region.bodyStart:region.end]...)
	body = removeEmptyMarker(body, emptyMarker)
	if len(body) > 0 && strings.TrimSpace(body[len(body)-1]) != "" {
		body = append(body, "")
	}
	body = append(body, strings.Split(entry, "\n")...)
	var updated []string
	updated = append(updated, lines[:region.bodyStart]...)
	updated = append(updated, body...)
	updated = append(updated, lines[region.end:]...)
	return strings.Join(updated, "\n"), nil
}

type headingRegion struct {
	bodyStart int
	end       int
}

func findHeadingRegion(lines []string, target string) (headingRegion, error) {
	targetLevel := 0
	start := -1
	for i, line := range lines {
		level, title, ok := parseHeading(line)
		if !ok {
			continue
		}
		if start == -1 {
			if title == target {
				targetLevel = level
				start = i
			}
			continue
		}
		if level <= targetLevel {
			return headingRegion{bodyStart: start + 1, end: i}, nil
		}
	}
	if start == -1 {
		return headingRegion{}, ErrTargetNotFound
	}
	return headingRegion{bodyStart: start + 1, end: len(lines)}, nil
}

func parseHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	return level, strings.TrimSpace(trimmed[level:]), true
}

func removeEmptyMarker(lines []string, marker string) []string {
	if marker == "" {
		return lines
	}
	out := lines[:0]
	for _, line := range lines {
		if isEmptyMarkerLine(line, marker) {
			continue
		}
		out = append(out, line)
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return out
}

func isEmptyMarkerLine(line, marker string) bool {
	marker = strings.TrimSpace(marker)
	line = strings.TrimSpace(line)
	if marker == "" {
		return false
	}
	return line == marker || strings.Contains(line, marker)
}

func emptyMarkerLine(op doccontract.AppendLogOperation) string {
	marker := strings.TrimSpace(op.EmptyMarker)
	if op.Format != "markdown-table" || marker == "" || strings.HasPrefix(marker, "|") {
		return marker
	}
	cells := make([]string, 0, len(op.Fields))
	for i := range op.Fields {
		if i == 0 {
			cells = append(cells, marker)
		} else {
			cells = append(cells, "")
		}
	}
	if len(cells) == 0 {
		cells = []string{marker}
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

func pruneLogEntries(op doccontract.AppendLogOperation, lines []string, config ResetConfig, now time.Time) ([]string, int, []string) {
	entries := parseEntries(op, lines)
	if len(entries) == 0 {
		return removeEmptyMarker(lines, op.EmptyMarker), 0, nil
	}
	keepMask, removeMask := selectEntries(entries, config.MaxAgeDays, config.KeepMinEntries, now)
	if op.Format == "markdown-table" {
		return pruneTableRows(op, lines, entries, keepMask, removeMask)
	}
	var keepLines []string
	var removedLabels []string
	keptCount := 0
	for i, entry := range entries {
		if keepMask[i] {
			keepLines = append(keepLines, lines[entry.start:entry.end]...)
			keptCount++
		}
		if removeMask[i] {
			removedLabels = append(removedLabels, strings.TrimSpace(lines[entry.start]))
		}
	}
	return keepLines, keptCount, removedLabels
}

func pruneTableRows(op doccontract.AppendLogOperation, lines []string, entries []parsedEntry, keepMask []bool, removeMask []bool) ([]string, int, []string) {
	entryByLine := map[int]int{}
	for i, entry := range entries {
		entryByLine[entry.start] = i
	}
	var keepLines []string
	var removedLabels []string
	keptCount := 0
	hasEmptyMarker := false
	for i, line := range lines {
		if isEmptyMarkerLine(line, op.EmptyMarker) {
			hasEmptyMarker = true
			continue
		}
		entryIndex, isEntry := entryByLine[i]
		if !isEntry {
			keepLines = append(keepLines, line)
			continue
		}
		if keepMask[entryIndex] {
			keepLines = append(keepLines, line)
			keptCount++
		}
		if removeMask[entryIndex] {
			removedLabels = append(removedLabels, strings.TrimSpace(line))
		}
	}
	if keptCount == 0 && op.EmptyMarker != "" && !hasEmptyMarker && len(removedLabels) > 0 {
		keepLines = append(keepLines, emptyMarkerLine(op))
	}
	return keepLines, keptCount, removedLabels
}

type parsedEntry struct {
	start     int
	end       int
	date      time.Time
	dateValid bool
}

func parseEntries(op doccontract.AppendLogOperation, lines []string) []parsedEntry {
	switch op.Format {
	case "dated-markdown-section":
		return parseDatedSections(lines)
	case "markdown-table":
		return parseTableRows(lines)
	default:
		return nil
	}
}

func parseDatedSections(lines []string) []parsedEntry {
	var entries []parsedEntry
	for i, line := range lines {
		m := datedHeading.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		if len(entries) > 0 {
			entries[len(entries)-1].end = i
		}
		entry := parsedEntry{start: i}
		if parsed, err := time.Parse("2006-01-02", m[1]); err == nil {
			entry.date = parsed
			entry.dateValid = true
		}
		entries = append(entries, entry)
	}
	if len(entries) > 0 {
		entries[len(entries)-1].end = len(lines)
	}
	return entries
}

func parseTableRows(lines []string) []parsedEntry {
	var entries []parsedEntry
	for i, line := range lines {
		m := tableDateRow.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		entry := parsedEntry{start: i, end: i + 1}
		if parsed, err := time.Parse("2006-01-02", m[1]); err == nil {
			entry.date = parsed
			entry.dateValid = true
		}
		entries = append(entries, entry)
	}
	return entries
}

func selectEntries(entries []parsedEntry, maxAgeDays int, keepMin int, now time.Time) ([]bool, []bool) {
	keep := make([]bool, len(entries))
	removed := make([]bool, len(entries))
	if maxAgeDays <= 0 {
		for i := range keep {
			keep[i] = true
		}
		return keep, removed
	}
	cutoff := now.AddDate(0, 0, -maxAgeDays)
	for i, entry := range entries {
		if !entry.dateValid || !entry.date.Before(cutoff) {
			keep[i] = true
			continue
		}
		removed[i] = true
	}
	keepCount := 0
	for _, value := range keep {
		if value {
			keepCount++
		}
	}
	if keepMin > 0 && keepCount < keepMin {
		candidates := make([]parsedEntry, 0, len(entries))
		for i, entry := range entries {
			if removed[i] {
				entry.start = i
				candidates = append(candidates, entry)
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].date.After(candidates[j].date) })
		for _, candidate := range candidates {
			if keepCount >= keepMin {
				break
			}
			keep[candidate.start] = true
			removed[candidate.start] = false
			keepCount++
		}
	}
	return keep, removed
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
