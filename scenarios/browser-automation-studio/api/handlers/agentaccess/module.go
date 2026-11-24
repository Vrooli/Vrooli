// Package agentaccess exposes the BAS-owned browser session broker over a
// typed Connect surface for governed Program Runtime bindings.
package agentaccess

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	access "github.com/vrooli/browser-automation-studio/automation/agentaccess"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	agentaccessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess"
	agentaccessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/agentaccess/agentaccessv1connect"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	identityHeader = "X-Agent-Identity-Token"
	grantHeader    = "X-Vrooli-Browser-Grant-Receipt"
	maxScreenshot  = 4 << 20
)

// Deps deliberately accepts the already-constructed broker. This module
// cannot create a session manager or browser process of its own.
type Deps struct {
	Broker        SessionBroker
	Workflows     access.WorkflowCatalog
	ProjectScopes access.ProjectScopeAuthorizer
}

// SessionBroker is the existing BAS broker contract consumed by this module.
// Keeping the seam narrow lets owner tests prove request mapping without
// launching a driver or creating another browser lifecycle.
type SessionBroker interface {
	Open(context.Context, access.OpenRequest) (access.SessionInfo, error)
	Attach(context.Context, access.AttachRequest) (access.SessionInfo, error)
	Observe(context.Context, string, access.ObserveRequest) (access.Observation, error)
	Act(context.Context, string, access.ActRequest) (access.ActionResult, error)
	Trace(context.Context, string) (access.Trace, error)
	DraftTrace(context.Context, string, access.PromoteRequest) (access.PromotedFlow, error)
	PromoteTrace(context.Context, string, access.PromoteRequest) (access.PromotedFlow, error)
	ReplayPromotedFlow(context.Context, string, *workflowsv1.WorkflowDefinitionV2, map[string]string) (access.ReplayResult, error)
	Close(context.Context, string) error
}

type service struct {
	broker        SessionBroker
	workflows     access.WorkflowCatalog
	projectScopes access.ProjectScopeAuthorizer
}

// Module returns the typed Connect mount. The interceptor copies exactly one
// opaque identity/grant pair into the request context; no RPC body field can
// supply or override caller authority.
func Module(d Deps) connectx.ServiceMount {
	if d.Broker == nil || d.Workflows == nil {
		panic("agentaccess.Module requires the BAS session broker and workflow catalog")
	}
	path, handler := agentaccessconnect.NewBrowserSessionServiceHandler(
		&service{broker: d.Broker, workflows: d.Workflows, projectScopes: d.ProjectScopes},
		connect.WithInterceptors(connect.UnaryInterceptorFunc(withBrowserCredentials)),
	)
	return connectx.ServiceMount{Path: path, Handler: handler}
}

func withBrowserCredentials(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		tokens := req.Header().Values(identityHeader)
		grants := req.Header().Values(grantHeader)
		if len(tokens) != 1 || strings.TrimSpace(tokens[0]) == "" || len(grants) != 1 || strings.TrimSpace(grants[0]) == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("one Agent Manager identity token and browser grant receipt are required"))
		}
		ctx = access.WithBrowserAgentCredentials(ctx, strings.TrimSpace(tokens[0]), strings.TrimSpace(grants[0]))
		return next(ctx, req)
	}
}

