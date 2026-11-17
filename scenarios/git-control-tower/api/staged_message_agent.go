package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/stagedmessages"
	"github.com/vrooli/api-core/provenance"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	stagedMessageWorkflowKey     = "git-control-tower/staged-message"
	stagedMessageDeepWorkflowKey = "git-control-tower/staged-message-deep"
	// AM validates workflow input through its 64 KiB structured-value boundary.
	// The complete subject stays in the existing GCT job; only this projection
	// crosses the workflow boundary and becomes model context.
	stagedMessageMaxWorkflowInputBytes = 56 << 10
	stagedMessageMaxPromptBytes        = 48 << 10
)

type stagedMessageWorkflowAPI interface {
	StartWorkflowExecution(context.Context, *connect.Request[apipb.StartWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error)
	GetWorkflowExecution(context.Context, *connect.Request[apipb.GetWorkflowExecutionRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error)
	GetWorkflowExecutionResult(context.Context, *connect.Request[apipb.GetWorkflowExecutionResultRequest]) (*connect.Response[apipb.WorkflowExecutionResponse], error)
	WaitWorkflowExecution(context.Context, *connect.Request[apipb.WaitWorkflowExecutionRequest]) (*connect.Response[apipb.WaitWorkflowExecutionResponse], error)
	CancelWorkflowExecution(context.Context, *connect.Request[apipb.WorkflowExecutionOperationRequest]) (*connect.Response[apipb.WorkflowExecutionOperationResponse], error)
}

type stagedMessageSearchAPI interface {
	SearchConversations(context.Context, *connect.Request[ampb.SearchConversationsRequest]) (*connect.Response[ampb.SearchConversationsResponse], error)
}

type stagedMessageAgent struct {
	resolve      func(context.Context) (string, error)
	httpClient   *http.Client
	workflow     func(string) stagedMessageWorkflowAPI
	search       func(string) stagedMessageSearchAPI
	repository   func(context.Context, string) (string, error)
	searchBudget time.Duration
	inventory    func(context.Context, string, string, string, string) (map[int]bool, error)
	evidence     func(context.Context, string, string, string, string) (map[int]bool, error)
}

func newStagedMessageProvider(s *Server) stagedmessages.Provider {
	client := s.agentManagerClient
	if client == nil {
		client = NewAgentManagerClient(70 * time.Second)
	}
	httpClient := client.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 70 * time.Second}
	}
	forwardingClient := *httpClient
	transport := httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	forwardingClient.Transport = provenance.ForwardingTransport{Base: transport}
	httpClient = &forwardingClient
	return &stagedMessageAgent{
		httpClient: httpClient,
		inventory: func(ctx context.Context, job, repo, digest, execution string) (map[int]bool, error) {
			if s.stagedMessages == nil {
				return nil, errors.New("durable evidence owner unavailable")
			}
			return s.stagedMessages.InventoryEligibility(ctx, job, repo, digest, execution)
		},
		evidence: func(ctx context.Context, job, repo, digest, execution string) (map[int]bool, error) {
			if s.stagedMessages == nil {
				return nil, errors.New("durable evidence owner unavailable")
			}
			return s.stagedMessages.EvidenceEligibility(ctx, job, repo, digest, execution)
		},
		resolve: client.resolveBaseURL,
		workflow: func(base string) stagedMessageWorkflowAPI {
			return apiconnect.NewAgentManagerServiceClient(httpClient, base)
		},
		search: func(base string) stagedMessageSearchAPI {
			return domainconnect.NewConversationSearchServiceClient(httpClient, base)
		},
		repository: func(ctx context.Context, id string) (string, error) {
			repo, err := s.resolveRepoForConnect(ctx, id, "")
			if err != nil {
				return "", err
			}
			root, err := filepath.EvalSymlinks(repo.Path)
			if err != nil {
				return "", fmt.Errorf("resolve registered checkout: %w", err)
			}
			return filepath.Abs(root)
		},
	}
}

