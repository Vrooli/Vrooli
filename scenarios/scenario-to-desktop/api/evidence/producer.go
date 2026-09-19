package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	readinessv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
	"scenario-to-desktop-api/captures"
	serviceauth "scenario-to-desktop-api/shared/auth"
	"scenario-to-desktop-api/smoketest"
)

type EvidenceClient interface {
	ReportTargetVerdict(context.Context, *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error)
}

type ReadinessClient interface {
	ReportEvidence(context.Context, *connect.Request[readinessv1.ReportEvidenceRequest]) (*connect.Response[readinessv1.ReportEvidenceResponse], error)
}

type ClientUpdateReceiptClient interface {
	RecordClientUpdateReceipt(context.Context, *connect.Request[releasesv1.RecordClientUpdateReceiptRequest]) (*connect.Response[releasesv1.RecordClientUpdateReceiptResponse], error)
}

// ClientUpdateReceiptInput is the owner-produced, finalized update result.
// The caller must supply the exact successor digest and installed version;
// this package never infers them from a downloaded or verified candidate.
type ClientUpdateReceiptInput struct {
	ReleaseID       string
	CandidateID     string
	PredecessorRef  string
	SuccessorDigest string
	TargetID        string
	VerifiedVersion string
	ExternalReceipt string
	ObservedAt      time.Time
}

// ConnectReporter reports reference-only evidence to deployment-manager.
type ConnectReporter struct {
	client    EvidenceClient
	readiness ReadinessClient
	releases  ClientUpdateReceiptClient
}

func NewConnectReporter(client EvidenceClient) *ConnectReporter {
	return &ConnectReporter{client: client}
}

func NewConnectReporterWithClients(evidence EvidenceClient, readiness ReadinessClient) *ConnectReporter {
	return &ConnectReporter{client: evidence, readiness: readiness}
}

func NewConnectReporterWithAllClients(evidence EvidenceClient, readiness ReadinessClient, releases ClientUpdateReceiptClient) *ConnectReporter {
	return &ConnectReporter{client: evidence, readiness: readiness, releases: releases}
}

func NewConnectReporterFromURL(baseURL string, httpClient *http.Client) *ConnectReporter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	httpClient = withDeploymentManagerServiceAuth(httpClient)
	baseURL = strings.TrimRight(baseURL, "/")
	return NewConnectReporterWithAllClients(
		evidencev1connect.NewEvidenceServiceClient(httpClient, baseURL),
		readinessv1connect.NewReadinessServiceClient(httpClient, baseURL),
		releasesv1connect.NewReleasesServiceClient(httpClient, baseURL),
	)
}

type deploymentManagerServiceAuthTransport struct {
	base http.RoundTripper
}