func (s *service) OpenSession(ctx context.Context, req *connect.Request[agentaccessv1.OpenSessionRequest]) (*connect.Response[agentaccessv1.OpenSessionResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("open session request is required"))
	}
	// The trusted AM run context selects the persisted permission profile in
	// Broker.Open. An omitted ref lets callers avoid selecting a profile; a
	// supplied ref remains only an assertion and the Broker rejects mismatch.
	opened, err := s.broker.Open(ctx, access.OpenRequest{
		PermissionRef:  strings.TrimSpace(req.Msg.GetPermissionRef()),
		ConsentReceipt: strings.TrimSpace(req.Msg.GetConsentReceipt()),
		TabID:          strings.TrimSpace(req.Msg.GetTabId()), Backend: strings.TrimSpace(req.Msg.GetBackend()),
		StartURL: strings.TrimSpace(req.Msg.GetStartUrl()), ReuseMode: strings.TrimSpace(req.Msg.GetReuseMode()),
		ViewportWidth: int(req.Msg.GetViewportWidth()), ViewportHeight: int(req.Msg.GetViewportHeight()),
		AudioMode: strings.TrimSpace(req.Msg.GetAudioMode()), AudioPlaybackPauseMS: int(req.Msg.GetAudioPlaybackPauseMs()),
	})
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.OpenSessionResponse{
		// Permission profile references are server authority metadata. The caller
		// needs the opaque session handle and capability summary, not the profile
		// selector that authorized it.
		SessionId: opened.SessionID, Mode: opened.Mode,
		Role: opened.Role, Capabilities: append([]string(nil), opened.Capabilities...), PairingCode: opened.PairingCode,
	}
	if !opened.PairingExpires.IsZero() {
		response.PairingExpires = timestamppb.New(opened.PairingExpires)
	}
	if evidence := opened.AudioDeviceEvidence; evidence != nil {
		response.AudioDeviceEnumerated = evidence.Enumerated
		response.AudioDeviceLabel = evidence.SelectedLabel
		response.AudioSampleRate = int32(evidence.SampleRate)
		response.AudioChannelCount = int32(evidence.ChannelCount)
	}
	if identity := opened.IdentityProfile; identity != nil {
		response.IdentityProfile = &agentaccessv1.BrowserIdentityProfile{
			Preset: identity.Preset, ViewportWidth: int32(identity.ViewportWidth), ViewportHeight: int32(identity.ViewportHeight),
			DeviceScaleFactor: identity.DeviceScaleFactor, Locale: identity.Locale, TimezoneId: identity.TimezoneID,
			ColorScheme: identity.ColorScheme,
		}
	}
	if viewport := opened.ActualViewport; viewport != nil {
		response.ActualViewport = &agentaccessv1.ObservedViewport{
			Width: int32(viewport.Width), Height: int32(viewport.Height), Source: string(viewport.Source), Reason: viewport.Reason,
		}
	}
	if evidence := opened.IdentityEvidence; evidence != nil {
		response.IdentityEvidence = mapIdentityEvidence(evidence)
	}
	return connect.NewResponse(response), nil
}

func mapIdentityEvidence(evidence *driver.IdentityEvidence) *agentaccessv1.EffectiveIdentityEvidence {
	if evidence == nil {
		return nil
	}
	return &agentaccessv1.EffectiveIdentityEvidence{
		Engine: evidence.Engine, EngineVersion: evidence.EngineVersion, EngineSource: evidence.EngineSource,
		UserAgent: mapIdentityDimension(evidence.UserAgent), Locale: mapIdentityDimension(evidence.Locale),
		Timezone: mapIdentityDimension(evidence.Timezone), ColorScheme: mapIdentityDimension(evidence.ColorScheme),
		DeviceScaleFactor: mapIdentityDimension(evidence.DeviceScaleFactor), Viewport: mapIdentityDimension(evidence.Viewport),
	}
}

func mapIdentityDimension(dimension driver.IdentityDimensionEvidence) *agentaccessv1.IdentityDimensionEvidence {
	result := &agentaccessv1.IdentityDimensionEvidence{Status: dimension.Status, Source: dimension.Source, Reason: dimension.Reason}
	if dimension.Configured != nil {
		result.Configured, _ = structpb.NewValue(dimension.Configured)
	}
	if dimension.Observed != nil {
		result.Observed, _ = structpb.NewValue(dimension.Observed)
	}
	if dimension.Effective != nil {
		result.Effective, _ = structpb.NewValue(dimension.Effective)
	}
	return result
}

