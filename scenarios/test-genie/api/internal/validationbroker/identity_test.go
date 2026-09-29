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

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
)

func TestResolveSourceIdentityIsReadOnlyAndDoesNotAcceptCallerIdentity(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testsqllite(t))
	want := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:post-promotion"}
	resolver := &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{want}}
	service := NewService(repo, nil)
	service.SetIdentityResolver(resolver)
	request := &validationv1.ResolveSourceIdentityRequest{
		Targets: []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: "demo"}},
		ContentInputs: []*validationv1.ContentInputRoot{{Name: "candidate", Root: "scenarios/demo", Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}}},
	}
	response, err := service.ResolveSourceIdentity(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(response.Msg.GetIdentity(), want) {
		t.Fatalf("identity = %#v, want %#v", response.Msg.GetIdentity(), want)
	}
	rows, _, err := repo.List(ctx, ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("identity resolution created %d validation receipts", len(rows))
	}
}

func TestResolveSourceIdentityRefusesMissingContentOrResolver(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewRepository(testsqllite(t)), nil)
	if _, err := service.ResolveSourceIdentity(ctx, connect.NewRequest(&validationv1.ResolveSourceIdentityRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing content error = %v, want invalid argument", err)
	}
	service.SetIdentityResolver(nil)
	request := &validationv1.ResolveSourceIdentityRequest{ContentInputs: []*validationv1.ContentInputRoot{{Name: "candidate", Root: "scenarios/demo"}}}
	if _, err := service.ResolveSourceIdentity(ctx, connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing resolver error = %v, want failed precondition", err)
	}
}

func TestAdmissionEnforcesRetainedFileManifest(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-IDENTITY-P0]
	for _, change := range []string{"unchanged", "reordered", "root-digest-only", "changed-bytes", "extra-file", "missing-file", "wrong-size", "wrong-root", "duplicate-root", "duplicate-file", "aggregate-conflict", "root-conflict", "empty-root", "schema-conflict", "aggregate-matches-files-conflict"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			source := filepath.Join(root, "scenarios/demo")
			if err := os.MkdirAll(source, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"a.go", "b.go"} {
				if err := os.WriteFile(filepath.Join(source, name), []byte("reviewed bytes"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			intent := validIntent("agent-manager", "retained-review")
			intent.ExpectedIdentity = nil
			resolver := NewContentIdentityResolver(root, 0)
			reviewed, err := resolver.Resolve(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			// The caller retains exact reviewed file bytes, not an unrelated hash
			// domain or a freshly observed aggregate identity claimed as review.
			expected := proto.Clone(reviewed).(*validationv1.SourceIdentity)
			expected.Identity = ""
			expected.Roots[0].Identity = ""
			wantOK := false
			switch change {
			case "unchanged":
				wantOK = true
			case "reordered":
				expected.Roots[0].Files[0], expected.Roots[0].Files[1] = expected.Roots[0].Files[1], expected.Roots[0].Files[0]
				wantOK = true
			case "root-digest-only":
				expected.Roots[0].Identity = reviewed.Roots[0].Identity
				expected.Roots[0].Files = nil
				wantOK = true
			case "changed-bytes":
				if err := os.WriteFile(filepath.Join(source, "a.go"), []byte("unreviewed now"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "extra-file":
				if err := os.WriteFile(filepath.Join(source, "c.go"), []byte("unreviewed"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "missing-file":
				if err := os.Remove(filepath.Join(source, "a.go")); err != nil {
					t.Fatal(err)
				}
			case "wrong-size":
				expected.Roots[0].Files[0].Size++
			case "wrong-root":
				expected.Roots[0].Name = "not-selected"
			case "duplicate-root":
				expected.Roots = append(expected.Roots, proto.Clone(expected.Roots[0]).(*validationv1.ContentRootIdentity))
			case "duplicate-file":
				expected.Roots[0].Files[1] = proto.Clone(expected.Roots[0].Files[0]).(*validationv1.ContentFileIdentity)
			case "aggregate-conflict":
				expected.Identity = "ci:v1:wrong"
			case "root-conflict":
				expected.Roots[0].Identity = "ri:v1:wrong"
			case "empty-root":
				expected.Roots[0].Files = nil
			case "schema-conflict":
				expected.SchemaVersion++
			case "aggregate-matches-files-conflict":
				expected.Identity = reviewed.Identity
				expected.Roots[0].Files[0].Size++
			}
			intent.ExpectedIdentity = expected
			original := proto.Clone(intent)
			repo := NewRepository(testsqllite(t))
			service := NewService(repo, nil)
			service.SetIdentityResolver(resolver)
			response, err := service.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
			if wantOK {
				if err != nil {
					t.Fatal(err)
				}
				if response.Msg.GetReceipt().GetAdmittedIdentity().GetIdentity() != reviewed.GetIdentity() {
					t.Fatal("receipt did not retain resolved identity")
				}
				repeated, err := service.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
				if err != nil || repeated.Msg.GetReceipt().GetReceiptId() != response.Msg.GetReceipt().GetReceiptId() {
					t.Fatalf("reattachment changed identity: %v", err)
				}
			} else {
				if connect.CodeOf(err) != connect.CodeFailedPrecondition {
					t.Fatalf("unreviewed or contradictory input: admitted=%t err=%v", response != nil, err)
				}
				rows, _, listErr := repo.List(ctx, ListFilter{})
				if listErr != nil || len(rows) != 0 {
					t.Fatalf("rejected input created a receipt: count=%d err=%v", len(rows), listErr)
				}
			}
			if !proto.Equal(original, intent) {
				t.Fatal("admission mutated caller's retained preconditions")
			}
		})
	}
}

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
