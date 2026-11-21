package scanner

import (
	"context"
	"testing"
)

type fakeImportScanner struct {
	extensions []string
}

func (f fakeImportScanner) Extensions() []string {
	return f.extensions
}

func (f fakeImportScanner) Scan(context.Context, string, string) (*ScanResult, error) {
	return &ScanResult{}, nil
}

func TestRegistryReturnsScannerByCaseInsensitiveExtension(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	fake := fakeImportScanner{extensions: []string{".ts", ".tsx"}}
	registry.Register(fake)

	if registry.Get("src/App.TSX") == nil {
		t.Fatal("expected scanner for TSX extension")
	}
	if !registry.HasScanner("src/model.ts") {
		t.Fatal("expected scanner for TS extension")
	}
	if registry.Get("README.md") != nil {
		t.Fatal("expected no scanner for markdown")
	}
}
