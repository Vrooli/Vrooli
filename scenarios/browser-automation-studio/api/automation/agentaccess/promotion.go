package agentaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
)

type Parameter struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type PromoteRequest struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description,omitempty"`
	ProjectID   string                        `json:"project_id"`
	Assertions  []*actionsv1.ActionDefinition `json:"assertions"`
}

type PromotedFlow struct {
	DraftDigest        string                            `json:"draft_digest"`
	Evidence           []TraceCompileEvidence            `json:"evidence"`
	TraceID            string                            `json:"trace_id"`
	SessionID          string                            `json:"session_id"`
	WorkflowID         string                            `json:"workflow_id,omitempty"`
	WorkflowVersion    int32                             `json:"workflow_version,omitempty"`
	Definition         *workflowsv1.WorkflowDefinitionV2 `json:"definition"`
	RequiredParameters []Parameter                       `json:"required_parameters"`
	ActionCount        int                               `json:"action_count"`
}

// WorkflowCatalog is BAS's durable workflow owner. Agent Access never writes
// workflow files or indexes directly.
type WorkflowCatalog interface {
	CreateWorkflow(context.Context, *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error)
	GetWorkflowAPI(context.Context, *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error)
}

// ProjectScopeAuthorizer is an owner supplied boundary for project reads and
// writes. WorkflowService verifies project existence, but does not establish
// that this browser caller may read or write it. Implementations must verify an
// Agent Manager issued scope receipt from trusted invocation context; callers
// cannot supply identity or a receipt in promotion/replay request bodies.
type ProjectScopeAuthorizer interface {
	AuthorizeProjectRead(context.Context, string) error
	AuthorizeProjectWrite(context.Context, string) error
}

// LoadScopedPromotedWorkflow loads a persisted candidate only after resolving
// its project through BAS's workflow owner and checking the exact project read
// scope. The caller cannot choose a project ID that differs from the saved
// workflow's owner. This keeps replay from becoming a workflow-ID oracle or an
// alternate path around the project authorization required for promotion.
func LoadScopedPromotedWorkflow(ctx context.Context, catalog WorkflowCatalog, scope ProjectScopeAuthorizer, workflowID string, version int32) (*basapi.GetWorkflowResponse, error) {
	if catalog == nil || scope == nil || strings.TrimSpace(workflowID) == "" || version <= 0 {
		return nil, errors.New("workflow catalog, project-scope verifier, workflow ID, and positive version are required")
	}
	wantedVersion := version
	loaded, err := catalog.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: strings.TrimSpace(workflowID), Version: &wantedVersion})
	if err != nil || loaded == nil || loaded.GetWorkflow() == nil || loaded.GetWorkflow().GetFlowDefinition() == nil {
		return nil, errors.New("saved workflow is unavailable")
	}
	projectID := strings.TrimSpace(loaded.GetWorkflow().GetProjectId())
	if projectID == "" {
		return nil, errors.New("saved workflow has no owning project")
	}
	if err := scope.AuthorizeProjectRead(ctx, projectID); err != nil {
		return nil, fmt.Errorf("project-scope receipt rejected workflow read: %w", err)
	}
	return loaded, nil
}

func PersistPromotedFlow(ctx context.Context, catalog WorkflowCatalog, scope ProjectScopeAuthorizer, projectID string, promoted PromotedFlow) (PromotedFlow, error) {
	if catalog == nil || strings.TrimSpace(projectID) == "" || promoted.Definition == nil || promoted.Definition.GetMetadata() == nil {
		return PromotedFlow{}, errors.New("workflow catalog, project ID, and promoted definition are required")
	}
	if scope == nil {
		return PromotedFlow{}, errors.New("project-scope receipt verifier is not configured; refusing cross-user workflow write")
	}
	if err := scope.AuthorizeProjectWrite(ctx, strings.TrimSpace(projectID)); err != nil {
		return PromotedFlow{}, fmt.Errorf("project-scope receipt rejected workflow write: %w", err)
	}
	created, err := catalog.CreateWorkflow(ctx, &basapi.CreateWorkflowRequest{
		ProjectId: strings.TrimSpace(projectID), Name: promoted.Definition.GetMetadata().GetName(),
		FolderPath: "agent-promoted", FlowDefinition: promoted.Definition,
	})
	if err != nil {
		return PromotedFlow{}, fmt.Errorf("persist promoted workflow through BAS workflow owner: %w", err)
	}
	if created == nil || created.GetWorkflow() == nil || strings.TrimSpace(created.GetWorkflow().GetId()) == "" {
		return PromotedFlow{}, errors.New("BAS workflow owner returned no persisted workflow")
	}
	promoted.WorkflowID = created.GetWorkflow().GetId()
	promoted.WorkflowVersion = int32(created.GetWorkflow().GetVersion())
	return promoted, nil
}

