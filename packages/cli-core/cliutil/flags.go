package cliutil

import (
	"flag"
	"os"
	"strings"
)

// StringList collects repeated flag values (e.g., --tag foo --tag bar).
type StringList []string

func (s *StringList) String() string {
	return strings.Join(*s, ",")
}

func (s *StringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// Values returns a copy of the collected values.
func (s *StringList) Values() []string {
	out := make([]string, len(*s))
	copy(out, *s)
	return out
}

// NewFlagSet creates a CLI flag set that reports parse errors to stderr.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// ParseCSV splits a comma-separated list into trimmed values, dropping blanks.
func ParseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var cleaned []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return cleaned
}

// ParseCSVValues parses repeated or comma-separated values without changing
// their order or duplicates.
func ParseCSVValues(values []string) []string {
	var parsed []string
	for _, value := range values {
		parsed = append(parsed, ParseCSV(value)...)
	}
	return parsed
}

// ParseUniqueCSVValues parses repeated or comma-separated values and keeps the
// first occurrence of each trimmed, nonblank value.
func ParseUniqueCSVValues(values []string) []string {
	seen := make(map[string]struct{})
	var parsed []string
	for _, value := range values {
		for _, part := range ParseCSV(value) {
			if _, exists := seen[part]; exists {
				continue
			}
			seen[part] = struct{}{}
			parsed = append(parsed, part)
		}
	}
	return parsed
}

// MergeArgs joins primary and extra positional arguments, trimming blanks.
func MergeArgs(primary, extra []string) []string {
	result := append([]string{}, primary...)
	for _, item := range extra {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

// ReadFileString reads a file and returns its contents as a string.
func ReadFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSONFlag adds a standard --json flag to a FlagSet.
func JSONFlag(fs *flag.FlagSet) *bool {
	return fs.Bool("json", false, "Output raw JSON")
}
