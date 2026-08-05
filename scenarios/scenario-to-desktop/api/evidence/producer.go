package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/smoketest"
)

type EvidenceClient interface {
	ReportTargetVerdict(context.Context, *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error)
}

// ConnectReporter reports reference-only evidence to deployment-manager.
type ConnectReporter struct {
	client EvidenceClient
}

func NewConnectReporter(client EvidenceClient) *ConnectReporter {
	return &ConnectReporter{client: client}
}

func NewConnectReporterFromURL(baseURL string, httpClient *http.Client) *ConnectReporter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return NewConnectReporter(evidencev1connect.NewEvidenceServiceClient(httpClient, strings.TrimRight(baseURL, "/")))
}

var _ smoketest.EvidenceReporter = (*ConnectReporter)(nil)

// CaptureToEvidenceRef maps the producer-owned capture metadata without
// reading or recomputing the artifact bytes. The consumer receives identity,
// checksum, and size only.
func CaptureToEvidenceRef(value captures.Capture) *commonv1.EvidenceRef {
	return &commonv1.EvidenceRef{
		Producer:   "scenario-to-desktop",
		ArtifactId: value.ID,
		Kind:       string(value.Type),
		Checksum:   value.Checksum,
		SizeBytes:  value.FileSizeBytes,
		CreatedAt:  timestamppb.New(value.CreatedAt),
	}
}

// TargetToEvidenceTarget maps the scenario target to the shared contract.
func TargetToEvidenceTarget(target *domainv1.EvidenceTarget, platform string) *commonv1.EvidenceTarget {
	result := &commonv1.EvidenceTarget{Ramp: "scenario-to-desktop", Platform: strings.ToLower(platform), Os: strings.ToLower(platform), DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}
	if target == nil {
		return result
	}
	if target.BridgeNodeId != nil {
		result.BridgeNodeId = target.BridgeNodeId
	}
	if target.BridgeJobId != nil {
		result.BridgeJobId = target.BridgeJobId
	}
	if target.Kind == domainv1.EvidenceTarget_KIND_BRIDGE_NODE {
		result.DeviceKind = commonv1.DeviceKind_DEVICE_KIND_PHYSICAL
	}
	return result
}

func disposition(value string) commonv1.Disposition {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pass", "passed":
		return commonv1.Disposition_DISPOSITION_PASSED
	case "degraded", "failed", "failure":
		return commonv1.Disposition_DISPOSITION_FAILED
	default:
		return commonv1.Disposition_DISPOSITION_FAILED
	}
}

// ReportJourney sends a terminal verdict and only references to the captures.
// A transport failure is returned to the caller and is never converted into a
// successful result.
func (r *ConnectReporter) ReportJourney(ctx context.Context, input smoketest.EvidenceReportInput) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("deployment-manager evidence client is unavailable")
	}
	if strings.TrimSpace(input.ProfileID) == "" || strings.TrimSpace(input.GitCommit) == "" {
		return fmt.Errorf("profile ID and git commit are required for evidence reporting")
	}
	refs := make([]*commonv1.EvidenceRef, 0, 2)
	var hasJourney, hasRecording bool
	for _, item := range input.Captures {
		switch item.Type {
		case captures.CaptureJourney:
			hasJourney = true
			refs = append(refs, CaptureToEvidenceRef(item))
		case captures.CaptureRecording:
			hasRecording = true
			refs = append(refs, CaptureToEvidenceRef(item))
		}
	}
	if !hasJourney || !hasRecording {
		return fmt.Errorf("journey and recording references are both required")
	}
	verdict := &commonv1.TargetVerdict{
		Target:      TargetToEvidenceTarget(input.Target, input.Platform),
		Disposition: disposition(input.Disposition),
		Refs:        refs,
		RunId:       input.RunID,
		Detail:      journeyDetail(input),
	}
	_, err := r.client.ReportTargetVerdict(ctx, connect.NewRequest(&evidencev1.ReportTargetVerdictRequest{ProfileId: input.ProfileID, GitCommitHash: input.GitCommit, Verdict: verdict}))
	if err != nil {
		return fmt.Errorf("report evidence verdict: %w", err)
	}
	return nil
}

func journeyDetail(input smoketest.EvidenceReportInput) string {
	type detail struct {
		Producer         string                   `json:"producer"`
		RecordingURL     string                   `json:"recording_url,omitempty"`
		JourneyCaptureID string                   `json:"journey_capture_id,omitempty"`
		RecordingID      string                   `json:"recording_id,omitempty"`
		Journey          *smoketest.JourneyResult `json:"journey,omitempty"`
	}
	value := detail{Producer: "scenario-to-desktop", Journey: input.Journey}
	for _, item := range input.Captures {
		switch item.Type {
		case captures.CaptureJourney:
			value.JourneyCaptureID = item.ID
		case captures.CaptureRecording:
			value.RecordingID = item.ID
			if input.ProducerBaseURL != "" {
				value.RecordingURL = strings.TrimRight(input.ProducerBaseURL, "/") + "/api/v1/captures/" + input.ScenarioName + "/" + item.ID + "/file"
			}
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "scenario-to-desktop scripted desktop journey"
	}
	return string(data)
}
