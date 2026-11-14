package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/scopecatalog"
	"switchboard/internal/channels"
	"switchboard/internal/threads"
)

type authorizationContextKey struct{}
type createRunIdentityContextKey struct{}
type createRunAmbiguousContextKey struct{}

// WithCreateRunCaller retains offered channels without substituting identity.
// This adapter currently has an existing human route, not a parent-run route.
func WithCreateRunCaller(ctx context.Context, authorization, runIdentity string) context.Context {
	return context.WithValue(WithAuthorization(ctx, authorization), createRunIdentityContextKey{}, runIdentity)
}

// WithCreateRunHeaders retains offered-channel presence and ambiguity so a
// new-start refusal cannot be bypassed by choosing only the first field value.
func WithCreateRunHeaders(ctx context.Context, headers http.Header) context.Context {
	ctx = WithCreateRunCaller(ctx, headers.Get("Authorization"), headers.Get("X-Agent-Identity-Token"))
	ambiguous := len(headers.Values("Authorization")) > 1 || len(headers.Values("X-Agent-Identity-Token")) != 0
	return context.WithValue(ctx, createRunAmbiguousContextKey{}, ambiguous)
}

// WithAuthorization carries the caller's already-issued bearer credential
// across the in-process dispatch boundary. It is intentionally request-scoped
// and is never written to the thread or run store.
func WithAuthorization(ctx context.Context, authorization string) context.Context {
	return context.WithValue(ctx, authorizationContextKey{}, strings.TrimSpace(authorization))
}

func authorization(ctx context.Context) string {
	value, _ := ctx.Value(authorizationContextKey{}).(string)
	return strings.TrimSpace(value)
}

// AgentManagerRunner submits a conversation turn to agent-manager's public
// task/run API. It deliberately owns no agent profile data; agent-manager
// resolves the profile reference and remains the authority for execution.
type AgentManagerRunner struct {
	Identity owneridentity.Validator
	Now      func() time.Time
	BaseURL  string
	Client   *http.Client
	Threads  *threads.Store
	Send     Reply
	Wait     time.Duration
}

func (r AgentManagerRunner) Run(ctx context.Context, agentID string, scopes []string, message string) (string, error) {
	reply, _, err := r.start(ctx, agentID, scopes, message)
	return reply, err
}

// RunConversation resumes the durable agent-manager run for a thread when one
// exists. A missing mapping creates the first run and records its id before
// returning, so process restart does not fork the conversation.
func (r AgentManagerRunner) RunConversation(ctx context.Context, agentID string, scopes []string, channelID, threadKey, message string) (string, error) {
	e := channels.Envelope{ChannelID: channelID, ThreadKey: threadKey}
	afterSequence := int64(0)
	if r.Threads != nil {
		runID, err := r.Threads.RunID(ctx, e)
		if err != nil {
			return "", err
		}
		if runID != "" {
			afterSequence = r.currentSequence(ctx, runID)
			if err := r.continueRun(ctx, runID, message); err != nil {
				return "", err
			}
			r.awaitReply(runID, afterSequence, channelID, threadKey)
			if r.Send != nil {
				return "", nil
			}
			return "agent-manager conversation continued: " + runID, nil
		}
	}
	reply, runID, err := r.start(ctx, agentID, scopes, message)
	if err != nil {
		return "", err
	}
	if r.Threads != nil {
		if err := r.Threads.SetRunID(ctx, e, runID); err != nil {
			return "", err
		}
	}
	r.awaitReply(runID, afterSequence, channelID, threadKey)
	if r.Send != nil {
		return "", nil
	}
	return reply, nil
}