func (t deploymentManagerServiceAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	token := serviceauth.DeploymentManagerServiceToken()
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

// ReportClientUpdateReceipt routes a finalized owner receipt to the durable
// release identity ledger. Incomplete receipts fail before transport.
func (r *ConnectReporter) ReportClientUpdateReceipt(ctx context.Context, input ClientUpdateReceiptInput) error {
	if r == nil || r.releases == nil {
		return fmt.Errorf("deployment-manager release client is unavailable")
	}
	input.ReleaseID = strings.TrimSpace(input.ReleaseID)
	input.CandidateID = strings.TrimSpace(input.CandidateID)
	input.PredecessorRef = strings.TrimSpace(input.PredecessorRef)
	input.SuccessorDigest = strings.TrimSpace(input.SuccessorDigest)
	input.TargetID = strings.TrimSpace(input.TargetID)
	input.VerifiedVersion = strings.TrimSpace(input.VerifiedVersion)
	input.ExternalReceipt = strings.TrimSpace(input.ExternalReceipt)
	if input.ReleaseID == "" || input.CandidateID == "" || input.PredecessorRef == "" || input.SuccessorDigest == "" || input.TargetID == "" || input.VerifiedVersion == "" || input.ExternalReceipt == "" || input.ObservedAt.IsZero() {
		return fmt.Errorf("complete client update receipt identity is required")
	}
	_, err := r.releases.RecordClientUpdateReceipt(ctx, connect.NewRequest(&releasesv1.RecordClientUpdateReceiptRequest{
		ReleaseId: input.ReleaseID,
		Receipt: &releasesv1.ClientUpdateReceipt{
			CandidateId: input.CandidateID, PredecessorRef: input.PredecessorRef, SuccessorDigest: input.SuccessorDigest,
			TargetId: input.TargetID, VerifiedVersion: input.VerifiedVersion,
			Outcome: releasesv1.ReceiptOutcome_RECEIPT_OUTCOME_VERIFIED, Producer: "scenario-to-desktop",
			ExternalReceipt: input.ExternalReceipt, ObservedAt: timestamppb.New(input.ObservedAt),
		},
	}))
	if err != nil {
		return fmt.Errorf("record client update receipt: %w", err)
	}
	return nil
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
		EvidenceClass: "desktop-journey",
		Refs:        refs,
		RunId:       input.RunID,
		Detail:      journeyDetail(input),
	}
	_, err := r.client.ReportTargetVerdict(ctx, connect.NewRequest(&evidencev1.ReportTargetVerdictRequest{ProfileId: input.ProfileID, GitCommitHash: input.GitCommit, Verdict: verdict}))
	if err != nil {
		return fmt.Errorf("report evidence verdict: %w", err)
	}
	if r.readiness != nil {
		if err := r.reportReadiness(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

func (r *ConnectReporter) reportReadiness(ctx context.Context, input smoketest.EvidenceReportInput) error {
	if r == nil || r.readiness == nil {
		return nil
	}
	if strings.TrimSpace(input.ArtifactDigest) == "" {
		return fmt.Errorf("report readiness evidence: artifact digest is required")
	}
	channel := strings.TrimSpace(input.Channel)
	if channel == "" {
		channel = "stable"
	}
	status := "failed"
	if strings.EqualFold(strings.TrimSpace(input.Disposition), "pass") {
		status = "passed"
	} else if strings.EqualFold(strings.TrimSpace(input.Disposition), "unavailable") {
		status = "unavailable"
	}
	policyVersion := input.PolicyVersion
	if policyVersion == 0 {
		policyVersion = 2
	}
	reference := "run:" + strings.TrimSpace(input.RunID)
	if strings.TrimSpace(input.RunID) == "" {
		return fmt.Errorf("report readiness evidence: run ID is required")
	}
	target := strings.TrimSpace(input.Platform)
	if target == "" {
		return fmt.Errorf("report readiness evidence: platform is required")
	}
	_, err := r.readiness.ReportEvidence(ctx, connect.NewRequest(&readinessv1.ReportEvidenceRequest{
		Scenario: input.ScenarioName, ProfileId: input.ProfileID, CandidateCommit: input.GitCommit,
		ArtifactDigest: input.ArtifactDigest, Targets: []string{target}, Channel: channel,
		PolicyVersion: int32(policyVersion), CriterionId: "platform-delivery-proven",
		ProducerBinding: "scenario-to-desktop.evidence.readiness", ProducerVersion: "scenario-to-desktop",
		Status: status, ObservedAt: timestamppb.New(time.Now().UTC()), EvidenceReference: reference,
		Detail:      "scenario-to-desktop owner journey completed: " + strings.TrimSpace(input.Disposition),
		CandidateId: input.CandidateID, DestinationRevisionId: input.DestinationRevisionID,
		AuthorizationEpoch: input.AuthorizationEpoch,
	}))
	if err != nil {
		return fmt.Errorf("report readiness evidence: %w", err)
	}
	return nil
}

func journeyDetail(input smoketest.EvidenceReportInput) string {
	type detail struct {
		Producer         string                      `json:"producer"`
		RecordingURL     string                      `json:"recording_url,omitempty"`
		JourneyCaptureID string                      `json:"journey_capture_id,omitempty"`
		RecordingID      string                      `json:"recording_id,omitempty"`
		Journey          *deliveryramp.JourneyResult `json:"journey,omitempty"`
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
