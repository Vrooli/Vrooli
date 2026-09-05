package cleanup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	cleanupcore "storage-manager/internal/cleanup"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/policy"

	"connectrpc.com/connect"
	corestorage "github.com/vrooli/api-core/storage"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
)

func TestConnectHandlerListsProviderMetadataAndPolicy(t *testing.T) {
	t.Parallel()

	svc := &fakeCleanupService{policy: samplePolicy()}
	svc.catalog = []cleanupcore.ProviderMetadata{sampleProvider("tmp")}
	handler := NewConnectHandler(svc)

	providers, err := handler.ListProviders(context.Background(), connect.NewRequest(&cleanupv1.ListProvidersRequest{}))
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if got := providers.Msg.GetProviders()[0]; got.GetId() != "tmp" || got.GetSafetyTier() != "safe" {
		t.Fatalf("provider proto = %#v, want tmp safe", got)
	}

	pol, err := handler.GetPolicy(context.Background(), connect.NewRequest(&cleanupv1.GetPolicyRequest{}))
	if err != nil {
		t.Fatalf("GetPolicy() error = %v", err)
	}
	if pol.Msg.GetPolicy().GetVersion() != "policy-test" || len(pol.Msg.GetPolicy().GetProviders()) != 1 {
		t.Fatalf("policy proto = %#v", pol.Msg.GetPolicy())
	}
}

func TestRuntimeHomeProviderConfigsRegisterOnlyContractEligibleEntries(t *testing.T) {
	configs := runtimeHomeProviderConfigs("/home/matthalloran8/Vrooli", t.TempDir())
	if len(configs) != 6 {
		t.Fatalf("runtime-home provider count = %d, want six regenerable cleanup entries (bin and artifacts are protected install roots)", len(configs))
	}
	for _, cfg := range configs {
		if cfg.ProtectActive != true || cfg.RetentionMaxAge <= 0 {
			t.Fatalf("runtime-home provider config = %#v, want active protection and age floor", cfg)
		}
		if cfg.ID == "runtime-home-bin" {
			t.Fatalf("control-plane install root registered for generic cleanup: %#v", cfg)
		}
		if cfg.ID == "runtime-home-artifacts" {
			t.Fatalf("managed artifact root registered for generic cleanup: %#v", cfg)
		}
		if cfg.ID == "runtime-home-backups" || cfg.ID == "runtime-home-secrets" || cfg.ID == "runtime-home-data" {
			t.Fatalf("protected runtime-home entry registered for cleanup: %#v", cfg)
		}
	}
}

func TestGovernedRootProviderConfigsAreDeclarativeAndExcludeLeasedRoots(t *testing.T) {
	configs := governedRootProviderConfigs("/home/matthalloran8/Vrooli", t.TempDir())
	if len(configs) == 0 {
		t.Fatal("governed root provider configs are empty")
	}
	seen := map[string]bool{}
	for _, cfg := range configs {
		if cfg.Tier != cleanupcore.SafetyTierSafe && cfg.Tier != cleanupcore.SafetyTierRegenerable || len(cfg.Roots) != 1 || cfg.RetentionMaxAge <= 0 {
			t.Fatalf("governed config = %#v, want safe or regenerable root with age bound", cfg)
		}
		if seen[cfg.ID] {
			t.Fatalf("duplicate governed root provider %q", cfg.ID)
		}
		seen[cfg.ID] = true
		if cfg.ID == "spec-browser-recordings" || cfg.ID == "spec-browser-captures" {
			t.Fatalf("owner-leased root was admitted to autonomous provider set: %q", cfg.ID)
		}
	}
	if !seen["spec-uv-cache"] || !seen["spec-go-build-cache"] || !seen["spec-go-module-cache"] || !seen["spec-go-work-dirs"] {
		t.Fatalf("expected declarative cache roots, got %v", seen)
	}
}

