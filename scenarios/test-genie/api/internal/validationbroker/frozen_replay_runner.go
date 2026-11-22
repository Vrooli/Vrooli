package validationbroker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	workspace "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace/workspaceconnect"
)

const (
	frozenReplayMaximumRunSeconds  = 600
	frozenReplayMaximumWaitSeconds = 30
)

type FrozenReplayTestOutcome string

const (
	FrozenReplayRunning              FrozenReplayTestOutcome = "running"
	FrozenReplayPassed               FrozenReplayTestOutcome = "passed"
	FrozenReplayAssertionFailed      FrozenReplayTestOutcome = "assertion_failed"
	FrozenReplayInfrastructureFailed FrozenReplayTestOutcome = "infrastructure_failed"
	FrozenReplayInterrupted          FrozenReplayTestOutcome = "interrupted"
)

type FrozenReplayTestResult struct {
	Success               bool
	OperationID           string
	CaseID                string
	Role                  validationv1.WorkspaceSandboxSealRole
	SealDigest            string
	EvaluatorBundleID     string
	EvaluatorBundleDigest string
	EntryPoint            string
	Outcome               FrozenReplayTestOutcome
	ResultDigest          string
	SealIntactBefore      bool
	SealIntactAfter       bool
	Detail                string
}

type FrozenReplayTestRunner interface {
	Start(context.Context, string, *validationv1.FrozenReplayExecution, FrozenEvaluatorBundle, time.Duration) (FrozenReplayTestResult, error)
	Wait(context.Context, FrozenReplayTestResult, time.Duration) (FrozenReplayTestResult, error)
}

// LiveFrozenReplayTestRunner is the only production bridge to Workspace
// Sandbox's contained replay-test operation. The request carries bounded
// evaluator bytes and content identities; it never carries a host path/argv.
type LiveFrozenReplayTestRunner struct{}

func (LiveFrozenReplayTestRunner) client(ctx context.Context) (workspaceconnect.WorkspaceSandboxServiceClient, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "workspace-sandbox")
	if err != nil {
		return nil, fmt.Errorf("resolve workspace-sandbox URL: %w", err)
	}
	return workspaceconnect.NewWorkspaceSandboxServiceClient(http.DefaultClient, url), nil
}

func (r LiveFrozenReplayTestRunner) Start(ctx context.Context, key string, execution *validationv1.FrozenReplayExecution, bundle FrozenEvaluatorBundle, timeout time.Duration) (FrozenReplayTestResult, error) {
	client, err := r.client(ctx)
	if err != nil {
		return FrozenReplayTestResult{}, err
	}
	request := &workspace.StartReplayTestRequest{
		OperationIdempotencyKey: key,
		CaseId:                  execution.GetCaseId(),
		Role:                    workspace.ReplaySealRole(execution.GetRole()),
		SealDigest:              execution.GetSealDigest(),
		EvaluatorBundleId:       execution.GetEvaluatorBundleId(),
		EvaluatorBundleDigest:   bundle.ContentDigest,
		EntryPoint:              execution.GetEvaluatorEntryPoint(),
		TimeoutSeconds:          boundedSeconds(timeout, frozenReplayMaximumRunSeconds),
	}
	for _, file := range bundle.Files {
		request.Files = append(request.Files, &workspace.ReplayTestFile{RelativePath: file.RelativePath, Content: append([]byte(nil), file.Content...), Sha256: file.SHA256})
	}
	response, err := client.StartReplayTest(ctx, connect.NewRequest(request))
	if err != nil {
		return FrozenReplayTestResult{}, err
	}
	return replayTestResultFromStart(response.Msg), nil
}

func (r LiveFrozenReplayTestRunner) Wait(ctx context.Context, prior FrozenReplayTestResult, timeout time.Duration) (FrozenReplayTestResult, error) {
	client, err := r.client(ctx)
	if err != nil {
		return FrozenReplayTestResult{}, err
	}
	response, err := client.WaitReplayTest(ctx, connect.NewRequest(&workspace.WaitReplayTestRequest{OperationId: prior.OperationID, CaseId: prior.CaseID, SealDigest: prior.SealDigest, WaitSeconds: boundedSeconds(timeout, frozenReplayMaximumWaitSeconds)}))
	if err != nil {
		return FrozenReplayTestResult{}, err
	}
	return replayTestResultFromWait(response.Msg), nil
}

func boundedSeconds(duration time.Duration, maximum uint32) uint32 {
	if duration <= 0 {
		return 1
	}
	seconds := duration / time.Second
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds > time.Duration(maximum) {
		return maximum
	}
	if seconds < 1 {
		return 1
	}
	return uint32(seconds)
}

func replayTestResultFromStart(response *workspace.StartReplayTestResponse) FrozenReplayTestResult {
	if response == nil {
		return FrozenReplayTestResult{}
	}
	return FrozenReplayTestResult{Success: response.GetSuccess(), OperationID: response.GetOperationId(), CaseID: response.GetCaseId(), Role: validationv1.WorkspaceSandboxSealRole(response.GetRole()), SealDigest: response.GetSealDigest(), EvaluatorBundleID: response.GetEvaluatorBundleId(), EvaluatorBundleDigest: response.GetEvaluatorBundleDigest(), EntryPoint: response.GetEntryPoint(), Outcome: replayTestOutcome(response.GetOutcome()), ResultDigest: response.GetResultDigest(), SealIntactBefore: response.GetSealIntactBefore(), SealIntactAfter: response.GetSealIntactAfter(), Detail: response.GetDetail()}
}

func replayTestResultFromWait(response *workspace.WaitReplayTestResponse) FrozenReplayTestResult {
	if response == nil {
		return FrozenReplayTestResult{}
	}
	return FrozenReplayTestResult{Success: response.GetSuccess(), OperationID: response.GetOperationId(), CaseID: response.GetCaseId(), Role: validationv1.WorkspaceSandboxSealRole(response.GetRole()), SealDigest: response.GetSealDigest(), EvaluatorBundleID: response.GetEvaluatorBundleId(), EvaluatorBundleDigest: response.GetEvaluatorBundleDigest(), EntryPoint: response.GetEntryPoint(), Outcome: replayTestOutcome(response.GetOutcome()), ResultDigest: response.GetResultDigest(), SealIntactBefore: response.GetSealIntactBefore(), SealIntactAfter: response.GetSealIntactAfter(), Detail: response.GetDetail()}
}

func replayTestOutcome(value workspace.ReplayTestOutcome) FrozenReplayTestOutcome {
	switch value {
	case workspace.ReplayTestOutcome_REPLAY_TEST_OUTCOME_RUNNING:
		return FrozenReplayRunning
	case workspace.ReplayTestOutcome_REPLAY_TEST_OUTCOME_PASSED:
		return FrozenReplayPassed
	case workspace.ReplayTestOutcome_REPLAY_TEST_OUTCOME_ASSERTION_FAILED:
		return FrozenReplayAssertionFailed
	case workspace.ReplayTestOutcome_REPLAY_TEST_OUTCOME_INFRASTRUCTURE_FAILED:
		return FrozenReplayInfrastructureFailed
	case workspace.ReplayTestOutcome_REPLAY_TEST_OUTCOME_INTERRUPTED:
		return FrozenReplayInterrupted
	default:
		return ""
	}
}

func resultDigestValid(value string) bool {
	value = strings.TrimPrefix(strings.ToLower(value), "sha256:")
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
