package accel_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Feature: internal/accel never runs a command
//
//	As the control plane
//	I want accelerator truth read through one host-inventory seam
//	So that a second copy of vendor-tool logic cannot grow here under time
//	pressure, and every test can run on a host with no accelerator.

// forbiddenImports are packages that would let this package execute a command
// or read a device directly. internal/hostinventory owns vendor-tool calls, and
// container probes arrive through an injected ContainerProbe.
var forbiddenImports = []string{
	"os/exec",
	"syscall",
	"golang.org/x/sys/unix",
	"github.com/vrooli/vrooli/internal/gpuaccess",
}

// Scenario: the package imports nothing that can execute a command.
func TestAccelPackageExecutesNothing(t *testing.T) {
	// Given every Go file in internal/accel, build tags included
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	fileSet := token.NewFileSet()
	inspected := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		// When its import block is read
		parsed, err := parser.ParseFile(fileSet, filepath.Join(".", entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		inspected++
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("%s: unquote import %s: %v", entry.Name(), spec.Path.Value, err)
			}
			// Then no forbidden package appears
			for _, forbidden := range forbiddenImports {
				if path == forbidden {
					t.Fatalf("%s imports %q; internal/accel must reach the host only through hostinventory and the injected ContainerProbe", entry.Name(), forbidden)
				}
			}
		}
	}

	// And the check actually inspected the package rather than silently
	// finding nothing to look at
	if inspected == 0 {
		t.Fatal("no non-test Go files were inspected; the import check would pass vacuously")
	}
	t.Logf("inspected %d files for forbidden imports", inspected)
}

// Scenario: no file in the package calls a vendor tool by name.
func TestAccelPackageNamesNoVendorTool(t *testing.T) {
	// Given the vendor tools only hostinventory may invoke
	vendorTools := []string{"nvidia-smi", "rocm-smi", "system_profiler", "wmic", "docker exec"}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		source, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, entry.Name(), source, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}

		// When each string literal is read
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			// Then none of them is a command this package could shell out to.
			// A tool named inside an explanatory message is fine; a tool name
			// that is the whole literal is a command waiting to happen.
			for _, tool := range vendorTools {
				if value == tool {
					t.Errorf("%s contains the bare vendor-tool literal %q; route the call through internal/hostinventory", entry.Name(), tool)
				}
			}
			return true
		})
	}
}
