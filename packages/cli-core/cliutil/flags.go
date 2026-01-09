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
