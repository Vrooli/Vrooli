package resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelativeResolverResolveFindsFilesAndIndexes(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeResolverFixture(t, repoRoot, "src/utils/format.ts")
	writeResolverFixture(t, repoRoot, "src/components/Button/index.ts")
	writeResolverFixture(t, repoRoot, "src/types/generated.d.ts")

	resolver := NewRelativeResolver()
	tests := []struct {
		name   string
		source string
		from   string
		want   string
	}{
		{name: "extension fallback", source: "../utils/format", from: "src/components/Button.tsx", want: "src/utils/format.ts"},
		{name: "directory index", source: "./Button", from: "src/components/App.tsx", want: "src/components/Button/index.ts"},
		{name: "exact path", source: "../types/generated.d.ts", from: "src/components/App.tsx", want: "src/types/generated.d.ts"},
		{name: "non-relative ignored", source: "react", from: "src/components/App.tsx", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.Resolve(tt.source, tt.from, repoRoot)
			if got != tt.want {
				t.Fatalf("Resolve(%q, %q) = %q, want %q", tt.source, tt.from, got, tt.want)
			}
		})
	}
}

func TestIsRelativeImport(t *testing.T) {
	t.Parallel()

	if !IsRelativeImport("./local") || !IsRelativeImport("../parent") {
		t.Fatal("expected ./ and ../ imports to be relative")
	}
	if IsRelativeImport(".hidden") || IsRelativeImport("react") {
		t.Fatal("expected non path-like imports to be non-relative")
	}
}

func TestTSConfigResolverResolvesPathsAndBaseURL(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeResolverFixture(t, repoRoot, "src/components/Button.tsx")
	writeResolverFixture(t, repoRoot, "src/lib/logger/index.ts")
	writeResolverFixture(t, repoRoot, "src/config.ts")
	writeResolverContent(t, repoRoot, "tsconfig.json", `{
  "compilerOptions": {
    "baseUrl": "src",
    "paths": {
      "@/*": ["./*"],
      "@lib/*": ["lib/*"]
    }
  }
}`)

	resolver := NewTSConfigResolver()
	tests := []struct {
		source string
		want   string
	}{
		{source: "@/components/Button", want: "src/components/Button.tsx"},
		{source: "@lib/logger", want: "src/lib/logger/index.ts"},
		{source: "config", want: "src/config.ts"},
		{source: "./relative", want: ""},
		{source: "missing/module", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := resolver.Resolve(tt.source, "src/App.tsx", repoRoot)
			if got != tt.want {
				t.Fatalf("Resolve(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}

	if got := resolver.GetBaseUrl(); got != "src" {
		t.Fatalf("baseUrl = %q, want src", got)
	}
	if got := resolver.GetPaths(); len(got) != 2 {
		t.Fatalf("paths = %#v, want 2 entries", got)
	}
}

func writeResolverFixture(t *testing.T, repoRoot, relPath string) {
	t.Helper()

	writeResolverContent(t, repoRoot, relPath, "fixture")
}

func writeResolverContent(t *testing.T, repoRoot, relPath, content string) {
	t.Helper()

	absPath := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(absPath), err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}
