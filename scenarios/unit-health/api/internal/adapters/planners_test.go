package adapters

import (
	"strings"
	"testing"
)

func TestDefaultPlannerRegistryPlansTypedCommands(t *testing.T) {
	r := DefaultPlannerRegistry()
	plan, err := r.Resolve(Facts{Language: "go", Framework: "go test"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Adapter.ID != "go" || plan.Test.Executable != "go" || len(plan.Test.Args) != 2 {
		t.Fatalf("plan=%+v", plan)
	}
	plan, err = r.Resolve(Facts{Language: "typescript", Framework: "vitest", PackageManager: "pnpm", CoverageScript: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Adapter.ID != "react-vitest" || plan.Coverage == nil || plan.Coverage.Args[1] != "test:coverage" || len(plan.Coverage.Artifacts) != 2 {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestDefaultPlannerRegistryRejectsUnsupportedHost(t *testing.T) {
	r := DefaultPlannerRegistry()
	if _, err := r.Resolve(Facts{Language: "bash", Framework: "bats", Platform: "windows"}); err == nil {
		t.Fatal("bats unexpectedly supported on Windows")
	}
	if _, err := r.Resolve(Facts{Language: "unknown", Framework: "unknown"}); err == nil {
		t.Fatal("unknown stack unexpectedly resolved")
	}
	plan, err := r.Resolve(Facts{Language: "rust", Framework: "cargo", Platform: "linux"})
	if err != nil || plan.Adapter.ID != "rust-cargo" || plan.Test.Executable != "cargo" {
		t.Fatalf("rust plan=%+v err=%v", plan, err)
	}
	plan, err = r.Resolve(Facts{Language: "powershell", Framework: "pester", Platform: "windows", TestPath: `C:\tests\sample.Tests.ps1`})
	if err != nil || plan.Adapter.ID != "powershell-pester" || len(plan.Test.Args) < 4 || plan.Test.Args[2] != "-File" {
		t.Fatalf("pester plan=%+v err=%v", plan, err)
	}
	plan, err = r.Resolve(Facts{Language: "python", Framework: "pytest", Platform: "windows"})
	if err != nil || plan.Test.Executable != "py" {
		t.Fatalf("Windows pytest plan=%+v err=%v; want py launcher", plan, err)
	}
}

func TestPlannerPreservesAdapterResolvedExecutable(t *testing.T) {
	plan, err := DefaultPlannerRegistry().Resolve(Facts{
		Language: "typescript", Framework: "vitest", PackageManager: "pnpm", Executable: "/tmp/tools/pnpm with spaces", CoverageScript: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Test.Executable != "/tmp/tools/pnpm with spaces" || plan.Coverage == nil || plan.Coverage.Executable != plan.Test.Executable {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestJestPlannerDeclaresNormalizedCoverageArtifacts(t *testing.T) {
	plan, err := DefaultPlannerRegistry().Resolve(Facts{Language: "typescript", Framework: "jest", PackageManager: "npm", CoverageScript: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Coverage == nil || len(plan.Coverage.Artifacts) != 2 || plan.Coverage.Artifacts[0].Kind != "istanbul-summary" || plan.Coverage.Artifacts[1].Kind != "lcov" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestPytestPlannerDeclaresCoberturaWhenCoverageIsAvailable(t *testing.T) {
	plan, err := DefaultPlannerRegistry().Resolve(Facts{Language: "python", Framework: "pytest", Platform: "linux", CoverageScript: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Coverage == nil || len(plan.Coverage.Artifacts) != 1 || plan.Coverage.Artifacts[0].Kind != "cobertura" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestCargoPlannerDeclaresLCOVWhenCargoLLVMcovIsAvailable(t *testing.T) {
	plan, err := DefaultPlannerRegistry().Resolve(Facts{Language: "rust", Framework: "cargo", Platform: "linux", CoverageScript: true, CoverageExecutable: "/tmp/cargo-llvm-cov with spaces"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Coverage == nil || plan.Coverage.Executable != "/tmp/cargo-llvm-cov with spaces" || len(plan.Coverage.Artifacts) != 1 || plan.Coverage.Artifacts[0].Kind != "lcov" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestFirstPartyPlannerConformanceProducesTypedCommands(t *testing.T) {
	cases := []struct {
		name  string
		facts Facts
		want  string
	}{
		{name: "go", facts: Facts{Language: "go", Framework: "go test"}, want: "go"},
		{name: "vitest", facts: Facts{Language: "typescript", Framework: "vitest", PackageManager: "pnpm"}, want: "react-vitest"},
		{name: "jest", facts: Facts{Language: "typescript", Framework: "jest", PackageManager: "npm"}, want: "node-jest"},
		{name: "pytest", facts: Facts{Language: "python", Framework: "pytest", Platform: "linux"}, want: "python-pytest"},
		{name: "cargo", facts: Facts{Language: "rust", Framework: "cargo", Platform: "linux"}, want: "rust-cargo"},
		{name: "bats", facts: Facts{Language: "bash", Framework: "bats", Platform: "linux", Executable: "/tmp/bats with spaces"}, want: "bash-bats"},
		{name: "pester", facts: Facts{Language: "powershell", Framework: "pester", Platform: "windows", TestPath: `C:\test path\sample.Tests.ps1`}, want: "powershell-pester"},
	}
	registry := DefaultPlannerRegistry()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := registry.Resolve(tc.facts)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Adapter.ID != tc.want || strings.TrimSpace(plan.Test.Executable) == "" || len(plan.Test.Args) == 0 {
				t.Fatalf("plan=%+v", plan)
			}
			if strings.ContainsAny(plan.Test.Display, ";&|`\n\r") {
				t.Fatalf("display command contains shell syntax: %q", plan.Test.Display)
			}
			for _, arg := range plan.Test.Args {
				if strings.Contains(arg, "&&") {
					t.Fatalf("argv contains shell composition: %+v", plan.Test.Args)
				}
			}
		})
	}
}

func TestGenericNodeAndJavaScriptJestUseTheNodeAdapter(t *testing.T) {
	for _, language := range []string{"javascript", "node"} {
		plan, err := DefaultPlannerRegistry().Resolve(Facts{Language: language, Framework: "jest", PackageManager: "npm"})
		if err != nil {
			t.Fatalf("language=%s: resolve: %v", language, err)
		}
		if plan.Adapter.ID != "node-jest" || plan.Test.Executable != "npm" || len(plan.Test.Args) != 2 {
			t.Fatalf("language=%s: plan=%+v", language, plan)
		}
	}
}
