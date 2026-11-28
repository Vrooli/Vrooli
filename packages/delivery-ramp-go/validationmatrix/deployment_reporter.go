package validationmatrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeploymentEvidenceClient interface {
	ReportTargetVerdict(context.Context, *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error)
}

// DeploymentReporter adapts a completed matrix to deployment-manager's
// existing target-verdict authority. It reports references only; deployment-
// manager remains responsible for release approval and promotion policy.
type DeploymentReporter struct {
	client   DeploymentEvidenceClient
	profile  string
	commit   string
	producer string
	ramp     string
	platform string
	os       string
}

// SupportsRunIdentity marks the reporter as able to accept deployment identity
// from an exact matrix request while leaving unrelated operator matrices
// unreported.
func (*DeploymentReporter) SupportsRunIdentity() bool { return true }

func (r *DeploymentReporter) HasConfiguredIdentity() bool {
	return r != nil && strings.TrimSpace(r.profile) != "" && strings.TrimSpace(r.commit) != ""
}

func NewDeploymentReporter(client DeploymentEvidenceClient, profileID, gitCommit string) *DeploymentReporter {
	return &DeploymentReporter{client: client, profile: profileID, commit: gitCommit, producer: "delivery-ramp", ramp: "delivery-ramp", platform: "unknown", os: "unknown"}
}

func WithDeploymentIdentity(producer, ramp, platform, operatingSystem string) func(*DeploymentReporter) {
	return func(reporter *DeploymentReporter) {
		reporter.producer, reporter.ramp = strings.TrimSpace(producer), strings.TrimSpace(ramp)
		reporter.platform, reporter.os = strings.TrimSpace(platform), strings.TrimSpace(operatingSystem)
	}
}

func NewDeploymentReporterFromURL(baseURL, profileID, gitCommit string, httpClient *http.Client, options ...func(*DeploymentReporter)) *DeploymentReporter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	httpClient = withDeploymentManagerServiceAuth(httpClient)
	reporter := NewDeploymentReporter(evidencev1connect.NewEvidenceServiceClient(httpClient, strings.TrimRight(baseURL, "/")), profileID, gitCommit)
	for _, option := range options {
		if option != nil {
			option(reporter)
		}
	}
	return reporter
}

type deploymentManagerServiceAuthTransport struct {
	base http.RoundTripper
}

func (t deploymentManagerServiceAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	token := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN"))
	if token == "" {
		if path := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN_FILE")); path != "" {
			if data, err := os.ReadFile(path); err == nil {
				token = strings.TrimSpace(string(data))
			}
		}
	}
	if token == "" {
		return base.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+token)
	return base.RoundTrip(clone)
}

func withDeploymentManagerServiceAuth(client *http.Client) *http.Client {
	copy := *client
	copy.Transport = deploymentManagerServiceAuthTransport{base: client.Transport}
	return &copy
}

