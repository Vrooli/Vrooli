/*
Rule: Library Shell Ownership
ID: standard_shell_ownership
Description: Scenarios declare a shell archetype and mount its library component;

	application chrome anywhere in production source requires a scoped ejection.

Why: Local chrome forks lose library accessibility and responsive behavior.
Category: standards
Severity: high
Slot: [D]
SlotFile: ui/manifest.json
TechStack: React
Recommendation: Mount the declared library archetype; record an evidence-backed,

	file-scoped shell-ejection in docs/reference/component-library-gaps.md when
	the library cannot yet carry this product.

Standard: vrooli-library-shell-ownership-v1
GoodExample:

	export function AppShell() { return <LibraryAppShell />; }

BadExample:

	export function AppShell() { return <nav><a href="/">Home</a></nav>; }
*/
package checks

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"ui-health/internal/uiinterop"

	"golang.org/x/net/html"
)

//go:embed shell_ownership.cjs
var shellOwnershipScanner string

type shellDeclaration struct {
	Archetype string `json:"archetype"`
	Asset     string `json:"asset"`
	Entry     string `json:"entry"`
	Export    string `json:"export"`
}
type shellFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Element string `json:"element"`
	Reason  string `json:"reason"`
}
type shellAnalysis struct {
	Mounted  bool           `json:"mounted"`
	Findings []shellFinding `json:"findings"`
}

var shellArchetypes = map[string]string{
	"navigated-console": "AppShell",
	"ambient-display":   "AmbientDisplayShell",
	"command-center":    "CommandCenterShell",
	"master-detail":     "MasterDetail",
	"inspector":         "InspectorLayout",
	"drawer":            "DrawerShell",
}

func init() { uiinterop.Register("standard_shell_ownership", checkShellOwnership) }

func checkShellOwnership(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const id = "standard_shell_ownership"
	result := uiinterop.RuleResult{RuleID: id, Passed: true, Message: "declared library shell owns application chrome"}
	fail := func(file string, line int, element, reason string) {
		result.Passed = false
		result.Message = "shell ownership could not be verified"
		result.Violations = append(result.Violations, uiinterop.Violation{
			RuleID: id, Severity: "high", Title: "Library shell ownership", FilePath: file, Line: line,
			CodeSnippet: element, Description: fmt.Sprintf("%s:%d %s: %s", file, line, element, reason),
			Recommendation: "Mount the declared library shell and move scenario controls into its slots or feature pages. If the archetype cannot carry the product, document a reason and exact files in a shell-ejection JSON block in docs/reference/component-library-gaps.md.",
		})
	}
	var manifest struct {
		Shell shellDeclaration `json:"shell"`
	}
	data, err := os.ReadFile(filepath.Join(ctx.ScenarioRoot, "ui", "manifest.json"))
	if err != nil || json.Unmarshal(data, &manifest) != nil {
		fail("ui/manifest.json", 1, "shell", "readable UI manifest with a shell declaration is required")
		return result
	}
	shell := manifest.Shell
	if shellArchetypes[shell.Archetype] == "" || shellArchetypes[shell.Archetype] != shell.Asset || !safeShellPath(shell.Entry) || strings.TrimSpace(shell.Export) == "" {
		fail("ui/manifest.json", 1, "shell", "declare a supported archetype, its library asset, production entry path and exported component")
		return result
	}
	ejected, err := shellEjectionFiles(ctx.ScenarioRoot, shell.Archetype)
	if err != nil {
		fail("docs/reference/component-library-gaps.md", 1, "shell-ejection", err.Error())
		return result
	}
	analysis, err := analyzeShellOwnership(ctx, shell)
	if err != nil {
		fail("ui/manifest.json", 1, "shell", "source analysis unavailable: "+err.Error())
		return result
	}
	for _, finding := range analysis.Findings {
		// Parse and declaration failures cannot be excused as a product gap.
		if ejected[finding.File] != "" && strings.HasPrefix(finding.Reason, "locally rendered") {
			continue
		}
		fail(finding.File, finding.Line, finding.Element, finding.Reason)
	}
	if !analysis.Mounted && ejected[shell.Entry] == "" {
		fail(shell.Entry, 1, shell.Export, "declared entry does not render "+shell.Asset+" from @vrooli/react-component-library")
	}
	if result.Passed && len(ejected) > 0 {
		reasons := map[string]bool{}
		for _, reason := range ejected {
			reasons[reason] = true
		}
		ordered := make([]string, 0, len(reasons))
		for reason := range reasons {
			ordered = append(ordered, reason)
		}
		sort.Strings(ordered)
		result.Message = "shell ownership verified with documented file-scoped ejection: " + strings.Join(ordered, "; ")
	}
	return result
}

func safeShellPath(value string) bool {
	return strings.HasPrefix(value, "ui/src/") && filepath.ToSlash(filepath.Clean(value)) == value && !strings.Contains(value, "\\")
}

// Ejections name exact browser source files; they do not change which module
// may serve as the declared application shell entry.
func safeShellEjectionPath(value string) bool {
	return shellBrowserSourcePath(value) &&
		filepath.ToSlash(filepath.Clean(value)) == value && !strings.Contains(value, "\\")
}

func shellBrowserSourcePath(value string) bool {
	if strings.HasPrefix(value, "ui/src/") || strings.HasPrefix(value, "ui/public/") {
		return true
	}
	switch value {
	case "ui/app.js", "ui/script.js", "ui/main.js", "ui/index.js", "ui/index.html":
		return true
	}
	return false
}