func (s *service) Observe(ctx context.Context, req *connect.Request[agentaccessv1.ObserveRequest]) (*connect.Response[agentaccessv1.ObserveResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	observed, err := s.broker.Observe(ctx, strings.TrimSpace(req.Msg.GetSessionId()), access.ObserveRequest{PageID: strings.TrimSpace(req.Msg.GetPageId())})
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.ObserveResponse{
		SessionId: observed.SessionID, Url: observed.URL, Title: observed.Title, Text: observed.Text,
	}
	if !observed.ObservedAt.IsZero() {
		response.ObservedAt = timestamppb.New(observed.ObservedAt)
	}
	if shot := observed.Screenshot; shot != nil {
		if len(shot.Data) > base64.StdEncoding.EncodedLen(maxScreenshot) {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("screenshot exceeds 4 MiB response limit"))
		}
		data, decodeErr := base64.StdEncoding.DecodeString(shot.Data)
		if decodeErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode broker screenshot: %w", decodeErr))
		}
		if len(data) > maxScreenshot {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("screenshot exceeds 4 MiB response limit"))
		}
		response.ScreenshotData = base64.StdEncoding.EncodeToString(data)
		response.ScreenshotMediaType = shot.MediaType
		response.ScreenshotWidth = int32(shot.Width)
		response.ScreenshotHeight = int32(shot.Height)
		if value, parseErr := parseTimestamp(shot.CapturedAt); parseErr == nil {
			response.ScreenshotCapturedAt = value
		}
	}
	return connect.NewResponse(response), nil
}

func (s *service) AttachSession(ctx context.Context, req *connect.Request[agentaccessv1.AttachSessionRequest]) (*connect.Response[agentaccessv1.AttachSessionResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" || strings.TrimSpace(req.Msg.GetPermissionRef()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id and permission_ref are required"))
	}
	attached, err := s.broker.Attach(ctx, access.AttachRequest{
		SessionID: req.Msg.GetSessionId(), PermissionRef: req.Msg.GetPermissionRef(), TabID: req.Msg.GetTabId(),
		ConsentReceipt: req.Msg.GetConsentReceipt(), ApprovalReceipt: req.Msg.GetApprovalReceipt(),
	})
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.AttachSessionResponse{
		SessionId: attached.SessionID, Mode: attached.Mode,
		Role: attached.Role, Capabilities: append([]string(nil), attached.Capabilities...), PairingCode: attached.PairingCode,
	}
	if !attached.PairingExpires.IsZero() {
		response.PairingExpires = timestamppb.New(attached.PairingExpires)
	}
	return connect.NewResponse(response), nil
}

// RunTask executes a bounded typed action plan on the exact broker-owned
// session. Every instruction re-enters Broker.Act, which re-checks the active
// caller, lease, profile, URL scope, and action policy. It never opens or
// borrows a driver session outside that broker.
func (s *service) RunTask(ctx context.Context, req *connect.Request[agentaccessv1.RunTaskRequest]) (*connect.Response[agentaccessv1.RunTaskResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" || len(req.Msg.GetInstructions()) == 0 || len(req.Msg.GetInstructions()) > 25 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id and 1..25 instructions are required"))
	}
	response := &agentaccessv1.RunTaskResponse{SessionId: req.Msg.GetSessionId()}
	for _, item := range req.Msg.GetInstructions() {
		if item == nil || item.GetAction() == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("every task instruction requires a typed action"))
		}
		result, err := s.broker.Act(ctx, req.Msg.GetSessionId(), access.ActRequest{
			Instruction: contracts.CompiledInstruction{Index: int(item.GetInstructionIndex()), NodeID: item.GetNodeId(), Action: item.GetAction()},
			TargetURL:   item.GetTargetUrl(), ApprovalReceipt: item.GetApprovalReceipt(),
		})
		row := &agentaccessv1.RunTaskActionResult{InstructionIndex: item.GetInstructionIndex()}
		if err != nil {
			row.FailureClass = "action_denied"
			response.Results = append(response.Results, row)
			return connect.NewResponse(response), nil
		}
		row.Success = result.Outcome.Success
		row.StepType = result.Outcome.StepType
		if !row.Success {
			row.FailureClass = "action_failed"
			if result.Outcome.Failure != nil && result.Outcome.Failure.Kind != "" {
				row.FailureClass = string(result.Outcome.Failure.Kind)
			}
		}
		response.Results = append(response.Results, row)
		if !row.Success {
			return connect.NewResponse(response), nil
		}
	}
	response.Completed = true
	return connect.NewResponse(response), nil
}

