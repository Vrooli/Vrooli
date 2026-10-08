package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestDefaultServiceCompilerCompile(t *testing.T) {
	compiler := &defaultServiceCompiler{
		platform: &defaultPlatformResolver{},
		fileOps:  &defaultFileOperations{},
	}

	t.Run("unsupported build type", func(t *testing.T) {
		svc := bundlemanifest.Service{
			ID: "test-svc",
			Build: &bundlemanifest.BuildConfig{
				Type: "unsupported",
			},
		}
		_, err := compiler.Compile(svc, "linux-amd64", "/tmp")
		if err == nil {
			t.Error("expected error for unsupported build type")
		}
	})
}

func TestFindNpmEntryPointPrefersRecognizedServerEntrypoints(t *testing.T) {
	dist := t.TempDir()
	if _, err := findNpmEntryPoint(dist); err == nil {
		t.Fatal("expected an error when dist has no supported entrypoint")
	}
	if err := os.WriteFile(filepath.Join(dist, "main.js"), []byte("main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := findNpmEntryPoint(dist); err != nil || got != "main.js" {
		t.Fatalf("findNpmEntryPoint() = %q, %v; want main.js, nil", got, err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.js"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "server.js"), []byte("server"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := findNpmEntryPoint(dist); err != nil || got != "server.js" {
		t.Fatalf("findNpmEntryPoint() = %q, %v; want server.js, nil", got, err)
	}
}

func TestAssembleNpmOutputProducesPortableLaunchers(t *testing.T) {
	src := t.TempDir()
	dist := filepath.Join(src, "dist")
	if err := os.MkdirAll(filepath.Join(dist, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "server.js"), []byte("console.log('ready')"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "nested", "asset.txt"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "node_modules", "example"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "node_modules", "example", "index.js"), []byte("module"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "package.json"), []byte(`{"name":"desktop-service"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "bundle")
	if err := assembleNpmOutput(src, out, dist, "server.js", "windows"); err != nil {
		t.Fatalf("assembleNpmOutput: %v", err)
	}
	for _, path := range []string{"dist/server.js", "dist/nested/asset.txt", "node_modules/example/index.js", "package.json", "run.sh", "run.cmd"} {
		if _, err := os.Stat(filepath.Join(out, path)); err != nil {
			t.Fatalf("expected packaged %s: %v", path, err)
		}
	}
	runSH, err := os.ReadFile(filepath.Join(out, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runSH), "exec node dist/server.js \"$@\"") {
		t.Fatalf("run.sh does not invoke the selected entrypoint: %s", runSH)
	}
	runCMD, err := os.ReadFile(filepath.Join(out, "run.cmd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(runCMD), `node dist\server.js %*`) {
		t.Fatalf("run.cmd does not invoke the selected entrypoint: %s", runCMD)
	}
}

func TestCompileCustomBinaryRequiresCommand(t *testing.T) {
	err := compileCustomBinary(t.TempDir(), filepath.Join(t.TempDir(), "output"), "linux", "amd64", &bundlemanifest.BuildConfig{})
	if err == nil || !strings.Contains(err.Error(), "requires args") {
		t.Fatalf("compileCustomBinary() error = %v, want missing-command error", err)
	}
}

func TestDefaultServiceCompilerCompilesGoAndCustomServices(t *testing.T) {
	compiler := &defaultServiceCompiler{platform: &defaultPlatformResolver{}, fileOps: &defaultFileOperations{}}

	t.Run("go service", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "service")
		if err := os.MkdirAll(src, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, "go.mod"), []byte("module fixture\n\ngo 1.22\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := compiler.Compile(bundlemanifest.Service{ID: "fixture", Build: &bundlemanifest.BuildConfig{Type: "go", SourceDir: "service", EntryPoint: "."}}, "linux-amd64", root)
		if err != nil {
			t.Fatalf("compile Go service: %v", err)
		}
		if info, err := os.Stat(out); err != nil || info.Size() == 0 {
			t.Fatalf("compiled output = %q, %v", out, err)
		}
	})

	t.Run("custom service substitutes output placeholder", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "service"), 0o755); err != nil {
			t.Fatal(err)
		}
		out, err := compiler.Compile(bundlemanifest.Service{ID: "custom", Build: &bundlemanifest.BuildConfig{
			Type: "custom", SourceDir: "service", OutputPattern: "out/{{platform}}/artifact", Args: []string{"sh", "-c", "printf built > \"$1\"", "ignored", "{{output}}"},
		}}, "linux-amd64", root)
		if err != nil {
			t.Fatalf("compile custom service: %v", err)
		}
		data, err := os.ReadFile(out)
		if err != nil || string(data) != "built" {
			t.Fatalf("custom output = %q, %v", data, err)
		}
	})
}

func TestCompilerRejectsMissingSourceAndUnknownRuntimeTarget(t *testing.T) {
	compiler := &defaultServiceCompiler{platform: &defaultPlatformResolver{}, fileOps: &defaultFileOperations{}}
	if _, err := compiler.Compile(bundlemanifest.Service{ID: "missing", Build: &bundlemanifest.BuildConfig{Type: "go", SourceDir: "missing"}}, "linux-amd64", t.TempDir()); err == nil {
		t.Fatal("expected missing source error")
	}
	if err := (&defaultRuntimeBuilder{}).Build(t.TempDir(), filepath.Join(t.TempDir(), "runtime"), "linux", "amd64", "unknown"); err == nil {
		t.Fatal("expected unknown runtime target error")
	}
}

func TestRustTarget(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		goarch    string
		expected  string
		expectErr bool
	}{
		{
			name:     "linux amd64",
			goos:     "linux",
			goarch:   "amd64",
			expected: "x86_64-unknown-linux-gnu",
		},
		{
			name:     "linux arm64",
			goos:     "linux",
			goarch:   "arm64",
			expected: "aarch64-unknown-linux-gnu",
		},
		{
			name:     "darwin amd64",
			goos:     "darwin",
			goarch:   "amd64",
			expected: "x86_64-apple-darwin",
		},
		{
			name:     "darwin arm64",
			goos:     "darwin",
			goarch:   "arm64",
			expected: "aarch64-apple-darwin",
		},
		{
			name:     "windows amd64",
			goos:     "windows",
			goarch:   "amd64",
			expected: "x86_64-pc-windows-msvc",
		},
		{
			name:     "windows arm64",
			goos:     "windows",
			goarch:   "arm64",
			expected: "aarch64-pc-windows-msvc",
		},
		{
			name:      "unsupported os",
			goos:      "freebsd",
			goarch:    "amd64",
			expectErr: true,
		},
		{
			name:      "unsupported arch",
			goos:      "linux",
			goarch:    "386",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rustTarget(tt.goos, tt.goarch)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("rustTarget(%q, %q) = %q, want %q", tt.goos, tt.goarch, result, tt.expected)
			}
		})
	}
}
