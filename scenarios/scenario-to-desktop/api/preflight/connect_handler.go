package preflight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	runtimeapi "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConnectService exposes the existing preflight domain service without creating
// a second execution path for bundle validation or session management.
type ConnectService struct {
	domainconnect.UnimplementedPreflightServiceHandler
	service Service
}

var _ domainconnect.PreflightServiceHandler = (*ConnectService)(nil)

func NewConnectService(service Service) *ConnectService { return &ConnectService{service: service} }

func (s *ConnectService) RunPreflight(_ context.Context, req *connect.Request[domainv1.PreflightRequest]) (*connect.Response[sharedv1.PreflightResponse], error) {
	result, err := s.service.RunBundlePreflight(preflightRequestFromProto(req.Msg))
	if err != nil {
		return nil, preflightConnectError(err)
	}
	return connect.NewResponse(ResponseToProto(result)), nil
}

func (s *ConnectService) GetPreflightJob(_ context.Context, req *connect.Request[domainv1.GetPreflightJobRequest]) (*connect.Response[domainv1.JobStatusResponse], error) {
	job, ok := s.service.GetJob(req.Msg.GetJobId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("preflight job %q not found", req.Msg.GetJobId()))
	}
	steps := make([]*domainv1.JobStep, 0, len(job.Steps))
	for _, step := range job.Steps {
		steps = append(steps, &domainv1.JobStep{Id: step.ID, Name: step.Name, State: checkStatusProto(step.State), Detail: stringPtr(step.Detail)})
	}
	response := &domainv1.JobStatusResponse{JobId: job.ID, Status: jobStatusProto(job.Status), Steps: steps, Error: stringPtr(job.Err), StartedAt: timestamppb.New(job.StartedAt), UpdatedAt: timestamppb.New(job.UpdatedAt)}
	if job.Result != nil {
		response.Result = ResponseToProto(job.Result)
	}
	return connect.NewResponse(response), nil
}

// InspectManifest returns the parsed, validated manifest that the runtime will
// use. Manifest creation belongs to the pipeline service; this operation is
// deliberately read-only.
func (s *ConnectService) InspectManifest(_ context.Context, req *connect.Request[domainv1.ManifestRequest]) (*connect.Response[domainv1.ManifestResponse], error) {
	if strings.TrimSpace(req.Msg.GetManifestPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("manifest_path is required"))
	}
	manifest, _, err := loadAndValidateManifest(req.Msg.GetManifestPath())
	if err != nil {
		return nil, preflightConnectError(err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal validated manifest: %w", err))
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode validated manifest: %w", err))
	}
	flat := make(map[string]string, len(values))
	for key, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode manifest field %q: %w", key, err))
		}
		flat[key] = string(encoded)
	}
	return connect.NewResponse(&domainv1.ManifestResponse{Manifest: flat}), nil
}

func preflightRequestFromProto(v *domainv1.PreflightRequest) Request {
	return Request{BundleManifestPath: v.GetBundleManifestPath(), BundleRoot: v.GetBundleRoot(), Secrets: v.GetSecrets(), TimeoutSeconds: int(v.GetTimeoutSeconds()), StartServices: v.GetStartServices(), LogTailLines: int(v.GetLogTailLines()), StatusOnly: v.GetStatusOnly(), SessionID: v.GetSessionId(), SessionTTLSeconds: int(v.GetSessionTtl()), SessionStop: v.GetSessionStop()}
}

