package backlog

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/workshop"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type workshopWorkflowResult struct {
	Outcome           string                     `json:"outcome"`
	Note              string                     `json:"note,omitempty"`
	Reason            string                     `json:"reason,omitempty"`
	DecisionQuestions []workshopDecisionProposal `json:"decisionQuestions,omitempty"`
	Readiness         map[string]int             `json:"readiness,omitempty"`
}

type workshopDecisionProposal struct {
	ID      string                   `json:"id"`
	Topic   string                   `json:"topic"`
	Context string                   `json:"context"`
	Options []workshopOptionProposal `json:"options"`
}

type workshopOptionProposal struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Rationale   string `json:"rationale"`
	Recommended bool   `json:"recommended"`
}

type workshopWorkflowProvenance struct {
	Kind             string `json:"kind"`
	ExecutionID      string `json:"execution_id"`
	DefinitionDigest string `json:"definition_digest"`
	RunID            string `json:"run_id"`
	ProfileIdentity  string `json:"profile_identity"`
	EntityVersion    string `json:"entity_version"`
	Outcome          string `json:"outcome"`
	Artifact         string `json:"artifact,omitempty"`
	RecordedAt       string `json:"recorded_at"`
}

type workshopWorkflowApplyResponse struct {
	ExecutionID string `json:"execution_id"`
	Applied     bool   `json:"applied"`
	Idempotent  bool   `json:"idempotent"`
	Outcome     string `json:"outcome"`
	Artifact    string `json:"artifact,omitempty"`
}

type workshopWorkflowPending struct {
	ExecutionID      string `json:"execution_id"`
	Kind             string `json:"kind"`
	Name             string `json:"name"`
	EntityVersion    string `json:"entity_version"`
	DefinitionDigest string `json:"definition_digest"`
	StartedAt        string `json:"started_at"`
}

func workshopSnapshotVersion(item BacklogItem, roundCount int) string {
	payload, _ := json.Marshal(struct {
		Item       BacklogItem `json:"item"`
		RoundCount int         `json:"round_count"`
	}{Item: item, RoundCount: roundCount})
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("sha256:%x", digest[:])
}

