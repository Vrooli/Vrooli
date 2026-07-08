package cleanup

import (
	"context"
	"errors"
	"testing"
	"time"

	cleanupcore "cleanup-manager/internal/cleanup"
	"cleanup-manager/internal/orchestrator"
	"cleanup-manager/internal/policy"
	"connectrpc.com/connect"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cleanup-manager/v1/cleanup"
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

type fakeCleanupService struct {
	catalog    []cleanupcore.ProviderMetadata
	policy     orchestrator.Policy
	plan       orchestrator.Plan
	apply      orchestrator.ApplyReport
	audit      []orchestrator.AuditEvent
	err        error
	applyInput orchestrator.ApplyInput
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

func (s *fakeCleanupService) Audit(context.Context) ([]orchestrator.AuditEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.audit, nil
}

func sampleProvider(id string) cleanupcore.ProviderMetadata {
	return cleanupcore.ProviderMetadata{
		ID: id, Name: "Temporary files", Version: "v1", OwnerScenario: "cleanup-manager",
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
