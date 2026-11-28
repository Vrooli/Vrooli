package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

const (
	basTargetMachineParam    = "cl2_target_machine_id"
	basTargetNodeParam       = "cl2_target_node_id"
	basTargetGenerationParam = "cl2_target_generation"
)

// BASAssertionOwner is the owner read boundary for BAS UI lane evidence.
type BASAssertionOwner interface {
	GetExecution(context.Context, string) (*basexecution.Execution, error)
	GetExecutionTimeline(context.Context, string) (*bastimeline.ExecutionTimeline, error)
}

// DiscoveredBASAssertionOwner resolves BAS on each read and injects the
// configured owner token. It never infers a target identity from the caller.
type DiscoveredBASAssertionOwner struct {
	HTTPClient    *http.Client
	ResolveURL    func(context.Context) (string, error)
	TokenProvider func(context.Context) (string, error)
}

func NewDiscoveredBASAssertionOwner(timeout time.Duration) *DiscoveredBASAssertionOwner {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &DiscoveredBASAssertionOwner{
		HTTPClient: &http.Client{Timeout: timeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
		},
		TokenProvider: basAssertionOwnerToken,
	}
}

func basAssertionOwnerToken(context.Context) (string, error) {
	for _, name := range []string{"VROOLI_BAS_API_TOKEN", "VROOLI_API_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("no Browser Automation Studio owner token is configured")
}

func (o *DiscoveredBASAssertionOwner) client(ctx context.Context) (basconnect.ExecutionsServiceClient, error) {
	if !hasBASOwnerProviders(o) {
		return nil, fmt.Errorf("BAS owner discovery and token provider are required")
	}
	baseURL, err := o.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve BAS owner: %w", err)
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("BAS owner URL is invalid")
	}
	if !validBASOwnerBaseURL(parsed) {
		return nil, fmt.Errorf("BAS owner URL is invalid")
	}
	token, err := o.TokenProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve BAS owner token: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("resolve BAS owner token: empty token")
	}
	baseClient := o.HTTPClient
	if baseClient == nil {
		baseClient = http.DefaultClient
	}
	securedClient := *baseClient
	transport := baseClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	securedClient.Transport = ownerBearerRoundTripper{base: transport, token: token}
	return basconnect.NewExecutionsServiceClient(&securedClient, strings.TrimRight(baseURL, "/")), nil
}

func hasBASOwnerProviders(owner *DiscoveredBASAssertionOwner) bool {
	return owner != nil && owner.ResolveURL != nil && owner.TokenProvider != nil
}

func validBASOwnerBaseURL(baseURL *url.URL) bool {
	return baseURL != nil && baseURL.Scheme != "" && baseURL.Host != ""
}

func (o *DiscoveredBASAssertionOwner) GetExecution(ctx context.Context, id string) (*basexecution.Execution, error) {
	client, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetExecution(ctx, connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: id}))
	if err != nil {
		return nil, fmt.Errorf("read BAS execution %q: %w", id, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetExecution() == nil {
		return nil, fmt.Errorf("BAS returned no execution for %q", id)
	}
	return response.Msg.GetExecution(), nil
}

func (o *DiscoveredBASAssertionOwner) GetExecutionTimeline(ctx context.Context, id string) (*bastimeline.ExecutionTimeline, error) {
	client, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetExecutionTimeline(ctx, connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: id}))
	if err != nil {
		return nil, fmt.Errorf("read BAS execution timeline %q: %w", id, err)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("BAS returned no timeline for %q", id)
	}
	return response.Msg, nil
}

// BASAssertionOwnerProof retains the BAS execution, exact target binding and
// successful assertion event that underlie a BAS UI cell.
type BASAssertionOwnerProof struct {
	Owner            string `json:"owner"`
	ExecutionID      string `json:"execution_id"`
	WorkflowID       string `json:"workflow_id"`
	NodeID           string `json:"node_id"`
	EventID          string `json:"event_id"`
	TargetMachine    string `json:"target_machine_id"`
	TargetNode       string `json:"target_node_id"`
	TargetGeneration uint64 `json:"target_generation"`
	Status           string `json:"status"`
	AssertionMode    string `json:"assertion_mode"`
	Selector         string `json:"selector"`
	ArtifactRef      string `json:"artifact_ref"`
	SHA256           string `json:"sha256"`
}

type basAssertionProofFields struct {
	Owner            string `json:"owner"`
	ExecutionID      string `json:"execution_id"`
	WorkflowID       string `json:"workflow_id"`
	NodeID           string `json:"node_id"`
	EventID          string `json:"event_id"`
	TargetMachine    string `json:"target_machine_id"`
	TargetNode       string `json:"target_node_id"`
	TargetGeneration uint64 `json:"target_generation"`
	Status           string `json:"status"`
	AssertionMode    string `json:"assertion_mode"`
	Selector         string `json:"selector"`
}

