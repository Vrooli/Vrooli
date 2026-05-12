// Package lint enforces the hand-authored replay-test contract for
// every temporal flow. It runs as part of `temporal-model check`:
//
//   - For each flow with a Go runtime, every *_test.go file in the
//     wrapper directory is scanned with go/ast. At least one test in
//     that directory must import the generated subpackage and call
//     <subpkg>.RunReplay(t, <non-trivial transition>).
//
//   - For each flow with a TypeScript runtime, every *.test.ts file
//     in the wrapper directory is scanned with a structural source
//     reader. At least one test must import runFormalReplay from
//     ./generated/<folder>/replay.helper, must import transition from
//     the wrapper module, must import fixtures from the fixtures
//     module, and must call runFormalReplay at module top level with
//     both transition and fixtures bound.
//
// Lint failures from check are fatal — there is no --no-lint flag.
package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"flow-verifier/internal/flows/layout"
	"flow-verifier/internal/flows/model"
)

// CheckAll runs the lint pass for every flow and returns a single
// aggregated error if any flow fails. Each per-flow failure includes
// the flow ID, the offending directory, and a shape diagram of what
// the lint expected.
func CheckAll(root string, flows []model.Flow) error {
	var failures []string
	for _, flow := range flows {
		if err := Check(root, flow); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) == 0 {
		return nil
	}
	sort.Strings(failures)
	return fmt.Errorf("temporal-model lint failed:\n%s", strings.Join(failures, "\n"))
}

// Check runs the lint pass for a single flow.
func Check(root string, flow model.Flow) error {
	switch flow.Layout.Language {
	case layout.LanguageGo:
		return checkGo(root, flow)
	case layout.LanguageTypeScript:
		return checkTypeScript(root, flow)
	default:
		return fmt.Errorf("%s: unsupported language %q", flow.FlowID, flow.Layout.Language)
	}
}

func listFiles(root string, baseDir string, suffix string) ([]string, error) {
	dir := filepath.Join(root, filepath.FromSlash(baseDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	sort.Strings(out)
	return out, nil
}

func shape(flow model.Flow, blockExample string) string {
	return fmt.Sprintf(
		"%s in %s:\n"+
			"  Expected a hand-authored test in %s that:\n"+
			"%s",
		flow.FlowID, flow.Layout.BaseDir, flow.Layout.BaseDir, blockExample,
	)
}