// ResponseToProto is the canonical boundary mapping for a completed preflight
// result. Pipeline stage details reuse it so standalone and orchestrated
// preflight responses cannot drift.
func ResponseToProto(v *Response) *sharedv1.PreflightResponse {
	if v == nil {
		return nil
	}
	response := &sharedv1.PreflightResponse{Status: preflightStatusProto(v.Status), Validation: bundleValidationToProto(v.Validation), Ports: servicePortsToProto(v.Ports), Errors: validationErrors(v.Errors), SessionId: stringPtr(v.SessionID), ExpiresAt: parseTimestamp(v.ExpiresAt)}
	for _, secret := range v.Secrets {
		response.Secrets = append(response.Secrets, &sharedv1.PreflightSecret{Id: secret.ID, SecretClass: secretClassProto(secret.Class), Required: secret.Required, HasValue: secret.HasValue, Description: stringPtr(secret.Description), Format: stringPtr(secret.Format), Prompt: secret.Prompt})
	}
	for _, check := range v.Checks {
		response.Checks = append(response.Checks, &sharedv1.PreflightCheck{Id: check.ID, Step: preflightCheckStepProto(check.Step), Name: check.Name, Status: checkStatusProto(check.Status), Detail: stringPtr(check.Detail)})
	}
	for _, tail := range v.LogTails {
		response.LogTails = append(response.LogTails, &sharedv1.ServiceLogTail{ServiceId: tail.ServiceID, Lines: int32(tail.Lines), Content: tail.Content, Error: stringPtr(tail.Error)})
	}
	if v.Telemetry != nil {
		response.Telemetry = &sharedv1.TelemetryInfo{Path: v.Telemetry.Path, UploadUrl: stringPtr(v.Telemetry.UploadURL)}
	}
	if v.Ready != nil {
		response.Ready = &sharedv1.PreflightReady{Ready: v.Ready.Ready, SnapshotAt: parseTimestamp(v.Ready.SnapshotAt), WaitedSeconds: int32(v.Ready.WaitedSeconds), Gpu: &sharedv1.GPUInfo{Available: v.Ready.GPU.Available, Method: stringPtr(v.Ready.GPU.Method), Reason: stringPtr(v.Ready.GPU.Reason), Requirements: v.Ready.GPU.Requirements}}
		for serviceID, status := range v.Ready.Details {
			response.Ready.Details = append(response.Ready.Details, &sharedv1.ServiceReadiness{ServiceId: serviceID, Ready: status.Ready, Skipped: status.Skipped, Message: stringPtr(status.Message), ExitCode: int32Ptr(status.ExitCode), StartedAt: timestampPtr(status.StartedAt), ReadyAt: timestampPtr(status.ReadyAt), UpdatedAt: timestampPtr(status.UpdatedAt)})
		}
	}
	if v.Runtime != nil {
		response.Runtime = &sharedv1.PreflightRuntime{InstanceId: v.Runtime.InstanceID, StartedAt: parseTimestamp(v.Runtime.StartedAt), AppDataDir: stringPtr(v.Runtime.AppDataDir), BundleRoot: stringPtr(v.Runtime.BundleRoot), DryRun: v.Runtime.DryRun, ManifestHash: stringPtr(v.Runtime.ManifestHash), ManifestSchema: stringPtr(v.Runtime.ManifestSchema), Target: stringPtr(v.Runtime.Target), AppName: stringPtr(v.Runtime.AppName), AppVersion: stringPtr(v.Runtime.AppVersion), IpcHost: stringPtr(v.Runtime.IPCHost), IpcPort: int32Ptr(&v.Runtime.IPCPort), RuntimeVersion: stringPtr(v.Runtime.RuntimeVersion), BuildVersion: stringPtr(v.Runtime.BuildVersion)}
	}
	for _, fingerprint := range v.Fingerprints {
		response.ServiceFingerprints = append(response.ServiceFingerprints, &sharedv1.ServiceFingerprint{ServiceId: fingerprint.ServiceID, Platform: fingerprint.Platform, BinaryPath: stringPtr(fingerprint.BinaryPath), BinaryResolvedPath: stringPtr(fingerprint.BinaryResolvedPath), BinarySha256: stringPtr(fingerprint.BinarySHA256), BinarySizeBytes: int64Ptr(fingerprint.BinarySizeBytes), BinaryMtime: parseTimestamp(fingerprint.BinaryMtime), Error: stringPtr(fingerprint.Error)})
	}
	return response
}

func preflightConnectError(err error) error {
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		switch statusErr.Status {
		case http.StatusBadRequest:
			return connect.NewError(connect.CodeInvalidArgument, statusErr.Err)
		case http.StatusNotFound:
			return connect.NewError(connect.CodeNotFound, statusErr.Err)
		case http.StatusConflict:
			return connect.NewError(connect.CodeFailedPrecondition, statusErr.Err)
		}
	}
	return connect.NewError(connect.CodeInternal, err)
}

