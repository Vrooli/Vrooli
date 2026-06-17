package facts

import (
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func writeFileFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func unitByLanguage(units []*factsv1.ParseUnit, lang string) []*factsv1.ParseUnit {
	var out []*factsv1.ParseUnit
	for _, u := range units {
		if u.GetLanguage() == lang {
			out = append(out, u)
		}
	}
	return out
}

func TestDiscoverNestedParseUnitsEmitsBashUnit(t *testing.T) {
	root := t.TempDir()
	// Shell scripts in a manifest-less tree → one collapsed bash unit at cli/.
	writeFileFixture(t, filepath.Join(root, "cli", "run.sh"), "#!/usr/bin/env bash\necho hi\n")
	writeFileFixture(t, filepath.Join(root, "cli", "sub", "helper.sh"), "echo helper\n")
	writeFileFixture(t, filepath.Join(root, "test", "smoke.bats"), "@test \"ok\" { true; }\n")
	// Shell scripts inside a Go module dir must NOT produce a bash unit.
	writeFileFixture(t, filepath.Join(root, "api", "go.mod"), "module x\n")
	writeFileFixture(t, filepath.Join(root, "api", "build.sh"), "go build ./...\n")

	units := discoverNestedParseUnits(root)

	bash := unitByLanguage(units, "bash")
	roots := map[string]bool{}
	for _, u := range bash {
		roots[u.GetRootPath()] = true
		if u.GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED {
			t.Errorf("bash unit %q status = %v, want UNSUPPORTED", u.GetRootPath(), u.GetStatus())
		}
	}
	if !roots[filepath.Join(root, "cli")] {
		t.Errorf("expected a bash unit at cli/, got roots=%v", roots)
	}
	if !roots[filepath.Join(root, "test")] {
		t.Errorf("expected a bash unit at test/ (.bats), got roots=%v", roots)
	}
	if roots[filepath.Join(root, "cli", "sub")] {
		t.Errorf("nested cli/sub should collapse into cli/, got roots=%v", roots)
	}
	if roots[filepath.Join(root, "api")] {
		t.Errorf("api/ has go.mod and must not become a bash unit, got roots=%v", roots)
	}
	if len(unitByLanguage(units, "go")) != 1 {
		t.Errorf("expected the Go unit to still be discovered, units=%v", units)
	}
}
