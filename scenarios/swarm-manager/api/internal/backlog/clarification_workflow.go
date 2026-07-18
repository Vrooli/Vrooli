package backlog

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type clarificationWorkflowPending struct {
	ExecutionID      string `json:"execution_id"`
	ThreadID         string `json:"thread_id"`
	Version          string `json:"version"`
	DefinitionDigest string `json:"definition_digest"`
}

type clarificationWorkflowResult struct {
	Outcome string `json:"outcome"`
	Reply   string `json:"reply"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
}

func clarificationWorkflowVersion(item BacklogItem, thread *workshop.ClarificationThread) string {
	threadSnapshot := *thread
	// Run correlation and mutation timestamps are assigned after the immutable
	// snapshot is started; neither changes the operator conversation.
	threadSnapshot.RunID = ""
	threadSnapshot.UpdatedAt = ""
	payload, _ := json.Marshal(struct {
		Item   BacklogItem                   `json:"item"`
		Thread *workshop.ClarificationThread `json:"thread"`
	}{item, &threadSnapshot})
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("sha256:%x", digest[:])
}

func clarificationWorkflowPendingPath(itemDir, threadID, executionID string) string {
	return filepath.Join(itemDir, "workshop", "clarification-"+threadID+"-workflow-"+executionID+".json")
}

func clarificationDecisionForThread(itemDir string, thread *workshop.ClarificationThread) (*workshop.Item, error) {
	rounds, err := workshop.LoadRounds(itemDir)
	if err != nil {
		return nil, err
	}
	for _, round := range rounds {
		if round.RoundNum == thread.RoundNumber {
			for i := range round.Items {
				if round.Items[i].ID == thread.ItemID {
					return &round.Items[i], nil
				}
			}
		}
	}
	return nil, errors.New("clarification decision no longer exists")
}

func (h *Handler) startClarificationWorkflow(ctx context.Context, item BacklogItem, decision *workshop.Item, thread *workshop.ClarificationThread) (agentmanager.WorkflowStart, error) {
	if h.clarificationWorkflow == nil {
		return agentmanager.WorkflowStart{}, agentmanager.ErrNotAvailable
	}
	version := clarificationWorkflowVersion(item, thread)
	decisionJSON, _ := json.Marshal(decision)
	threadJSON, _ := json.Marshal(thread)
	var decisionSnapshot, threadSnapshot map[string]any
	if err := json.Unmarshal(decisionJSON, &decisionSnapshot); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	if err := json.Unmarshal(threadJSON, &threadSnapshot); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": string(item.Kind), "name": item.Name, "version": version}, "decision": decisionSnapshot, "thread": threadSnapshot})
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	start, err := h.clarificationWorkflow.StartWorkflow(ctx, agentmanager.Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/backlog-clarify", Input: input, IdempotencyKey: "backlog-clarify/" + item.Name + "/" + thread.ID + "/" + strings.TrimPrefix(version, "sha256:"), FirstRunNodeID: "clarify"})
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	dir := h.store.ItemDir(item.Kind, item.Name)
	if err := os.MkdirAll(filepath.Join(dir, "workshop"), 0o750); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	pending := clarificationWorkflowPending{ExecutionID: start.ExecutionID, ThreadID: thread.ID, Version: version, DefinitionDigest: start.DefinitionDigest}
	if err := writeJSONRedacted(clarificationWorkflowPendingPath(dir, thread.ID, start.ExecutionID), pending); err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	return start, nil
}

func (h *Handler) ApplyClarificationWorkflow(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "clarification-workflow-apply")
	if !ok {
		return
	}
	threadID, executionID := mux.Vars(r)["threadId"], mux.Vars(r)["executionID"]
	if err := h.applyClarificationWorkflow(r.Context(), kind, name, threadID, executionID); err != nil {
		apierr.MapError(w, "[backlog] clarification-workflow-apply", apierr.Conflict("%s", err))
		return
	}
	thread, _ := workshop.LoadClarificationByID(h.store.ItemDir(kind, name), threadID)
	_ = httputil.JSON(w, map[string]any{"thread": thread, "execution_id": executionID})
}

func (h *Handler) applyClarificationWorkflow(ctx context.Context, kind BacklogKind, name, threadID, executionID string) error {
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return errors.New("backlog item not found")
	}
	dir := h.store.ItemDir(kind, name)
	thread, err := workshop.LoadClarificationByID(dir, threadID)
	if err != nil || thread == nil {
		return errors.New("clarification thread not found")
	}
	pendingData, err := os.ReadFile(clarificationWorkflowPendingPath(dir, threadID, executionID))
	if err != nil {
		return errors.New("workflow is not pending for clarification")
	}
	var pending clarificationWorkflowPending
	if json.Unmarshal(pendingData, &pending) != nil || pending.ThreadID != threadID {
		return errors.New("invalid clarification workflow correlation")
	}
	if h.clarificationWorkflow == nil {
		return agentmanager.ErrNotAvailable
	}
	completion, err := h.clarificationWorkflow.CollectWorkflow(ctx, executionID)
	if err != nil {
		return err
	}
	if completion.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED || completion.DefinitionDigest != pending.DefinitionDigest {
		return errors.New("workflow terminal snapshot is not applicable")
	}
	if clarificationWorkflowVersion(item, thread) != pending.Version {
		return errors.New("clarification changed after workflow start")
	}
	if completion.Input == nil {
		return errors.New("workflow omitted immutable input")
	}
	payload, ok := completion.Input.AsInterface().(map[string]any)
	if !ok {
		return errors.New("workflow input is invalid")
	}
	entity, ok := payload["entity"].(map[string]any)
	if !ok || entity["version"] != pending.Version {
		return errors.New("workflow input does not match clarification")
	}
	if completion.Output == nil {
		return errors.New("workflow omitted result")
	}
	out, ok := completion.Output.AsInterface().(map[string]any)
	if !ok {
		return errors.New("workflow output invalid")
	}
	raw, ok := out["result"]
	if !ok {
		return errors.New("workflow output omitted result")
	}
	data, _ := json.Marshal(raw)
	var result clarificationWorkflowResult
	if json.Unmarshal(data, &result) != nil {
		return errors.New("workflow result invalid")
	}
	if result.Outcome == "clarified" && strings.TrimSpace(result.Reply) != "" {
		thread.Messages = append(thread.Messages, workshop.ClarificationMessage{Role: "assistant", Content: result.Reply, CreatedAt: time.Now().UTC().Format(time.RFC3339)})
		if result.Status == "resolved" {
			thread.Status = "resolved"
		}
		thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := workshop.SaveClarification(dir, thread); err != nil {
			return err
		}
	} else if result.Outcome == "abstained" && strings.TrimSpace(result.Reason) != "" {
		thread.Status = "abstained"
		thread.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := workshop.SaveClarification(dir, thread); err != nil {
			return err
		}
	} else {
		return errors.New("workflow result has invalid outcome")
	}
	return os.Remove(clarificationWorkflowPendingPath(dir, threadID, executionID))
}
