package validationbroker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type identityPlannerFunc func(orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error)

func (f identityPlannerFunc) Preview(_ context.Context, r orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error) {
	return f(r)
}

func TestEffectiveSuiteConfigurationInvalidatesCallerIdentity(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios/demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios/demo/main.go"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	version := "effective-config-1"
	intent := validIntent("plan-manager", "configuration")
	planner := identityPlannerFunc(func(request orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error) {
		if request.ScenarioName != "demo" || request.Preset != presetForStrength(intent.GetRequiredStrength()) {
			t.Fatalf("request differs from execution: %+v", request)
		}
		return &execution.ExecutionPlanPreview{ConfigurationFingerprint: version, Phases: []execution.PlannedPhase{{Name: "unit"}}}, nil
	})
	resolver := NewContentIdentityResolver(root, 0).WithExecutionPlanner(planner)
	before, err := resolver.Resolve(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	intent.ExpectedIdentity = before
	version = "effective-config-2"
	after, err := resolver.Resolve(context.Background(), intent)
	if err != nil || after.GetIdentity() == before.GetIdentity() || after.GetConfiguration()["suite/demo"] != version {
		t.Fatalf("stale owner configuration survived: %v, %v", after, err)
	}
}

func TestControlPlaneInputsLive(t *testing.T) {
	root := os.Getenv("VALIDATION_BROKER_LIVE_REPO")
	if root == "" {
		t.Skip("set VALIDATION_BROKER_LIVE_REPO for installed control-plane integration")
	}
	intent := validIntent("plan-manager", "live-inputs")
	intent.Targets[0].Id = "plan-manager"
	intent.ContentInputs[0].Root = "scenarios/plan-manager"
	planner, err := orchestrator.NewSuiteOrchestrator(filepath.Join(root, "scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewContentIdentityResolver(root, 1<<20).WithControlPlaneInputs().WithExecutionPlanner(execution.NewExecutionPlanService(planner, emptyPlanHistory{}))
	identity, err := resolver.Resolve(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	if len(identity.GetRoots()) < 2 || identity.GetToolchain()["build/plan-manager/api/toolchain"] == "" {
		t.Fatalf("authoritative inputs missing: roots=%d keys=%v", len(identity.GetRoots()), identity.GetToolchain())
	}
	t.Logf("resolved %d roots with authoritative build keys", len(identity.GetRoots()))
}

type emptyPlanHistory struct{}

func (emptyPlanHistory) ListPhaseSamples(context.Context, string, []string, time.Time, int) ([]execution.PhaseDurationSample, error) {
	return nil, nil
}
func (emptyPlanHistory) ListPlanSamples(context.Context, string, time.Time, int) ([]execution.PlanDurationSample, error) {
	return nil, nil
}

func TestOwnerBuildInputsInvalidateReuseOutsidePlanBoundary(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"scenarios/demo/main.go", "packages/shared/value.go", "packages/unrelated/value.go"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte("initial"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	resolver := NewContentIdentityResolver(root, 0)
	toolchain := "go1"
	resolver.buildInputs = func(context.Context, string) (*cliv1.ScenarioFreshnessInputs, error) {
		return &cliv1.ScenarioFreshnessInputs{Paths: []string{"packages/shared/value.go"}, BuildKeys: map[string]string{"api/toolchain": toolchain}}, nil
	}
	intent := validIntent("caller", "closure")
	before, err := resolver.Resolve(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages/unrelated/value.go"), []byte("unrelated edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	unchanged, err := resolver.Resolve(context.Background(), intent)
	if err != nil || unchanged.GetIdentity() != before.GetIdentity() {
		t.Fatalf("unrelated edit invalidates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages/shared/value.go"), []byte("relevant edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := resolver.Resolve(context.Background(), intent)
	if err != nil || changed.GetIdentity() == before.GetIdentity() {
		t.Fatalf("dependency edit reused: %v", err)
	}
	toolchain = "go2"
	after, err := resolver.Resolve(context.Background(), intent)
	if err != nil || after.GetIdentity() == changed.GetIdentity() {
		t.Fatalf("toolchain change reused: %v", err)
	}
	resolver.buildInputs = func(context.Context, string) (*cliv1.ScenarioFreshnessInputs, error) {
		return nil, errors.New("owner unavailable")
	}
	if _, err := resolver.Resolve(context.Background(), intent); err == nil {
		t.Fatal("missing closure accepted")
	}
}

func TestContentIdentityResolverResolvesBuildInputsWithBoundedConcurrency(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios/demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios/demo/main.go"), []byte("package demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	intent := validIntent("plan-manager", "parallel-inputs")
	for _, name := range []string{"one", "two", "three", "four", "five"} {
		intent.Targets = append(intent.Targets, &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO,
			Id:   name,
		})
	}
	resolver := NewContentIdentityResolver(root, 0)
	var mu sync.Mutex
	active, maxActive := 0, 0
	resolver.buildInputs = func(ctx context.Context, name string) (*cliv1.ScenarioFreshnessInputs, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		defer func() {
			mu.Lock()
			active--
			mu.Unlock()
		}()
		select {
		case <-time.After(25 * time.Millisecond):
			return &cliv1.ScenarioFreshnessInputs{BuildKeys: map[string]string{"target": name}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if _, err := resolver.Resolve(context.Background(), intent); err != nil {
		t.Fatal(err)
	}
	if maxActive < 2 {
		t.Fatalf("build-input resolution was serialized; max concurrent calls = %d", maxActive)
	}
}

func TestContentIdentityResolverUsesDeclaredRepositoryRelativeInputs(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, "src", "main.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := validIntent("plan-manager", "resolver")
	resolver := NewContentIdentityResolver(repoRoot, 1<<20)
	identity, err := resolver.Resolve(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(identity.GetIdentity(), "ci:v1:") || len(identity.GetRoots()) != 1 || identity.GetRoots()[0].GetName() != "scenario" {
		t.Fatalf("resolved identity = %#v", identity)
	}
	files := identity.GetRoots()[0].GetFiles()
	if len(files) != 1 || files[0].GetPath() != "src/main.go" || files[0].GetDigest() == "" {
		t.Fatalf("missing frozen manifest files: %v", files)
	}
	if err := os.WriteFile(filepath.Join(scenarioRoot, "src", "main.go"), []byte("package changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := resolver.Resolve(context.Background(), intent)
	if err != nil || changed.GetIdentity() == identity.GetIdentity() {
		t.Fatalf("changed identity = %#v err=%v", changed, err)
	}
}

func TestContentIdentityResolverRejectsEscapingRoot(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	intent := validIntent("plan-manager", "escape")
	intent.ContentInputs[0].Root = "../outside"
	_, err := NewContentIdentityResolver(t.TempDir(), 0).Resolve(context.Background(), intent)
	if err == nil || !strings.Contains(err.Error(), "repository-relative") {
		t.Fatalf("escaping root error = %v", err)
	}
}