// awaitReply bridges agent-manager's asynchronous run lifecycle back to the
// channel that admitted the turn. The HTTP request remains bounded; a slow
// provider does not hold an inbound channel request open indefinitely.
func (r AgentManagerRunner) awaitReply(runID string, afterSequence int64, channelID, threadKey string) {
	if r.Send == nil {
		return
	}
	wait := r.Wait
	if wait <= 0 {
		wait = 30 * time.Second
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			if reply, ok := r.assistantReply(ctx, runID, afterSequence); ok {
				_ = r.Send(ctx, channels.Outbound{ChannelID: channelID, ThreadKey: threadKey, Text: reply})
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (r AgentManagerRunner) currentSequence(ctx context.Context, runID string) int64 {
	events, err := r.events(ctx, runID, 0)
	if err != nil {
		return 0
	}
	var latest int64
	for _, event := range events {
		if sequence, ok := event["sequence"].(float64); ok && int64(sequence) > latest {
			latest = int64(sequence)
		}
	}
	return latest
}

func (r AgentManagerRunner) assistantReply(ctx context.Context, runID string, afterSequence int64) (string, bool) {
	events, err := r.events(ctx, runID, afterSequence)
	if err != nil {
		return "", false
	}
	for index := len(events) - 1; index >= 0; index-- {
		data, ok := events[index]["message"].(map[string]any)
		if !ok || data["role"] != "assistant" {
			continue
		}
		content, _ := data["content"].(string)
		if strings.TrimSpace(content) != "" {
			return content, true
		}
	}
	return "", false
}

func (r AgentManagerRunner) events(ctx context.Context, runID string, afterSequence int64) ([]map[string]any, error) {
	base := strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}
	query := url.Values{}
	query.Set("after_sequence", strconv.FormatInt(afterSequence, 10))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/runs/"+url.PathEscape(runID)+"/events?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get agent events returned %s", resp.Status)
	}
	var payload struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Events, nil
}

func (r AgentManagerRunner) start(ctx context.Context, agentID string, scopes []string, message string) (string, string, error) {
	// Verify the existing owner proof before the two-step task/run mutation.
	// Nil verifier, lineage text and legacy subject headers supply no identity.
	if ambiguous, _ := ctx.Value(createRunAmbiguousContextKey{}).(bool); ambiguous {
		return "", "", fmt.Errorf("Switchboard CreateRun refuses ambiguous or unsupported offered identity")
	}
	if runProof, _ := ctx.Value(createRunIdentityContextKey{}).(string); runProof != "" {
		return "", "", fmt.Errorf("Switchboard CreateRun has no qualified exact-parent run route")
	}
	parts := strings.Fields(authorization(ctx))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || r.Identity == nil {
		return "", "", fmt.Errorf("Switchboard CreateRun requires a verified human owner")
	}
	owner, verifyErr := r.Identity.Validate(ctx, parts[1])
	now := time.Now()
	if r.Now != nil {
		now = r.Now()
	}
	if verifyErr != nil || strings.TrimSpace(owner.Subject) == "" || !owner.ExpiresAt.After(now) || !scopecatalog.MatchCapability(owner.Scopes, "agent-manager:write") {
		return "", "", fmt.Errorf("Switchboard CreateRun owner proof is invalid, expired or lacks agent-manager:write")
	}

	base := strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	if base == "" {
		return "", "", fmt.Errorf("agent-manager URL is unavailable")
	}
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}
	// Even same-origin redirects must not move the caller's credential to a
	// different operation. This copy changes no continuation/ingest contract.
	creationClient := *client
	creationClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &creationClient
	task, err := r.post(ctx, client, base+"/api/v1/tasks", map[string]any{
		"task": map[string]string{"title": "Switchboard conversation", "description": message, "scope_path": "."},
	})
	if err != nil {
		return "", "", fmt.Errorf("create agent task: %w", err)
	}
	taskID := nestedID(task, "task")
	if taskID == "" {
		return "", "", fmt.Errorf("agent-manager task response has no id")
	}
	run, err := r.post(ctx, client, base+"/api/v1/runs", map[string]any{
		"task_id":     taskID,
		"profile_ref": map[string]string{"profile_key": agentID},
		"prompt":      message,
		"inline_config": map[string]any{
			"allowed_tools":       scopes,
			"clear_allowed_tools": true,
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("create agent run: %w", err)
	}
	runID := nestedID(run, "run")
	if runID == "" {
		return "", "", fmt.Errorf("agent-manager run response has no id")
	}
	return "agent-manager run accepted: " + runID, runID, nil
}

func (r AgentManagerRunner) continueRun(ctx context.Context, runID, message string) error {
	base := strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("agent-manager URL is unavailable")
	}
	client := r.Client
	if client == nil {
		client = http.DefaultClient
	}
	_, err := r.post(ctx, client, base+"/api/v1/runs/"+runID+"/continue", map[string]string{"run_id": runID, "message": message})
	if err != nil {
		return fmt.Errorf("continue agent run: %w", err)
	}
	return nil
}

func (r AgentManagerRunner) post(ctx context.Context, client *http.Client, url string, body any) (map[string]any, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if value := authorization(ctx); value != "" {
		req.Header.Set("Authorization", value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(data))
		if parts := strings.Fields(authorization(ctx)); len(parts) == 2 {
			message = strings.ReplaceAll(message, parts[1], "[redacted]")
		}
		return nil, fmt.Errorf("agent-manager returned %s: %s", resp.Status, message)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func nestedID(value map[string]any, key string) string {
	if nested, ok := value[key].(map[string]any); ok {
		if id, ok := nested["id"].(string); ok {
			return id
		}
	}
	if id, ok := value["id"].(string); ok {
		return id
	}
	return ""
}
