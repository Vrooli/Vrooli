package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// startPreparedAllowlist names the only files permitted to hand the runner a
// pre-built snapshot instead of going through the registered input builder.
//
// The entry is not a convenience: plan.repair's start carries an operator input
// (maxRepairAttempts) that is not persisted on the backlog item, so its start
// cannot be reconstructed from durable state alone. Every other transition must
// use StartWith. Adding a file here means accepting that its start path and its
// registered builder can drift, which is exactly how six transitions ended up
// with placeholder builders that disabled the apply-time staleness guard.
var startPreparedAllowlist = map[string]string{
	"internal/backlog/plan_repair.go": "start carries operator-supplied maxRepairAttempts",
}

// TestPreBuiltTransitionInputsStayAllowlisted keeps the number of input paths
// per transition at one. A domain that builds its own snapshot and passes it to
// StartPrepared has two projections of the same subject, and nothing forces
// them to agree.
func TestPreBuiltTransitionInputsStayAllowlisted(t *testing.T) {
	root := filepath.Join(swarmScenarioRoot(t), "api")
	var offenders []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if strings.HasPrefix(rel, "internal/transitionrunner/") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !strings.Contains(string(source), "StartPrepared(") {
			return nil
		}
		if _, allowed := startPreparedAllowlist[filepath.ToSlash(rel)]; !allowed {
			offenders = append(offenders, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("these files pass a pre-built transition input without an allowlist entry: %s\n"+
			"Use StartWith so the registered input builder is the single projection, or add an entry to startPreparedAllowlist explaining why the start cannot be rebuilt from durable subject state.",
			strings.Join(offenders, ", "))
	}
}

// TestAllowlistedPreBuiltInputFilesExist stops the allowlist from outliving the
// code it excuses.
func TestAllowlistedPreBuiltInputFilesExist(t *testing.T) {
	root := filepath.Join(swarmScenarioRoot(t), "api")
	for rel, reason := range startPreparedAllowlist {
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("allowlisted file %s (%s) no longer exists; remove the entry", rel, reason)
			continue
		}
		if !strings.Contains(string(source), "StartPrepared(") {
			t.Errorf("allowlisted file %s no longer calls StartPrepared; remove the entry", rel)
		}
	}
}
