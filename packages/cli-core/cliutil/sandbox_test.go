package cliutil

import (
	"errors"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func repoContractLoader(t *testing.T) func() (*repocontract.Contract, string, error) {
	t.Helper()
	root, err := repoContractRoot()
	if err != nil {
		t.Fatalf("Abs(repo root) error = %v", err)
	}
	return func() (*repocontract.Contract, string, error) {
		contract, err := repocontract.LoadDefault(root)
		return contract, root, err
	}
}

func repoContractRoot() (string, error) {
	return filepath.Abs(filepath.Join("..", "..", ".."))
}

func TestDetectSandbox(t *testing.T) {
	t.Run("returns zero value when env vars not set", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_ID", "")
		t.Setenv("VROOLI_SANDBOX_MERGED", "")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "")

		sbx := DetectSandbox()
		if sbx.ID != "" || sbx.Merged != "" || sbx.Scope != "" {
			t.Fatalf("expected zero-value SandboxEnv, got %+v", sbx)
		}
	})

	t.Run("returns populated struct when env vars set", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_ID", "abc-123")
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/abc/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/my-scenario")

		sbx := DetectSandbox()
		if sbx.ID != "abc-123" {
			t.Errorf("ID = %q, want %q", sbx.ID, "abc-123")
		}
		if sbx.Merged != "/tmp/sandbox/abc/merged" {
			t.Errorf("Merged = %q, want %q", sbx.Merged, "/tmp/sandbox/abc/merged")
		}
		if sbx.Scope != "scenarios/my-scenario" {
			t.Errorf("Scope = %q, want %q", sbx.Scope, "scenarios/my-scenario")
		}
	})
}

func TestIsSandboxActive(t *testing.T) {
	tests := []struct {
		name   string
		merged string
		scope  string
		want   bool
	}{
		{"both set", "/tmp/merged", "scenarios/foo", true},
		{"merged empty", "", "scenarios/foo", false},
		{"scope empty means full repo", "/tmp/merged", "", true},
		{"both empty", "", "", false},
		// ID is optional and does not affect active status.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sbx := SandboxEnv{Merged: tt.merged, Scope: tt.scope}
			if got := sbx.IsSandboxActive(); got != tt.want {
				t.Errorf("IsSandboxActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSandboxEnvNormalizedScope(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  string
	}{
		{"empty", "", "."},
		{"dot", ".", "."},
		{"scenario", "scenarios/foo", "scenarios/foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (SandboxEnv{Scope: tt.scope}).NormalizedScope(); got != tt.want {
				t.Fatalf("NormalizedScope() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScenarioInScope(t *testing.T) {
	oldLoadContract := loadContract
	loadContract = repoContractLoader(t)
	t.Cleanup(func() { loadContract = oldLoadContract })

	tests := []struct {
		name         string
		scenarioName string
		scope        string
		want         bool
	}{
		// Empty/root scope — everything matches.
		{"empty scope", "foo", "", true},
		{"dot scope", "foo", ".", true},
		{"slash scope", "foo", "/", true},

		// "scenarios" scope — all scenarios match.
		{"scenarios scope", "foo", "scenarios", true},
		{"scenarios scope with trailing slash", "foo", "scenarios/", true},

		// Specific scenario scope — only that scenario matches.
		{"exact scenario match", "foo", "scenarios/foo", true},
		{"exact scenario with trailing slash", "foo", "scenarios/foo/", true},
		{"different scenario", "bar", "scenarios/foo", false},

		// Deeper sub-path — still matches the scenario.
		{"deeper sub-path api", "foo", "scenarios/foo/api", true},
		{"deeper sub-path ui/src", "foo", "scenarios/foo/ui/src", true},

		// Scope outside scenarios/ — nothing matches.
		{"packages scope", "foo", "packages/shared", false},
		{"root file scope", "foo", "go.mod", false},

		// Edge cases.
		{"scenario name prefix overlap", "foo-bar", "scenarios/foo", false},
		{"scenario name suffix overlap", "fo", "scenarios/foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScenarioInScope(tt.scenarioName, tt.scope)
			if got != tt.want {
				t.Errorf("ScenarioInScope(%q, %q) = %v, want %v",
					tt.scenarioName, tt.scope, got, tt.want)
			}
		})
	}
}

func TestResolveMergedPath(t *testing.T) {
	oldLoadContract := loadContract
	loadContract = repoContractLoader(t)
	t.Cleanup(func() { loadContract = oldLoadContract })

	merged := "/tmp/sandbox/abc/merged"

	tests := []struct {
		name         string
		scenarioName string
		scope        string
		want         string
	}{
		// Empty/root scope — merged/ is the project root.
		{"empty scope", "foo", "", filepath.Join(merged, "scenarios", "foo")},
		{"dot scope", "foo", ".", filepath.Join(merged, "scenarios", "foo")},
		{"slash scope", "foo", "/", filepath.Join(merged, "scenarios", "foo")},

		// "scenarios" scope — merged/ contains scenario dirs.
		{"scenarios scope", "foo", "scenarios", filepath.Join(merged, "foo")},
		{"scenarios scope trailing slash", "foo", "scenarios/", filepath.Join(merged, "foo")},

		// Exact scenario scope — merged/ IS the scenario.
		{"exact scenario scope", "foo", "scenarios/foo", merged},
		{"exact scenario scope trailing slash", "foo", "scenarios/foo/", merged},

		// Deeper scope — merged/ is a subdirectory of the scenario. This case
		// shouldn't normally happen (scope should be the scenario dir or broader),
		// but the fallback produces a safe path.
		{"deeper scope", "foo", "scenarios/foo/api", filepath.Join(merged, "scenarios", "foo")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveMergedPath(tt.scenarioName, tt.scope, merged)
			if got != tt.want {
				t.Errorf("ResolveMergedPath(%q, %q, %q) = %q, want %q",
					tt.scenarioName, tt.scope, merged, got, tt.want)
			}
		})
	}
}

func TestResolveScenarioPath(t *testing.T) {
	oldFindRepoRoot := findRepoRoot
	oldLoadContract := loadContract
	t.Cleanup(func() {
		findRepoRoot = oldFindRepoRoot
		loadContract = oldLoadContract
	})

	loadContract = repoContractLoader(t)
	root, err := repoContractRoot()
	if err != nil {
		t.Fatalf("repoContractRoot() error = %v", err)
	}

	t.Run("sandbox active and in scope returns merged path", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/my-app")
		findRepoRoot = func() (string, error) { return "/home/user/Vrooli", nil }

		got := ResolveScenarioPath("my-app")
		want := "/tmp/sandbox/merged"
		if got != want {
			t.Errorf("ResolveScenarioPath(%q) = %q, want %q", "my-app", got, want)
		}
	})

	t.Run("sandbox active but out of scope returns real path", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/my-app")
		findRepoRoot = func() (string, error) { return root, nil }

		got := ResolveScenarioPath("other-scenario")
		want := filepath.Join(root, "scenarios", "other-scenario")
		if got != want {
			t.Errorf("ResolveScenarioPath(%q) = %q, want %q", "other-scenario", got, want)
		}
	})

	t.Run("no sandbox falls back to VROOLI_ROOT", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "")
		findRepoRoot = func() (string, error) { return root, nil }

		got := ResolveScenarioPath("my-app")
		want := filepath.Join(root, "scenarios", "my-app")
		if got != want {
			t.Errorf("ResolveScenarioPath(%q) = %q, want %q", "my-app", got, want)
		}
	})

	t.Run("no sandbox with no repo root returns empty", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "")
		findRepoRoot = func() (string, error) { return "", errors.New("not found") }

		got := ResolveScenarioPath("my-app")
		if got != "" {
			t.Errorf("ResolveScenarioPath(%q) = %q, want empty path", "my-app", got)
		}
	})

	t.Run("broad scope redirects all scenarios", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios")
		findRepoRoot = func() (string, error) { return "/home/user/Vrooli", nil }

		got := ResolveScenarioPath("any-scenario")
		want := filepath.Join("/tmp/sandbox/merged", "any-scenario")
		if got != want {
			t.Errorf("ResolveScenarioPath(%q) = %q, want %q", "any-scenario", got, want)
		}
	})
}