func (r *DeploymentReporter) ReportValidationGate(ctx context.Context, verdict ReleaseVerdict) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("deployment-manager evidence client is unavailable")
	}
	profile, commit := r.profile, r.commit
	if strings.TrimSpace(verdict.ProfileID) != "" || strings.TrimSpace(verdict.GitCommit) != "" {
		if strings.TrimSpace(verdict.ProfileID) == "" || strings.TrimSpace(verdict.GitCommit) == "" {
			return fmt.Errorf("deployment profile and git commit must be provided together")
		}
		profile, commit = verdict.ProfileID, verdict.GitCommit
	}
	if strings.TrimSpace(profile) == "" || strings.TrimSpace(commit) == "" {
		return fmt.Errorf("deployment-manager profile and git commit are required")
	}
	if len(verdict.Targets) > 1 {
		return fmt.Errorf("deployment evidence requires one exact target per validation run")
	}
	disposition := commonv1.Disposition_DISPOSITION_FAILED
	if verdict.Gate != nil && verdict.Gate.GetPassed() {
		disposition = commonv1.Disposition_DISPOSITION_PASSED
	}
	detailData := struct {
		Producer       string           `json:"producer"`
		MatrixID       string           `json:"matrix_id"`
		ScenarioName   string           `json:"scenario_name"`
		ArtifactDigest string           `json:"artifact_digest"`
		ArtifactRef    string           `json:"artifact_ref,omitempty"`
		TargetID       string           `json:"target_id,omitempty"`
		Gate           *ReleaseGateView `json:"gate"`
	}{
		Producer:       r.producer,
		MatrixID:       verdict.MatrixID,
		ScenarioName:   verdict.ScenarioName,
		ArtifactDigest: verdict.ArtifactDigest,
		ArtifactRef:    verdict.ArtifactRef,
		Gate:           releaseGateView(verdict.Gate),
	}
	if len(verdict.Targets) == 1 && verdict.Targets[0].Descriptor != nil {
		detailData.TargetID = verdict.Targets[0].Descriptor.GetTargetId()
	}
	detail, err := json.Marshal(detailData)
	if err != nil {
		return fmt.Errorf("encode matrix release detail: %w", err)
	}
	refs := make([]*commonv1.EvidenceRef, 0, len(verdict.Evidence))
	createdAt := time.Now().UTC()
	for _, evidence := range verdict.Evidence {
		if evidence == nil {
			continue
		}
		refs = append(refs, &commonv1.EvidenceRef{Producer: r.producer, ArtifactId: evidence.GetEvidenceId(), Kind: evidence.GetKind().String(), Checksum: evidence.GetSha256(), CreatedAt: timestamppb.New(createdAt)})
	}
	if verdict.ArtifactRef != "" {
		refs = append(refs, &commonv1.EvidenceRef{Producer: r.producer, ArtifactId: verdict.ArtifactRef, Kind: "android-apk", Checksum: verdict.ArtifactDigest, CreatedAt: timestamppb.New(createdAt)})
	}
	deviceKind := commonv1.DeviceKind_DEVICE_KIND_HOST
	osName := r.os
	if strings.EqualFold(r.platform, "android") {
		osName = "android"
		if len(verdict.Targets) == 1 && verdict.Targets[0].Descriptor != nil {
			if strings.HasPrefix(verdict.Targets[0].Descriptor.GetTargetId(), "android:emulator:") {
				deviceKind = commonv1.DeviceKind_DEVICE_KIND_EMULATOR
			} else {
				deviceKind = commonv1.DeviceKind_DEVICE_KIND_PHYSICAL
			}
		}
	}
	target := &commonv1.EvidenceTarget{Ramp: r.ramp, Platform: r.platform, Os: osName, DeviceKind: deviceKind}
	_, err = r.client.ReportTargetVerdict(ctx, connect.NewRequest(&evidencev1.ReportTargetVerdictRequest{ProfileId: profile, GitCommitHash: commit, Verdict: &commonv1.TargetVerdict{Target: target, Disposition: disposition, Refs: refs, RunId: verdict.RunID, Detail: string(detail), EvidenceClass: "release-grade"}}))
	if err != nil {
		return fmt.Errorf("report matrix release verdict: %w", err)
	}
	return nil
}

type ReleaseGateView struct {
	MatrixID          string   `json:"matrix_id,omitempty"`
	Passed            bool     `json:"passed"`
	Disposition       string   `json:"disposition,omitempty"`
	RequiredCellCount int32    `json:"required_cell_count"`
	PassingCellCount  int32    `json:"passing_cell_count"`
	MissingCellIDs    []string `json:"missing_cell_ids,omitempty"`
	FailedCellIDs     []string `json:"failed_cell_ids,omitempty"`
	Reason            string   `json:"reason,omitempty"`
}

func releaseGateView(gate interface{ GetMatrixId() string }) *ReleaseGateView {
	if gate == nil {
		return nil
	}
	value, ok := gate.(*domainv1.ReleaseGate)
	if !ok || value == nil {
		return nil
	}
	return &ReleaseGateView{MatrixID: value.GetMatrixId(), Passed: value.GetPassed(), Disposition: value.GetDisposition().String(), RequiredCellCount: value.GetRequiredCellCount(), PassingCellCount: value.GetPassingCellCount(), MissingCellIDs: append([]string(nil), value.GetMissingCellIds()...), FailedCellIDs: append([]string(nil), value.GetFailedCellIds()...), Reason: value.GetReason()}
}