func TestOwnerScenarioProviderConfigsComeFromDeclarations(t *testing.T) {
	repoRoot := t.TempDir()
	writeOwnerProviderManifest(t, repoRoot, "z-owner", `{"storage":{"entries":{"cache":{"regenerable":true,"budget":{"max_age":"7d"}}},"cleanup_providers":[{"id":"z-retention","name":"Z retained data","safety_tier":"safe_with_owner","default_mode":"disabled","default_approval":"owner"}]}}`)
	writeOwnerProviderManifest(t, repoRoot, "a-owner", `{"storage":{"entries":{"cache":{"regenerable":true,"budget":{"max_age":"7d"}}},"cleanup_providers":[{"id":"a-retention","name":"A retained data","safety_tier":"conditional","default_mode":"enabled","default_approval":"operator","storage_entries":["cache"]}]}}`)
	writeOwnerProviderManifest(t, repoRoot, "undeclared", `{"storage":{"entries":{}}}`)

	configs := ownerScenarioProviderConfigs(repoRoot)
	if len(configs) != 2 {
		t.Fatalf("owner provider count = %d, want two declared providers: %#v", len(configs), configs)
	}
	if configs[0].ID != "a-retention" || configs[0].OwnerScenario != "a-owner" || configs[0].SafetyTier != cleanupcore.SafetyTierConditional || configs[0].DefaultApproval != cleanupcore.ApprovalModeOperator {
		t.Fatalf("first declaration = %#v", configs[0])
	}
	if !configs[0].OwnerBudget || len(configs[0].StorageEntries) != 1 || configs[0].StorageEntries[0] != "cache" {
		t.Fatalf("linked owner budget = %#v, want explicit cache authority", configs[0])
	}
	if configs[1].ID != "z-retention" || configs[1].OwnerScenario != "z-owner" || configs[1].DefaultMode != cleanupcore.ProviderModeDisabled {
		t.Fatalf("second declaration = %#v", configs[1])
	}
	if configs[1].OwnerBudget {
		t.Fatalf("unlinked budget incorrectly granted authority: %#v", configs[1])
	}
}