func (s *service) Act(ctx context.Context, req *connect.Request[agentaccessv1.ActRequest]) (*connect.Response[agentaccessv1.ActResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" || req.Msg.GetAction() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id and typed action are required"))
	}
	result, err := s.broker.Act(ctx, strings.TrimSpace(req.Msg.GetSessionId()), access.ActRequest{
		Instruction: contracts.CompiledInstruction{Index: int(req.Msg.GetInstructionIndex()), NodeID: req.Msg.GetNodeId(), Action: req.Msg.GetAction()},
		TargetURL:   strings.TrimSpace(req.Msg.GetTargetUrl()), ApprovalReceipt: strings.TrimSpace(req.Msg.GetApprovalReceipt()),
	})
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.ActResponse{SessionId: result.SessionID, Success: result.Outcome.Success, StepType: result.Outcome.StepType}
	if failure := result.Outcome.Failure; failure != nil {
		response.FailureClass = string(failure.Kind) + ":" + failure.Code
		response.FailureDetail = failure.Message
	}
	return connect.NewResponse(response), nil
}

func (s *service) ReadTrace(ctx context.Context, req *connect.Request[agentaccessv1.ReadTraceRequest]) (*connect.Response[agentaccessv1.ReadTraceResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	trace, err := s.broker.Trace(ctx, strings.TrimSpace(req.Msg.GetSessionId()))
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.ReadTraceResponse{TraceId: trace.ID, SessionId: trace.SessionID}
	for _, event := range trace.Events {
		row := &agentaccessv1.TraceEvent{Sequence: event.Sequence, Kind: event.Kind, Url: event.URL, Title: event.Title, UnpromotableReason: event.UnpromotableReason}
		if !event.OccurredAt.IsZero() {
			row.OccurredAt = timestamppb.New(event.OccurredAt)
		}
		if event.Instruction != nil {
			row.Instruction = event.Instruction.Action
		}
		if event.Outcome != nil {
			row.ActionSucceeded = event.Outcome.Success
		}
		for _, parameter := range event.Parameters {
			row.Parameters = append(row.Parameters, &agentaccessv1.TraceParameter{Name: parameter.Name, Kind: parameter.Kind})
		}
		response.Events = append(response.Events, row)
	}
	return connect.NewResponse(response), nil
}

func (s *service) PromoteTrace(ctx context.Context, req *connect.Request[agentaccessv1.PromoteTraceRequest]) (*connect.Response[agentaccessv1.PromoteTraceResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" || strings.TrimSpace(req.Msg.GetProjectId()) == "" || strings.TrimSpace(req.Msg.GetName()) == "" || len(req.Msg.GetAssertions()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id, project_id, name, and at least one typed assertion are required"))
	}
	promoted, err := s.broker.PromoteTrace(ctx, strings.TrimSpace(req.Msg.GetSessionId()), access.PromoteRequest{
		Name: strings.TrimSpace(req.Msg.GetName()), Description: strings.TrimSpace(req.Msg.GetDescription()), Assertions: req.Msg.GetAssertions(),
	})
	if err != nil {
		return nil, brokerError(err)
	}
	promoted, err = access.PersistPromotedFlow(ctx, s.workflows, s.projectScopes, req.Msg.GetProjectId(), promoted)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("BAS workflow persistence failed"))
	}
	response := &agentaccessv1.PromoteTraceResponse{
		TraceId: promoted.TraceID, SessionId: promoted.SessionID, Definition: promoted.Definition, ActionCount: int32(promoted.ActionCount),
		WorkflowId: promoted.WorkflowID, WorkflowVersion: promoted.WorkflowVersion,
	}
	for _, parameter := range promoted.RequiredParameters {
		response.RequiredParameters = append(response.RequiredParameters, &agentaccessv1.TraceParameter{Name: parameter.Name, Kind: parameter.Kind})
	}
	return connect.NewResponse(response), nil
}