// TraceCompileEvidence ties each generated assertion to the accepted action and
// the next Broker observation. No permission/grant secret is copied into V2.
type TraceCompileEvidence struct {
	ActionSequence      uint64 `json:"action_sequence"`
	ObservationSequence uint64 `json:"observation_sequence"`
	SessionID           string `json:"session_id"`
	PermissionClass     string `json:"permission_class"`
	EffectClass         string `json:"effect_class"`
	ObservedOrigin      string `json:"observed_origin"`
	ObservedAt          int64  `json:"observed_at_unix_nano"`
}

// DraftTrace authorizes compilation against the existing active Broker session
// and returns a deterministic V2 draft without writing the project catalog.
func (b *Broker) DraftTrace(ctx context.Context, sessionID string, req PromoteRequest) (PromotedFlow, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return PromotedFlow{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return PromotedFlow{}, err
	}
	defer e.opMu.Unlock()
	currentURL := e.tabURL
	if e.session != nil {
		if err := e.session.AuthorizeAction(ctx, "trace_promote", "", ""); err != nil {
			return PromotedFlow{}, err
		}
	} else {
		if e.attachedTab != nil {
			currentURL, err = e.attachedTab.CurrentURL(ctx)
			if err != nil {
				return PromotedFlow{}, err
			}
		}
		if err := b.authorize(ctx, e, actionRequest("trace_promote", currentURL)); err != nil {
			return PromotedFlow{}, err
		}
	}
	if e.session == nil && currentURL != "" && !e.profile.AllowsURL(currentURL) {
		return PromotedFlow{}, sessionpermission.ErrDenied
	}
	trace := b.traceSnapshot(e)
	if trace.SessionID != e.sessionID || trace.PermissionRef != e.permissionRef {
		return PromotedFlow{}, ErrSessionNotOwned
	}
	return compileTraceDraft(trace, sessionID, req)
}

// PromoteTrace remains the existing in-memory compiler contract. Persistence
// continues to belong exclusively to PersistPromotedFlow and its project gate.
func (b *Broker) PromoteTrace(ctx context.Context, sessionID string, req PromoteRequest) (PromotedFlow, error) {
	return b.DraftTrace(ctx, sessionID, req)
}

var stableTraceSelector = regexp.MustCompile(`^(#[\p{L}_][\p{L}\p{N}_-]{0,63}|\[data-testid="[\p{L}\p{N}_.:-]{1,80}"\]|\[aria-label="[\p{L}\p{N} _.-]{1,80}"\])$`)

// These attributes have stable, non-secret semantics for replay assertions.
// In particular, href, value, and arbitrary data attributes can carry user
// state and are deliberately excluded.
var stableTraceAssertionAttributes = map[string]struct{}{
	"aria-label":  {},
	"data-testid": {},
	"role":        {},
	"type":        {},
}

