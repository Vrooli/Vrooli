package invokers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// DocPath is the reference page rendered from the registry.
const DocPath = "docs/reference/cli-invokers.md"

// RenderDoc renders the registry as the reference page. The doc test fails
// when the committed page differs from this rendering.
func RenderDoc() (string, error) {
	items, err := All()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# CLI invokers\n\n")
	b.WriteString("Every process that builds an argv for the `vrooli` CLI is registered in\n")
	b.WriteString("`internal/cli/rootcli/invokers`. The test `TestEveryInvokerResolvesThroughTheRootParser`\n")
	b.WriteString("parses each argv below through the real root parser and fails on an unknown\n")
	b.WriteString("command or on a retired global flag, so a change to the CLI's argv contract\n")
	b.WriteString("fails a test instead of a boot. `TestEveryDirectSpawnIsRegistered` fails when a\n")
	b.WriteString("Go file builds a `cliinvoke.Invocation` without being a registered owner.\n\n")
	b.WriteString("This page is generated: run `go test ./internal/cli/rootcli/invokers -run TestDocIsCurrent -update`\n")
	b.WriteString("after changing the registry.\n\n")
	b.WriteString("| Invoker | Owner | Argv | Resolves to |\n|---|---|---|---|\n")
	for _, item := range items {
		argv, err := item.Argv()
		if err != nil {
			return "", fmt.Errorf("%s: %w", item.Name, err)
		}
		res, err := rootcli.ResolveArgv(argv)
		resolved := "unknown"
		if err == nil {
			resolved = strings.TrimSpace(res.Command + " " + res.Subcommand)
		}
		fmt.Fprintf(&b, "| `%s` | `%s` | `vrooli %s` | `%s` |\n", item.Name, item.Owner, strings.Join(argv, " "), resolved)
	}
	b.WriteString("\n## Runners\n\nThese files execute an invocation whose argv a registered owner supplies; they produce no argv of their own.\n\n")
	runners := append([]string(nil), Runners...)
	sort.Strings(runners)
	for _, runner := range runners {
		fmt.Fprintf(&b, "- `%s`\n", runner)
	}
	b.WriteString("\n## Scenario-level callers not yet migrated\n\n")
	b.WriteString("These scenarios spawn `vrooli` with their own resolution and are outside the\n")
	b.WriteString("registry; migrating them onto `repo-contract-go/cliinvoke` is a follow-up.\n\n")
	callers, err := DeferredCallers()
	if err != nil {
		return "", err
	}
	b.WriteString("| Scenario | File:line | Call |\n|---|---|---|\n")
	for _, caller := range callers {
		fmt.Fprintf(&b, "| `%s` | `%s:%d` | `%s` |\n", caller.Scenario, caller.File, caller.Line, caller.Call)
	}
	return b.String(), nil
}

// DeferredCaller is a scenario-level `vrooli` spawn left for a follow-up.
type DeferredCaller struct {
	Scenario string
	File     string
	Line     int
	Call     string
}

var adHocSpawn = regexp.MustCompile(`exec\.Command(Context)?\([^)]*"vrooli"[^)]*\)`)

// DeferredCallers is the census of ad-hoc `vrooli` spawns under scenarios/
// that do not go through the invoker seam. It is computed from the tree so
// the reference page cannot drift from the code.
func DeferredCallers() ([]DeferredCaller, error) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return nil, err
	}
	var items []DeferredCaller
	scenarioDirs, _ := filepath.Glob(filepath.Join(root, "scenarios", "*"))
	for _, dir := range scenarioDirs {
		scenario := filepath.Base(dir)
		if scenario == "vrooli-autoheal" {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				switch d.Name() {
				case "node_modules", "vendor", "testdata", "ui", ".git":
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if match := adHocSpawn.FindString(line); match != "" {
					rel, _ := filepath.Rel(root, path)
					items = append(items, DeferredCaller{Scenario: scenario, File: filepath.ToSlash(rel), Line: i + 1, Call: match})
				}
			}
			return nil
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].File != items[j].File {
			return items[i].File < items[j].File
		}
		return items[i].Line < items[j].Line
	})
	return items, nil
}
