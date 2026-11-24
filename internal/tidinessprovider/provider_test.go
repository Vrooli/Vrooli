package tidinessprovider

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tidiness-manager/v1/validation"
	"google.golang.org/protobuf/types/known/anypb"
)

type fakeValidationService struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	mu       sync.Mutex
	target   *commonv1.ValidationTarget
	exclude  []string
	response *scenariovalidationv1.ValidateTargetResponse
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
	f.mu.Lock()
	f.target = req.Msg.GetTarget()
	f.exclude = append([]string(nil), req.Msg.GetExclude()...)
	f.mu.Unlock()
	if f.response != nil {
		return connect.NewResponse(f.response), nil
	}
	native, err := anypb.New(&validationv1.TidinessScanResponse{
		Status: "passed",
		Findings: []*validationv1.TidinessFinding{{
			RuleId:      "clean",
			Severity:    "info",
			Description: "clean",
		}},
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target:       req.Msg.GetTarget(),
		Status:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		NativeDetail: native,
		Assessment:   &commonv1.MaturityAssessment{},
	}), nil
}

func TestValidateDelegatesControlPlaneTargetAndExcludesOwnedChildren(t *testing.T) {
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	demands := &demandStub{}

	result, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Demand: demands}).Validate(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if result.Status != "VALIDATION_STATUS_PASSED" || len(result.Findings) != 1 {
		t.Fatalf("result = %+v", result)
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.target.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE || service.target.GetId() != TargetID || service.target.GetRoot() != TargetRoot {
		t.Fatalf("target = %+v", service.target)
	}
	if len(service.exclude) != 2 || service.exclude[0] != TargetToolGlob || service.exclude[1] != TargetSafeGlob {
		t.Fatalf("exclude = %v", service.exclude)
	}
	if len(demands.acquired) != 1 || len(demands.released) != 1 || demands.released[0] != demands.acquired[0].LeaseID {
		t.Fatalf("demand lifecycle = acquired %d released %d", len(demands.acquired), len(demands.released))
	}
	lease := demands.acquired[0]
	if lease.Scenario != ScenarioName || lease.ConsumerID != "control-plane:tidiness-provider" || lease.Kind != demand.KindDependency || lease.Metadata != "owner=tidiness-provider" || !strings.HasPrefix(lease.RequestID, "validate:/repo:") || lease.LeaseID != demand.StableLeaseID(lease.ConsumerID, ScenarioName, lease.RequestID) || len(demands.releaseReasons) != 1 || demands.releaseReasons[0] != "tidiness validation complete" {
		t.Fatalf("demand policy = %+v, release reasons = %v", lease, demands.releaseReasons)
	}
}

func TestValidateReturnsExplicitUnavailableError(t *testing.T) {
	_, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return "", context.Canceled }}).Validate(context.Background(), "/repo")
	if err == nil || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}

func TestValidateRejectsIncompleteTargetCoverage(t *testing.T) {
	native, err := anypb.New(&validationv1.TidinessScanResponse{})
	if err != nil {
		t.Fatal(err)
	}
	validTarget := &commonv1.ValidationTarget{
		Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE,
		Id:   TargetID,
		Root: TargetRoot,
	}
	tests := []struct {
		name     string
		response *scenariovalidationv1.ValidateTargetResponse
	}{
		{
			name: "mismatched target",
			response: &scenariovalidationv1.ValidateTargetResponse{
				Target: &commonv1.ValidationTarget{Kind: validTarget.Kind, Id: "wrong", Root: validTarget.Root},
				Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, NativeDetail: native, Assessment: &commonv1.MaturityAssessment{},
			},
		},
		{
			name: "missing assessment evidence",
			response: &scenariovalidationv1.ValidateTargetResponse{
				Target: validTarget, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, NativeDetail: native,
			},
		},
		{
			name: "missing canonical native detail",
			response: &scenariovalidationv1.ValidateTargetResponse{
				Target: validTarget, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: &commonv1.MaturityAssessment{},
			},
		},
		{
			name: "unspecified status",
			response: &scenariovalidationv1.ValidateTargetResponse{
				Target: validTarget, NativeDetail: native, Assessment: &commonv1.MaturityAssessment{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeValidationService{response: test.response}
			_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
			server := httptest.NewServer(handler)
			defer server.Close()
			_, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Demand: &demandStub{}}).Validate(context.Background(), "/repo")
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("Validate error = %v, want explicit incomplete coverage", err)
			}
		})
	}
}

func TestResultFromResponsePrefersCanonicalNativeFindingOverAssessmentDuplicate(t *testing.T) {
	native, err := anypb.New(&validationv1.TidinessScanResponse{
		Findings: []*validationv1.TidinessFinding{{
			RuleId: "long-file", Severity: "warning", FilePath: "api/scan.go", LineNumber: 12,
			Description: "canonical source description", RecommendedRemediation: "canonical fix",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := &scenariovalidationv1.ValidateTargetResponse{
		Status:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
		NativeDetail: native,
		Assessment: &commonv1.MaturityAssessment{Findings: []*commonv1.AssessmentFinding{
			{Code: "long-file", Severity: "FINDING_SEVERITY_WARNING", Location: "api/scan.go:12", Message: "derived assessment description"},
			{Code: "other-rule", Severity: "FINDING_SEVERITY_ERROR", Location: "api/other.go:4", Message: "assessment-only evidence"},
		}},
	}
	got := resultFromResponse(response)
	if len(got.Findings) != 2 {
		t.Fatalf("findings = %+v, want one canonical finding plus one assessment-only finding", got.Findings)
	}
	if got.Findings[0].Location != "api/scan.go:12" || got.Findings[0].Message != "canonical source description" {
		t.Fatalf("canonical source was not preserved: %+v", got.Findings[0])
	}
	if got.Findings[1].Code != "other-rule" {
		t.Fatalf("assessment-only evidence was dropped: %+v", got.Findings)
	}
}

func TestValidateWrapsDemandAcquireFailure(t *testing.T) {
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	demands := &demandStub{acquireErr: errors.New("lease denied")}
	_, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Demand: demands}).Validate(context.Background(), "/repo")
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "acquire tidiness-manager demand lease") {
		t.Fatalf("Validate error = %v, want provider-specific unavailable error", err)
	}
}
