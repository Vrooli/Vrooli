package gates

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeTokenFallbackFixture(t *testing.T, root, config string) {
	t.Helper()
	manifestDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Fixture")
	versionDir := filepath.Join(manifestDir, "versions", "1.0.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "react-component-library", "catalog"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "component.json"), []byte(`{"catalogId":"fixture","libraryId":"react-component-library:Fixture","latest":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "Fixture.tsx"), []byte(`export const Fixture = () => <div style={{color: "var(--color-foreground, #111827)"}} />;`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTokenFallbackLiteralRejectsLiteral(t *testing.T) {
	root := t.TempDir()
	writeTokenFallbackFixture(t, root, `{"x-token-fallback-exemptions":[]}`)
	result, err := ValidateTokenFallbackLiteral(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.token-fallback-literal" {
		t.Fatalf("expected literal fallback finding, got %#v", result.Findings)
	}
}

func TestValidateTokenFallbackLiteralHonorsNamedExemption(t *testing.T) {
	root := t.TempDir()
	writeTokenFallbackFixture(t, root, `{"x-token-fallback-exemptions":[{"token":"--color-foreground","reason":"fixture exemption"}]}`)
	result, err := ValidateTokenFallbackLiteral(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected named exemption to clear finding, got %#v", result.Findings)
	}
}
