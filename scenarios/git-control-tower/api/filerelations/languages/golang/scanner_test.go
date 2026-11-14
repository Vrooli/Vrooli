package golang

import (
	"context"
	"testing"

	"git-control-tower/filerelations/scanner"
)

func TestScannerScanExtractsSingleAndBlockImports(t *testing.T) {
	t.Parallel()

	result, err := New().Scan(context.Background(), `package main

import "fmt"

import (
	"net/http"
	"../local"
)
`, "main.go")
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	assertGoImport(t, result, "fmt", false)
	assertGoImport(t, result, "net/http", false)
	assertGoImport(t, result, "../local", true)
}

func TestScannerScanHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := New().Scan(ctx, `package main`, "main.go"); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func assertGoImport(t *testing.T, result *scanner.ScanResult, source string, isRelative bool) {
	t.Helper()

	for _, imp := range result.Imports {
		if imp.Source == source {
			if imp.IsRelative != isRelative {
				t.Fatalf("import %s IsRelative = %v, want %v", source, imp.IsRelative, isRelative)
			}
			if imp.Kind != scanner.ImportKindStatic {
				t.Fatalf("import %s Kind = %s, want static", source, imp.Kind)
			}
			return
		}
	}
	t.Fatalf("missing import %s in %#v", source, result.Imports)
}
