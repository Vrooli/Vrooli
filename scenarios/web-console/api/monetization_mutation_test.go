package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Removing the local Vrooli-credits session check must make the executable
// journey fail: a cached lease/account hint alone cannot authorize metered AI.
func TestWebConsoleExecutableJourneyRejectsSessionGateRemovalMutation(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve mutation test location")
	}
	apiDir := filepath.Dir(testFile)
	productionSource := filepath.Join(apiDir, "internal", "ai", "gateway.go")
	source, err := os.ReadFile(productionSource)
	if err != nil {
		t.Fatal(err)
	}
	gate := "\t\tif consumerToken(ctx) == \"\" {\n" +
		"\t\t\treturn \"\", fmt.Errorf(\"%w: Vrooli credits require a signed-in subscription session\", ErrCreditsRequired)\n" +
		"\t\t}\n"
	if count := strings.Count(string(source), gate); count != 1 {
		t.Fatalf("found %d exact Vrooli-credits session gates; want 1 before mutation", count)
	}
	mutant := strings.Replace(string(source), gate, "", 1)
	tmp := t.TempDir()
	mutantPath := filepath.Join(tmp, "gateway.go")
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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "-overlay", overlayPath, "-run", "^TestWebConsoleOfflineLeaseCannotGrantMeteredAI$", "-count=1")
	command.Dir = apiDir
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("mutated integration test did not finish: %v\n%s", ctx.Err(), output)
	}
	if err == nil {
		t.Fatalf("production journey passed after removing the credits session gate:\n%s", output)
	}
	if !strings.Contains(string(output), "--- FAIL: TestWebConsoleOfflineLeaseCannotGrantMeteredAI") ||
		!strings.Contains(string(output), "offline cached lease plus local identity hint decision") {
		t.Fatalf("mutation did not fail at the unauthenticated paid-call assertion:\n%s", output)
	}
}
