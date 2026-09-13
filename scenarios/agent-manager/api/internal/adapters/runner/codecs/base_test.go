package codecs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestBaseCodecAvailabilityAndProbeReflectFilesystemState(t *testing.T) {
	base := baseCodec{runnerType: domain.RunnerTypeCodex, binaryDesc: "Codex", installHint: "install codex"}
	missing := resolveBinary(base, "agent-manager-definitely-not-a-binary")
	if available, message := missing.Available(context.Background()); available || !strings.Contains(message, "install codex") {
		t.Fatalf("missing binary availability = %v, %q", available, message)
	}
	if err := missing.ProbeModel(context.Background(), "any"); err == nil {
		t.Fatal("probe accepted unavailable codec")
	}
	stub := testBase(base, "/test/codex", "test codec unavailable")
	if ok, message := stub.Available(context.Background()); ok || message != "test codec unavailable" {
		t.Fatalf("test base availability = %v, %q", ok, message)
	}

	available := baseCodec{binaryPath: t.TempDir() + "/removed", available: true, binaryDesc: "Codex", installHint: "install codex", runnerType: domain.RunnerTypeCodex}
	if ok, _ := available.Available(context.Background()); ok {
		t.Fatal("deleted binary path reported available")
	}
	resolved := resolveBinary(base, "sh")
	if ok, message := resolved.Available(context.Background()); !ok || message != "Codex is available" {
		t.Fatalf("resolved binary availability = %v, %q", ok, message)
	}
	if err := resolved.ProbeModel(context.Background(), "any"); err != nil {
		t.Fatalf("available probe failed: %v", err)
	}
}

func TestBaseCodecIdentityAndBuildEnvironmentContracts(t *testing.T) {
	id := uuid.New()
	base := baseCodec{
		binaryPath: "codex-bin", available: true, runnerType: domain.RunnerTypeCodex,
		binaryDesc: "Codex", tagEnvKey: "CODEX_AGENT_TAG", continuePrefix: "codex", labels: Labels{StartMessage: "starting"},
	}
	if base.Type() != domain.RunnerTypeCodex || base.BinaryPath() != "codex-bin" || base.BinaryDescription() != "Codex" || base.TagEnvKey() != "CODEX_AGENT_TAG" || base.Labels().StartMessage != "starting" {
		t.Fatalf("identity contract broken: %+v", base)
	}
	if got := base.ContinueTag(runner.ContinueRequest{RunID: id}); got != "codex-continue-"+id.String()[:8] {
		t.Fatalf("continue tag = %q", got)
	}
	if base.OnEarlyTerminate(nil, "") {
		t.Fatal("base codec should drain after early termination")
	}
	base.PostClassify(nil, nil)

	env := standardBuildEnv("CODEX_AGENT_TAG", "run-tag", map[string]string{"EXTRA": "value", " ": "ignored"}, "FIXED=yes")
	joined := strings.Join(env, "\n")
	for _, entry := range []string{"CODEX_AGENT_TAG=run-tag", "FIXED=yes", "EXTRA=value"} {
		if !strings.Contains(joined, entry) {
			t.Fatalf("environment missing %q: %v", entry, env)
		}
	}
	if strings.Contains(joined, " =ignored") {
		t.Fatalf("blank environment key leaked: %v", env)
	}
}

func TestContinuationRetainsCanonicalProcessTag(t *testing.T) {
	base := opencodeBase()
	if got := base.ContinueTag(runner.ContinueRequest{RunID: uuid.New(), Tag: "effort/platform"}); got != "effort/platform" {
		t.Fatalf("reconciler cannot find continuation tag %q; want canonical effort/platform", got)
	}
}

func TestBaseCodecRuntimeVersionReadsFirstLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-cli")
	script := "#!/bin/sh\necho '1.18.30'\necho 'second line ignored'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	base := baseCodec{binaryPath: path, available: true, binaryDesc: "fake CLI", runnerType: domain.RunnerTypeOpenCode}

	got, err := base.RuntimeVersion(context.Background())
	if err != nil {
		t.Fatalf("RuntimeVersion error = %v", err)
	}
	if got != "1.18.30" {
		t.Fatalf("RuntimeVersion = %q, want 1.18.30", got)
	}
}

func TestBaseCodecRuntimeVersionFailsClosed(t *testing.T) {
	unavailable := baseCodec{binaryDesc: "fake CLI", runnerType: domain.RunnerTypeOpenCode}
	if _, err := unavailable.RuntimeVersion(context.Background()); err == nil {
		t.Fatal("unavailable codec reported a runtime version")
	}

	failing := baseCodec{binaryPath: filepath.Join(t.TempDir(), "missing"), available: true, binaryDesc: "fake CLI", runnerType: domain.RunnerTypeOpenCode}
	if _, err := failing.RuntimeVersion(context.Background()); err == nil {
		t.Fatal("missing binary reported a runtime version")
	}
}
