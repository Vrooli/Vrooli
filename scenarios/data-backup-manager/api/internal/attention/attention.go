// Package attention reads the research supervisor's attention file, so a
// decision's technical details can name the other items that need the
// operator now. It is context for the next agent, never part of the question.
package attention

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var heading = regexp.MustCompile(`(?m)^###\s+([^\s]+)\s+—\s+(.+?)\s*$`)

// File is the attention file under the operator's home.
func File() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join("~", ".vrooli", "research", "ATTENTION.md")
	}
	return filepath.Join(home, ".vrooli", "research", "ATTENTION.md")
}

// Digest lists up to max "Actionable now" items from the file read returns,
// or says why it cannot.
func Digest(read func(string) ([]byte, error), max int) string {
	if read == nil {
		return "unverified (reader unavailable)"
	}
	data, err := read(File())
	if err != nil {
		return "unverified (file unreadable)"
	}
	content := string(data)
	start := strings.Index(content, "## Actionable now")
	if start < 0 {
		return "unverified (no Actionable now section)"
	}
	section := content[start+len("## Actionable now"):]
	if next := strings.Index(section, "\n## "); next >= 0 {
		section = section[:next]
	}
	items := heading.FindAllStringSubmatch(section, -1)
	if len(items) == 0 {
		if strings.Contains(section, "No actionable items are recorded.") {
			return "none recorded"
		}
		return "unverified (entries could not be parsed)"
	}
	shown := min(len(items), max)
	parts := make([]string, 0, shown+1)
	for _, item := range items[:shown] {
		parts = append(parts, item[1]+" — "+item[2])
	}
	if len(items) > shown {
		parts = append(parts, fmt.Sprintf("and %d more", len(items)-shown))
	}
	return strings.Join(parts, "; ")
}
