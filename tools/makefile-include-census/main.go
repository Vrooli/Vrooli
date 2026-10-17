// Command makefile-include-census keeps every scenario Makefile on the shared
// toolchain include. It lists the Makefiles that miss the include line, that
// still carry a copied body of a shared target, or that define a custom body
// for a shared target without declaring it in SCENARIO_CUSTOM_TARGETS.
//
// With --apply it inserts the include line, deletes bodies that are
// byte-identical to mk/scenario.mk's, and declares the remaining custom
// bodies. Run it once and review the summary; the check mode is the test.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	includeLine  = "-include ../../mk/scenario.mk"
	includeNote  = "# Shared toolchain floor (mk/toolchain.mk) and build targets (mk/scenario.mk); a custom body for one of them is declared in SCENARIO_CUSTOM_TARGETS."
	customPrefix = "SCENARIO_CUSTOM_TARGETS +="
	scenarioMK   = "mk/scenario.mk"
)

// sharedTargets are the recipes mk/scenario.mk supplies.
var sharedTargets = []string{"build", "fmt-go", "fmt-ui", "lint-go", "lint-ui"}

// legacyVariants are bodies an older template generation shipped. They are
// drift, not intent: the current recipe is a superset (GOWORK=off, the cli
// module, the strings check) or guards on the same file the no-op assumed
// absent. A body that matches one is replaced by the include like a copy.
var legacyVariants = map[string][]string{
	"build": {normalizeText(`build:
	@if [ -f api/go.mod ]; then \
		cd api && go build -o "$(SCENARIO_NAME)-api" .; \
	fi
	@if [ -f ui/package.json ]; then \
		cd ui && pnpm run build; \
	fi`)},
	"fmt-go": {normalizeText(`fmt-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v gofumpt >/dev/null 2>&1; then \
			cd api && gofumpt -w .; \
		elif command -v gofmt >/dev/null 2>&1; then \
			cd api && gofmt -w .; \
		fi; \
	fi`)},
	"lint-go": {normalizeText(`lint-go:
	@if [ -d api ] && find api -name "*.go" | head -1 | grep -q .; then \
		if command -v golangci-lint >/dev/null 2>&1; then \
			cd api && golangci-lint run ./...; \
		elif command -v go >/dev/null 2>&1; then \
			cd api && go vet ./... && go fmt ./...; \
		fi; \
	fi`)},
	"lint-ui": {normalizeText("lint-ui:\n\t@true")},
	"fmt-ui":  {normalizeText("fmt-ui:\n\t@true")},
}

func normalizeText(body string) string { return normalize(strings.Split(body, "\n")) }

func replaceable(target, body, canonical string) bool {
	if body == canonical {
		return true
	}
	for _, variant := range legacyVariants[target] {
		if body == variant {
			return true
		}
	}
	return false
}

var targetLine = regexp.MustCompile(`^([A-Za-z0-9_.-]+):(?:[^=]|$)`)

type outlier struct {
	Path   string
	Reason string
}

