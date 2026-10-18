package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

// taskDispatchOwner is deliberately narrower than AgentClient: the task-create
// recovery boundary must only inspect or create the exact frozen task. In
// particular, it cannot cancel/delete a task whose identity or terms differ.
type taskDispatchOwner interface {
	CreateTask(context.Context, *Task) (*Task, error)
	GetTask(context.Context, string) (*Task, error)
}

// freezeTaskDispatchRequest serializes the complete caller-authored task before
// its first owner request. The UUID is the idempotency key for task creation.
func freezeTaskDispatchRequest(task *Task) (string, error) {
	if task == nil {
		return "", errors.New("task dispatch request is nil")
	}
	if err := validateTaskDispatchID(task.ID); err != nil {
		return "", err
	}
	for index, phaseID := range task.PhasePromptIDs {
		if err := validateTaskDispatchID(phaseID); err != nil {
			return "", fmt.Errorf("task dispatch request phase prompt ID %d: %w", index, err)
		}
	}
	encoded, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("marshal task dispatch request: %w", err)
	}
	return string(encoded), nil
}

func parseTaskDispatchRequest(snapshot string) (*Task, error) {
	if strings.TrimSpace(snapshot) == "" {
		return nil, errors.New("task dispatch request snapshot is empty")
	}
	decoder := json.NewDecoder(strings.NewReader(snapshot))
	decoder.DisallowUnknownFields()
	var task Task
	if err := decoder.Decode(&task); err != nil {
		return nil, fmt.Errorf("decode task dispatch request snapshot: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("task dispatch request snapshot has trailing JSON")
		}
		return nil, fmt.Errorf("read task dispatch request snapshot: %w", err)
	}
	if err := validateTaskDispatchID(task.ID); err != nil {
		return nil, err
	}
	for index, phaseID := range task.PhasePromptIDs {
		if err := validateTaskDispatchID(phaseID); err != nil {
			return nil, fmt.Errorf("task dispatch request phase prompt ID %d: %w", index, err)
		}
	}
	return &task, nil
}

