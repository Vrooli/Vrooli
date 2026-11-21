package ai

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

// TestBASGateRemovalMutationFails proves the existing route regression detects
// removal of the production ai-feature gate. The child test compiles the real
// package with a temporary Go overlay; repository source remains untouched.
func TestBASGateRemovalMutationFails(t *testing.T) {
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	productionPath := filepath.Join(workDir, "gateway_client.go")
	source, err := os.ReadFile(productionPath)
	if err != nil {
		t.Fatal(err)
	}
	gate := "ent := monetization.SnapshotFromContext(ctx)\n\t\tif ent == nil || !ent.HasFeatureForApp(\"browser-automation-studio\", entitlement.FeatureAI) {\n\t\t\treturn \"\", sharedv1.Profile_PROFILE_UNSPECIFIED, \"\", errors.New(\"BAS AI requires the signed session's ai feature entitlement when using Vrooli credits\")\n\t\t}"
	if count := strings.Count(string(source), gate); count != 2 {
		t.Fatalf("expected two production Vrooli-credit feature gates to mutate, found %d", count)
	}
	mutated := strings.ReplaceAll(string(source), gate, "")
	mutated = strings.Replace(mutated, "\t\"github.com/vrooli/browser-automation-studio/services/entitlement\"\n", "", 1)

	tmp := t.TempDir()
	mutatedPath := filepath.Join(tmp, "gateway_client.go")
	if err := os.WriteFile(mutatedPath, []byte(mutated), 0o600); err != nil {
		t.Fatal(err)
	}
	overlay, err := json.Marshal(struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{productionPath: mutatedPath}})
	if err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(tmp, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-overlay", overlayPath, "-run", "^TestBASVrooliCreditGateIsServerAuthoritativeAndKeepsGatewayMeter$", "-count=1")
	cmd.Dir = workDir
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("mutated route regression timed out: %v\n%s", ctx.Err(), output)
	}
	if runErr == nil {
		t.Fatalf("route regression passed after removing the production ai-feature gate; expected mutation to fail\n%s", output)
	}
	if !strings.Contains(string(output), "--- FAIL: TestBASVrooliCreditGateIsServerAuthoritativeAndKeepsGatewayMeter") ||
		!strings.Contains(string(output), "An error is expected but got nil") {
		t.Fatalf("mutated package failed outside the intended gate assertion: %v\n%s", runErr, output)
	}
}
