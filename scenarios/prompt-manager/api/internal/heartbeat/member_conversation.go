package heartbeat

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"prompt-manager/internal/store"
)

// MemberConversationRequest starts an operator conversation with a selected
// persona. The server owns context, profile and attribution; this is not a
// heartbeat trigger. RequestID remains stable when the caller retries a send.
type MemberConversationRequest struct {
	TeamID    string `json:"team_id,omitempty"`
	AgentID   string `json:"agent_id"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func (h *Handlers) createMemberConversation(w http.ResponseWriter, r *http.Request, req *MemberConversationRequest, raw CreateRunRequest) {
	fail := func(status int, err error) { http.Error(w, err.Error(), status) }
	requestID, err := uuid.Parse(req.RequestID)
	if err != nil || requestID == uuid.Nil || strings.TrimSpace(req.AgentID) == "" || strings.TrimSpace(req.Message) == "" || len(req.Message) > 16000 {
		fail(http.StatusBadRequest, fmt.Errorf("conversation requires agent_id, a UUID request_id, and a message of 1–16000 bytes"))
		return
	}
	if raw.TaskID != "" || raw.ProfileRef != nil || raw.Tag != nil || raw.RunMode != "" || len(raw.Environment) > 0 || raw.IdempotencyKey != "" {
		fail(http.StatusBadRequest, fmt.Errorf("conversation cannot be combined with task, profile, tag, environment or run options"))
		return
	}
	if h.executor == nil || h.agentStore == nil || h.teamStore == nil {
		fail(http.StatusServiceUnavailable, fmt.Errorf("member context is not configured"))
		return
	}
	ctx := r.Context()
	agent, err := h.agentStore.Get(ctx, req.AgentID)
	if err != nil {
		fail(http.StatusNotFound, fmt.Errorf("agent not found"))
		return
	}
	runtimeMode, override := "", ""
	if req.TeamID != "" {
		team, err := h.teamStore.Get(ctx, req.TeamID)
		if err != nil {
			fail(http.StatusNotFound, fmt.Errorf("team not found"))
			return
		}
		if err := h.requireMember(ctx, req.TeamID, req.AgentID); err != nil {
			fail(http.StatusNotFound, err)
			return
		}
		runtimeMode = team.Runtime.Mode
		config, err := h.teamStore.GetHeartbeatConfig(ctx, req.TeamID, req.AgentID)
		if err != nil {
			fail(http.StatusInternalServerError, err)
			return
		}
		if config != nil {
			override = config.ProfileKey
		}
	}
	profile, err := DefaultProfileKeyForRuntimeMode(runtimeMode)
	if err != nil {
		fail(http.StatusServiceUnavailable, err)
		return
	}
	if override != "" {
		profile = override
	}

	// Bind an idempotent task to this exact member and first message. Keep the
	// binding separate from assembled context, which may change between retries.
	binding, _ := json.Marshal([]string{req.TeamID, req.AgentID, req.Message})
	title := fmt.Sprintf("Member conversation · %x", sha256.Sum256(binding))
	taskID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("prompt-manager/conversation/"+requestID.String())).String()
	tag := "conversation-" + taskID
	task, err := h.agentClient.GetTask(ctx, taskID)
	if err != nil {
		fail(http.StatusBadGateway, err)
		return
	}
	if task == nil {
		context, err := h.executor.BuildContext(ctx, req.TeamID, req.AgentID)
		if err != nil {
			fail(http.StatusInternalServerError, err)
			return
		}
		task = &Task{ID: taskID, Title: title, ScopePath: h.executor.vrooliRoot, ProjectRoot: h.executor.vrooliRoot,
			Description: context + fmt.Sprintf("\n\nYou are %s (agent %s). The operator has started a conversation with you. Respond directly to their message, using your reference context. Do not execute an automatic heartbeat task.\n\nOperator message:\n%s", agent.DisplayName, req.AgentID, req.Message)}
		if _, err := h.agentClient.CreateTask(ctx, task); err != nil {
			// A timeout or racing retry can follow a successful task write.
			existing, readErr := h.agentClient.GetTask(ctx, taskID)
			if readErr != nil || existing == nil {
				fail(http.StatusBadGateway, err)
				return
			}
			task = existing
		}
	}
	if task.Title != title {
		fail(http.StatusConflict, fmt.Errorf("request_id was already used for a different member or message"))
		return
	}
	// Read durable runs as well as using Agent Manager's short-lived idempotency
	// key. A reload/retry tomorrow must not spawn the same conversation again.
	existing, err := h.agentClient.ListRuns(ctx, ListRunsOptions{TaskID: taskID, TagPrefix: tag, Limit: 100})
	if err != nil {
		fail(http.StatusBadGateway, err)
		return
	}
	if existing != nil {
		for _, run := range existing.Runs {
			if run.TaskID == taskID && run.Tag == tag {
				writeConversationRun(w, run)
				return
			}
		}
	}
	runReq := &CreateRunRequest{TaskID: taskID, ProfileRef: &ProfileRef{ProfileKey: profile}, Tag: &tag, IdempotencyKey: "prompt-manager-conversation:" + taskID}
	// Unassigned personas have no team-member authority to inherit.
	if req.TeamID != "" {
		key, value := buildMemberAttributionEnv(req.TeamID, req.AgentID, store.SpawnOriginConversation)
		runReq.Environment = map[string]string{key: value}
	}
	run, err := h.agentClient.CreateRun(ctx, runReq)
	if err != nil {
		fail(http.StatusBadGateway, err)
		return
	}
	writeConversationRun(w, run)
}

func writeConversationRun(w http.ResponseWriter, run *Run) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": run})
}