func main() {
	root := flag.String("root", ".", "repository root")
	apply := flag.Bool("apply", false, "rewrite scenario Makefiles onto the include")
	flag.Parse()
	outliers, err := run(*root, *apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, o := range outliers {
		fmt.Printf("%s: %s\n", o.Path, o.Reason)
	}
	if len(outliers) > 0 && !*apply {
		os.Exit(1)
	}
}

// Run is the check-mode entry point the drift test calls.
func Run(root string) ([]string, error) {
	outliers, err := run(root, false)
	if err != nil {
		return nil, err
	}
	lines := make([]string, 0, len(outliers))
	for _, o := range outliers {
		lines = append(lines, o.Path+": "+o.Reason)
	}
	return lines, nil
}

func run(root string, apply bool) ([]outlier, error) {
	canonical, err := canonicalBodies(filepath.Join(root, scenarioMK))
	if err != nil {
		return nil, err
	}
	makefiles, err := filepath.Glob(filepath.Join(root, "scenarios", "*", "Makefile"))
	if err != nil {
		return nil, err
	}
	sort.Strings(makefiles)
	var outliers []outlier
	for _, path := range makefiles {
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		declared := declaredCustom(lines)
		hasInclude := containsLine(lines, includeLine)
		blocks := targetBlocks(lines)
		var custom []string
		var identical []string
		for _, target := range sharedTargets {
			block, ok := blocks[target]
			if !ok {
				continue
			}
			if replaceable(target, normalize(lines[block.start:block.end]), canonical[target]) {
				identical = append(identical, target)
			} else if !declared[target] {
				custom = append(custom, target)
			}
		}
		if !apply {
			if !hasInclude {
				outliers = append(outliers, outlier{rel, "missing " + includeLine})
			}
			for _, t := range identical {
				outliers = append(outliers, outlier{rel, "carries a copied body for " + t + " that mk/scenario.mk supplies"})
			}
			for _, t := range custom {
				outliers = append(outliers, outlier{rel, "custom body for " + t + " is not declared in SCENARIO_CUSTOM_TARGETS"})
			}
			continue
		}
		if hasInclude && len(identical) == 0 && len(custom) == 0 {
			continue
		}
		rewritten := rewrite(lines, blocks, identical, custom, hasInclude)
		if err := os.WriteFile(path, []byte(strings.Join(rewritten, "\n")+"\n"), 0o644); err != nil {
			return nil, err
		}
		reason := fmt.Sprintf("applied: include=%t removed=%v custom=%v", !hasInclude, identical, custom)
		outliers = append(outliers, outlier{rel, reason})
	}
	return outliers, nil
}

type span struct{ start, end int }

// targetBlocks maps a target name to the line span of its rule: the target
// line and the recipe lines that follow, up to the next blank line.
func targetBlocks(lines []string) map[string]span {
	blocks := map[string]span{}
	for i := 0; i < len(lines); i++ {
		m := targetLine.FindStringSubmatch(lines[i])
		if m == nil || strings.HasPrefix(lines[i], ".") {
			continue
		}
		end := i + 1
		for end < len(lines) && strings.TrimSpace(lines[end]) != "" && (strings.HasPrefix(lines[end], "\t") || strings.HasSuffix(lines[end-1], "\\")) {
			end++
		}
		if _, seen := blocks[m[1]]; !seen {
			blocks[m[1]] = span{i, end}
		}
	}
	return blocks
}

// canonicalBodies reads the shared recipes out of mk/scenario.mk.
func canonicalBodies(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	blocks := targetBlocks(lines)
	out := map[string]string{}
	for _, target := range sharedTargets {
		block, ok := blocks[target]
		if !ok {
			return nil, fmt.Errorf("%s does not define %s", path, target)
		}
		out[target] = normalize(lines[block.start:block.end])
	}
	return out, nil
}

// normalize compares recipes on their tokens: the help comment on the
// target line and indentation width do not make a body custom.
func normalize(block []string) string {
	var b strings.Builder
	for i, line := range block {
		if i == 0 {
			if idx := strings.Index(line, "##"); idx >= 0 {
				line = line[:idx]
			}
		}
		b.WriteString(strings.Join(strings.Fields(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

func declaredCustom(lines []string) map[string]bool {
	out := map[string]bool{}
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), customPrefix) {
			for _, name := range strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), customPrefix)) {
				out[name] = true
			}
		}
	}
	return out
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

// rewrite drops the identical blocks, declares the custom targets and
// inserts the include after SCENARIO_NAME (or .DEFAULT_GOAL, or .PHONY).
func rewrite(lines []string, blocks map[string]span, identical, custom []string, hasInclude bool) []string {
	drop := map[int]bool{}
	for _, t := range identical {
		b := blocks[t]
		for i := b.start; i < b.end; i++ {
			drop[i] = true
		}
		if b.end < len(lines) && strings.TrimSpace(lines[b.end]) == "" {
			drop[b.end] = true
		}
	}
	anchor := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "SCENARIO_NAME"):
			anchor = i
		case anchor < 0 && strings.HasPrefix(trimmed, ".DEFAULT_GOAL"):
			anchor = i
		case anchor < 0 && strings.HasPrefix(trimmed, ".PHONY"):
			anchor = i
		}
		if strings.HasPrefix(trimmed, "SCENARIO_NAME") {
			break
		}
	}
	var out []string
	for i, line := range lines {
		if drop[i] {
			continue
		}
		out = append(out, line)
		if i == anchor && !hasInclude {
			out = append(out, "")
			if len(custom) > 0 {
				out = append(out, customPrefix+" "+strings.Join(custom, " "))
			}
			out = append(out, includeNote, includeLine)
		}
	}
	if hasInclude && len(custom) > 0 {
		for i, line := range out {
			if strings.TrimSpace(line) == includeLine {
				out = append(out[:i], append([]string{customPrefix + " " + strings.Join(custom, " ")}, out[i:]...)...)
				break
			}
		}
	}
	// Collapse runs of blank lines the removals may have left behind.
	var compact []string
	for i, line := range out {
		if strings.TrimSpace(line) == "" && i > 0 && strings.TrimSpace(out[i-1]) == "" {
			continue
		}
		compact = append(compact, line)
	}
	return compact
}
