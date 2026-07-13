package execution

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	internalvalidation "plan-manager/internal/validation"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

// gctCollectionClient is duplicated at the two application composition roots
// intentionally: each module builds its own validation service and neither
// internal validation nor execution imports a handler package.
type gctCollectionClient struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http    connect.HTTPClient
	timeout time.Duration
}

func newGCTCollectionClient() internalvalidation.BaselineCollectionClient {
	return gctCollectionClient{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: http.DefaultClient, timeout: 2 * time.Minute}
}

func (c gctCollectionClient) StartCollectionCapture(ctx context.Context, req internalvalidation.BaselineCollectionCaptureRequest) (internalvalidation.BaselineCollectionCaptureResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, "git-control-tower")
	if err != nil {
		return internalvalidation.BaselineCollectionCaptureResult{}, fmt.Errorf("resolve git-control-tower URL: %w", err)
	}
	targets := make([]*baselinesv1.CollectionTarget, 0, len(req.Scenarios))
	for _, scenario := range req.Scenarios {
		targets = append(targets, &baselinesv1.CollectionTarget{Scenario: scenario, BaselineName: req.Name, Required: true})
	}
	client := baselinesconnect.NewBaselinesServiceClient(c.http, baseURL)
	resp, err := client.StartCollectionCapture(callCtx, connect.NewRequest(&baselinesv1.StartCollectionCaptureRequest{Name: req.Name, Targets: targets, PathSelections: req.RepoPaths, CreatedBy: "plan-manager", Reason: "execution-start baseline set"}))
	if err != nil {
		return internalvalidation.BaselineCollectionCaptureResult{}, fmt.Errorf("start git-control-tower collection: %w", err)
	}
	result := collectionResult(resp.Msg.GetCollection())
	if result.Complete() {
		return result, nil
	}
	settled, err := client.GetCollection(callCtx, connect.NewRequest(&baselinesv1.GetCollectionRequest{Name: req.Name, Wait: true}))
	if err != nil {
		return internalvalidation.BaselineCollectionCaptureResult{}, fmt.Errorf("wait for git-control-tower collection: %w", err)
	}
	return collectionResult(settled.Msg.GetCollection()), nil
}

func collectionResult(collection *baselinesv1.BaselineCollection) internalvalidation.BaselineCollectionCaptureResult {
	if collection == nil || collection.GetCoverage() == nil {
		return internalvalidation.BaselineCollectionCaptureResult{}
	}
	coverage := collection.GetCoverage()
	result := internalvalidation.BaselineCollectionCaptureResult{Name: collection.GetName(), Branch: collection.GetBranch(), Required: int(coverage.GetRequired()), Ready: int(coverage.GetReady()), Pending: int(coverage.GetPending()), Failed: int(coverage.GetFailed()), Skipped: int(coverage.GetSkipped()), Stale: int(coverage.GetStale())}
	for _, member := range collection.GetMembers() {
		result.Members = append(result.Members, internalvalidation.BaselineCollectionMember{Scenario: member.GetScenario(), BaselineName: member.GetBaselineName(), Required: member.GetRequired(), Status: member.GetStatus(), RunID: member.GetRunId(), GitSHA: member.GetGitSha(), Error: member.GetError()})
	}
	for _, snapshot := range collection.GetPathSnapshots() {
		result.PathSnapshots = append(result.PathSnapshots, internalvalidation.BaselinePathSnapshot{Name: snapshot.GetName(), Branch: snapshot.GetBranch(), CreatedAt: snapshot.GetCreatedAt()})
	}
	return result
}