// VerifyBASAssertionOwnerReceipt reads BAS's durable execution and timeline,
// requiring the exact matrix cell's named assertion node and target identity.
func VerifyBASAssertionOwnerReceipt(ctx context.Context, owner BASAssertionOwner, result DriverResult, assertion TestGenieAssertionProof) (BASAssertionOwnerProof, error) {
	if err := validateBASOwnerRequest(owner, result, assertion); err != nil {
		return BASAssertionOwnerProof{}, err
	}
	executionID, nodeID, err := parseBASOwnerReceipt(assertion.Envelope.OwnerReceiptRef)
	if err != nil {
		return BASAssertionOwnerProof{}, err
	}
	execution, err := owner.GetExecution(ctx, executionID)
	if err != nil {
		return BASAssertionOwnerProof{}, err
	}
	if err := validateBASExecution(execution, executionID); err != nil {
		return BASAssertionOwnerProof{}, err
	}
	if err := validateBASExecutionTarget(execution, result.Target); err != nil {
		return BASAssertionOwnerProof{}, err
	}
	timeline, err := owner.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return BASAssertionOwnerProof{}, err
	}
	if err := validateBASTimeline(timeline, executionID, execution.GetWorkflowId()); err != nil {
		return BASAssertionOwnerProof{}, err
	}
	assertionEvent, err := successfulBASAssertionEvent(timeline, executionID, nodeID)
	if err != nil {
		return BASAssertionOwnerProof{}, err
	}
	assertionResult := assertionEvent.GetContext().GetAssertion()
	fields := basAssertionProofFields{Owner: "Browser Automation Studio", ExecutionID: executionID, WorkflowID: execution.GetWorkflowId(), NodeID: nodeID, EventID: assertionEvent.GetId(), TargetMachine: result.Target.MachineID, TargetNode: result.Target.NodeID, TargetGeneration: result.Target.Generation, Status: execution.GetStatus().String(), AssertionMode: assertionResult.GetMode().String(), Selector: assertionResult.GetSelector()}
	return buildBASAssertionOwnerProof(fields)
}

func validateBASOwnerRequest(owner BASAssertionOwner, result DriverResult, assertion TestGenieAssertionProof) error {
	if result.Cell.Lane != LaneBASUI {
		return fmt.Errorf("BAS assertion owner only supports bas_ui lane cells")
	}
	if owner == nil {
		return fmt.Errorf("BAS assertion owner is required for bas_ui lane")
	}
	if assertion.Envelope.AssertionOwner != "Browser Automation Studio" {
		return fmt.Errorf("BAS assertion artifact has the wrong owner")
	}
	return nil
}

func parseBASOwnerReceipt(receiptRef string) (executionID, nodeID string, err error) {
	u, err := url.Parse(strings.TrimSpace(receiptRef))
	if err != nil || !validBASOwnerReceiptURL(u) {
		return "", "", fmt.Errorf("BAS owner receipt must use bas-execution://<execution-uuid>/<node-uuid>")
	}
	parsedExecutionID, err := uuid.Parse(u.Host)
	if err != nil {
		return "", "", fmt.Errorf("BAS owner receipt execution id must be a UUID")
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 1 {
		return "", "", fmt.Errorf("BAS owner receipt must identify one assertion node")
	}
	parsedNodeID, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("BAS owner receipt node id is invalid")
	}
	if _, err := uuid.Parse(parsedNodeID); err != nil {
		return "", "", fmt.Errorf("BAS assertion node id must be a UUID")
	}
	return parsedExecutionID.String(), parsedNodeID, nil
}

func validBASOwnerReceiptURL(u *url.URL) bool {
	return u != nil && u.Scheme == "bas-execution" && strings.TrimSpace(u.Host) != "" && u.User == nil && !strings.ContainsAny(u.Host, ":@") && u.RawQuery == "" && u.Fragment == ""
}

func validateBASExecution(execution *basexecution.Execution, executionID string) error {
	if execution == nil || execution.GetExecutionId() != executionID || execution.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED || execution.GetCompletedAt() == nil || execution.GetResult() == nil || !execution.GetResult().GetSuccess() {
		return fmt.Errorf("BAS execution %s is not successfully completed", executionID)
	}
	return nil
}

func validateBASExecutionTarget(execution *basexecution.Execution, target Target) error {
	params := execution.GetParameters()
	if params == nil || params.GetEnv()[basTargetMachineParam].GetStringValue() != target.MachineID || params.GetEnv()[basTargetNodeParam].GetStringValue() != target.NodeID || params.GetEnv()[basTargetGenerationParam].GetIntValue() != int64(target.Generation) {
		return fmt.Errorf("BAS execution target environment does not match the admitted target identity")
	}
	return nil
}

func validateBASTimeline(timeline *bastimeline.ExecutionTimeline, executionID, workflowID string) error {
	if timeline == nil || timeline.GetExecutionId() != executionID || timeline.GetWorkflowId() != workflowID || timeline.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED || timeline.GetCompletedAt() == nil {
		return fmt.Errorf("BAS execution timeline identity or completion state does not match")
	}
	return nil
}

