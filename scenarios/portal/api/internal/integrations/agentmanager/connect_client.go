package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// AuthorizationApprovalPendingError is definitive: Agent Manager did not
// start the requested operation. The caller must wait for approval before
// retrying.
type AuthorizationApprovalPendingError struct {
	Subject string
	Detail  string
	cause   error
}

func (e *AuthorizationApprovalPendingError) Error() string {
	if e == nil || strings.TrimSpace(e.Detail) == "" {
		return "agent-manager authorization approval pending; operation was not started"
	}
	return fmt.Sprintf("agent-manager authorization approval pending; operation was not started: %s", e.Detail)
}

func (e *AuthorizationApprovalPendingError) Unwrap() error { return e.cause }

type createCallerInterceptor struct{}

func (createCallerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if req.Spec().Procedure == apiconnect.AgentManagerServiceCreateRunProcedure {
			if err := RequireCreateCaller(ctx); err != nil {
				return nil, err
			}
			req.Header().Set("Authorization", createCallerAuthorization(ctx))
		}
		resp, err := next(ctx, req)
		if err != nil && req.Spec().Procedure == apiconnect.AgentManagerServiceCreateRunProcedure {
			return resp, mapApprovalPending(ctx, err)
		}
		if err != nil && req.Spec().Procedure == apiconnect.AgentManagerServiceCreateTaskProcedure {
			return resp, mapApprovalPending(ctx, err)
		}
		return resp, err
	}
}

// ConnectClient uses Agent Manager's generated Connect contract for every
// proto-owned operation. Multipart attachment upload remains on its dedicated
// REST route because the owner has no corresponding RPC.
type ConnectClient struct {
	baseURL string
	http    *http.Client
	rpc     apiconnect.AgentManagerServiceClient
}

func NewConnectClient(baseURL string, client *http.Client) *ConnectClient {
	if client == nil {
		client = &http.Client{}
	}
	transport := *client
	transport.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	baseURL = strings.TrimRight(baseURL, "/")
	return &ConnectClient{
		baseURL: baseURL,
		http:    &transport,
		rpc: apiconnect.NewAgentManagerServiceClient(&transport, baseURL,
			connect.WithInterceptors(createCallerInterceptor{}),
		),
	}
}

func (c *ConnectClient) Start(ctx context.Context, input StartInput) (Session, error) {
	if err := RequireCreateCaller(ctx); err != nil {
		return Session{}, err
	}
	tag, err := admissionTag(input.AdmissionID)
	if err != nil {
		return Session{}, err
	}
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return Session{}, fmt.Errorf("agent prompt is required")
	}
	projectRoot := strings.TrimSpace(input.ProjectRoot)
	if projectRoot == "" {
		projectRoot = defaultProjectRoot()
	}
	attachments := make([]*domain.ContextAttachment, 0, len(input.AttachmentIDs))
	for _, id := range input.AttachmentIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return Session{}, fmt.Errorf("agent attachment IDs must be non-empty")
		}
		attachments = append(attachments, &domain.ContextAttachment{Type: "image", AttachmentId: id})
	}
	parent := strings.TrimSpace(input.ParentRunID)
	var parentID *string
	if parent != "" {
		if !canonicalUUID(parent) {
			return Session{}, fmt.Errorf("canonical parent run UUID required")
		}
		parentID = &parent
	}
	if existing, findErr := c.FindAdmission(ctx, input.AdmissionID); findErr == nil {
		return existing, nil
	} else {
		var rpcErr *connect.Error
		if !errors.As(findErr, &rpcErr) || rpcErr.Code() != connect.CodeNotFound {
			return Session{}, findErr
		}
	}
	if err := c.reconcileProfile(ctx); err != nil {
		return Session{}, err
	}
	taskResp, err := c.rpc.CreateTask(ctx, connect.NewRequest(&api.CreateTaskRequest{Task: &domain.Task{
		Title: "Portal agent chat", Description: prompt, ScopePath: projectRoot,
		ProjectRoot: projectRoot, CreatedBy: "portal", ContextAttachments: attachments,
	}}))
	if err != nil {
		return Session{}, fmt.Errorf("create agent task: %w", redactCreateCallerError(ctx, mapAgentManagerError(ctx, err)))
	}
	taskID := taskResp.Msg.GetTask().GetId()
	if !canonicalUUID(taskID) {
		return Session{}, fmt.Errorf("agent-manager returned invalid task identity")
	}
	mode := domain.RunMode_RUN_MODE_SANDBOXED
	promptValue := prompt
	tagValue := tag
	idempotencyKey := tag
	runResp, err := c.rpc.CreateRun(ctx, connect.NewRequest(&api.CreateRunRequest{
		TaskId: taskID, Tag: &tagValue, RunMode: &mode, IdempotencyKey: &idempotencyKey,
		ProfileRef: &api.ProfileRef{ProfileKey: "portal/agent-chat"},
		Prompt:     &promptValue, ParentRunId: parentID,
	}))
	if err != nil {
		return Session{TaskID: taskID}, fmt.Errorf("create agent run: %w", redactCreateCallerError(ctx, mapAgentManagerError(ctx, err)))
	}
	run := runResp.Msg.GetRun()
	if run == nil || !canonicalUUID(run.GetId()) || run.GetTag() != tag || run.GetTaskId() != taskID {
		return Session{TaskID: taskID}, fmt.Errorf("agent-manager returned mismatched run identity")
	}
	return Session{TaskID: taskID, RunID: run.GetId()}, nil
}