func (s *service) DraftTrace(ctx context.Context, req *connect.Request[agentaccessv1.DraftTraceRequest]) (*connect.Response[agentaccessv1.DraftTraceResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" || strings.TrimSpace(req.Msg.GetName()) == "" || len(req.Msg.GetAssertions()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id, name, and at least one typed assertion are required"))
	}
	draft, err := s.broker.DraftTrace(ctx, strings.TrimSpace(req.Msg.GetSessionId()), access.PromoteRequest{
		Name: strings.TrimSpace(req.Msg.GetName()), Description: strings.TrimSpace(req.Msg.GetDescription()), Assertions: req.Msg.GetAssertions(),
	})
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.DraftTraceResponse{TraceId: draft.TraceID, SessionId: draft.SessionID, Definition: draft.Definition, ActionCount: int32(draft.ActionCount), DraftDigest: draft.DraftDigest}
	for _, parameter := range draft.RequiredParameters {
		response.RequiredParameters = append(response.RequiredParameters, &agentaccessv1.TraceParameter{Name: parameter.Name, Kind: parameter.Kind})
	}
	return connect.NewResponse(response), nil
}

func (s *service) ReplayWorkflow(ctx context.Context, req *connect.Request[agentaccessv1.ReplayWorkflowRequest]) (*connect.Response[agentaccessv1.ReplayWorkflowResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" || strings.TrimSpace(req.Msg.GetWorkflowId()) == "" || req.Msg.GetWorkflowVersion() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id, workflow_id, and positive workflow_version are required"))
	}
	loaded, err := access.LoadScopedPromotedWorkflow(ctx, s.workflows, s.projectScopes, req.Msg.GetWorkflowId(), req.Msg.GetWorkflowVersion())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("saved workflow is unavailable or project scope was denied"))
	}
	replayed, err := s.broker.ReplayPromotedFlow(ctx, strings.TrimSpace(req.Msg.GetSessionId()), loaded.GetWorkflow().GetFlowDefinition(), req.Msg.GetParameters())
	if err != nil {
		return nil, brokerError(err)
	}
	response := &agentaccessv1.ReplayWorkflowResponse{SessionId: replayed.SessionID, WorkflowId: loaded.GetWorkflow().GetId(), WorkflowVersion: int32(loaded.GetWorkflow().GetVersion()), Completed: replayed.Completed}
	for _, result := range replayed.Results {
		response.Results = append(response.Results, &agentaccessv1.ReplayWorkflowActionResult{NodeId: result.NodeID, Success: result.Success, StepType: result.StepType, FailureClass: result.FailureClass})
	}
	return connect.NewResponse(response), nil
}

func (s *service) CloseSession(ctx context.Context, req *connect.Request[agentaccessv1.CloseSessionRequest]) (*connect.Response[agentaccessv1.CloseSessionResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetSessionId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	if err := s.broker.Close(ctx, strings.TrimSpace(req.Msg.GetSessionId())); err != nil {
		return nil, brokerError(err)
	}
	return connect.NewResponse(&agentaccessv1.CloseSessionResponse{Closed: true}), nil
}

func brokerError(err error) error {
	return connect.NewError(connect.CodeFailedPrecondition, err)
}

func parseTimestamp(raw string) (*timestamppb.Timestamp, error) {
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(value), nil
}

var _ agentaccessconnect.BrowserSessionServiceHandler = (*service)(nil)
