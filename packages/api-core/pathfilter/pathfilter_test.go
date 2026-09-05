package pathfilter

import "testing"

func TestSkipDir_ExplicitNames(t *testing.T) {
	t.Parallel()

	names := []string{
		// Build/deployment artifacts
		"platforms", "dist", "build", "bin", "bundle", "artifacts",
		// Dependencies
		"node_modules", "vendor",
		// Runtime data
		"data", "logs", "coverage", "playwright-driver",
		// Language caches
		"__pycache__", "target", "obj",
		// Temporary
		"tmp", "temp", "storybook-static",
		// Python venvs
		"venv",
	}
	for _, name := range names {
		if !SkipDir(name) {
			t.Errorf("SkipDir(%q) = false, want true", name)
		}
	}
}

func TestSkipDir_DotDirectories(t *testing.T) {
	t.Parallel()

	names := []string{
		".git", ".cache", ".vrooli", ".claude", ".pnpm-store",
		".vscode", ".idea", ".next", ".vite", ".nuxt",
		".venv", ".pytest_cache", ".nyc_output", ".husky", ".turbo",
		".DS_Store",
	}
	for _, name := range names {
		if !SkipDir(name) {
			t.Errorf("SkipDir(%q) = false, want true", name)
		}
	}
}

func TestSkipDir_SourceDirsNotSkipped(t *testing.T) {
	t.Parallel()

	names := []string{
		"api", "ui", "cli", "bas", "docs",
		"requirements", "test", "tests", "prompts", "scripts",
		"chore", "execute", "fix", "ideas", "research", "captures",
		"src", "sidecar", "schemas", "config", "support-agent-docs",
	}
	for _, name := range names {
		if SkipDir(name) {
			t.Errorf("SkipDir(%q) = true, want false", name)
		}
	}
}

func TestSkipDir_EmptyString(t *testing.T) {
	t.Parallel()

	if SkipDir("") {
		t.Error("SkipDir(\"\") = true, want false")
	}
}

func TestSkipDir_CaseSensitive(t *testing.T) {
	t.Parallel()

	names := []string{"Node_Modules", "Dist", "BUILD", "Vendor", "PLATFORMS"}
	for _, name := range names {
		if SkipDir(name) {
			t.Errorf("SkipDir(%q) = true, want false (case-sensitive)", name)
		}
	}
}

func TestSkipDirSet_ReturnsCopy(t *testing.T) {
	t.Parallel()

	s1 := SkipDirSet()
	s1["should_not_persist"] = true

	s2 := SkipDirSet()
	if s2["should_not_persist"] {
		t.Error("SkipDirSet did not return independent copies")
	}
}

func TestIsSourceExt_Recognized(t *testing.T) {
	t.Parallel()

	exts := []string{
		".go", ".ts", ".tsx", ".js", ".jsx", ".py",
		".rs", ".rb", ".java", ".kt", ".cs", ".swift",
	}
	for _, ext := range exts {
		if !IsSourceExt(ext) {
			t.Errorf("IsSourceExt(%q) = false, want true", ext)
		}
	}
}

func TestIsSourceExt_NotRecognized(t *testing.T) {
	t.Parallel()

	exts := []string{".md", ".json", ".yaml", ".yml", ".html", ".css", ".sql", ".sh", ""}
	for _, ext := range exts {
		if IsSourceExt(ext) {
			t.Errorf("IsSourceExt(%q) = true, want false", ext)
		}
	}
}

func TestSourceExts_ReturnsCopy(t *testing.T) {
	t.Parallel()

	s1 := SourceExts()
	s1[".fake"] = true

	s2 := SourceExts()
	if s2[".fake"] {
		t.Error("SourceExts did not return independent copies")
	}
}

func TestIsGeneratedFile_Protobuf(t *testing.T) {
	t.Parallel()

	names := []string{"user.pb.go", "user_pb.go", "user_pb2.go"}
	for _, name := range names {
		if !IsGeneratedFile(name) {
			t.Errorf("IsGeneratedFile(%q) = false, want true", name)
		}
	}
}

func TestIsGeneratedFile_Codegen(t *testing.T) {
	t.Parallel()

	names := []string{"types_gen.go", "schema_generated.go"}
	for _, name := range names {
		if !IsGeneratedFile(name) {
			t.Errorf("IsGeneratedFile(%q) = false, want true", name)
		}
	}
}

func TestIsGeneratedFile_Mock(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want bool
	}{
		{"mock_service.go", true},
		{"service_mock.go", true},
		{"mock_repo.go", true},
		{"mocking.go", false},
	}
	for _, tc := range cases {
		if got := IsGeneratedFile(tc.name); got != tc.want {
			t.Errorf("IsGeneratedFile(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsGeneratedFile_TypeScriptDecl(t *testing.T) {
	t.Parallel()

	if !IsGeneratedFile("index.d.ts") {
		t.Error("IsGeneratedFile(\"index.d.ts\") = false, want true")
	}
	if IsGeneratedFile("index.ts") {
		t.Error("IsGeneratedFile(\"index.ts\") = true, want false")
	}
}

func TestIsGeneratedFile_Minified(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want bool
	}{
		{"app.min.js", true},
		{"styles.min.css", true},
		{"app.js", false},
		{"styles.css", false},
	}
	for _, tc := range cases {
		if got := IsGeneratedFile(tc.name); got != tc.want {
			t.Errorf("IsGeneratedFile(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsGeneratedFile_NormalFiles(t *testing.T) {
	t.Parallel()

	names := []string{"main.go", "App.tsx", "utils.py", "server.js", "handler.go"}
	for _, name := range names {
		if IsGeneratedFile(name) {
			t.Errorf("IsGeneratedFile(%q) = true, want false", name)
		}
	}
}

func BenchmarkSkipDir(b *testing.B) {
	names := []string{"api", "node_modules", ".git", "platforms", "src", "vendor"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SkipDir(names[i%len(names)])
	}
}