func compileTraceDraft(trace Trace, sessionID string, req PromoteRequest) (PromotedFlow, error) {
	if strings.TrimSpace(trace.ID) == "" || strings.TrimSpace(sessionID) == "" || trace.SessionID != sessionID || strings.TrimSpace(trace.PermissionRef) == "" {
		return PromotedFlow{}, errors.New("trace identity, permission binding, and matching session are required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return PromotedFlow{}, errors.New("flow name is required")
	}
	if len(trace.Events) == 0 {
		return PromotedFlow{}, errors.New("trace has no accepted events")
	}
	var previousEventAt time.Time
	for index, event := range trace.Events {
		if event.Sequence != uint64(index+1) || event.SessionID != sessionID || event.OccurredAt.IsZero() {
			return PromotedFlow{}, fmt.Errorf("trace event %d has missing, forged, or out-of-sequence provenance", index+1)
		}
		if !previousEventAt.IsZero() && !event.OccurredAt.After(previousEventAt) {
			return PromotedFlow{}, fmt.Errorf("trace event %d timestamp does not follow the prior event", index+1)
		}
		previousEventAt = event.OccurredAt
		if event.Kind != "action" && event.Kind != "observation" {
			return PromotedFlow{}, fmt.Errorf("trace event %d has unsupported effect kind %q", event.Sequence, event.Kind)
		}
		if event.Kind == "observation" && (event.PermissionClass != "observe" || event.EffectClass != "none" || !isSafeObservedOrigin(event.URL)) {
			return PromotedFlow{}, fmt.Errorf("trace observation %d is incomplete or outside the accepted observation contract", event.Sequence)
		}
	}

	actionIndexes := make([]int, 0)
	for i, event := range trace.Events {
		if event.Kind == "action" {
			actionIndexes = append(actionIndexes, i)
		}
	}
	if len(actionIndexes) == 0 {
		return PromotedFlow{}, errors.New("trace has no accepted actions to compile")
	}
	if len(req.Assertions) != len(actionIndexes) {
		return PromotedFlow{}, errors.New("one typed post-action assertion is required for each accepted action")
	}

	nodes := make([]*workflowsv1.WorkflowNodeV2, 0, len(actionIndexes)*2)
	edges := make([]*workflowsv1.WorkflowEdgeV2, 0, len(actionIndexes)*2-1)
	parameters := make([]Parameter, 0)
	evidence := make([]TraceCompileEvidence, 0, len(actionIndexes))
	for actionIndex, eventIndex := range actionIndexes {
		event := trace.Events[eventIndex]
		if event.UnpromotableReason != "" {
			return PromotedFlow{}, fmt.Errorf("trace action %d cannot be promoted: %s", event.Sequence, event.UnpromotableReason)
		}
		if event.Instruction == nil || event.Instruction.Action == nil || event.Outcome == nil || !event.Outcome.Success {
			return PromotedFlow{}, fmt.Errorf("trace action %d is incomplete or failed", event.Sequence)
		}
		if event.PermissionClass != "act" && event.PermissionClass != "unrestricted" {
			return PromotedFlow{}, fmt.Errorf("trace action %d lacks accepted act permission evidence", event.Sequence)
		}
		if event.SessionID != trace.SessionID {
			return PromotedFlow{}, fmt.Errorf("trace action %d belongs to another session", event.Sequence)
		}
		instruction, actionParameters, err := promotableTraceInstruction(*event.Instruction, event.Sequence)
		if err != nil {
			return PromotedFlow{}, fmt.Errorf("trace action %d cannot be promoted: %w", event.Sequence, err)
		}
		if event.EffectClass != traceEffectClass(instruction.Action.Type) {
			return PromotedFlow{}, fmt.Errorf("trace action %d has inconsistent effect evidence", event.Sequence)
		}

		observationIndex := -1
		for i := eventIndex + 1; i < len(trace.Events); i++ {
			if trace.Events[i].Kind == "action" {
				break
			}
			if trace.Events[i].Kind == "observation" {
				observationIndex = i
			}
		}
		if observationIndex < 0 {
			return PromotedFlow{}, fmt.Errorf("trace action %d has no complete post-action observation", event.Sequence)
		}
		observation := trace.Events[observationIndex]
		expectedOrigin := event.URL
		if instruction.Action.Type == actionsv1.ActionType_ACTION_TYPE_NAVIGATE {
			expectedOrigin = event.TargetOrigin
		}
		if !isSafeObservedOrigin(expectedOrigin) || observation.URL != expectedOrigin {
			return PromotedFlow{}, fmt.Errorf("trace action %d observation is incomplete or shows a redirect outside its expected origin", event.Sequence)
		}
		assertion := req.Assertions[actionIndex]
		if err := validatePostActionAssertion(assertion); err != nil {
			return PromotedFlow{}, fmt.Errorf("assertion %d is not a stable read-only observation: %w", actionIndex+1, err)
		}
		redactedAssertion, assertionParameters, err := redactAssertion(assertion, event.Sequence)
		if err != nil {
			return PromotedFlow{}, fmt.Errorf("assertion %d cannot be safely parameterized: %w", actionIndex+1, err)
		}

		actionNodeID := deterministicTraceNodeID(trace.ID, event.Sequence, "action")
		assertNodeID := deterministicTraceNodeID(trace.ID, event.Sequence, "assert")
		nodes = append(nodes,
			&workflowsv1.WorkflowNodeV2{Id: actionNodeID, Action: proto.Clone(instruction.Action).(*actionsv1.ActionDefinition)},
			&workflowsv1.WorkflowNodeV2{Id: assertNodeID, Action: redactedAssertion},
		)
		parameters = append(parameters, actionParameters...)
		parameters = append(parameters, assertionParameters...)
		evidence = append(evidence, TraceCompileEvidence{
			ActionSequence: event.Sequence, ObservationSequence: observation.Sequence, SessionID: sessionID,
			PermissionClass: event.PermissionClass, EffectClass: event.EffectClass,
			ObservedOrigin: observation.URL, ObservedAt: observation.OccurredAt.UnixNano(),
		})
	}
	for i := 1; i < len(nodes); i++ {
		edges = append(edges, &workflowsv1.WorkflowEdgeV2{
			Id:     deterministicTraceEdgeID(trace.ID, nodes[i-1].Id, nodes[i].Id),
			Source: nodes[i-1].Id, Target: nodes[i].Id,
		})
	}
	flow := &workflowsv1.WorkflowDefinitionV2{
		Metadata: &workflowsv1.WorkflowMetadataV2{
			Name: proto.String(strings.TrimSpace(req.Name)), Description: proto.String(req.Description),
			Labels:        map[string]string{"source": "agent-browser-trace", "trace_id": trace.ID, "trace_permission": "broker-accepted"},
			ExecutionMode: workflowsv1.ExecutionMode_EXECUTION_MODE_MUTATING,
		},
		Settings: &workflowsv1.WorkflowSettingsV2{}, Nodes: nodes, Edges: edges,
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(flow)
	if err != nil {
		return PromotedFlow{}, fmt.Errorf("canonicalize V2 trace draft: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return PromotedFlow{
		TraceID: trace.ID, SessionID: sessionID, Definition: flow,
		RequiredParameters: uniqueParameters(parameters), ActionCount: len(actionIndexes),
		DraftDigest: hex.EncodeToString(digest[:]), Evidence: evidence,
	}, nil
}

func promotableTraceInstruction(instruction contracts.CompiledInstruction, sequence uint64) (contracts.CompiledInstruction, []Parameter, error) {
	if instruction.Action == nil {
		return instruction, nil, errors.New("typed action is missing")
	}
	typeID := instruction.Action.Type
	if typeID != actionsv1.ActionType_ACTION_TYPE_NAVIGATE && typeID != actionsv1.ActionType_ACTION_TYPE_CLICK && typeID != actionsv1.ActionType_ACTION_TYPE_INPUT {
		return instruction, nil, fmt.Errorf("action type %s is outside the deterministic trace subset", typeID.String())
	}
	selector := traceActionSelector(instruction.Action)
	if typeID != actionsv1.ActionType_ACTION_TYPE_NAVIGATE && !stableTraceSelector.MatchString(selector) {
		return instruction, nil, errors.New("selector is not a stable trace allowlist form")
	}
	redacted, parameters, err := redactInstruction(instruction, int(sequence))
	if err != nil {
		return instruction, nil, err
	}
	return redacted, parameters, nil
}

func traceActionSelector(action *actionsv1.ActionDefinition) string {
	if click := action.GetClick(); click != nil {
		return strings.TrimSpace(click.Selector)
	}
	if input := action.GetInput(); input != nil {
		return strings.TrimSpace(input.Selector)
	}
	return ""
}

func traceEffectClass(kind actionsv1.ActionType) string {
	switch kind {
	case actionsv1.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigation"
	case actionsv1.ActionType_ACTION_TYPE_CLICK, actionsv1.ActionType_ACTION_TYPE_INPUT:
		return "ui"
	default:
		return "unsupported"
	}
}

func validatePostActionAssertion(action *actionsv1.ActionDefinition) error {
	if action == nil || action.Type != actionsv1.ActionType_ACTION_TYPE_ASSERT || action.GetAssert() == nil {
		return errors.New("expected typed ACTION_TYPE_ASSERT")
	}
	params := action.GetAssert()
	if !stableTraceSelector.MatchString(strings.TrimSpace(params.Selector)) {
		return errors.New("assertion selector is not stable")
	}
	switch params.Mode {
	case basev1.AssertionMode_ASSERTION_MODE_VISIBLE:
		return nil
	case basev1.AssertionMode_ASSERTION_MODE_TEXT_EQUALS, basev1.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS:
		if params.Expected == nil {
			return errors.New("text assertion requires an expected value")
		}
		return nil
	case basev1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS, basev1.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS:
		if params.Expected == nil {
			return errors.New("attribute assertion requires an expected value")
		}
		if _, ok := params.Expected.GetKind().(*commonv1.JsonValue_StringValue); !ok {
			return errors.New("attribute assertion expected value must be a string")
		}
		if _, allowed := stableTraceAssertionAttributes[strings.ToLower(strings.TrimSpace(params.GetAttributeName()))]; !allowed {
			return errors.New("attribute assertion name is outside the stable trace allowlist")
		}
		return nil
	default:
		return errors.New("assertion mode is not read-only and deterministic")
	}
}

func isSafeObservedOrigin(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" && parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return parsed.Path == "" || parsed.Path == "/"
}

func deterministicTraceNodeID(traceID string, sequence uint64, kind string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("bas-trace-v2:%s:%d:%s", traceID, sequence, kind))).String()
}

func deterministicTraceEdgeID(traceID, source, target string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("bas-trace-v2:%s:edge:%s:%s", traceID, source, target))).String()
}