func validateTaskDispatchID(value string) error {
	if strings.TrimSpace(value) != value {
		return errors.New("task dispatch request has a non-canonical UUID")
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil || id.String() != value {
		return errors.New("task dispatch request has an invalid UUID")
	}
	return nil
}

// resolveTaskDispatch first looks up the persisted task UUID. Only a proven
// 404 permits posting the frozen request, and every ambiguous POST result is
// reconciled against that same UUID. It never generates a replacement ID.
func resolveTaskDispatch(ctx context.Context, owner taskDispatchOwner, snapshot string) (*Task, error) {
	if owner == nil {
		return nil, errors.New("task dispatch owner is unavailable")
	}
	request, err := parseTaskDispatchRequest(snapshot)
	if err != nil {
		return nil, err
	}

	existing, err := owner.GetTask(ctx, request.ID)
	if err != nil {
		// A canceled preflight is a read-only failure. When this is the fresh
		// Execute path, the caller can safely cancel its just-persisted intent;
		// when this is a retained replay, ReconcileTaskDispatch keeps it held.
		// Do not turn an ordinary lookup failure into absence.
		if ctx.Err() != nil {
			return nil, NewDispatchRejectedError(ctx.Err())
		}
		return nil, NewDispatchUncertainError(fmt.Errorf("lookup task %s before dispatch: %w", request.ID, err))
	}
	if existing != nil {
		if err := verifyTaskDispatchMatch(request, existing); err != nil {
			return nil, NewDispatchUncertainError(err)
		}
		return existing, nil
	}

	created, createErr := owner.CreateTask(ctx, request)
	if createErr == nil {
		if err := verifyTaskDispatchMatch(request, created); err != nil {
			return nil, NewDispatchUncertainError(err)
		}
		return created, nil
	}
	// A typed approval response means the owner has not created the task; keep
	// its existing consent path intact instead of interpreting the 404 lookup
	// as permission to issue another request in this call.
	if IsApprovalPending(createErr) {
		return nil, createErr
	}
	// Agent Manager's pre-send cancellation guard proves no task POST was
	// attempted. Preserve that signal instead of issuing a follow-up GET with
	// the canceled context and converting it into a replayable uncertainty.
	if isPreSendCancellation(createErr) {
		return nil, createErr
	}

	observed, lookupErr := owner.GetTask(ctx, request.ID)
	if lookupErr != nil {
		return nil, NewDispatchUncertainError(errors.Join(
			fmt.Errorf("create task %s: %w", request.ID, createErr),
			fmt.Errorf("reconcile task %s after create result: %w", request.ID, lookupErr),
		))
	}
	if observed == nil {
		if IsDispatchRejected(createErr) {
			return nil, fmt.Errorf("create task %s: %w", request.ID, createErr)
		}
		return nil, NewDispatchUncertainError(fmt.Errorf("create task %s returned an ambiguous result and its exact-ID lookup still reports absence: %w", request.ID, createErr))
	}
	if err := verifyTaskDispatchMatch(request, observed); err != nil {
		return nil, NewDispatchUncertainError(errors.Join(fmt.Errorf("create task %s: %w", request.ID, createErr), err))
	}
	return observed, nil
}

func verifyTaskDispatchMatch(request, observed *Task) error {
	if request == nil || observed == nil {
		return errors.New("task dispatch owner returned no task")
	}
	if err := validateTaskDispatchID(observed.ID); err != nil || observed.ID != request.ID {
		return fmt.Errorf("task dispatch identity mismatch: expected %q, received %q", request.ID, observed.ID)
	}
	if request.Title != observed.Title {
		return fmt.Errorf("task %s exists with a title different from the frozen heartbeat request", request.ID)
	}
	if request.Description != observed.Description {
		return fmt.Errorf("task %s exists with a description different from the frozen heartbeat request", request.ID)
	}
	if request.ScopePath != observed.ScopePath {
		return fmt.Errorf("task %s exists with a scope path different from the frozen heartbeat request", request.ID)
	}
	if request.ProjectRoot != observed.ProjectRoot {
		return fmt.Errorf("task %s exists with a project root different from the frozen heartbeat request", request.ID)
	}
	if len(request.PhasePromptIDs) != len(observed.PhasePromptIDs) {
		return fmt.Errorf("task %s exists with phase prompt IDs different from the frozen heartbeat request", request.ID)
	}
	for index := range request.PhasePromptIDs {
		if request.PhasePromptIDs[index] != observed.PhasePromptIDs[index] {
			return fmt.Errorf("task %s exists with phase prompt IDs different from the frozen heartbeat request", request.ID)
		}
	}
	// Heartbeat run creation is only allowed for a newly queued task. Existing
	// running/review/terminal tasks may belong to an earlier or foreign owner
	// action; accepting them could resurrect a cancelled/completed task or create
	// a duplicate run. Keep such rows unresolved and do not mutate them.
	if !isQueuedTaskStatus(observed.Status) {
		return fmt.Errorf("task %s has non-runnable owner status %q; expected queued", request.ID, observed.Status)
	}
	requestedAttachments, err := canonicalTaskAttachments(request.ContextAttachments)
	if err != nil {
		return fmt.Errorf("frozen task %s has invalid context attachments: %w", request.ID, err)
	}
	observedAttachments, err := canonicalTaskAttachments(observed.ContextAttachments)
	if err != nil {
		return fmt.Errorf("owner task %s has invalid context attachments: %w", request.ID, err)
	}
	if !bytes.Equal(requestedAttachments, observedAttachments) {
		return fmt.Errorf("task %s exists with context attachments different from the frozen heartbeat request", request.ID)
	}
	return nil
}

// Agent Manager's HTTP owner serializes protobuf enums using their exact
// symbolic names. The lower-case domain value is retained for local test
// doubles; no other spelling or case is treated as queued.
func isQueuedTaskStatus(status string) bool {
	switch status {
	case "queued", "TASK_STATUS_QUEUED":
		return true
	default:
		return false
	}
}

func canonicalTaskAttachments(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
