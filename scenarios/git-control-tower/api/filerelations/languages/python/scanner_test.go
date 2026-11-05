package python

import (
	"context"
	"testing"

	"git-control-tower/filerelations/scanner"
)

func TestScannerScanExtractsImportForms(t *testing.T) {
	t.Parallel()

	result, err := New().Scan(context.Background(), `import os
import package.module, other.module
from .local import thing
from app.services import worker
`, "main.py")
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	assertPythonImport(t, result, "os", false)
	assertPythonImport(t, result, "package.module", false)
	assertPythonImport(t, result, "other.module", false)
	assertPythonImport(t, result, ".local", true)
	assertPythonImport(t, result, "app.services", false)
}

func TestScannerScanHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := New().Scan(ctx, `import os`, "main.py"); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func assertPythonImport(t *testing.T, result *scanner.ScanResult, source string, isRelative bool) {
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