func successfulBASAssertionEvent(timeline *bastimeline.ExecutionTimeline, executionID, nodeID string) (*bastimeline.TimelineEntry, error) {
	var assertionEvent *bastimeline.TimelineEntry
	for _, entry := range timeline.GetEntries() {
		if entry == nil || entry.GetNodeId() != nodeID {
			continue
		}
		if assertionEvent != nil {
			return nil, fmt.Errorf("BAS assertion node %s has duplicate timeline events", nodeID)
		}
		assertionEvent = entry
	}
	if !isSuccessfulBASAssertionEvent(assertionEvent, executionID) {
		return nil, fmt.Errorf("BAS execution has no successful assertion event for node %s", nodeID)
	}
	return assertionEvent, nil
}

func isSuccessfulBASAssertionEvent(entry *bastimeline.TimelineEntry, executionID string) bool {
	if entry == nil || entry.GetContext() == nil || entry.GetContext().GetExecutionId() != executionID || !entry.GetContext().GetSuccess() {
		return false
	}
	assertion := entry.GetContext().GetAssertion()
	return assertion != nil && assertion.GetSuccess()
}

func buildBASAssertionOwnerProof(fields basAssertionProofFields) (BASAssertionOwnerProof, error) {
	raw, err := json.Marshal(fields)
	if err != nil {
		return BASAssertionOwnerProof{}, fmt.Errorf("encode BAS assertion owner proof: %w", err)
	}
	digest := sha256.Sum256(raw)
	sha := hex.EncodeToString(digest[:])
	artifactRef := fmt.Sprintf("bas-execution://%s/%s?sha256=%s", fields.ExecutionID, url.PathEscape(fields.NodeID), sha)
	return BASAssertionOwnerProof{Owner: fields.Owner, ExecutionID: fields.ExecutionID, WorkflowID: fields.WorkflowID, NodeID: fields.NodeID, EventID: fields.EventID, TargetMachine: fields.TargetMachine, TargetNode: fields.TargetNode, TargetGeneration: fields.TargetGeneration, Status: fields.Status, AssertionMode: fields.AssertionMode, Selector: fields.Selector, ArtifactRef: artifactRef, SHA256: sha}, nil
}

func validateBASAssertionOwnerProof(proof *BASAssertionOwnerProof, target Target, assertion *TestGenieAssertionProof) error {
	if proof == nil || assertion == nil {
		return fmt.Errorf("BAS UI passed receipt requires BAS and Test Genie owner proofs")
	}
	if !validBASAssertionProof(proof, target) {
		return fmt.Errorf("BAS assertion proof has incomplete or mismatched identity")
	}
	if !matchesBASAssertionOwnerReceipt(proof, assertion) {
		return fmt.Errorf("BAS assertion proof does not match the lane owner receipt reference")
	}
	fields := basAssertionProofFields{Owner: proof.Owner, ExecutionID: proof.ExecutionID, WorkflowID: proof.WorkflowID, NodeID: proof.NodeID, EventID: proof.EventID, TargetMachine: proof.TargetMachine, TargetNode: proof.TargetNode, TargetGeneration: proof.TargetGeneration, Status: proof.Status, AssertionMode: proof.AssertionMode, Selector: proof.Selector}
	raw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	sha := hex.EncodeToString(digest[:])
	if !matchesBASAssertionProofDigest(proof, sha) {
		return fmt.Errorf("BAS assertion proof digest or ref does not match the owner fields")
	}
	return nil
}

func validBASAssertionProof(proof *BASAssertionOwnerProof, target Target) bool {
	return hasBASAssertionProofIdentity(proof) && matchesBASAssertionProofTarget(proof, target) && hasValidBASAssertionDetails(proof)
}

func hasValidBASAssertionDetails(proof *BASAssertionOwnerProof) bool {
	return proof.Status == basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED.String() && proof.AssertionMode != "" && proof.Selector != "" && proof.SHA256 != ""
}

func hasBASAssertionProofIdentity(proof *BASAssertionOwnerProof) bool {
	return proof.Owner == "Browser Automation Studio" && proof.ExecutionID != "" && proof.WorkflowID != "" && proof.NodeID != "" && proof.EventID != ""
}

func matchesBASAssertionProofTarget(proof *BASAssertionOwnerProof, target Target) bool {
	return proof.TargetMachine == target.MachineID && proof.TargetNode == target.NodeID && proof.TargetGeneration == target.Generation
}

func matchesBASAssertionOwnerReceipt(proof *BASAssertionOwnerProof, assertion *TestGenieAssertionProof) bool {
	return assertion.Envelope.AssertionOwner == proof.Owner && assertion.Envelope.OwnerReceiptRef == "bas-execution://"+proof.ExecutionID+"/"+url.PathEscape(proof.NodeID)
}

func matchesBASAssertionProofDigest(proof *BASAssertionOwnerProof, sha string) bool {
	return proof.SHA256 == sha && proof.ArtifactRef == fmt.Sprintf("bas-execution://%s/%s?sha256=%s", proof.ExecutionID, url.PathEscape(proof.NodeID), sha)
}
