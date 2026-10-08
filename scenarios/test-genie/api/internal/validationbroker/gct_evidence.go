package validationbroker

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

// GCTEvidenceClient is Test Genie's narrow child-operation seam to Git Control
// Tower. Start and wait are separate so the receipt can durably record the GCT
// handle before blocking. Both starts are idempotent by collection name or
// operation ID, which makes a crash between provider admission and receipt
// projection safe to replay.
type GCTEvidenceClient interface {
	StartCapture(context.Context, GCTCaptureRequest) (string, error)
	WaitCapture(context.Context, GCTCaptureRequest) (GCTEvidenceResult, error)
	StartDiff(context.Context, GCTDiffRequest) (string, error)
	WaitDiff(context.Context, GCTDiffRequest) (GCTEvidenceResult, error)
}

type GCTCaptureRequest struct {
	Collection      string
	ParentReceiptID string
	Targets         []string
	Paths           []string
	Actor           string
}

type GCTDiffRequest struct {
	Collection            string
	ParentReceiptID       string
	OperationID           string
	Targets               []string
	RequireSourceSnapshot bool
}

type GCTEvidenceResult struct {
	Passed   bool
	Detail   string
	Evidence []*validationv1.EvidenceReference
}

type scenarioURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type liveGCTEvidenceClient struct {
	resolver scenarioURLResolver
	http     connect.HTTPClient
}

func NewLiveGCTEvidenceClient() GCTEvidenceClient {
	return &liveGCTEvidenceClient{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: http.DefaultClient}
}

func (c *liveGCTEvidenceClient) client(ctx context.Context) (baselinesconnect.BaselinesServiceClient, error) {
	if c == nil || c.resolver == nil {
		return nil, fmt.Errorf("git-control-tower discovery unavailable")
	}
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, "git-control-tower")
	if err != nil {
		return nil, fmt.Errorf("resolve git-control-tower URL: %w", err)
	}
	httpClient := c.http
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return baselinesconnect.NewBaselinesServiceClient(httpClient, baseURL), nil
}

func (c *liveGCTEvidenceClient) StartCapture(ctx context.Context, req GCTCaptureRequest) (string, error) {
	client, err := c.client(ctx)
	if err != nil {
		return "", err
	}
	targets := make([]*baselinesv1.CollectionTarget, 0, len(req.Targets))
	for _, scenario := range req.Targets {
		targets = append(targets, &baselinesv1.CollectionTarget{Scenario: scenario, BaselineName: req.Collection, Required: true})
	}
	resp, err := client.StartCollectionCapture(ctx, connect.NewRequest(&baselinesv1.StartCollectionCaptureRequest{
		Name: req.Collection, Targets: targets, PathSelections: req.Paths, CreatedBy: req.Actor,
		Reason: "validation receipt behavioral-before evidence", ParentReceiptId: req.ParentReceiptID,
	}))
	if err != nil {
		return "", fmt.Errorf("start GCT collection %s: %w", req.Collection, err)
	}
	if resp.Msg.GetCollection().GetName() == "" {
		return "", fmt.Errorf("GCT returned no collection identity")
	}
	return resp.Msg.GetCollection().GetName(), nil
}

func (c *liveGCTEvidenceClient) WaitCapture(ctx context.Context, req GCTCaptureRequest) (GCTEvidenceResult, error) {
	client, err := c.client(ctx)
	if err != nil {
		return GCTEvidenceResult{}, err
	}
	resp, err := client.WaitCollectionCapture(ctx, connect.NewRequest(&baselinesv1.WaitCollectionCaptureRequest{Name: req.Collection}))
	if err != nil {
		return GCTEvidenceResult{}, fmt.Errorf("wait for GCT collection %s: %w", req.Collection, err)
	}
	collection := resp.Msg.GetCollection()
	coverage := collection.GetCoverage()
	passed := coverage.GetComplete() && coverage.GetRequired() > 0
	detail := fmt.Sprintf("GCT collection %s coverage required=%d ready=%d pending=%d failed=%d skipped=%d stale=%d", req.Collection, coverage.GetRequired(), coverage.GetReady(), coverage.GetPending(), coverage.GetFailed(), coverage.GetSkipped(), coverage.GetStale())
	evidence := []*validationv1.EvidenceReference{{EvidenceId: req.Collection, Kind: "gct-baseline-collection", Owner: "git-control-tower", SubjectId: strings.Join(req.Targets, ","), Uri: "git-control-tower://baseline/collections/" + req.Collection}}
	for _, snapshot := range collection.GetPathSnapshots() {
		evidence = append(evidence, &validationv1.EvidenceReference{EvidenceId: snapshot.GetName(), Kind: "gct-source-snapshot", Owner: "git-control-tower", SubjectId: strings.Join(req.Paths, ","), Uri: "git-control-tower://baseline/path-snapshots/" + snapshot.GetName()})
	}
	return GCTEvidenceResult{Passed: passed, Detail: detail, Evidence: evidence}, nil
}

