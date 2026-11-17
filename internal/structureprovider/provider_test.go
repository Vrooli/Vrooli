package structureprovider

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/vrooli/internal/providerdemand"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type fakeValidationService struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	mu         sync.Mutex
	gotTargets []*commonv1.ValidationTarget
}

type demandStub struct {
	acquired       []demand.AcquireRequest
	released       []string
	releaseReasons []string
	acquireErr     error
}

func (s *demandStub) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	s.acquired = append(s.acquired, req)
	if s.acquireErr != nil {
		return demand.Lease{}, s.acquireErr
	}
	return demand.Lease{LeaseID: req.LeaseID}, nil
}
func (s *demandStub) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, nil
}
func (s *demandStub) Release(_ context.Context, id, reason string) (demand.Lease, error) {
	s.released = append(s.released, id)
	s.releaseReasons = append(s.releaseReasons, reason)
	return demand.Lease{LeaseID: id}, nil
}

func (f *fakeValidationService) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	f.mu.Lock()
	f.gotTargets = append(f.gotTargets, target)
	f.mu.Unlock()
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target: target,
		Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
		Assessment: &commonv1.MaturityAssessment{Findings: []*commonv1.AssessmentFinding{
			{Code: "PROJECT_CONFIG_SURFACE", Message: "unapproved entry", Location: ".vrooli/baselines"},
		}},
	}), nil
}

func TestValidateDelegatesProjectTargetAndPreservesFinding(t *testing.T) {
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	demands := &demandStub{}

	output, err := (Provider{
		ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil },
		Demand:     demands,
	}).Validate(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	var projectTarget *commonv1.ValidationTarget
	for _, target := range service.gotTargets {
		if target.GetKind() == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT {
			projectTarget = target
			break
		}
	}
	if projectTarget == nil || projectTarget.GetId() != TargetID || projectTarget.GetRoot() != "/repo" {
		t.Fatalf("project target = %#v", projectTarget)
	}
	if output.Success {
		t.Fatal("output.Success = true, want delegated failure")
	}
	if len(demands.acquired) != 1 || len(demands.released) != 1 || demands.released[0] != demands.acquired[0].LeaseID {
		t.Fatalf("demand lifecycle = acquired %d released %d", len(demands.acquired), len(demands.released))
	}
	lease := demands.acquired[0]
	if lease.Scenario != ScenarioName || lease.ConsumerID != "control-plane:structure-provider" || lease.Kind != demand.KindDependency || lease.Metadata != "owner=structure-provider" || !strings.HasPrefix(lease.RequestID, "validate:/repo:") || lease.LeaseID != demand.StableLeaseID(lease.ConsumerID, ScenarioName, lease.RequestID) || len(demands.releaseReasons) != 1 || demands.releaseReasons[0] != "structure validation complete" {
		t.Fatalf("demand policy = %+v, release reasons = %v", lease, demands.releaseReasons)
	}
	if output.Report.Checks[4].Name != "project_config_surface" || output.Report.Checks[4].Passed {
		t.Fatalf("project config check = %#v", output.Report.Checks[4])
	}
}

func TestValidateReturnsExplicitUnavailableError(t *testing.T) {
	_, err := (Provider{
		ResolveURL: func(context.Context, string) (string, error) { return "", errors.New("offline") },
	}).Validate(context.Background(), "/repo")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}

func TestValidateWrapsDemandAcquireFailure(t *testing.T) {
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	demands := &demandStub{acquireErr: errors.New("lease denied")}
	_, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Demand: demands}).Validate(context.Background(), "/repo")
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "acquire structure-health demand lease") {
		t.Fatalf("Validate error = %v, want provider-specific unavailable error", err)
	}
}

func TestDemandConsumerIDIsTrimmed(t *testing.T) {
	demands := &demandStub{}
	release, err := providerdemand.Acquire(context.Background(), demands, demand.AcquireRequest{ConsumerID: "  custom-consumer  ", Scenario: ScenarioName}, "fallback", "done", "acquire failed")
	if err != nil {
		t.Fatal(err)
	}
	if demands.acquired[0].ConsumerID != "custom-consumer" {
		t.Fatalf("consumer ID = %q", demands.acquired[0].ConsumerID)
	}
	release()
}

func TestDemandAcquireWithoutClientIsNoop(t *testing.T) {
	release, err := providerdemand.Acquire(context.Background(), nil, demand.AcquireRequest{}, "fallback", "done", "acquire failed")
	if err != nil || release != nil {
		t.Fatalf("Acquire(nil) returned release=%t, err=%v; want no-op", release != nil, err)
	}
}

func TestProjectOnlySkipsFleetTargetTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "large-scenario"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "large-resource"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()

	_, err := (Provider{
		ResolveURL:  func(context.Context, string) (string, error) { return server.URL, nil },
		ProjectOnly: true,
	}).Validate(context.Background(), root)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if len(service.gotTargets) != 1 || service.gotTargets[0].GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT {
		t.Fatalf("targets = %#v, want only project target", service.gotTargets)
	}
}

func TestOutputFromResponsesRejectsMissingOrMismatchedRequiredEvidence(t *testing.T) {
	project := validationTarget{kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT, id: TargetID, root: "/repo"}
	passed := func(target *commonv1.ValidationTarget, assessment *commonv1.MaturityAssessment) *scenariovalidationv1.ValidateTargetResponse {
		return &scenariovalidationv1.ValidateTargetResponse{Target: target, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: assessment}
	}
	target := &commonv1.ValidationTarget{Kind: project.kind, Id: project.id, Root: project.root}
	tests := []struct {
		name      string
		responses []*scenariovalidationv1.ValidateTargetResponse
	}{
		{name: "missing response"},
		{name: "missing assessment", responses: []*scenariovalidationv1.ValidateTargetResponse{passed(target, nil)}},
		{name: "mismatched target", responses: []*scenariovalidationv1.ValidateTargetResponse{passed(&commonv1.ValidationTarget{Kind: project.kind, Id: "other", Root: project.root}, &commonv1.MaturityAssessment{})}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := outputFromResponses("/repo", []validationTarget{project}, test.responses)
			if output.Success || output.Report.Success {
				t.Fatalf("missing or malformed required evidence was reported clean: %+v", output)
			}
			found := false
			for _, check := range output.Report.Checks {
				if check.Name == "provider_coverage" && !check.Passed && check.Message != "" {
					found = true
				}
			}
			if !found {
				t.Fatalf("report lacks a concrete provider_coverage blocker: %+v", output.Report.Checks)
			}
		})
	}
}

func TestOutputFromResponsesAcceptsCompleteRequiredEvidence(t *testing.T) {
	target := validationTarget{kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT, id: TargetID, root: "/repo"}
	response := &scenariovalidationv1.ValidateTargetResponse{
		Target:     &commonv1.ValidationTarget{Kind: target.kind, Id: target.id, Root: target.root},
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: &commonv1.MaturityAssessment{},
	}
	output := outputFromResponses("/repo", []validationTarget{target}, []*scenariovalidationv1.ValidateTargetResponse{response})
	if !output.Success || !output.Report.Success {
		t.Fatalf("complete passed evidence should remain clean: %+v", output)
	}
}
