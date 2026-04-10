package docschema

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AppendConfig describes a new entry to append to a structured document.
type AppendConfig struct {
	DocType DocType
	Title   string
	Body    string // Freeform for problems; maps to Notes column for progress
	Author  string // Used only for progress table rows
	Status  string // Used only for progress table rows
}

// AppendResult reports the outcome of an append operation.
type AppendResult struct {
	FilePath   string
	EntryAdded string // The raw text that was appended
}

// AppendEntry appends a format-compatible entry to a known-structure document.
func AppendEntry(filePath string, config AppendConfig) (*AppendResult, error) {
	return appendEntryWithClock(filePath, config, time.Now())
}

func appendEntryWithClock(filePath string, config AppendConfig, now time.Time) (*AppendResult, error) {
	if config.DocType != DocTypeProblems && config.DocType != DocTypeProgress {
		return nil, errors.New("append is only supported for problems and progress documents")
	}
	title := strings.TrimSpace(config.Title)
	if title == "" {
		return nil, errors.New("title is required")
	}

	var entry string
	dateStr := now.Format("2006-01-02")
	switch config.DocType {
	case DocTypeProblems:
		entry = formatProblemEntry(dateStr, title, strings.TrimSpace(config.Body))
	case DocTypeProgress:
		entry = formatProgressEntry(dateStr, title, strings.TrimSpace(config.Author), strings.TrimSpace(config.Status))
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if mkErr := os.MkdirAll(filepath.Dir(filePath), 0o755); mkErr != nil {
			return nil, mkErr
		}
		content = []byte(defaultHeader(config.DocType))
	}

	text := string(content)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += entry

	if err := os.WriteFile(filePath, []byte(text), 0o644); err != nil {
		return nil, err
	}
	return &AppendResult{FilePath: filePath, EntryAdded: entry}, nil
}

func formatProblemEntry(date, title, body string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s: %s\n\n", date, title)
	if body != "" {
		fmt.Fprintf(&sb, "### Problem\n%s\n\n", body)
	}
	sb.WriteString("---\n")
	return sb.String()
}

func formatProgressEntry(date, notes, author, status string) string {
	if author == "" {
		author = "system"
	}
	if status == "" {
		status = "done"
	}
	return fmt.Sprintf("| %s | %s | %s | %s |\n", date, author, status, notes)
}

func defaultHeader(dt DocType) string {
	switch dt {
	case DocTypeProblems:
		return "# Problems\n\n"
	case DocTypeProgress:
		return "# Progress\n\n| Date | Author | Status | Notes |\n|------|--------|--------|-------|\n"
	default:
		return ""
	}
}
