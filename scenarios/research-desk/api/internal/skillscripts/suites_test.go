package skillscripts

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// The research skills' scripts are what agents run when every service is
// down, so their own test suites are part of this scenario's gate: a script
// suite nobody runs stops describing the script.
func TestSkillScriptSuitesPass(t *testing.T) {
	suites, err := filepath.Glob(filepath.Join(skillsDir, "*", "scripts", "test_*.py"))
	if err != nil {
		t.Fatal(err)
	}
	if len(suites) == 0 {
		t.Fatalf("no skill script suites under %s", skillsDir)
	}
	for _, suite := range suites {
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(suite)))+"/"+filepath.Base(suite), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "python3", "-I", filepath.Base(suite))
			cmd.Dir = filepath.Dir(suite)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s failed: %v\n%s", suite, err, out)
			}
		})
	}
}
