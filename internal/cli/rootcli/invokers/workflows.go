package invokers

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// workflowsDir holds the CI definitions scanned for vrooli invocations.
const workflowsDir = ".github/workflows"

// invocationPrefixes are the ways a workflow step names the CLI. The scan is
// line-based and tolerant: a line carrying a `${{ }}` expression is skipped,
// pipes and redirections end the argv, and other `vrooli-*` binaries are not
// matched.
var invocationPrefixes = []string{"go run ./cmd/vrooli ", "vrooli "}

func workflowInvokers() ([]Invoker, error) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root for %s: %w", workflowsDir, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, workflowsDir))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml")) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var items []Invoker
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join(workflowsDir, name))
		found, err := scanWorkflow(filepath.Join(root, workflowsDir, name), rel)
		if err != nil {
			return nil, err
		}
		items = append(items, found...)
	}
	return items, nil
}

func scanWorkflow(path, rel string) ([]Invoker, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var items []Invoker
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		text := scanner.Text()
		if strings.Contains(text, "${{") {
			continue
		}
		for _, segment := range strings.Split(text, "|") {
			argv, ok := argvFromSegment(segment)
			if !ok {
				continue
			}
			items = append(items, static(fmt.Sprintf("ci/%s:%d", name(rel), line), rel, argv))
		}
	}
	return items, scanner.Err()
}

func name(rel string) string {
	return strings.TrimSuffix(strings.TrimSuffix(filepath.Base(rel), ".yml"), ".yaml")
}

// argvFromSegment extracts the vrooli argv from one pipeline segment.
func argvFromSegment(segment string) ([]string, bool) {
	segment = strings.TrimSpace(segment)
	if strings.HasPrefix(segment, "#") {
		return nil, false
	}
	// Strip a remote-shell wrapper: `ssh ... 'cd X && vrooli deploy'`.
	if idx := strings.Index(segment, "&& "); idx >= 0 {
		segment = strings.TrimSpace(segment[idx+3:])
	}
	for _, prefix := range invocationPrefixes {
		idx := strings.Index(segment, prefix)
		if idx < 0 {
			continue
		}
		if idx > 0 {
			before := segment[idx-1]
			if before != ' ' && before != '\t' && before != '"' && before != '\'' && before != '(' {
				continue
			}
		}
		rest := segment[idx+len(prefix):]
		fields := strings.Fields(rest)
		var argv []string
		for _, field := range fields {
			field = strings.Trim(field, "'\"")
			if field == "" {
				continue
			}
			if strings.HasPrefix(field, ">") || strings.HasPrefix(field, "2>") || field == "&&" || field == ";" || strings.HasPrefix(field, "<") {
				break
			}
			argv = append(argv, field)
		}
		if len(argv) == 0 {
			return nil, false
		}
		return argv, true
	}
	return nil, false
}