func TestResolveRepoRoot(t *testing.T) {
	oldFindRepoRoot := findRepoRoot
	oldLoadContract := loadContract
	t.Cleanup(func() {
		findRepoRoot = oldFindRepoRoot
		loadContract = oldLoadContract
	})

	loadContract = repoContractLoader(t)
	root, err := repoContractRoot()
	if err != nil {
		t.Fatalf("repoContractRoot() error = %v", err)
	}

	t.Run("sandbox with root scope returns merged path", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", ".")
		findRepoRoot = func() (string, error) { return "/home/user/Vrooli", nil }

		got := ResolveRepoRoot()
		if got != "/tmp/sandbox/merged" {
			t.Errorf("ResolveRepoRoot() = %q, want %q", got, "/tmp/sandbox/merged")
		}
	})

	t.Run("sandbox with scenario scope returns real root", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/my-app")
		findRepoRoot = func() (string, error) { return root, nil }

		got := ResolveRepoRoot()
		if got != root {
			t.Errorf("ResolveRepoRoot() = %q, want %q", got, root)
		}
	})

	t.Run("no sandbox returns repo root", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "")
		findRepoRoot = func() (string, error) { return root, nil }

		got := ResolveRepoRoot()
		if got != root {
			t.Errorf("ResolveRepoRoot() = %q, want %q", got, root)
		}
	})

	t.Run("sandbox with empty scope returns merged path", func(t *testing.T) {
		t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/sandbox/merged")
		t.Setenv("VROOLI_SANDBOX_SCOPE", "")
		findRepoRoot = func() (string, error) { return root, nil }

		got := ResolveRepoRoot()
		if got != "/tmp/sandbox/merged" {
			t.Errorf("ResolveRepoRoot() = %q, want %q", got, "/tmp/sandbox/merged")
		}
	})
}
