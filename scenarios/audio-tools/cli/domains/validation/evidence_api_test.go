package validation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
)

type qualificationEvidenceProbe struct {
	experimentconnect.ExperimentServiceClient
	err error
}

func (p qualificationEvidenceProbe) ListQualificationEvidence(context.Context, *connect.Request[experimentv1.ListQualificationEvidenceRequest]) (*connect.Response[experimentv1.ListQualificationEvidenceResponse], error) {
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&experimentv1.ListQualificationEvidenceResponse{}), nil
}

func TestRequireLabQualificationEvidenceAPIGivesActionableRouteForProductAPI(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeNotFound, connect.CodeUnimplemented} {
		t.Run(code.String(), func(t *testing.T) {
			client := qualificationEvidenceProbe{err: connect.NewError(code, errors.New("missing method"))}
			err := requireLabQualificationEvidenceAPI(context.Background(), client, "validation run-host-portability")
			if err == nil || !strings.Contains(err.Error(), "audio-tools --instance lab validation run-host-portability") {
				t.Fatalf("expected lab route guidance, got %v", err)
			}
		})
	}
}

func TestRequireLabQualificationEvidenceAPIPropagatesOtherFailures(t *testing.T) {
	want := connect.NewError(connect.CodeUnavailable, errors.New("lab is restarting"))
	err := requireLabQualificationEvidenceAPI(context.Background(), qualificationEvidenceProbe{err: want}, "validation run-host-portability")
	if !errors.Is(err, want) {
		t.Fatalf("preflight error = %v; want wrapped original error %v", err, want)
	}
}

func TestRequireLabQualificationEvidenceAPIAcceptsAvailableService(t *testing.T) {
	if err := requireLabQualificationEvidenceAPI(context.Background(), qualificationEvidenceProbe{}, "validation run-host-portability"); err != nil {
		t.Fatalf("available evidence service rejected: %v", err)
	}
}