var shellEjectionBlock = regexp.MustCompile("(?s)```shell-ejection\\s*\\n(.*?)\\n```")

func shellEjectionFiles(root, archetype string) (map[string]string, error) {
	files := map[string]string{}
	data, err := os.ReadFile(filepath.Join(root, "docs/reference/component-library-gaps.md"))
	if os.IsNotExist(err) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	for _, block := range shellEjectionBlock.FindAllSubmatch(data, -1) {
		var record struct {
			Archetype string   `json:"archetype"`
			Reason    string   `json:"reason"`
			Files     []string `json:"files"`
		}
		if err := json.Unmarshal(block[1], &record); err != nil {
			return nil, fmt.Errorf("invalid shell-ejection JSON: %w", err)
		}
		if record.Archetype != archetype {
			continue
		}
		if strings.TrimSpace(record.Reason) == "" || len(record.Files) == 0 {
			return nil, fmt.Errorf("shell-ejection requires a reason and exact production files")
		}
		for _, file := range record.Files {
			if !safeShellEjectionPath(file) {
				return nil, fmt.Errorf("invalid shell-ejection file %q", file)
			}
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(file))); err != nil || info.IsDir() {
				return nil, fmt.Errorf("shell-ejection file %q does not exist", file)
			}
			files[file] = strings.TrimSpace(record.Reason)
		}
	}
	return files, nil
}

func analyzeShellOwnership(ctx uiinterop.CheckContext, shell shellDeclaration) (shellAnalysis, error) {
	var result shellAnalysis
	repo := findRepoRoot(ctx.ScenarioRoot)
	if repo == "" {
		if cwd, err := os.Getwd(); err == nil {
			repo = findRepoRoot(cwd)
		}
	}
	typescript := filepath.Join(repo, "scenarios/ui-health/ui/node_modules/typescript")
	if _, err := os.Stat(filepath.Join(typescript, "package.json")); err != nil {
		return result, fmt.Errorf("ui-health TypeScript parser is not installed: %w", err)
	}
	type source struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	input := struct {
		Shell      shellDeclaration `json:"shell"`
		Typescript string           `json:"typescript"`
		Files      []source         `json:"files"`
	}{Shell: shell, Typescript: typescript, Files: []source{}}
	for _, file := range ctx.Sources {
		if safeShellEjectionPath(file.RelPath) {
			input.Files = append(input.Files, source{file.RelPath, file.Content})
		}
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return result, err
	}
	timeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timeout, "node", "-e", shellOwnershipScanner)
	cmd.Stdin = bytes.NewReader(encoded)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return result, fmt.Errorf("JSX parser: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if err = json.Unmarshal(output, &result); err != nil {
		return result, fmt.Errorf("JSX parser response: %w", err)
	}
	for _, file := range input.Files {
		if strings.HasSuffix(file.Path, ".html") || strings.HasSuffix(file.Path, ".htm") {
			findings, err := analyzeShellHTML(file.Path, file.Content)
			if err != nil {
				return result, err
			}
			result.Findings = append(result.Findings, findings...)
		}
	}
	return result, nil
}

// Parse HTML rather than matching source text: comments, raw script contents and
// inert templates must not become evidence of application navigation.
func analyzeShellHTML(file, content string) ([]shellFinding, error) {
	root, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("HTML parser %s: %w", file, err)
	}
	var findings []shellFinding
	var visit func(*html.Node, bool, bool)
	visit = func(node *html.Node, contentScope, overlay bool) {
		if node.Type == html.ElementNode {
			if node.Data == "template" || node.Data == "script" || node.Data == "style" {
				return
			}
			role := ""
			for _, attr := range node.Attr {
				if attr.Key == "role" {
					role = attr.Val
				}
			}
			overlay = overlay || node.Data == "dialog" || role == "dialog" || role == "alertdialog"
			contentScope = contentScope || node.Data == "main" || role == "main"
			if !overlay && (role == "banner" || (node.Data == "header" && !contentScope && !htmlSectioningAncestor(node))) {
				findings = append(findings, shellFinding{File: file, Line: 1, Element: node.Data, Reason: "locally rendered application header in HTML"})
			}
			if !overlay && (node.Data == "nav" || role == "navigation") && (!contentScope || htmlNavigationLeavesPage(node)) {
				findings = append(findings, shellFinding{File: file, Line: 1, Element: node.Data, Reason: "locally rendered application navigation in HTML"})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child, contentScope, overlay)
		}
	}
	visit(root, false, false)
	return findings, nil
}

// Section anchors are page content. Navigating to a different URL remains
// application navigation even when its landmark is nested under main.
func htmlNavigationLeavesPage(node *html.Node) bool {
	if node.Type == html.ElementNode {
		if node.Data == "template" || node.Data == "script" || node.Data == "style" {
			return false
		}
		if node.Data == "a" {
			for _, attr := range node.Attr {
				value := strings.TrimSpace(attr.Val)
				if attr.Key == "href" && value != "" && !strings.HasPrefix(value, "#") {
					return true
				}
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if htmlNavigationLeavesPage(child) {
			return true
		}
	}
	return false
}

func htmlSectioningAncestor(node *html.Node) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type != html.ElementNode {
			continue
		}
		switch parent.Data {
		case "article", "aside", "nav", "section":
			return true
		}
	}
	return false
}