func (c *ConnectClient) FindAdmission(ctx context.Context, id string) (Session, error) {
	tag, err := admissionTag(id)
	if err != nil {
		return Session{}, err
	}
	resp, err := c.rpc.GetRunByTag(ctx, connect.NewRequest(&api.GetRunByTagRequest{Tag: tag}))
	if err != nil {
		return Session{}, fmt.Errorf("find agent admission: %w", mapAgentManagerError(ctx, err))
	}
	run := resp.Msg.GetRun()
	if run == nil || run.GetTag() != tag || !canonicalUUID(run.GetId()) || !canonicalUUID(run.GetTaskId()) {
		return Session{}, fmt.Errorf("agent admission has no exact run identity")
	}
	return Session{RunID: run.GetId(), TaskID: run.GetTaskId()}, nil
}

func (c *ConnectClient) Stop(ctx context.Context, runID string) (RunState, error) {
	if !canonicalUUID(runID) {
		return RunState{}, fmt.Errorf("canonical agent run UUID required")
	}
	resp, err := c.rpc.StopRun(ctx, connect.NewRequest(&api.StopRunRequest{RunId: runID}))
	err = mapAgentManagerError(ctx, err)
	if err == nil {
		state, parseErr := stateFromRun(resp.Msg.GetRun(), runID)
		if parseErr == nil && state.Terminal {
			return state, nil
		}
		if parseErr != nil {
			err = parseErr
		} else {
			err = ErrStopUnconfirmed
		}
	}
	// A lost reply may follow an applied stop. Reconcile with an authoritative
	// read and never repeat the mutation automatically.
	state, readErr := c.Run(ctx, runID)
	if readErr == nil && state.Terminal {
		return state, nil
	}
	readErr = mapAgentManagerError(ctx, readErr)
	return state, errors.Join(ErrStopUnconfirmed, err, readErr)
}

// Run reads authoritative owner state without repeating a possibly applied Stop.
func (c *ConnectClient) Run(ctx context.Context, runID string) (RunState, error) {
	if !canonicalUUID(runID) {
		return RunState{}, fmt.Errorf("canonical agent run UUID required")
	}
	resp, err := c.rpc.GetRun(ctx, connect.NewRequest(&api.GetRunRequest{RunId: runID}))
	if err != nil {
		return RunState{}, fmt.Errorf("read agent run: %w", mapAgentManagerError(ctx, err))
	}
	return stateFromRun(resp.Msg.GetRun(), runID)
}

func (c *ConnectClient) reconcileProfile(ctx context.Context) error {
	resp, err := c.rpc.ReconcileScenarioProfiles(ctx, connect.NewRequest(&api.ReconcileScenarioProfilesRequest{Scenario: "portal"}))
	if err != nil {
		return fmt.Errorf("reconcile agent profile: %w", mapAgentManagerError(ctx, err))
	}
	for _, result := range resp.Msg.GetResults() {
		if result.GetProfileKey() == "portal/agent-chat" && result.GetProfileId() != "" {
			return nil
		}
	}
	return fmt.Errorf("reconcile agent profile returned no portal/agent-chat profile")
}

func (c *ConnectClient) UploadAttachment(ctx context.Context, content []byte, filename, mime string) (string, error) {
	if len(content) == 0 || len(content) > 32*1024*1024 || strings.TrimSpace(filename) == "" || strings.TrimSpace(mime) == "" {
		return "", fmt.Errorf("invalid agent attachment")
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", err
	}
	if _, err := part.Write(content); err != nil {
		return "", err
	}
	if err := form.Close(); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/attachments/upload", &body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("upload agent attachment: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upload agent attachment: status %d", response.StatusCode)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result); err != nil || strings.TrimSpace(result.ID) == "" {
		return "", fmt.Errorf("upload agent attachment: invalid response")
	}
	return result.ID, nil
}

func canonicalUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed != uuid.Nil && parsed.String() == value
}

func stateFromRun(run *domain.Run, expectedID string) (RunState, error) {
	if run == nil || run.GetId() != expectedID {
		return RunState{}, fmt.Errorf("agent-manager returned a different or missing run identity")
	}
	status := normalizeRunStatus(run.GetStatus().String())
	if status == "" || status == "unspecified" || status == "unknown" {
		return RunState{}, fmt.Errorf("agent-manager returned no run status")
	}
	return RunState{RunID: expectedID, Status: status, Terminal: terminalStatus(status)}, nil
}

func mapApprovalPending(ctx context.Context, err error) error {
	var alreadyMapped *AuthorizationApprovalPendingError
	if errors.As(err, &alreadyMapped) {
		return err
	}
	var rpcErr *connect.Error
	if !errors.As(err, &rpcErr) || rpcErr.Code() != connect.CodeFailedPrecondition {
		return err
	}
	for _, detail := range rpcErr.Details() {
		message, decodeErr := detail.Value()
		if decodeErr != nil {
			continue
		}
		pending, ok := message.(*api.AuthorizationApprovalPending)
		if ok {
			return &AuthorizationApprovalPendingError{
				Subject: redactCreateCaller(ctx, pending.GetSubject()),
				Detail:  redactCreateCaller(ctx, pending.GetDetail()),
				cause:   err,
			}
		}
	}
	return err
}

func mapAgentManagerError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if mapped := mapApprovalPending(ctx, err); mapped != err {
		return mapped
	}
	var rpcErr *connect.Error
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		(errors.As(err, &rpcErr) && (rpcErr.Code() == connect.CodeUnavailable || rpcErr.Code() == connect.CodeCanceled || rpcErr.Code() == connect.CodeDeadlineExceeded)) {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return err
}
