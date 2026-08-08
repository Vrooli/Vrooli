package validationmatrix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
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
	platform string
}

func NewDeploymentReporter(client DeploymentEvidenceClient, profileID, gitCommit string) *DeploymentReporter {
	return &DeploymentReporter{client: client, profile: profileID, commit: gitCommit, platform: runtime.GOOS}
}

func NewDeploymentReporterFromURL(baseURL, profileID, gitCommit string, httpClient *http.Client) *DeploymentReporter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return NewDeploymentReporter(evidencev1connect.NewEvidenceServiceClient(httpClient, strings.TrimRight(baseURL, "/")), profileID, gitCommit)
}

func (r *DeploymentReporter) ReportValidationGate(ctx context.Context, verdict ReleaseVerdict) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("deployment-manager evidence client is unavailable")
	}
	if strings.TrimSpace(r.profile) == "" || strings.TrimSpace(r.commit) == "" {
		return fmt.Errorf("deployment-manager profile and git commit are required")
	}
	disposition := commonv1.Disposition_DISPOSITION_FAILED
	if verdict.Gate != nil && verdict.Gate.GetPassed() {
		disposition = commonv1.Disposition_DISPOSITION_PASSED
	}
	detail, err := json.Marshal(struct {
		Producer       string           `json:"producer"`
		MatrixID       string           `json:"matrix_id"`
		ScenarioName   string           `json:"scenario_name"`
		ArtifactDigest string           `json:"artifact_digest"`
		Gate           *ReleaseGateView `json:"gate"`
	}{
		Producer:       "scenario-to-desktop",
		MatrixID:       verdict.MatrixID,
		ScenarioName:   verdict.ScenarioName,
		ArtifactDigest: verdict.ArtifactDigest,
		Gate:           releaseGateView(verdict.Gate),
	})
	if err != nil {
		return fmt.Errorf("encode matrix release detail: %w", err)
	}
	refs := make([]*commonv1.EvidenceRef, 0, len(verdict.Evidence))
	for _, evidence := range verdict.Evidence {
		if evidence == nil {
			continue
		}
		refs = append(refs, &commonv1.EvidenceRef{Producer: "scenario-to-desktop", ArtifactId: evidence.GetEvidenceId(), Kind: evidence.GetKind().String(), Checksum: evidence.GetSha256()})
	}
	target := &commonv1.EvidenceTarget{Ramp: "scenario-to-desktop", Platform: r.platform, Os: r.platform, DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}
	_, err = r.client.ReportTargetVerdict(ctx, connect.NewRequest(&evidencev1.ReportTargetVerdictRequest{ProfileId: r.profile, GitCommitHash: r.commit, Verdict: &commonv1.TargetVerdict{Target: target, Disposition: disposition, Refs: refs, RunId: verdict.RunID, Detail: string(detail)}}))
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