func (c gctCollectionClient) StartCollectionDiff(ctx context.Context, req internalvalidation.BaselineCollectionDiffRequest) (internalvalidation.BaselineCollectionDiffResult, error) {
	if c.resolver == nil {
		return internalvalidation.BaselineCollectionDiffResult{}, fmt.Errorf("git-control-tower discovery unavailable")
	}
	if c.http == nil {
		c.http = http.DefaultClient
	}
	if c.timeout <= 0 {
		c.timeout = 2 * time.Minute
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, "git-control-tower")
	if err != nil {
		return internalvalidation.BaselineCollectionDiffResult{}, fmt.Errorf("resolve git-control-tower URL: %w", err)
	}
	client := baselinesconnect.NewBaselinesServiceClient(c.http, baseURL)
	if _, err := client.StartCollectionDiff(callCtx, connect.NewRequest(&baselinesv1.StartCollectionDiffRequest{Name: req.Name, OperationId: req.OperationID, Scenarios: req.Scenarios})); err != nil {
		return internalvalidation.BaselineCollectionDiffResult{}, fmt.Errorf("start git-control-tower collection diff: %w", err)
	}
	settled, err := client.GetCollectionDiff(callCtx, connect.NewRequest(&baselinesv1.GetCollectionDiffRequest{Name: req.Name, OperationId: req.OperationID, Wait: true}))
	if err != nil {
		return internalvalidation.BaselineCollectionDiffResult{}, fmt.Errorf("wait for git-control-tower collection diff: %w", err)
	}
	parts := make([]string, 0, len(settled.Msg.GetMembers()))
	for _, member := range settled.Msg.GetMembers() {
		parts = append(parts, member.GetScenario()+":"+member.GetStatus()+":"+member.GetVerdict())
	}
	return internalvalidation.BaselineCollectionDiffResult{OperationID: settled.Msg.GetOperationId(), Classification: settled.Msg.GetClassification(), Detail: strings.Join(parts, ", ")}, nil
}

func (c gctCollectionClient) DiffPathEvidence(ctx context.Context, req internalvalidation.BaselinePathDiffRequest) (internalvalidation.BaselinePathDiffResult, error) {
	if c.resolver == nil {
		return internalvalidation.BaselinePathDiffResult{}, fmt.Errorf("git-control-tower discovery unavailable")
	}
	if c.http == nil {
		c.http = http.DefaultClient
	}
	if c.timeout <= 0 {
		c.timeout = 2 * time.Minute
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, "git-control-tower")
	if err != nil {
		return internalvalidation.BaselinePathDiffResult{}, fmt.Errorf("resolve git-control-tower URL: %w", err)
	}
	client := baselinesconnect.NewBaselinesServiceClient(c.http, baseURL)
	before, err := client.GetPathSnapshot(callCtx, connect.NewRequest(&baselinesv1.GetPathSnapshotRequest{Name: req.BeforeName, Branch: req.Branch}))
	if err != nil {
		return internalvalidation.BaselinePathDiffResult{}, fmt.Errorf("get source evidence snapshot: %w", err)
	}
	if len(before.Msg.GetSnapshot().GetSelections()) == 0 {
		return internalvalidation.BaselinePathDiffResult{}, fmt.Errorf("source evidence snapshot %q has no selections", req.BeforeName)
	}
	afterName := pathEvidenceAfterName(req.BeforeName, req.OperationID)
	if _, err := client.CapturePathSnapshot(callCtx, connect.NewRequest(&baselinesv1.CapturePathSnapshotRequest{Name: afterName, Branch: req.Branch, Selections: before.Msg.GetSnapshot().GetSelections()})); err != nil {
		return internalvalidation.BaselinePathDiffResult{}, fmt.Errorf("capture current source evidence: %w", err)
	}
	diff, err := client.DiffPathSnapshots(callCtx, connect.NewRequest(&baselinesv1.DiffPathSnapshotsRequest{BeforeName: req.BeforeName, AfterName: afterName, Branch: req.Branch, Selections: req.Paths}))
	if err != nil {
		return internalvalidation.BaselinePathDiffResult{}, fmt.Errorf("diff source evidence: %w", err)
	}
	return internalvalidation.BaselinePathDiffResult{AfterName: afterName, Deltas: len(diff.Msg.GetDeltas()), Detail: fmt.Sprintf("informational source evidence: %d scoped path delta(s), after snapshot %s", len(diff.Msg.GetDeltas()), afterName)}, nil
}

func pathEvidenceAfterName(beforeName, operationID string) string {
	var b strings.Builder
	for _, r := range beforeName + "-after-" + operationID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
