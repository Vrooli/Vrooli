package testutil

// This test documents the scripts-free census seam used by the production
// readiness baseline. The implementation intentionally remains a test helper:
// callers can add a deterministic AST walk without introducing a shell script
// or a production dependency on repository layout.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type ElevationInvocation struct {
	File string   `json:"file"`
	Line int      `json:"line"`
	Argv []string `json:"argv"`
}

type ElevationCensus struct {
	Root        string                `json:"root"`
	Invocations []ElevationInvocation `json:"invocations"`
}

func CensusElevations(root string) (ElevationCensus, error) {
	var report ElevationCensus
	report.Root = root
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			var argv []string
			for _, arg := range call.Args {
				lit, ok := arg.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					value = lit.Value
				}
				argv = append(argv, value)
			}
			if len(argv) == 0 {
				return true
			}
			first := argv[0]
			joined := strings.ToLower(strings.Join(argv, " "))
			if first == "sudo" || first == "pkexec" || first == "doas" || first == "runas" || (strings.Contains(joined, "start-process") && strings.Contains(joined, "-verb") && strings.Contains(joined, "runas")) {
				report.Invocations = append(report.Invocations, ElevationInvocation{File: path, Line: fset.Position(call.Pos()).Line, Argv: argv})
			}
			return true
		})
		return nil
	})
	sort.Slice(report.Invocations, func(i, j int) bool {
		if report.Invocations[i].File != report.Invocations[j].File {
			return report.Invocations[i].File < report.Invocations[j].File
		}
		return report.Invocations[i].Line < report.Invocations[j].Line
	})
	return report, err
}

func (c ElevationCensus) JSON() ([]byte, error) { return json.MarshalIndent(c, "", "  ") }

func TestElevationCensusHelperPlaceholder(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	report, err := CensusElevations(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := report.JSON(); err != nil {
		t.Fatal(err)
	}
}
