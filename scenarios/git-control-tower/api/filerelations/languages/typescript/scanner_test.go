package typescript

import (
	"context"
	"testing"

	"git-control-tower/filerelations/scanner"
)

func TestScannerScanExtractsImportsAndReExports(t *testing.T) {
	t.Parallel()

	result, err := New().Scan(context.Background(), `
import React from "react";
import { helper } from "./helper";
import "./styles.css";
const mod = await import("../dynamic");
export { Widget } from "./Widget";
`, "src/App.tsx")
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	assertTSImport(t, result, "react", false, scanner.ImportKindStatic)
	assertTSImport(t, result, "./helper", true, scanner.ImportKindStatic)
	assertTSImport(t, result, "./styles.css", true, scanner.ImportKindSideEffect)
	assertTSImport(t, result, "../dynamic", true, scanner.ImportKindDynamic)

	if len(result.Exports) != 1 {
		t.Fatalf("exports = %#v, want 1 export", result.Exports)
	}
	if result.Exports[0].Source != "./Widget" || !result.Exports[0].IsRelative {
		t.Fatalf("export = %#v, want relative ./Widget", result.Exports[0])
	}
}

func TestScannerScanHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := New().Scan(ctx, `import React from "react";`, "App.tsx"); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func assertTSImport(t *testing.T, result *scanner.ScanResult, source string, isRelative bool, kind scanner.ImportKind) {
	t.Helper()

	for _, imp := range result.Imports {
		if imp.Source == source {
			if imp.IsRelative != isRelative {
				t.Fatalf("import %s IsRelative = %v, want %v", source, imp.IsRelative, isRelative)
			}
			if imp.Kind != kind {
				t.Fatalf("import %s Kind = %s, want %s", source, imp.Kind, kind)
			}
			return
		}
	}
	t.Fatalf("missing import %s in %#v", source, result.Imports)
}
