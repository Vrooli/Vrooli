package structureprovider

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
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
	acquired []demand.AcquireRequest
	released []string
}

func (s *demandStub) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	s.acquired = append(s.acquired, req)
	return demand.Lease{LeaseID: req.LeaseID}, nil
}
func (s *demandStub) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, nil
}
func (s *demandStub) Release(_ context.Context, id, _ string) (demand.Lease, error) {
	s.released = append(s.released, id)
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