func actionRequest(kind, currentURL string) sessionpermission.ActionRequest {
	return sessionpermission.ActionRequest{Kind: kind, CurrentURL: currentURL}
}

func uniqueParameters(values []Parameter) []Parameter {
	seen := make(map[string]bool, len(values))
	result := make([]Parameter, 0, len(values))
	for _, value := range values {
		if value.Name == "" || seen[value.Name] {
			continue
		}
		seen[value.Name] = true
		result = append(result, value)
	}
	return result
}

func redactInstruction(instruction contracts.CompiledInstruction, sequence int) (contracts.CompiledInstruction, []Parameter, error) {
	if instruction.Action == nil {
		return instruction, nil, fmt.Errorf("instruction action is missing")
	}
	copy, ok := proto.Clone(instruction.Action).(*actionsv1.ActionDefinition)
	if !ok {
		return instruction, nil, errors.New("could not clone typed action")
	}
	parameters := make([]Parameter, 0, 1)
	switch copy.Type {
	case actionsv1.ActionType_ACTION_TYPE_NAVIGATE:
		if nav := copy.GetNavigate(); nav != nil && nav.Url != "" {
			name := fmt.Sprintf("navigation_%d", sequence)
			nav.Url = "{{@params/" + name + "}}"
			parameters = append(parameters, Parameter{Name: name, Kind: "url"})
		}
	case actionsv1.ActionType_ACTION_TYPE_INPUT:
		if input := copy.GetInput(); input != nil && input.Value != "" {
			name := fmt.Sprintf("input_%d", sequence)
			input.Value = "{{@params/" + name + "}}"
			parameters = append(parameters, Parameter{Name: name, Kind: func() string {
				if input.GetIsSensitive() {
					return "sensitive"
				}
				return "text"
			}()})
		}
	case actionsv1.ActionType_ACTION_TYPE_CLICK,
		actionsv1.ActionType_ACTION_TYPE_WAIT,
		actionsv1.ActionType_ACTION_TYPE_ASSERT,
		actionsv1.ActionType_ACTION_TYPE_SCROLL,
		actionsv1.ActionType_ACTION_TYPE_SCREENSHOT,
		actionsv1.ActionType_ACTION_TYPE_EXTRACT,
		actionsv1.ActionType_ACTION_TYPE_HOVER,
		actionsv1.ActionType_ACTION_TYPE_FOCUS,
		actionsv1.ActionType_ACTION_TYPE_BLUR,
		actionsv1.ActionType_ACTION_TYPE_KEYBOARD,
		actionsv1.ActionType_ACTION_TYPE_SHORTCUT:
		// These typed page-level actions contain selectors or key names but no
		// arbitrary credentials, protocol payloads, or host paths.
	default:
		return instruction, nil, fmt.Errorf("action type %s is not eligible for automatic trace promotion", copy.Type.String())
	}
	// Freeform execution metadata has no replay semantics and can contain
	// caller data or authority receipts. Keep only the typed Action in a draft.
	instruction.Metadata = nil
	instruction.Action = copy
	return instruction, parameters, nil
}

// redactAssertion clones the caller's typed assertion and parameterizes literal
// text so a draft records the expected check without retaining observed text.
func redactAssertion(assertion *actionsv1.ActionDefinition, sequence uint64) (*actionsv1.ActionDefinition, []Parameter, error) {
	if assertion == nil || assertion.Type != actionsv1.ActionType_ACTION_TYPE_ASSERT || assertion.GetAssert() == nil {
		return nil, nil, errors.New("typed assertion is required")
	}
	copy := proto.Clone(assertion).(*actionsv1.ActionDefinition)
	params := copy.GetAssert()
	// Failure messages are freeform caller data and are not part of the
	// deterministic assertion contract.
	params.FailureMessage = nil
	if params.Expected == nil {
		return copy, nil, nil
	}
	if _, ok := params.Expected.GetKind().(*commonv1.JsonValue_StringValue); !ok {
		return nil, nil, errors.New("text assertion expected value must be a string")
	}
	name := fmt.Sprintf("assertion_text_%d", sequence)
	params.Expected = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "{{@params/" + name + "}}"}}
	return copy, []Parameter{{Name: name, Kind: "assertion-text"}}, nil
}