func (p *stagedMessageAgent) Readiness(ctx context.Context) stagedmessages.ProviderReadiness {
	unknown := stagedmessages.ProviderReadiness{Provider: "workspace-sandbox", State: "unknown", Known: false, Message: "Workspace Sandbox readiness could not be confirmed."}
	if p == nil || p.resolve == nil || p.httpClient == nil {
		return unknown
	}
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	base, err := p.resolve(readCtx)
	if err != nil || strings.TrimSpace(base) == "" {
		return unknown
	}
	request, err := http.NewRequestWithContext(readCtx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/provider-readiness", nil)
	if err != nil {
		return unknown
	}
	for name, values := range policygate.VerifiedOwnerHeaders(ctx) {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := p.httpClient.Do(request)
	if err != nil {
		return unknown
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return unknown
	}
	var readiness stagedmessages.ProviderReadiness
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<10)).Decode(&readiness); err != nil {
		return unknown
	}
	if readiness.Provider != "workspace-sandbox" || !readiness.Known || (readiness.State != "open" && readiness.State != "closed") || readiness.ObservedAt == nil || readiness.ObservedAt.IsZero() {
		return unknown
	}
	if age := time.Since(*readiness.ObservedAt); age < -time.Minute || age > 30*time.Second {
		return unknown
	}
	return readiness
}

func stagedMessageRequest[T any](ctx context.Context, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	for name, values := range policygate.VerifiedOwnerHeaders(ctx) {
		req.Header()[name] = append([]string(nil), values...)
	}
	// Verified agent credentials use api-core's request-context forwarding
	// transport. Never substitute the server's own agent identity or grant.
	return req
}