func workshopOperatorNote(req interface {
	GetPrompt() string
	GetContextPaths() []string
	GetContextTargetIds() []string
	GetContextRequirementIds() []string
},
) string {
	payload := map[string]any{}
	if value := strings.TrimSpace(req.GetPrompt()); value != "" {
		payload["prompt"] = value
	}
	if value := req.GetContextPaths(); len(value) > 0 {
		payload["context_paths"] = value
	}
	if value := req.GetContextTargetIds(); len(value) > 0 {
		payload["context_target_ids"] = value
	}
	if value := req.GetContextRequirementIds(); len(value) > 0 {
		payload["context_requirement_ids"] = value
	}
	if len(payload) == 0 {
		return ""
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

func (h *Handler) startWorkshopRoundWorkflow(ctx context.Context, item BacklogItem, operatorNote string) (agentmanager.WorkflowStart, error) {
	if h.workshopWorkflow == nil {
		return agentmanager.WorkflowStart{}, agentmanager.ErrNotAvailable
	}
	_, roundCount, err := workshop.LoadLatestRound(h.store.ItemDir(item.Kind, item.Name))
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	version := workshopSnapshotVersion(item, roundCount)
	digest := sha256.Sum256([]byte(operatorNote))
	key := fmt.Sprintf("backlog-workshop/%s/%s/%s/%x", item.Kind, item.Name, strings.TrimPrefix(version, "sha256:")[:16], digest[:8])
	started, err := h.workshopWorkflow.StartWorkshopRound(ctx, agentmanager.BacklogWorkshopSnapshot{
		Kind: string(item.Kind), Name: item.Name, Version: version, Title: item.Title,
		Description: item.Description, Status: string(item.Status), Priority: item.Priority,
		Tags: append([]string(nil), item.Tags...), OperatorNote: operatorNote,
	}, key)
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	workshopDir := filepath.Join(h.store.ItemDir(item.Kind, item.Name), "workshop")
	if err := os.MkdirAll(workshopDir, 0o750); err != nil {
		return agentmanager.WorkflowStart{}, fmt.Errorf("persist workshop workflow correlation: %w", err)
	}
	pending := workshopWorkflowPending{
		ExecutionID: started.ExecutionID, Kind: string(item.Kind), Name: item.Name,
		EntityVersion: version, DefinitionDigest: started.DefinitionDigest,
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := writeJSONRedacted(workshopWorkflowPendingPath(workshopDir, started.ExecutionID), pending); err != nil {
		return agentmanager.WorkflowStart{}, fmt.Errorf("persist workshop workflow correlation: %w", err)
	}
	return started, nil
}

func workshopWorkflowPendingPath(workshopDir, executionID string) string {
	return filepath.Join(workshopDir, "workflow-pending-"+executionID+".json")
}

// ApplyWorkshopWorkflowResult is the backlog-owned command-result handler. It
// validates identity and snapshot version before materializing a round and
// uses an execution-specific provenance claim as its replay guard.
func (h *Handler) ApplyWorkshopWorkflowResult(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-workflow-apply")
	if !ok {
		return
	}
	executionID := strings.TrimSpace(r.PathValue("executionID"))
	if executionID == "" {
		executionID = strings.TrimSpace(mux.Vars(r)["executionID"])
	}
	if _, err := uuid.Parse(executionID); err != nil {
		apierr.MapError(w, "[backlog] workshop-workflow-apply", apierr.BadRequest("invalid workflow execution id"))
		return
	}
	result, err := h.applyWorkshopWorkflowResult(r.Context(), kind, name, executionID)
	if err != nil {
		mapWorkshopApplyError(w, err)
		return
	}
	_ = httputil.JSON(w, result)
}

type workshopApplyError struct {
	status int
	err    error
}

func (e *workshopApplyError) Error() string { return e.err.Error() }
func (e *workshopApplyError) Unwrap() error { return e.err }

func workshopApplyFailure(status int, format string, args ...any) error {
	return &workshopApplyError{status: status, err: fmt.Errorf(format, args...)}
}

func mapWorkshopApplyError(w http.ResponseWriter, err error) {
	var applyErr *workshopApplyError
	if !errors.As(err, &applyErr) {
		apierr.MapError(w, "[backlog] workshop-workflow-apply", apierr.Internal("%s", err.Error()))
		return
	}
	switch applyErr.status {
	case http.StatusBadRequest:
		apierr.MapError(w, "[backlog] workshop-workflow-apply", apierr.BadRequest("%s", applyErr.Error()))
	case http.StatusNotFound:
		apierr.MapError(w, "[backlog] workshop-workflow-apply", apierr.NotFound("%s", applyErr.Error()))
	case http.StatusConflict:
		apierr.MapError(w, "[backlog] workshop-workflow-apply", apierr.Conflict("%s", applyErr.Error()))
	default:
		apierr.MapError(w, "[backlog] workshop-workflow-apply", apierr.Internal("%s", applyErr.Error()))
	}
}

func (h *Handler) applyWorkshopWorkflowResult(ctx context.Context, kind BacklogKind, name, executionID string) (workshopWorkflowApplyResponse, error) {
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusNotFound, "backlog item not found")
	}
	itemDir := h.store.ItemDir(kind, name)
	workshopDir := filepath.Join(itemDir, "workshop")
	provenancePath := filepath.Join(workshopDir, "workflow-provenance-"+executionID+".json")
	var existingProvenance *workshopWorkflowProvenance
	if existing, readErr := readWorkshopWorkflowProvenance(provenancePath); readErr == nil {
		if existing.Artifact == "" {
			return workshopWorkflowApplyResponse{ExecutionID: executionID, Applied: false, Idempotent: true, Outcome: existing.Outcome}, nil
		}
		if _, statErr := os.Stat(filepath.Join(itemDir, existing.Artifact)); statErr == nil {
			return workshopWorkflowApplyResponse{ExecutionID: executionID, Applied: false, Idempotent: true, Outcome: existing.Outcome, Artifact: existing.Artifact}, nil
		}
		existingProvenance = &existing
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusInternalServerError, "failed to read workflow provenance")
	}
	completion, err := h.workshopWorkflow.CollectWorkshopRound(ctx, executionID)
	if err != nil {
		return workshopWorkflowApplyResponse{}, &workshopApplyError{status: http.StatusConflict, err: fmt.Errorf("workflow result is not ready: %w", err)}
	}
	if completion.EntityKind != string(kind) || completion.EntityName != name {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusConflict, "workflow result belongs to a different backlog item")
	}
	_, roundCount, err := workshop.LoadLatestRound(itemDir)
	if err != nil {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusInternalServerError, "failed to load workshop rounds")
	}
	if existingProvenance == nil && completion.EntityVersion != workshopSnapshotVersion(item, roundCount) {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusConflict, "backlog item or workshop version changed after workflow start")
	}
	var result workshopWorkflowResult
	if err := json.Unmarshal(completion.Result, &result); err != nil {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusBadRequest, "workflow returned an invalid typed result")
	}
	if err := validateWorkshopWorkflowResult(result); err != nil {
		return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusBadRequest, "invalid workshop proposal: %s", err.Error())
	}
	artifact := ""
	if result.Outcome != "abstained" {
		artifact = filepath.Join("workshop", fmt.Sprintf("round-%03d.json", roundCount+1))
	}
	provenance := workshopWorkflowProvenance{
		Kind: "agent-manager-workflow", ExecutionID: executionID, DefinitionDigest: completion.DefinitionDigest,
		RunID: completion.RunID, ProfileIdentity: completion.ProfileIdentity, EntityVersion: completion.EntityVersion,
		Outcome: result.Outcome, Artifact: artifact, RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if existingProvenance == nil {
		if err := claimWorkshopWorkflowProvenance(workshopDir, provenancePath, provenance); err != nil {
			if errors.Is(err, os.ErrExist) {
				return workshopWorkflowApplyResponse{ExecutionID: executionID, Idempotent: true, Outcome: result.Outcome, Artifact: artifact}, nil
			}
			return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusInternalServerError, "failed to claim workflow result")
		}
	} else {
		artifact = existingProvenance.Artifact
		provenance = *existingProvenance
	}
	if artifact != "" {
		round := workshop.Round{RoundNum: roundCount + 1, GeneratedAt: provenance.RecordedAt, Mode: "workshop", Readiness: result.Readiness, PlanUpdates: result.Note}
		for _, question := range result.DecisionQuestions {
			item := workshop.Item{ID: question.ID, Type: "decision", Topic: question.Topic, Context: question.Context}
			for _, option := range question.Options {
				item.Options = append(item.Options, workshop.Option{Key: option.Key, Label: option.Label, Rationale: option.Rationale, Recommended: option.Recommended})
			}
			round.Items = append(round.Items, item)
		}
		if result.Outcome == "no_questions" && strings.TrimSpace(result.Note) != "" {
			round.Items = append(round.Items, workshop.Item{ID: "workflow-note", Type: "info", Text: result.Note})
		}
		round.PendingSynthesis = workshop.NeedsSynthesis(&round)
		if err := writeJSONRedacted(filepath.Join(itemDir, artifact), round); err != nil {
			return workshopWorkflowApplyResponse{}, workshopApplyFailure(http.StatusInternalServerError, "failed to persist workflow workshop round")
		}
	}
	return workshopWorkflowApplyResponse{ExecutionID: executionID, Applied: true, Outcome: result.Outcome, Artifact: artifact}, nil
}

