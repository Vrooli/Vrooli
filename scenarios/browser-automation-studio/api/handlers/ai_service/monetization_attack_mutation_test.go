package ai_service

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func apiModuleRoot(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("could not locate API module root")
		}
		root = parent
	}
}

// This mutation proves the full backend attack journey detects removal of
// BAS's signed features.ai gate. The overlay exists only in a temporary dir;
// production source is never rewritten.
func TestBASExecutableJourneyRejectsGateRemovalMutation(t *testing.T) {
	apiDir := apiModuleRoot(t)
	productionSource := filepath.Join(apiDir, "handlers", "ai", "gateway_client.go")
	source, err := os.ReadFile(productionSource)
	if err != nil {
		t.Fatal(err)
	}
	featureGate := "\t\tent := monetization.SnapshotFromContext(ctx)\n" +
		"\t\tif ent == nil || !ent.HasFeatureForApp(\"browser-automation-studio\", entitlement.FeatureAI) {\n" +
		"\t\t\treturn \"\", sharedv1.Profile_PROFILE_UNSPECIFIED, \"\", errors.New(\"BAS AI requires the signed session's ai feature entitlement when using Vrooli credits\")\n" +
		"\t\t}\n"
	if count := strings.Count(string(source), featureGate); count != 2 {
		t.Fatalf("found %d exact features.ai gate blocks; want 2 before mutation", count)
	}
	mutant := strings.ReplaceAll(string(source), featureGate, "")
	mutant = strings.Replace(mutant, "\t\"github.com/vrooli/browser-automation-studio/services/entitlement\"\n", "", 1)

	tmp := t.TempDir()
	mutantPath := filepath.Join(tmp, "gateway_client.go")
	if err := os.WriteFile(mutantPath, []byte(mutant), 0o600); err != nil {
		t.Fatal(err)
	}
	overlay, err := json.Marshal(map[string]any{"Replace": map[string]string{productionSource: mutantPath}})
	if err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(tmp, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-overlay", overlayPath, "-run", "^TestBASExecutableLeaseAttackJourney$", "-count=1")
	command.Dir = filepath.Join(apiDir, "handlers", "ai_service")
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("mutated integration test did not finish: %v\n%s", ctx.Err(), output)
	}
	if err == nil {
		t.Fatalf("production journey passed after removing both features.ai checks:\n%s", output)
	}
	if !strings.Contains(string(output), "--- FAIL: TestBASExecutableLeaseAttackJourney") ||
		!strings.Contains(string(output), "forged lease was accepted") {
		t.Fatalf("mutation did not fail at the forged-lease denial assertion:\n%s", output)
	}
}
