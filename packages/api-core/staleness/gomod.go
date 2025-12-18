package staleness

import (
	"bufio"
	"os"
	"strings"
)

// ParseReplaceDirectives extracts local (relative path) replace directive targets
// from a go.mod file. Returns paths like "../../../packages/proto" for directives like:
//
//	replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto
//
// Only relative paths (starting with "./" or "../") are returned, as these represent
// local dependencies that should be checked for staleness.
func ParseReplaceDirectives(goModPath string) ([]string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var replacePaths []string
	scanner := bufio.NewScanner(file)
	inReplaceBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Handle replace block: replace ( ... )
		// Use HasPrefix/HasSuffix to handle whitespace variations like "replace  ("
		if strings.HasPrefix(line, "replace") && strings.HasSuffix(line, "(") {
			inReplaceBlock = true
			continue
		}
		if inReplaceBlock && line == ")" {
			inReplaceBlock = false
			continue
		}

		// Handle single-line replace or line within replace block
		var replaceLine string
		if strings.HasPrefix(line, "replace ") {
			replaceLine = strings.TrimPrefix(line, "replace ")
		} else if inReplaceBlock && strings.Contains(line, "=>") {
			replaceLine = line
		} else {
			continue
		}

		// Parse the replacement path after =>
		parts := strings.Split(replaceLine, "=>")
		if len(parts) != 2 {
			continue
		}

		replacement := strings.TrimSpace(parts[1])

		// Only include relative paths (local dependencies)
		if strings.HasPrefix(replacement, "../") || strings.HasPrefix(replacement, "./") {
			replacePaths = append(replacePaths, replacement)
		}
	}

	if err := scanner.Err(); err != nil {
		return replacePaths, err
	}

	return replacePaths, nil
}
