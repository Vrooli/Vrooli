package preflight

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	runtimeapi "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/health"
)

func TestConnectServiceReturnsTypedValidationAndJobErrors(t *testing.T) {
	service := NewService()
	handler := NewConnectService(service)

	_, err := handler.RunPreflight(context.Background(), connect.NewRequest(&domainv1.PreflightRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("RunPreflight() error code = %v, want invalid_argument (err %v)", connect.CodeOf(err), err)
	}

	_, err = handler.GetPreflightJob(context.Background(), connect.NewRequest(&domainv1.GetPreflightJobRequest{JobId: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetPreflightJob() error code = %v, want not_found (err %v)", connect.CodeOf(err), err)
	}

	job := service.CreateJob()
	got, err := handler.GetPreflightJob(context.Background(), connect.NewRequest(&domainv1.GetPreflightJobRequest{JobId: job.ID}))
	if err != nil {
		t.Fatalf("GetPreflightJob() error = %v", err)
	}
	if got.Msg.GetJobId() != job.ID || got.Msg.GetStatus() != domainv1.JobStatus_JOB_STATUS_RUNNING || len(got.Msg.GetSteps()) == 0 {
		t.Fatalf("GetPreflightJob() = %+v, want running job with steps", got.Msg)
	}
}

func TestResponseToProtoPreservesStructuredPreflightEvidence(t *testing.T) {
	exitCode := 17
	observedAt := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	result := ResponseToProto(&Response{
		Status: "passed",
		Validation: &runtimeapi.BundleValidationResult{
			Valid:           false,
			Errors:          []runtimeapi.BundleError{{Code: "MISSING_BINARY", Service: "api", Path: "bin/api", Message: "missing"}},
			Warnings:        []runtimeapi.BundleWarning{{Code: "OPTIONAL_ASSET", Service: "ui", Path: "assets/logo", Message: "optional"}},
			MissingBinaries: []runtimeapi.MissingBinary{{ServiceID: "api", Platform: "linux", Path: "bin/api"}},
		},
		Ports:        map[string]map[string]int{"api": {"http": 8080}},
		Checks:       []Check{{ID: "validate", Step: "validation", Status: "pass"}},
		Ready:        &Ready{Details: map[string]health.Status{"api": {Ready: true, ExitCode: &exitCode, UpdatedAt: observedAt}}},
		Runtime:      &Runtime{InstanceID: "instance-1", IPCPort: 7777},
		Fingerprints: []ServiceFingerprint{{ServiceID: "api", BinarySHA256: "abc", BinarySizeBytes: 42}},
	})

	if result.GetValidation().GetErrors()[0].GetCode() != "MISSING_BINARY" || result.GetValidation().GetMissingBinaries()[0].GetPath() != "bin/api" {
		t.Fatalf("validation evidence was not preserved: %+v", result.GetValidation())
	}
	if len(result.GetPorts()) != 1 || result.GetPorts()[0].GetServiceId() != "api" || result.GetPorts()[0].GetName() != "http" || result.GetPorts()[0].GetPort() != 8080 {
		t.Fatalf("ports were not preserved structurally: %+v", result.GetPorts())
	}
	if result.GetChecks()[0].GetStep() != sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_VALIDATION || result.GetChecks()[0].GetStatus() != sharedv1.CheckStatus_CHECK_STATUS_PASSED {
		t.Fatalf("check vocabulary was not preserved: %+v", result.GetChecks()[0])
	}
	if result.GetReady().GetDetails()[0].GetExitCode() != int32(exitCode) || result.GetReady().GetDetails()[0].GetUpdatedAt().AsTime() != observedAt {
		t.Fatalf("readiness detail was not preserved: %+v", result.GetReady().GetDetails()[0])
	}
	if result.GetRuntime().GetInstanceId() != "instance-1" || result.GetRuntime().GetIpcPort() != 7777 || result.GetServiceFingerprints()[0].GetBinarySha256() != "abc" {
		t.Fatalf("runtime or fingerprint evidence was not preserved: runtime=%+v fingerprints=%+v", result.GetRuntime(), result.GetServiceFingerprints())
	}
}

func TestConnectServiceRejectsEmptyManifestInspection(t *testing.T) {
	handler := NewConnectService(NewService())
	_, err := handler.InspectManifest(context.Background(), connect.NewRequest(&domainv1.ManifestRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("InspectManifest() error code = %v, want invalid_argument (err %v)", connect.CodeOf(err), err)
	}
}

func TestConnectServiceInspectsValidatedManifestAndMapsInvalidPath(t *testing.T) {
	handler := NewConnectService(NewService())
	fixture, err := filepath.Abs("../../runtime/testdata/fixture-bundle/bundle.json")
	if err != nil {
		t.Fatal(err)
	}
	response, err := handler.InspectManifest(context.Background(), connect.NewRequest(&domainv1.ManifestRequest{ManifestPath: fixture}))
	if err != nil || response.Msg.GetManifest()["schema_version"] == "" || response.Msg.GetManifest()["app"] == "" {
		t.Fatalf("InspectManifest() = %#v, %v", response.Msg, err)
	}
	_, err = handler.InspectManifest(context.Background(), connect.NewRequest(&domainv1.ManifestRequest{ManifestPath: filepath.Join(t.TempDir(), "missing.json")}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing manifest code = %v, want invalid_argument", connect.CodeOf(err))
	}
}