func bundleValidationToProto(value *runtimeapi.BundleValidationResult) *sharedv1.BundleValidationResult {
	if value == nil {
		return nil
	}
	result := &sharedv1.BundleValidationResult{Valid: value.Valid}
	for _, issue := range value.Errors {
		result.Errors = append(result.Errors, &sharedv1.BundleValidationIssue{Code: issue.Code, Service: stringPtr(issue.Service), Path: stringPtr(issue.Path), Message: issue.Message})
	}
	for _, issue := range value.Warnings {
		result.Warnings = append(result.Warnings, &sharedv1.BundleValidationIssue{Code: issue.Code, Service: stringPtr(issue.Service), Path: stringPtr(issue.Path), Message: issue.Message})
	}
	for _, item := range value.MissingBinaries {
		result.MissingBinaries = append(result.MissingBinaries, &sharedv1.MissingBinary{ServiceId: item.ServiceID, Platform: item.Platform, Path: item.Path})
	}
	for _, item := range value.MissingAssets {
		result.MissingAssets = append(result.MissingAssets, &sharedv1.MissingAsset{ServiceId: item.ServiceID, Path: item.Path})
	}
	for _, item := range value.InvalidChecksums {
		result.InvalidChecksums = append(result.InvalidChecksums, &sharedv1.InvalidChecksum{ServiceId: item.ServiceID, Path: item.Path, Expected: stringPtr(item.Expected), Actual: stringPtr(item.Actual)})
	}
	return result
}

func servicePortsToProto(ports map[string]map[string]int) []*sharedv1.ServicePort {
	result := []*sharedv1.ServicePort{}
	for service, values := range ports {
		for name, port := range values {
			result = append(result, &sharedv1.ServicePort{ServiceId: service, Name: name, Port: int32(port)})
		}
	}
	return result
}

func validationErrors(values []string) []*sharedv1.ValidationError {
	result := make([]*sharedv1.ValidationError, 0, len(values))
	for _, value := range values {
		result = append(result, &sharedv1.ValidationError{Message: value})
	}
	return result
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func parseTimestamp(value string) *timestamppb.Timestamp {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return timestamppb.New(parsed)
}

func timestampPtr(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

func int32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	result := int32(*value)
	return &result
}

func int64Ptr(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func preflightStatusProto(value string) sharedv1.PreflightStatus {
	switch value {
	case "passed", "completed":
		return sharedv1.PreflightStatus_PREFLIGHT_STATUS_PASSED
	case "warnings":
		return sharedv1.PreflightStatus_PREFLIGHT_STATUS_WARNINGS
	case "failed":
		return sharedv1.PreflightStatus_PREFLIGHT_STATUS_FAILED
	default:
		return sharedv1.PreflightStatus_PREFLIGHT_STATUS_RUNNING
	}
}

func checkStatusProto(value string) sharedv1.CheckStatus {
	switch value {
	case "pass", "passed", "completed":
		return sharedv1.CheckStatus_CHECK_STATUS_PASSED
	case "failed":
		return sharedv1.CheckStatus_CHECK_STATUS_FAILED
	case "skipped":
		return sharedv1.CheckStatus_CHECK_STATUS_SKIPPED
	case "running":
		return sharedv1.CheckStatus_CHECK_STATUS_RUNNING
	default:
		return sharedv1.CheckStatus_CHECK_STATUS_PENDING
	}
}

func preflightCheckStepProto(value string) sharedv1.PreflightCheckStep {
	switch value {
	case "validation":
		return sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_VALIDATION
	case "secrets":
		return sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_SECRETS
	case "runtime":
		return sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_RUNTIME
	case "services":
		return sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_SERVICES
	case "diagnostics":
		return sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_DIAGNOSTICS
	default:
		return sharedv1.PreflightCheckStep_PREFLIGHT_CHECK_STEP_UNSPECIFIED
	}
}

func jobStatusProto(value string) domainv1.JobStatus {
	switch value {
	case "completed", "passed":
		return domainv1.JobStatus_JOB_STATUS_COMPLETED
	case "failed":
		return domainv1.JobStatus_JOB_STATUS_FAILED
	case "pending":
		return domainv1.JobStatus_JOB_STATUS_PENDING
	default:
		return domainv1.JobStatus_JOB_STATUS_RUNNING
	}
}

func secretClassProto(value string) sharedv1.SecretClass {
	switch value {
	case "api_key":
		return sharedv1.SecretClass_SECRET_CLASS_API_KEY
	case "password":
		return sharedv1.SecretClass_SECRET_CLASS_PASSWORD
	case "token":
		return sharedv1.SecretClass_SECRET_CLASS_TOKEN
	case "connection_string":
		return sharedv1.SecretClass_SECRET_CLASS_CONNECTION_STRING
	case "certificate":
		return sharedv1.SecretClass_SECRET_CLASS_CERTIFICATE
	default:
		return sharedv1.SecretClass_SECRET_CLASS_GENERIC
	}
}