func writeOwnerProviderManifest(t *testing.T, repoRoot, scenario, body string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// [REQ:CLN-P0-005]
func TestConnectHandlerPlanApplyAndAuditUseServiceContracts(t *testing.T) {
	t.Parallel()

	svc := &fakeCleanupService{
		plan: orchestrator.Plan{
			ID:            "plan-test",
			PolicyVersion: "policy-test",
			CreatedAt:     fixedTime(),
			TotalBytes:    64,
			TotalItems:    1,
			Providers: []orchestrator.ProviderPlan{{
				ProviderID:      "tmp",
				ProviderVersion: "v1",
				Estimate:        cleanupcore.Estimate{EstimatedBytes: 64, ItemCount: 1},
				Preview: cleanupcore.Preview{Items: []cleanupcore.PreviewItem{{
					ID: "tmp/old", Path: "/redacted", Bytes: 64, SafetyTier: cleanupcore.SafetyTierSafe,
				}}},
				Policy: cleanupcore.ProviderPolicy{Enabled: true, ApprovalMode: cleanupcore.ApprovalModeNone},
			}},
		},
		apply: orchestrator.ApplyReport{
			PlanID:         "plan-test",
			IdempotencyKey: "idem-test",
			ReclaimedBytes: 64,
			Results:        []cleanupcore.ApplyResult{{ProviderID: "tmp", Applied: true, ReclaimedBytes: 64}},
		},
		audit: []orchestrator.AuditEvent{{
			ID: "audit-test", Time: fixedTime(), Type: "apply.completed", PlanID: "plan-test", Message: "64 bytes",
		}},
	}
	handler := NewConnectHandler(svc)

	planResp, err := handler.CreatePlan(context.Background(), connect.NewRequest(&cleanupv1.CreatePlanRequest{}))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if planResp.Msg.GetPlan().GetId() != "plan-test" || planResp.Msg.GetPlan().GetProviders()[0].GetItems()[0].GetBytes() != 64 {
		t.Fatalf("plan proto = %#v", planResp.Msg.GetPlan())
	}

	applyResp, err := handler.ApplyPlan(context.Background(), connect.NewRequest(&cleanupv1.ApplyPlanRequest{
		PlanId: "plan-test", PolicyVersion: "policy-test", IdempotencyKey: "idem-test",
	}))
	if err != nil {
		t.Fatalf("ApplyPlan() error = %v", err)
	}
	if applyResp.Msg.GetReclaimedBytes() != 64 || svc.applyInput.IdempotencyKey != "idem-test" {
		t.Fatalf("apply response/input = %#v / %#v", applyResp.Msg, svc.applyInput)
	}

	auditResp, err := handler.ListAudit(context.Background(), connect.NewRequest(&cleanupv1.ListAuditRequest{}))
	if err != nil {
		t.Fatalf("ListAudit() error = %v", err)
	}
	if auditResp.Msg.GetEvents()[0].GetType() != "apply.completed" {
		t.Fatalf("audit proto = %#v", auditResp.Msg.GetEvents())
	}
}

func TestConnectHandlerMapsValidationErrorsToConnectCodes(t *testing.T) {
	t.Parallel()

	handler := NewConnectHandler(&fakeCleanupService{err: errors.New("bad profile")})
	_, err := handler.SetPolicyProfile(context.Background(), connect.NewRequest(&cleanupv1.SetPolicyProfileRequest{Profile: "invalid"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("SetPolicyProfile() code = %s, want invalid_argument", connect.CodeOf(err))
	}
	_, err = handler.ApplyPlan(context.Background(), connect.NewRequest(&cleanupv1.ApplyPlanRequest{PlanId: "missing"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("ApplyPlan() code = %s, want failed_precondition", connect.CodeOf(err))
	}
}

func TestModuleShape(t *testing.T) {
	t.Parallel()

	module := ModuleWithService(&fakeCleanupService{policy: samplePolicy()})
	if module.Name != "cleanup" || module.Mount == nil || len(module.Endpoints) != len(Endpoints) {
		t.Fatalf("module shape = %#v", module)
	}
}

func TestDefaultRegistryWiresContractResolvedScenarioBinaryRoot(t *testing.T) {
	t.Parallel()

	registry, err := defaultRegistry(nil)
	if err != nil {
		t.Fatalf("defaultRegistry() error = %v", err)
	}
	provider, ok := registry.Get("scenario-binaries")
	if !ok {
		t.Fatal("scenario-binaries provider is not registered")
	}
	preview, err := provider.Preview(context.Background(), cleanupcore.PreviewRequest{
		Policy: cleanupcore.ProviderPolicy{Enabled: true, ApprovalMode: cleanupcore.ApprovalModeOwner},
	})
	if err != nil {
		t.Fatalf("scenario-binaries Preview() error = %v", err)
	}
	if preview.BlockedReason != "" {
		t.Fatalf("scenario-binaries provider is not wired to a usable contract root: %q", preview.BlockedReason)
	}
}

func TestAutohealLiveDatabasePathIgnoresCallerStorageNamespace(t *testing.T) {
	t.Setenv(corestorage.EnvStorageNamespace, "storage-manager")

	got := autohealLiveDatabasePath("/home/matthalloran8")
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}
	want, err := resolver.Path(corestorage.Options{ScenarioID: "vrooli-autoheal"}, corestorage.ClassData, "autoheal.sqlite")
	if err != nil {
		t.Fatalf("resolver.Path() error = %v", err)
	}
	if got != want {
		t.Fatalf("autohealLiveDatabasePath() = %q, want explicit autoheal path %q", got, want)
	}
}

type fakeCleanupService struct {
	catalog    []cleanupcore.ProviderMetadata
	policy     orchestrator.Policy
	plan       orchestrator.Plan
	apply      orchestrator.ApplyReport
	audit      []orchestrator.AuditEvent
	err        error
	applyInput orchestrator.ApplyInput

	pressureSignal  orchestrator.PressureSignal
	pressureOutcome orchestrator.PressureOutcome
}

func (s *fakeCleanupService) Catalog() []cleanupcore.ProviderMetadata { return s.catalog }
func (s *fakeCleanupService) CurrentPolicy(context.Context) (orchestrator.Policy, error) {
	if s.err != nil {
		return orchestrator.Policy{}, s.err
	}
	return s.policy, nil
}

func (s *fakeCleanupService) SetPolicyProfile(context.Context, policy.ProfileName) (orchestrator.Policy, error) {
	if s.err != nil {
		return orchestrator.Policy{}, s.err
	}
	return s.policy, nil
}

func (s *fakeCleanupService) Plan(context.Context, cleanupcore.ObservationScope) (orchestrator.Plan, error) {
	if s.err != nil {
		return orchestrator.Plan{}, s.err
	}
	return s.plan, nil
}

func (s *fakeCleanupService) Apply(_ context.Context, input orchestrator.ApplyInput) (orchestrator.ApplyReport, error) {
	s.applyInput = input
	if s.err != nil {
		return orchestrator.ApplyReport{}, s.err
	}
	return s.apply, nil
}

func (s *fakeCleanupService) ReportPressure(_ context.Context, signal orchestrator.PressureSignal) (orchestrator.PressureOutcome, error) {
	s.pressureSignal = signal
	if s.err != nil {
		return orchestrator.PressureOutcome{}, s.err
	}
	return s.pressureOutcome, nil
}

func (s *fakeCleanupService) Audit(context.Context) ([]orchestrator.AuditEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.audit, nil
}

func sampleProvider(id string) cleanupcore.ProviderMetadata {
	return cleanupcore.ProviderMetadata{
		ID: id, Name: "Temporary files", Version: "v1", OwnerScenario: "storage-manager",
		SafetyTier: cleanupcore.SafetyTierSafe, DefaultMode: cleanupcore.ProviderModeEnabled,
		DefaultApproval: cleanupcore.ApprovalModeNone, SupportedPlatforms: []string{"linux"},
		IrreversibleEffects: []string{"delete temp files"}, TestSubstitute: "fake filesystem",
	}
}

func samplePolicy() orchestrator.Policy {
	return orchestrator.Policy{
		Version: "policy-test", Profile: policy.ProfileConservative, CreatedAt: fixedTime(),
		Providers: map[string]cleanupcore.ProviderPolicy{
			"tmp": {Enabled: true, MinAge: time.Hour, ApprovalMode: cleanupcore.ApprovalModeNone},
		},
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)
}
