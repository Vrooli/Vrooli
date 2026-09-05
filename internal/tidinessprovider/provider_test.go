package tidinessprovider

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tidiness-manager/v1/validation"
	"google.golang.org/protobuf/types/known/anypb"
)

type fakeValidationService struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	mu      sync.Mutex
	target  *commonv1.ValidationTarget
	exclude []string
}

func (f *fakeValidationService) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	f.mu.Lock()
	f.target = req.Msg.GetTarget()
	f.exclude = append([]string(nil), req.Msg.GetExclude()...)
	f.mu.Unlock()
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
	}), nil
}

func TestValidateDelegatesControlPlaneTargetAndExcludesOwnedChildren(t *testing.T) {
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()

	result, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }}).Validate(context.Background(), "/repo")
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
}

func TestValidateReturnsExplicitUnavailableError(t *testing.T) {
	_, err := (Provider{ResolveURL: func(context.Context, string) (string, error) { return "", context.Canceled }}).Validate(context.Background(), "/repo")
	if err == nil || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}
