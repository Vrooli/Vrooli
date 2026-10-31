// Package transcriptredact removes fixture-unsafe data from runner transcripts.
// It is shared by the corpus harvester and committed-fixture gate so their
// safety rules cannot drift apart.
package transcriptredact

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var replacements = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)("(?:api[_-]?key|token|secret|authorization|access_token|refresh_token|password)"\s*:\s*)"[^"]*"`), `${1}"<REDACTED>"`},
	{regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|authorization|access_token|refresh_token|password)\s*[=:]\s*)[^\s"']+`), `${1}<REDACTED>`},
	{regexp.MustCompile(`(?i)(--(?:api[_-]?key|token|secret|password)\s+)[^\s"']+`), `${1}<REDACTED>`},
	{regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-]{8,}`), `Bearer <REDACTED>`},
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]{8,}`), `<REDACTED>`},
	{regexp.MustCompile(`(?i)(?:/home/|/users/)[^/"\\\s]+`), `<HOME>`},
	{regexp.MustCompile(`(?i)[a-z]:\\users\\[^\\"\s]+`), `<HOME>`},
}

// Redact preserves transcript structure and classification-relevant fields
// while replacing credential-shaped values and user-home prefixes.
func Redact(value string) string {
	for _, rule := range replacements {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	return value
}

// ScanDir reports fixture files that are not already in their canonical
// redacted form. This gives the repository gate exactly the same policy as the
// harvesting write path.
func ScanDir(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if Redact(string(contents)) != string(contents) {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			violations = append(violations, fmt.Sprintf("%s contains a credential or absolute home path", filepath.ToSlash(relative)))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(violations)
	return violations, nil
}