func validateWorkshopWorkflowResult(result workshopWorkflowResult) error {
	switch result.Outcome {
	case "abstained":
		if strings.TrimSpace(result.Reason) == "" {
			return errors.New("abstention reason is required")
		}
		return nil
	case "no_questions":
		return validateReadiness(result.Readiness)
	case "proposals":
		if err := validateReadiness(result.Readiness); err != nil {
			return err
		}
		seen := map[string]struct{}{}
		for _, question := range result.DecisionQuestions {
			if strings.TrimSpace(question.ID) == "" || strings.TrimSpace(question.Topic) == "" || len(question.Options) < 2 {
				return errors.New("each decision requires id, topic, and at least two options")
			}
			if _, exists := seen[question.ID]; exists {
				return fmt.Errorf("duplicate decision id %q", question.ID)
			}
			seen[question.ID] = struct{}{}
		}
		return nil
	default:
		return fmt.Errorf("unsupported outcome %q", result.Outcome)
	}
}

func validateReadiness(scores map[string]int) error {
	want := append([]string(nil), workshop.ReadinessDimensions...)
	sort.Strings(want)
	for _, dimension := range want {
		if score, ok := scores[dimension]; !ok || score < 0 || score > 3 {
			return fmt.Errorf("readiness %s must be between 0 and 3", dimension)
		}
	}
	return nil
}

func claimWorkshopWorkflowProvenance(dir, path string, provenance workshopWorkflowProvenance) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func readWorkshopWorkflowProvenance(path string) (workshopWorkflowProvenance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return workshopWorkflowProvenance{}, err
	}
	var provenance workshopWorkflowProvenance
	err = json.Unmarshal(data, &provenance)
	return provenance, err
}

// ProcessWorkshopWorkflows reconciles durable workflow correlations into
// backlog-owned workshop rounds. Pending files are the restart boundary: they
// are removed only after an idempotently applied terminal result.
func (h *Handler) ProcessWorkshopWorkflows(ctx context.Context) error {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		return err
	}
	for _, item := range items {
		workshopDir := filepath.Join(h.store.ItemDir(item.Kind, item.Name), "workshop")
		paths, globErr := filepath.Glob(filepath.Join(workshopDir, "workflow-pending-*.json"))
		if globErr != nil {
			return globErr
		}
		for _, path := range paths {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				slog.Warn("workshop workflow reconciliation could not read correlation", "path", path, "err", readErr)
				continue
			}
			var pending workshopWorkflowPending
			if json.Unmarshal(data, &pending) != nil || pending.Kind != string(item.Kind) || pending.Name != item.Name {
				slog.Warn("workshop workflow reconciliation found invalid correlation", "path", path)
				continue
			}
			if _, parseErr := uuid.Parse(pending.ExecutionID); parseErr != nil || path != workshopWorkflowPendingPath(workshopDir, pending.ExecutionID) {
				slog.Warn("workshop workflow reconciliation found invalid execution identity", "path", path)
				continue
			}
			_, applyErr := h.applyWorkshopWorkflowResult(ctx, item.Kind, item.Name, pending.ExecutionID)
			if applyErr != nil {
				if !errors.Is(applyErr, agentmanager.ErrWorkflowNotReady) {
					slog.Warn("workshop workflow result reconciliation failed", "kind", item.Kind, "name", item.Name, "execution_id", pending.ExecutionID, "err", applyErr)
				}
				continue
			}
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				slog.Warn("workshop workflow result applied but correlation cleanup failed", "path", path, "err", removeErr)
			}
		}
	}
	return nil
}
