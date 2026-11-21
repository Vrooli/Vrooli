package decisionpage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// DefaultProfileKey is the declared Agent Manager profile writer runs use:
// Codex on the code.cheap role (GPT-6 Luna), no sandbox, so the run can
// write its job folder and photograph the page with a headless browser.
const DefaultProfileKey = "notification-hub/decision-page-writer"

// AgentManager launches writer runs as Agent Manager task+run pairs with the
// writer's declared profile. Run creation is authority: every request carries
// the writer's service credential from Authorization, and without one the
// launcher refuses rather than relying on Agent Manager admitting a caller
// that presents none. A refusal is retried after the writer's cool-down, so a
// credential issued later takes effect without a new request.
type AgentManager struct {
	// BaseURL resolves Agent Manager's API; nil uses lifecycle discovery.
	BaseURL func(context.Context) (string, error)
	HTTP    *http.Client
	// ProfileKey is the declared profile; empty uses DefaultProfileKey.
	ProfileKey string
	// ScopePath is the task's scope inside the repository; empty uses the
	// notification-hub skill folder the writer reads.
	ScopePath string
	// CreatedBy names the creator on the task; empty uses "notification-hub".
	CreatedBy string
	// Authorization returns the Authorization header value of the writer's
	// declared service identity, holding a standing run-creation grant for
	// the writer profile. Nil refuses every launch (ErrRefused).
	Authorization func(context.Context) (string, error)
}

// ErrNoCredential is why a launcher with no service credential refuses.
var ErrNoCredential = fmt.Errorf("%w: the decision-page writer has no service credential for Agent Manager run creation", ErrRefused)

func (a *AgentManager) base(ctx context.Context) (string, error) {
	if a.BaseURL != nil {
		return a.BaseURL(ctx)
	}
	return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
}

func (a *AgentManager) client() *http.Client {
	if a.HTTP != nil {
		return a.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// Launch creates the task and its run.
func (a *AgentManager) Launch(ctx context.Context, spec LaunchSpec) (RunRef, error) {
	if a.Authorization == nil {
		return RunRef{}, ErrNoCredential
	}
	var task struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	}
	if err := a.call(ctx, http.MethodPost, "/api/v1/tasks", map[string]any{"task": map[string]any{
		"title": spec.Title, "description": spec.Description,
		"scopePath": firstNonEmpty(a.ScopePath, "scenarios/notification-hub/skills/notification-hub"),
		"createdBy": firstNonEmpty(a.CreatedBy, "notification-hub"),
	}}, &task); err != nil {
		return RunRef{}, fmt.Errorf("create task: %w", err)
	}
	var run runEnvelope
	if err := a.call(ctx, http.MethodPost, "/api/v1/runs", map[string]any{
		"taskId": task.Task.ID, "force": true, "prompt": spec.Prompt, "tag": spec.Tag,
		"idempotencyKey": spec.IdempotencyKey,
		"profileRef":     map[string]any{"profileKey": firstNonEmpty(a.ProfileKey, DefaultProfileKey)},
	}, &run); err != nil {
		return RunRef{}, fmt.Errorf("create run: %w", err)
	}
	ref := run.Run.ref()
	ref.TaskID = firstNonEmpty(ref.TaskID, task.Task.ID)
	return ref, nil
}

// Run reads one run.
func (a *AgentManager) Run(ctx context.Context, id string) (RunRef, error) {
	var resp runEnvelope
	if err := a.call(ctx, http.MethodGet, "/api/v1/runs/"+id, nil, &resp); err != nil {
		return RunRef{}, err
	}
	return resp.Run.ref(), nil
}

// Stop stops one run; best effort, since an unattended caller may not be
// allowed to.
func (a *AgentManager) Stop(ctx context.Context, id string) error {
	return a.call(ctx, http.MethodPost, "/api/v1/runs/"+id+"/stop", map[string]any{"runId": id}, nil)
}

// runEnvelope is Agent Manager's run JSON, read in either field spelling.
type runEnvelope struct {
	Run amRun `json:"run"`
}

type amRun struct {
	ID             string `json:"id"`
	TaskID         string `json:"task_id"`
	TaskIDCamel    string `json:"taskId"`
	Status         string `json:"status"`
	ActualModel    string `json:"actual_model"`
	ActualCamel    string `json:"actualModel"`
	RequestedModel string `json:"requested_model"`
	ErrorMsg       string `json:"error_msg"`
	ErrorCamel     string `json:"errorMsg"`
	Resolved       struct {
		Model     string `json:"model"`
		Admission struct {
			// EffectiveModel is the model the runner was admitted with: the
			// model flag Agent Manager passed, which the runner cannot change.
			EffectiveModel string `json:"effective_model"`
		} `json:"admission"`
	} `json:"resolved_config"`
}

func (r amRun) ref() RunRef {
	ref := RunRef{
		ID: r.ID, TaskID: firstNonEmpty(r.TaskID, r.TaskIDCamel),
		Model: firstNonEmpty(r.ActualModel, r.ActualCamel, r.Resolved.Admission.EffectiveModel, r.Resolved.Model, r.RequestedModel),
		Error: strings.TrimSpace(firstNonEmpty(r.ErrorMsg, r.ErrorCamel)),
	}
	ref.Status = strings.ToLower(strings.TrimPrefix(r.Status, "RUN_STATUS_"))
	switch ref.Status {
	case "complete", "needs_review":
		ref.Ended, ref.Succeeded = true, true
	case "failed", "cancelled":
		ref.Ended = true
	}
	return ref
}

// call sends one JSON request. 401 and 403 are refusals; a transport error
// or a 5xx is unavailability; anything else non-2xx is a plain error.
func (a *AgentManager) call(ctx context.Context, method, path string, in any, out any) error {
	base, err := a.base(ctx)
	if err != nil {
		return fmt.Errorf("%w: agent-manager: %v", ErrUnavailable, err)
	}
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.Authorization != nil {
		auth, err := a.Authorization(ctx)
		if err != nil {
			return fmt.Errorf("%w: the writer's service credential: %v", ErrRefused, err)
		}
		req.Header.Set("Authorization", auth)
	}
	resp, err := a.client().Do(req)
	if err != nil {
		return fmt.Errorf("%w: agent-manager: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	switch {
	// 202 approval_pending: Agent Manager has asked the operator to approve a
	// standing grant for the writer's service principal. 429: that ask is
	// already outstanding. Both are refusals the writer retries after its
	// cool-down, by which time the operator may have answered.
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode == http.StatusAccepted && strings.Contains(string(data), "approval"):
		return fmt.Errorf("%w: agent-manager %s: %s", ErrRefused, resp.Status, errorText(data))
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w: agent-manager %s: %s", ErrUnavailable, resp.Status, errorText(data))
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return fmt.Errorf("agent-manager %s: %s", resp.Status, errorText(data))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func errorText(data []byte) string {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &e) == nil && firstNonEmpty(e.Message, e.Error) != "" {
		return firstNonEmpty(e.Message, e.Error)
	}
	return clip(string(data), 300)
}