func (p *stagedMessageAgent) Start(ctx context.Context, in stagedmessages.DraftInput, key string) (stagedmessages.Execution, error) {
	if in.Snapshot == nil || in.Snapshot.GetDigest() == "" || len(in.Snapshot.GetFiles()) == 0 || strings.TrimSpace(key) == "" {
		return stagedmessages.Execution{}, connect.NewError(connect.CodeInvalidArgument, errors.New("immutable staged input and request key are required"))
	}
	if len(in.Contexts) > stagedmessages.MaxSelectedContexts || len(in.OperatorNote) > stagedmessages.MaxOperatorNoteBytes {
		return stagedmessages.Execution{}, connect.NewError(connect.CodeInvalidArgument, errors.New("draft input exceeds context or note bounds"))
	}
	// Agent Manager owns the workflow's trigger policy. When GCT has no
	// forwardable browser or agent proof, this request remains programmatic;
	// it is never promoted to a human initiator. The owner may admit or refuse
	// that initiator according to the workflow definition.
	if p.repository == nil {
		return stagedmessages.Execution{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("registered repository resolver unavailable"))
	}
	directory, err := p.repository(ctx, in.Snapshot.GetRepositoryId())
	if err != nil {
		return stagedmessages.Execution{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("registered repository unavailable: %w", err))
	}
	if len(directory) > 4096 || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return stagedmessages.Execution{}, connect.NewError(connect.CodeFailedPrecondition, errors.New("registered repository directory is invalid"))
	}
	input, err := packageStagedMessageInput(in, directory)
	if err != nil {
		return stagedmessages.Execution{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	value, err := structpb.NewValue(input)
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	base, err := p.resolve(ctx)
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	req := stagedMessageRequest(ctx, &apipb.StartWorkflowExecutionRequest{Owner: "git-control-tower", WorkflowKey: stagedMessageInputWorkflowKey(in), Input: value, IdempotencyKey: key})
	response, err := p.workflow(base).StartWorkflowExecution(ctx, req)
	if err != nil {
		return stagedmessages.Execution{}, fmt.Errorf("start message workflow: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetExecution() == nil {
		return stagedmessages.Execution{}, errors.New("message workflow returned no execution")
	}
	execution := response.Msg.GetExecution()
	if execution.GetIdempotencyKey() != key {
		return stagedmessages.Execution{}, errors.New("message workflow returned an execution for a different request key")
	}
	return p.project(ctx, p.workflow(base), execution, in)
}

// CancelByKey atomically installs an owner-side cancellation fence and cancels
// an execution already admitted with the exact workflow key, if one exists.
func (p *stagedMessageAgent) CancelByKey(ctx context.Context, key string, in stagedmessages.DraftInput) (stagedmessages.DispatchCancellation, error) {
	if strings.TrimSpace(key) == "" {
		return stagedmessages.DispatchCancellation{}, connect.NewError(connect.CodeInvalidArgument, errors.New("workflow idempotency key is required"))
	}
	base, err := p.resolve(ctx)
	if err != nil {
		return stagedmessages.DispatchCancellation{}, err
	}
	workflowKey := stagedMessageInputWorkflowKey(in)
	body, err := json.Marshal(struct {
		Owner          string `json:"owner"`
		WorkflowKey    string `json:"workflowKey"`
		IdempotencyKey string `json:"idempotencyKey"`
		OperationKey   string `json:"operationKey"`
		Reason         string `json:"reason"`
	}{
		Owner: "git-control-tower", WorkflowKey: workflowKey, IdempotencyKey: key,
		OperationKey: stagedMessageCancelOperationKey(key), Reason: "Operator cancelled message drafting",
	})
	if err != nil {
		return stagedmessages.DispatchCancellation{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/workflow-executions/cancel-by-idempotency-key", bytes.NewReader(body))
	if err != nil {
		return stagedmessages.DispatchCancellation{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	for name, values := range policygate.VerifiedOwnerHeaders(ctx) {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := p.httpClient.Do(request)
	if err != nil {
		return stagedmessages.DispatchCancellation{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return stagedmessages.DispatchCancellation{}, fmt.Errorf("Agent Manager workflow cancellation fence returned HTTP %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return stagedmessages.DispatchCancellation{}, fmt.Errorf("read Agent Manager workflow cancellation response: %w", err)
	}
	var payload struct {
		Fenced    bool            `json:"fenced"`
		Execution json.RawMessage `json:"execution"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return stagedmessages.DispatchCancellation{}, fmt.Errorf("decode Agent Manager workflow cancellation response: %w", err)
	}
	if !payload.Fenced {
		return stagedmessages.DispatchCancellation{}, errors.New("Agent Manager did not confirm the cancellation fence")
	}
	result := stagedmessages.DispatchCancellation{Fenced: true}
	if len(payload.Execution) == 0 || string(payload.Execution) == "null" {
		return result, nil
	}
	execution := &ampb.WorkflowExecution{}
	if err := protojson.Unmarshal(payload.Execution, execution); err != nil {
		return stagedmessages.DispatchCancellation{}, fmt.Errorf("decode fenced Agent Manager execution: %w", err)
	}
	if execution.GetIdempotencyKey() != key {
		return stagedmessages.DispatchCancellation{}, errors.New("Agent Manager cancellation returned a different idempotency key")
	}
	if err := stagedMessageExecutionIdentity(execution, workflowKey); err != nil {
		return stagedmessages.DispatchCancellation{}, connect.NewError(connect.CodePermissionDenied, err)
	}
	projected, err := p.project(ctx, p.workflow(base), execution, in)
	if err != nil {
		return stagedmessages.DispatchCancellation{}, err
	}
	result.Execution = &projected
	return result, nil
}

func stagedMessageCancelOperationKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return "gct/staged-message/cancel-by-key/" + hex.EncodeToString(digest[:])
}

func (p *stagedMessageAgent) Get(ctx context.Context, id string, in stagedmessages.DraftInput) (stagedmessages.Execution, error) {
	base, err := p.resolve(ctx)
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	client := p.workflow(base)
	response, err := client.GetWorkflowExecution(ctx, stagedMessageRequest(ctx, &apipb.GetWorkflowExecutionRequest{ExecutionId: id}))
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	if response.Msg.GetExecution().GetId() != id {
		return stagedmessages.Execution{}, errors.New("execution owner returned a different requested handle")
	}
	return p.project(ctx, client, response.Msg.GetExecution(), in)
}

func (p *stagedMessageAgent) Wait(ctx context.Context, id string, in stagedmessages.DraftInput, timeout time.Duration) (stagedmessages.Execution, error) {
	base, err := p.resolve(ctx)
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	seconds := int32((timeout + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 60 {
		seconds = 60
	}
	client := p.workflow(base)
	response, err := client.WaitWorkflowExecution(ctx, stagedMessageRequest(ctx, &apipb.WaitWorkflowExecutionRequest{ExecutionId: id, TimeoutSeconds: seconds}))
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	if response.Msg.GetExecution().GetId() != id {
		return stagedmessages.Execution{}, errors.New("execution owner returned a different requested handle")
	}
	return p.project(ctx, client, response.Msg.GetExecution(), in)
}

func (p *stagedMessageAgent) Cancel(ctx context.Context, id string, in stagedmessages.DraftInput) (stagedmessages.Execution, error) {
	base, err := p.resolve(ctx)
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	client := p.workflow(base)
	current, err := client.GetWorkflowExecution(ctx, stagedMessageRequest(ctx, &apipb.GetWorkflowExecutionRequest{ExecutionId: id}))
	if err != nil {
		return stagedmessages.Execution{}, err
	}
	execution := current.Msg.GetExecution()
	if execution.GetId() != id {
		return stagedmessages.Execution{}, errors.New("execution owner returned a different requested handle")
	}
	if err := stagedMessageExecutionIdentity(execution, stagedMessageInputWorkflowKey(in)); err != nil {
		return stagedmessages.Execution{}, err
	}
	if status := stagedMessageStatus(execution.GetStatus()); status != "queued" && status != "running" && status != "cancelling" {
		return p.project(ctx, client, execution, in)
	}
	response, err := client.CancelWorkflowExecution(ctx, stagedMessageRequest(ctx, &apipb.WorkflowExecutionOperationRequest{
		ExecutionId: id, IdempotencyKey: "gct/staged-message/cancel/" + id, ExpectedVersion: execution.GetVersion(), Reason: "Operator cancelled message drafting",
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeFailedPrecondition {
			return p.Get(ctx, id, in)
		}
		return stagedmessages.Execution{}, err
	}
	if response.Msg.GetExecution().GetId() != id {
		return stagedmessages.Execution{}, errors.New("execution owner returned a different requested handle")
	}
	return p.project(ctx, client, response.Msg.GetExecution(), in)
}

func stagedMessageInputWorkflowKey(in stagedmessages.DraftInput) string {
	if in.FormatVersion >= 3 {
		return stagedMessageDeepWorkflowKey
	}
	return stagedMessageWorkflowKey
}

func stagedMessageExecutionIdentity(e *ampb.WorkflowExecution, expected ...string) error {
	key := stagedMessageWorkflowKey
	if len(expected) > 0 {
		key = expected[0]
	}
	if e == nil || e.GetId() == "" || e.GetOwner() != "git-control-tower" || e.GetWorkflowKey() != key {
		return errors.New("message workflow returned missing or foreign execution identity")
	}
	return nil
}

func (p *stagedMessageAgent) project(ctx context.Context, client stagedMessageWorkflowAPI, e *ampb.WorkflowExecution, in stagedmessages.DraftInput) (stagedmessages.Execution, error) {
	if err := stagedMessageExecutionIdentity(e, stagedMessageInputWorkflowKey(in)); err != nil {
		return stagedmessages.Execution{}, err
	}
	out := stagedmessages.Execution{ID: e.GetId(), Status: stagedMessageStatus(e.GetStatus()), Phase: e.GetCurrentNodeId()}
	out.Error = stagedMessageTerminalError(e)
	if reason := e.GetTerminalReason(); reason != nil {
		out.ErrorCode = reason.GetCode()
		out.Retryable = reason.GetRetryable()
		if reason.GetCode() == "structured_result_invalid" {
			// Legacy staged-message revisions required JSON even when the agent
			// returned the new XML contract. A new GCT retry adopts the saved input
			// under the active raw-handoff revision, so this format failure is
			// recoverable without changing the staged selection.
			out.Retryable = true
		}
		if reason.GetCode() == "provider_admission_maintenance" {
			out.Error = "Agent execution is paused for Workspace Sandbox maintenance; no agent work started."
			out.ExecutionStarted = false
			out.ExecutionStartedKnown = true
			out.TokensUsed, out.TokensKnown = 0, true
		}
	}
	if e.GetStatus() == ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED {
		out.ErrorCode = "budget_exhausted"
		out.Retryable = false
	}
	if charge := e.GetChargeReceipt(); charge != nil && charge.GetMeasured() && charge.AmountMicroUsd != nil {
		out.ChargeMicroUSD = charge.GetAmountMicroUsd()
		out.ChargeKnown = true
	}
	if usage := e.GetBudgetUsage(); usage != nil {
		out.TokensUsed = int64(usage.GetTokens())
		out.TokensKnown = usage.GetAccountingComplete()
	}
	if e.GetStatus() == ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		out.ExecutionStarted, out.ExecutionStartedKnown = true, true
	}
	if out.ErrorCode == "provider_admission_maintenance" {
		// The terminal reason is an owner-issued pre-launch receipt; aggregate
		// workflow accounting can remain incomplete for charging without making
		// this provider's known zero token usage unknown.
		out.ExecutionStarted = false
		out.TokensUsed, out.TokensKnown = 0, true
		out.ChargeMicroUSD, out.ChargeKnown = 0, true
	}
	if out.Status != "succeeded" {
		return out, nil
	}
	result, err := client.GetWorkflowExecutionResult(ctx, stagedMessageRequest(ctx, &apipb.GetWorkflowExecutionResultRequest{ExecutionId: e.GetId(), ExplicitlyAuthorized: true}))
	if err != nil {
		return out, stagedmessages.EvidenceReadError{Cause: fmt.Errorf("read message workflow result: %w", err)}
	}
	finished := result.Msg.GetExecution()
	if err := stagedMessageExecutionIdentity(finished, stagedMessageInputWorkflowKey(in)); err != nil {
		return stagedmessages.Execution{}, err
	}
	if finished.GetId() != e.GetId() || finished.GetIdempotencyKey() != e.GetIdempotencyKey() || finished.GetStatus() != ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		return stagedmessages.Execution{}, errors.New("message workflow result identity or terminal state changed")
	}
	var eligible map[int]bool
	if in.FormatVersion >= 3 {
		if p.evidence == nil {
			return out, stagedmessages.EvidenceReadError{Cause: errors.New("durable supplied evidence reader unavailable")}
		}
		eligible, err = p.evidence(ctx, in.JobID, in.Snapshot.RepositoryId, in.Snapshot.Digest, e.Id)
		if err != nil {
			return out, stagedmessages.EvidenceReadError{Cause: err}
		}
	}
	var inventory map[int]bool
	if in.FormatVersion == 4 {
		if p.inventory == nil {
			return out, stagedmessages.EvidenceReadError{Cause: errors.New("durable inventory reader unavailable")}
		}
		inventory, err = p.inventory(ctx, in.JobID, in.Snapshot.RepositoryId, in.Snapshot.Digest, e.Id)
		if err != nil {
			return out, stagedmessages.EvidenceReadError{Cause: err}
		}
	}
	suggestion, err := decodeStagedMessageSynthesisResult(finished, in, eligible, inventory)
	if err != nil {
		// The workflow completed successfully, but GCT rejected its draft. Report
		// the validation failure as its own retryable category instead of leaking
		// the workflow's successful terminal code (for example, "completed").
		out.Status = "failed"
		out.ErrorCode = "invalid_structured_output"
		out.Retryable = true
		out.Error = "Invalid structured message result: " + err.Error()
		return out, nil
	}
	out.Result = suggestion
	return out, nil
}

// Owner messages are authoritative. Some budget terminal records contain only
// typed coordinates, so preserve their meaning when no message was retained.
func stagedMessageTerminalError(e *ampb.WorkflowExecution) string {
	reason := e.GetTerminalReason()
	if message := reason.GetMessage(); message != "" {
		return message
	}
	if e.GetStatus() == ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED || reason.GetCode() == "budget_exhausted" {
		switch reason.GetBudgetName() {
		case "tokens":
			return "Drafting stopped because the workflow token budget was exhausted."
		case "charge":
			return "Drafting stopped because the workflow spending budget was exhausted."
		case "wall_time":
			return "Drafting stopped because the workflow time budget was exhausted."
		default:
			return "Drafting stopped because a workflow budget was exhausted."
		}
	}
	return ""
}

func (p *stagedMessageAgent) AuthorizeEvidence(ctx context.Context, input stagedmessages.DraftInput, key, execution string) error {
	base, err := p.resolve(ctx)
	if err != nil {
		return err
	}
	// This exact input read is intentional: the ordinary Get response is
	// metadata only and cannot prove the immutable subject before GCT saves
	// its handle. Reuse the existing explicit payload read, including while running.
	response, err := p.workflow(base).GetWorkflowExecutionResult(ctx, stagedMessageRequest(ctx, &apipb.GetWorkflowExecutionResultRequest{ExecutionId: execution, ExplicitlyAuthorized: true}))
	if err != nil {
		return err
	}
	owner := response.Msg.GetExecution()
	if err = stagedMessageExecutionIdentity(owner, stagedMessageDeepWorkflowKey); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	raw, err := json.Marshal(owner.GetInput().AsInterface())
	if err != nil {
		return err
	}
	var compact stagedMessageCompactInput
	if err = json.Unmarshal(raw, &compact); err != nil {
		return err
	}
	digest, err := stagedMessageInputDigest(input)
	if err != nil {
		return err
	}
	if owner.Id != execution || owner.IdempotencyKey != key || compact.FormatVersion != input.FormatVersion || (input.FormatVersion != 3 && input.FormatVersion != 4) || compact.DraftInputDigest != digest || compact.DraftContext.EvidenceJobID != input.JobID || compact.DraftContext.SnapshotDigest != input.Snapshot.Digest {
		return connect.NewError(connect.CodePermissionDenied, errors.New("workflow does not own this immutable evidence job"))
	}
	return nil
}

func stagedMessageStatus(status ampb.WorkflowExecutionStatus) string {
	switch status {
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING:
		return "queued"
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "running"
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_WAITING:
		return "awaiting_input"
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLING:
		return "cancelling"
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED:
		return "succeeded"
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	case ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED,
		ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED,
		ampb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED:
		return "failed"
	default:
		return "unknown"
	}
}

type stagedMessageResult struct {
	// Older retained workflow results may still contain this field. Current
	// workflows bind the result to their immutable input instead of echoing it.
	SnapshotDigest string   `json:"snapshot_digest,omitempty"`
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	Trailers       []any    `json:"trailers"`
	CoveredPaths   []string `json:"covered_paths"`
	UncoveredPaths []string `json:"uncovered_paths"`
	Partial        bool     `json:"partial"`
	SourceIDs      []string `json:"source_ids"`
	Unknowns       []string `json:"unknowns"`
}

func decodeStrictStagedMessageJSON(encoded []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("multiple result values")
	}
	return nil
}

func decodeStagedMessageResult(e *ampb.WorkflowExecution) (*advisoryv1.StagedMessageSuggestion, error) {
	input, ok := e.GetInput().AsInterface().(map[string]any)
	if !ok {
		return nil, errors.New("immutable input unavailable")
	}
	snapshot, ok := input["snapshot"].(map[string]any)
	if !ok {
		return nil, errors.New("immutable snapshot unavailable")
	}
	output, ok := e.GetOutput().AsInterface().(map[string]any)
	if !ok {
		return nil, errors.New("output unavailable")
	}
	encoded, err := json.Marshal(output["result"])
	if err != nil {
		return nil, err
	}
	if len(encoded) > 262144 {
		return nil, errors.New("output exceeds bounds")
	}
	var result stagedMessageResult
	if err := decodeStrictStagedMessageJSON(encoded, &result); err != nil {
		return nil, err
	}
	if strings.TrimSpace(result.Title) == "" || len(result.Title) > 200 || strings.ContainsAny(result.Title, "\r\n") || len(result.Body) > 8000 || len(result.Trailers) > 0 || len(result.Unknowns) > 32 || len(result.SourceIDs) > 20 {
		return nil, errors.New("message fields exceed contract or contain inferred trailers")
	}
	paths := map[string]bool{}
	fileEdits := map[string]int32{}
	var totalEdits, matchedEdits int32
	files, _ := snapshot["files"].([]any)
	for _, raw := range files {
		if file, ok := raw.(map[string]any); ok {
			if path, ok := file["path"].(string); ok {
				paths[path] = true
				if reason, _ := file["omission_reason"].(string); reason != "" {
					paths[path] = false
				}
				if binary, _ := file["binary"].(bool); binary {
					paths[path] = false
				}
				edits := int32(1)
				if count, ok := file["edit_count"].(float64); ok && count > 0 {
					edits = int32(count)
				}
				fileEdits[path] = edits
				totalEdits += edits
			}
		}
	}
	covered := map[string]bool{}
	for _, path := range result.CoveredPaths {
		if !paths[path] || covered[path] {
			return nil, errors.New("invalid or duplicate covered path")
		}
		covered[path] = true
		matchedEdits += fileEdits[path]
	}
	uncovered := map[string]bool{}
	for _, path := range result.UncoveredPaths {
		if _, exists := fileEdits[path]; !exists || covered[path] || uncovered[path] {
			return nil, errors.New("invalid or duplicate uncovered path")
		}
		uncovered[path] = true
	}
	for path := range paths {
		if !covered[path] && !uncovered[path] {
			return nil, errors.New("result omits file coverage")
		}
	}
	if (len(uncovered) > 0) != result.Partial {
		return nil, errors.New("uncovered files require partial disclosure")
	}
	sources := map[string]bool{}
	contexts, _ := input["contexts"].([]any)
	for _, raw := range contexts {
		if source, ok := raw.(map[string]any); ok {
			if id, ok := source["id"].(string); ok {
				sources[id] = true
			}
		}
	}
	seenSources := map[string]bool{}
	for _, id := range result.SourceIDs {
		if !sources[id] || seenSources[id] {
			return nil, errors.New("unknown or duplicate context source")
		}
		seenSources[id] = true
	}
	for _, unknown := range result.Unknowns {
		if len(unknown) > 1000 {
			return nil, errors.New("unknown disclosure exceeds bounds")
		}
	}
	return &advisoryv1.StagedMessageSuggestion{Id: e.GetId(), Title: result.Title, Body: result.Body, Coverage: &advisoryv1.StagedMessageCoverage{CoveredPaths: result.CoveredPaths, UncoveredPaths: result.UncoveredPaths, Partial: result.Partial, TotalEdits: totalEdits, MatchedEdits: matchedEdits}, SourceIds: result.SourceIDs, Unknowns: result.Unknowns}, nil
}