func (c *liveGCTEvidenceClient) StartDiff(ctx context.Context, req GCTDiffRequest) (string, error) {
	client, err := c.client(ctx)
	if err != nil {
		return "", err
	}
	resp, err := client.StartCollectionDiff(ctx, connect.NewRequest(&baselinesv1.StartCollectionDiffRequest{Name: req.Collection, OperationId: req.OperationID, Scenarios: req.Targets, ParentReceiptId: req.ParentReceiptID}))
	if err != nil {
		return "", fmt.Errorf("start GCT collection diff %s: %w", req.OperationID, err)
	}
	if resp.Msg.GetOperationId() == "" {
		return "", fmt.Errorf("GCT returned no collection diff identity")
	}
	return resp.Msg.GetOperationId(), nil
}

func (c *liveGCTEvidenceClient) WaitDiff(ctx context.Context, req GCTDiffRequest) (GCTEvidenceResult, error) {
	client, err := c.client(ctx)
	if err != nil {
		return GCTEvidenceResult{}, err
	}
	resp, err := client.WaitCollectionDiff(ctx, connect.NewRequest(&baselinesv1.WaitCollectionDiffRequest{Name: req.Collection, OperationId: req.OperationID}))
	if err != nil {
		return GCTEvidenceResult{}, fmt.Errorf("wait for GCT collection diff %s: %w", req.OperationID, err)
	}
	classification := strings.TrimSpace(resp.Msg.GetClassification())
	detail := "GCT collection diff " + req.OperationID + " classified " + classification
	if standing := resp.Msg.GetStanding(); standing != nil && standing.GetDetail() != "" {
		detail += ": " + standing.GetDetail()
	}
	result := GCTEvidenceResult{
		Passed:   classification == "clean",
		Detail:   detail,
		Evidence: []*validationv1.EvidenceReference{{EvidenceId: req.OperationID, Kind: "gct-collection-diff", Owner: "git-control-tower", SubjectId: strings.Join(req.Targets, ","), Uri: "git-control-tower://baseline/collection-diffs/" + req.Collection + "/" + req.OperationID}},
	}
	if !req.RequireSourceSnapshot {
		return result, nil
	}
	for _, before := range resp.Msg.GetCollection().GetPathSnapshots() {
		snapshot, getErr := client.GetPathSnapshot(ctx, connect.NewRequest(&baselinesv1.GetPathSnapshotRequest{Name: before.GetName(), Branch: before.GetBranch()}))
		if getErr != nil {
			return GCTEvidenceResult{}, fmt.Errorf("get GCT source snapshot %s: %w", before.GetName(), getErr)
		}
		afterName := safeEvidenceName(before.GetName() + "-after-" + req.OperationID)
		if _, captureErr := client.CapturePathSnapshot(ctx, connect.NewRequest(&baselinesv1.CapturePathSnapshotRequest{Name: afterName, Branch: before.GetBranch(), Selections: snapshot.Msg.GetSnapshot().GetSelections()})); captureErr != nil {
			return GCTEvidenceResult{}, fmt.Errorf("capture current GCT source snapshot %s: %w", afterName, captureErr)
		}
		diff, diffErr := client.DiffPathSnapshots(ctx, connect.NewRequest(&baselinesv1.DiffPathSnapshotsRequest{BeforeName: before.GetName(), AfterName: afterName, Branch: before.GetBranch()}))
		if diffErr != nil {
			return GCTEvidenceResult{}, fmt.Errorf("diff GCT source snapshot %s: %w", before.GetName(), diffErr)
		}
		result.Evidence = append(result.Evidence, &validationv1.EvidenceReference{EvidenceId: afterName, Kind: "gct-source-snapshot", Owner: "git-control-tower", SubjectId: before.GetName(), Uri: "git-control-tower://baseline/path-snapshots/" + afterName})
		result.Detail += fmt.Sprintf("; source snapshot %s has %d scoped delta(s)", before.GetName(), len(diff.Msg.GetDeltas()))
	}
	return result, nil
}

func safeEvidenceName(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
