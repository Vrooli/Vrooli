package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// Mirror the canonical control-plane executor-scope-v1 bounds. One inventory
// attachment uses one indexed host snapshot, including retained managed history.
const (
	executorScopeMaxReferences = 16384
	executorScopeMaxBytes      = 8 << 20
)

type executorScopeRef struct {
	RunID     string     `json:"runId"`
	Tag       string     `json:"tag"`
	LegacyTag string     `json:"legacyTag"`
	PID       int        `json:"pid"`
	PGID      int        `json:"pgid"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

type executorScopeReport struct {
	SchemaVersion string `json:"schemaVersion"`
	Complete      bool   `json:"complete"`
	Executors     []struct {
		RunID   string   `json:"runId"`
		State   string   `json:"state"`
		PIDs    []int    `json:"pids"`
		Reasons []string `json:"reasons"`
	} `json:"executors"`
}

// ControlPlaneObserver delegates all process identity, PID reuse, group and
// descendant inspection to the canonical read-only control-plane command. It
// never signals a process, changes lifecycle, or exchanges an owner credential.
type ControlPlaneObserver struct {
	lookup      func(context.Context, uuid.UUID) (*domain.Run, error)
	lookupBatch func(context.Context, []ExecutorRef) ([]*domain.Run, error)
	read        func(context.Context, []byte) ([]byte, error)
}

func NewControlPlaneInventory(db InventoryDatabase, lookup func(context.Context, uuid.UUID) (*domain.Run, error)) *InventoryReader {
	owner := &ControlPlaneObserver{lookup: lookup, read: readExecutorScope}
	return NewBatchedInventory(db, owner.Observe)
}

// NewControlPlaneInventoryBatch avoids one full run/action hydration per
// historical PID. SQL ownership supplies the existing timestamp decoder.
func NewControlPlaneInventoryBatch(db InventoryDatabase, lookup func(context.Context, []ExecutorRef) ([]*domain.Run, error)) *InventoryReader {
	owner := &ControlPlaneObserver{lookupBatch: lookup, read: readExecutorScope}
	return NewBatchedInventory(db, owner.Observe)
}

// ExcludeExecutor returns nil only when the control plane completely excludes
// the durable run's executor scope. Positive, ambiguous or unavailable evidence
// refuses exclusion. The caller must hold its admission/continuation claim
// across this read and dispatch; absence alone grants no lifecycle authority.
func ExcludeExecutor(ctx context.Context, run *domain.Run) error {
	return excludeExecutor(ctx, run, readExecutorScope)
}

func excludeExecutor(ctx context.Context, run *domain.Run, read func(context.Context, []byte) ([]byte, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if run == nil || run.ID == uuid.Nil {
		return fmt.Errorf("executor exclusion requires a durable run identity")
	}
	ref := ExecutorRef{WorkRef: WorkRef{ID: run.ID.String(), Kind: "run", Status: string(run.Status)}, PID: run.RunnerPID, PGID: run.RunnerPGID}
	owner := &ControlPlaneObserver{
		lookupBatch: func(context.Context, []ExecutorRef) ([]*domain.Run, error) { return []*domain.Run{run}, nil },
		read:        read,
	}
	evidence, err := owner.Observe(ctx, []ExecutorRef{ref})
	if err != nil {
		return err
	}
	if len(evidence) != 1 || evidence[0].ID != ref.ID {
		return fmt.Errorf("executor exclusion lacks exact run coverage")
	}
	if evidence[0].Alive || evidence[0].HasChildren || len(evidence[0].PIDs) > 0 {
		return fmt.Errorf("executor scope remains present for %s: pids %v", ref.ID, evidence[0].PIDs)
	}
	return nil
}

func (o *ControlPlaneObserver) Observe(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
	if (o.lookup == nil && o.lookupBatch == nil) || o.read == nil || len(refs) < 1 || len(refs) > executorScopeMaxReferences {
		return nil, fmt.Errorf("executor-scope requires an owner reader and 1..%d references", executorScopeMaxReferences)
	}
	request := struct {
		Executors []executorScopeRef `json:"executors"`
	}{}
	expected := map[string]ExecutorRef{}
	identities := map[uuid.UUID]*domain.Run{}
	if o.lookupBatch != nil {
		runs, err := o.lookupBatch(ctx, refs)
		if err != nil {
			return nil, fmt.Errorf("hydrate executor scopes: %w", err)
		}
		for _, run := range runs {
			if run == nil {
				return nil, fmt.Errorf("missing durable executor identity")
			}
			if _, exists := identities[run.ID]; exists {
				return nil, fmt.Errorf("duplicate durable executor identity")
			}
			identities[run.ID] = run
		}
	}
	for _, ref := range refs {
		id, err := uuid.Parse(ref.ID)
		if err != nil {
			return nil, fmt.Errorf("executor scope has invalid durable run ID")
		}
		run := identities[id]
		if o.lookupBatch == nil {
			run, err = o.lookup(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("hydrate executor scope: %w", err)
			}
		}
		if run == nil || run.ID != id || run.RunnerPID != ref.PID || run.RunnerPGID != ref.PGID {
			return nil, fmt.Errorf("durable executor identity changed during inventory")
		}
		if _, exists := expected[ref.ID]; exists {
			return nil, fmt.Errorf("duplicate executor scope")
		}
		expected[ref.ID] = ref
		request.Executors = append(request.Executors, executorScopeRef{RunID: ref.ID, Tag: run.GetTag(), LegacyTag: "opencode-continue-" + id.String()[:8], PID: ref.PID, PGID: ref.PGID, StartedAt: run.StartedAt, EndedAt: run.EndedAt})
	}
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(input) > executorScopeMaxBytes {
		return nil, fmt.Errorf("executor-scope request exceeds owner limit")
	}
	data, err := o.read(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("control-plane executor-scope unavailable: %w", err)
	}
	if len(data) > executorScopeMaxBytes {
		return nil, fmt.Errorf("executor-scope evidence exceeds limit")
	}
	var report executorScopeReport
	if err := json.Unmarshal(data, &report); err != nil || report.SchemaVersion != "executor-scope-v1" {
		return nil, fmt.Errorf("invalid control-plane executor-scope evidence")
	}
	var evidence []ExecutorEvidence
	var incomplete error
	if !report.Complete {
		incomplete = fmt.Errorf("control-plane executor-scope inventory is incomplete")
	}
	for _, item := range report.Executors {
		ref, ok := expected[item.RunID]
		if !ok {
			return nil, fmt.Errorf("unexpected or duplicate executor-scope identity")
		}
		delete(expected, item.RunID)
		value := ExecutorEvidence{ExecutorRef: ref}
		switch item.State {
		case "present":
			if len(item.PIDs) == 0 {
				return nil, fmt.Errorf("present executor has no physical evidence")
			}
			for _, pid := range item.PIDs {
				if pid <= 0 {
					return nil, fmt.Errorf("invalid executor PID evidence")
				}
			}
			value.Alive = true
			value.PIDs = item.PIDs
			value.HasChildren = len(item.PIDs) > 1 || item.PIDs[0] != ref.PID
		case "absent":
			if len(item.PIDs) > 0 || len(item.Reasons) > 0 {
				incomplete = fmt.Errorf("contradictory executor exclusion evidence")
			}
		case "unknown":
			reason := strings.Join(item.Reasons, "; ")
			if len(reason) > 512 {
				reason = reason[:512] + " (truncated)"
			}
			incomplete = fmt.Errorf("control-plane executor-scope exclusion is unknown for %s: %s", item.RunID, reason)
		default:
			return nil, fmt.Errorf("unknown executor-scope verdict")
		}
		evidence = append(evidence, value)
	}
	if len(expected) > 0 {
		return evidence, fmt.Errorf("control-plane omitted executor scopes")
	}
	if err := ctx.Err(); err != nil {
		return evidence, err
	}
	return evidence, incomplete
}

type boundedScopeOutput struct{ bytes.Buffer }

func (b *boundedScopeOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > executorScopeMaxBytes {
		return 0, fmt.Errorf("executor-scope output exceeds evidence limit")
	}
	return b.Buffer.Write(p)
}

func readExecutorScope(parent context.Context, input []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 4500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "vrooli", "runtime", "executor-scope", "--json")
	cmd.Stdin = bytes.NewReader(input)
	var out boundedScopeOutput
	cmd.Stdout, cmd.Stderr = &out, io.Discard
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
