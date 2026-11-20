package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

// Mirror the canonical control-plane executor-scope-v1 request bound. The root
// owner internally pages large inventories while retaining one host snapshot.
const (
	executorScopeMaxBytes      = 32 << 20
	executorScopeMaxErrorBytes = 4 << 10
)

var errExecutorScopeRequestTooLarge = errors.New("executor-scope request exceeds owner limit")
var errExecutorScopeReferenceLimit = errors.New("executor-scope reference limit exceeded")
var errExecutorScopeInventoryNonAtomic = errors.New("executor-scope inventory required multiple root requests; host snapshots are non-atomic")

type executorScopeRef struct {
	HarnessKind      string                            `json:"harnessKind,omitempty"`
	HarnessSessionID string                            `json:"harnessSessionId,omitempty"`
	BindingProof     *cliutil.AgentSessionBindingProof `json:"bindingProof,omitempty"`
	RunID            string                            `json:"runId"`
	Tag              string                            `json:"tag"`
	LegacyTag        string                            `json:"legacyTag"`
	PID              int                               `json:"pid"`
	PGID             int                               `json:"pgid"`
	StartedAt        *time.Time                        `json:"startedAt,omitempty"`
	EndedAt          *time.Time                        `json:"endedAt,omitempty"`
}

type executorScopeReport struct {
	SchemaVersion   string `json:"schemaVersion"`
	IdentityVersion string `json:"identityVersion,omitempty"`
	Complete        bool   `json:"complete"`
	Executors       []struct {
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
	readBinding func(context.Context, attachedBindingRequest) (cliutil.AgentSessionBindingProof, error)
}

type attachedBindingRequest struct {
	RunID, HarnessSessionID, WebConsoleSessionID, HarnessKind string
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
	if (o.lookup == nil && o.lookupBatch == nil) || o.read == nil || len(refs) < 1 {
		return nil, fmt.Errorf("executor-scope requires an owner reader and at least one reference")
	}
	// Reject duplicate identities across page boundaries before any host read.
	// Exact coverage is part of the proof; processing the same run twice in
	// separate requests cannot turn a duplicate input into valid coverage.
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, exists := seen[ref.ID]; exists {
			return nil, fmt.Errorf("duplicate executor scope")
		}
		seen[ref.ID] = struct{}{}
	}
	return o.observeBoundedSnapshot(ctx, refs)
}

// observeBoundedSnapshot sends one full inventory request whenever it fits the
// root CLI's 32 MiB contract. The root owner performs internal reference pages
// against one process/birth snapshot. If the request itself exceeds the byte
// limit, split it before reading; multiple root calls are useful evidence but
// cannot establish one atomic complete inventory.
func (o *ControlPlaneObserver) observeBoundedSnapshot(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
	evidence, err := o.observePage(ctx, refs)
	if (!errors.Is(err, errExecutorScopeRequestTooLarge) && !errors.Is(err, errExecutorScopeReferenceLimit)) || len(refs) == 1 {
		return evidence, err
	}
	middle := len(refs) / 2
	left, leftErr := o.observeBoundedSnapshot(ctx, refs[:middle])
	if ctxErr := ctx.Err(); ctxErr != nil {
		return left, ctxErr
	}
	right, rightErr := o.observeBoundedSnapshot(ctx, refs[middle:])
	if ctxErr := ctx.Err(); ctxErr != nil {
		return append(left, right...), ctxErr
	}
	if leftErr != nil || rightErr != nil {
		return append(left, right...), errors.Join(errExecutorScopeInventoryNonAtomic, err, leftErr, rightErr)
	}
	return append(left, right...), errors.Join(errExecutorScopeInventoryNonAtomic, err)
}

func (o *ControlPlaneObserver) observePage(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
	if len(refs) < 1 {
		return nil, fmt.Errorf("executor-scope request requires at least one reference")
	}
	request := struct {
		Executors []executorScopeRef `json:"executors"`
	}{}
	attachedIdentityRefs := map[string]bool{}
	attachedIdentityReady := map[string]bool{}
	attachedIdentityFailure := map[string]string{}
	effectivePIDs := map[string]int{}
	expected := map[string]ExecutorRef{}
	var sessionDuplicates []ExecutorEvidence
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
	hydrated := make([]*domain.Run, len(refs))
	for i, ref := range refs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
		hydrated[i] = run
	}
	reattached := reattachedSessionDuplicates(hydrated)
	for i, ref := range refs {
		run := hydrated[i]
		id := run.ID
		if _, exists := expected[ref.ID]; exists {
			return nil, fmt.Errorf("duplicate executor scope")
		}
		if reattached[ref.ID] {
			// Same harness session and process as an earlier live attach; that
			// run carries the session's single physical executor verdict.
			sessionDuplicates = append(sessionDuplicates, ExecutorEvidence{ExecutorRef: ref})
			continue
		}
		expected[ref.ID] = ref
		scope := executorScopeRef{RunID: ref.ID, Tag: run.GetTag(), LegacyTag: "opencode-continue-" + id.String()[:8], PID: ref.PID, PGID: ref.PGID, StartedAt: run.StartedAt, EndedAt: run.EndedAt}
		effectivePIDs[ref.ID] = ref.PID
		if run.ExecutionMode == domain.ExecutionModeAttached {
			attachedIdentityRefs[ref.ID] = true
			// The native lease is anchored to HarnessSessionID, not the run ID
			// or mutable transcript SessionID. Retain even partial/malformed
			// anchors as attached, but send only a complete shape accepted by the
			// owner. A malformed partial anchor must not reject ordinary refs in
			// the same request, and its own verdict remains unknown below.
			kind, sessionID := run.HarnessKind, run.HarnessSessionID
			pid := ref.PID
			if pid <= 1 && ref.PGID == 0 && kind != "" && sessionID != "" && run.WebConsoleSessionID != "" && strings.TrimSpace(kind) == kind && strings.TrimSpace(sessionID) == sessionID && len(kind) <= 100 && len(sessionID) <= 512 {
				proof, proofErr := o.readAttachedBinding(ctx, attachedBindingRequest{
					RunID: id.String(), HarnessSessionID: sessionID,
					WebConsoleSessionID: run.WebConsoleSessionID, HarnessKind: kind,
				})
				if proofErr != nil {
					attachedIdentityFailure[ref.ID] = "attached-session owner identity proof unavailable"
				} else if validAttachedBindingProof(proof, attachedBindingRequest{
					RunID: id.String(), HarnessSessionID: sessionID,
					WebConsoleSessionID: run.WebConsoleSessionID, HarnessKind: kind,
				}) {
					pid = proof.Binding.PID
					scope.BindingProof = &proof
					effectivePIDs[ref.ID] = pid
				} else {
					attachedIdentityFailure[ref.ID] = "attached-session owner identity proof mismatched or incomplete"
				}
			}
			if pid > 1 && kind != "" && sessionID != "" && strings.TrimSpace(kind) == kind && strings.TrimSpace(sessionID) == sessionID && len(kind) <= 100 && len(sessionID) <= 512 {
				scope.HarnessKind, scope.HarnessSessionID = kind, sessionID
				attachedIdentityReady[ref.ID] = true
			}
		}
		request.Executors = append(request.Executors, scope)
	}
	if len(request.Executors) == 0 {
		return sessionDuplicates, nil
	}
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(input) > executorScopeMaxBytes {
		return nil, errExecutorScopeRequestTooLarge
	}
	data, readErr := o.read(ctx, input)
	if readErr != nil && len(data) == 0 {
		return nil, fmt.Errorf("control-plane executor-scope unavailable: %w", readErr)
	}
	if len(data) > executorScopeMaxBytes {
		return nil, fmt.Errorf("executor-scope evidence exceeds limit")
	}
	var report executorScopeReport
	if err := json.Unmarshal(data, &report); err != nil || report.SchemaVersion != "executor-scope-v1" {
		return nil, fmt.Errorf("invalid control-plane executor-scope evidence")
	}
	// Older owners can silently ignore additive request fields while returning
	// the same schema. Their exact report coverage remains useful for unanchored
	// refs, but no verdict for an attached ref is trusted without the marker.
	attachedIdentitySupported := report.IdentityVersion == "attached-key-v1"
	evidence := sessionDuplicates
	var incomplete error
	unknownVerdicts := 0
	for _, item := range report.Executors {
		ref, ok := expected[item.RunID]
		if !ok {
			return nil, fmt.Errorf("unexpected or duplicate executor-scope identity")
		}
		delete(expected, item.RunID)
		value := ExecutorEvidence{ExecutorRef: ref}
		if err := validateExecutorScopeVerdict(item); err != nil {
			return nil, err
		}
		if attachedIdentityRefs[item.RunID] && (!attachedIdentityReady[item.RunID] || !attachedIdentitySupported) {
			unknownVerdicts++
			if incomplete == nil {
				if failure := attachedIdentityFailure[item.RunID]; failure != "" {
					incomplete = fmt.Errorf("%s for run %s", failure, item.RunID)
				} else if !attachedIdentityReady[item.RunID] {
					incomplete = fmt.Errorf("attached-session durable harness identity is incomplete for run %s", item.RunID)
				} else {
					incomplete = fmt.Errorf("control-plane executor-scope lacks attached-key-v1 identity support for attached run %s", item.RunID)
				}
			}
			evidence = append(evidence, value)
			continue
		}
		switch item.State {
		case "present":
			value.Alive = true
			value.PIDs = item.PIDs
			value.HasChildren = len(item.PIDs) > 1 || item.PIDs[0] != effectivePIDs[item.RunID]
		case "absent":
			if len(item.PIDs) > 0 || len(item.Reasons) > 0 {
				incomplete = fmt.Errorf("contradictory executor exclusion evidence")
			}
		case "unknown":
			unknownVerdicts++
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
	if readErr != nil && (report.Complete || unknownVerdicts == 0) && incomplete == nil {
		incomplete = fmt.Errorf("control-plane executor-scope command failed: %w", readErr)
	}
	if !report.Complete && unknownVerdicts == 0 && incomplete == nil {
		incomplete = fmt.Errorf("control-plane executor-scope inventory is incomplete")
	}
	if len(expected) > 0 {
		return evidence, fmt.Errorf("control-plane omitted executor scopes")
	}
	if err := ctx.Err(); err != nil {
		return evidence, err
	}
	return evidence, incomplete
}

// reattachedSessionDuplicates names live attached runs that re-attach a
// harness session already bound to an earlier live run with the same process.
// Every attach of one harness session is a new run of the same conversation
// (attachedConversationID), so the session has one physical executor. The
// control plane proves a leased PID only inside the run's interval, so the
// earliest run of the session is the one whose interval contains the process
// birth; later re-attaches were reported unknown ("leased PID birth is outside
// the recorded Agent Manager run interval") and kept every inventory
// incomplete. Only the earliest run is probed; its verdict covers the session,
// and the later runs remain logical work without a second physical executor.
func reattachedSessionDuplicates(runs []*domain.Run) map[string]bool {
	type sessionKey struct {
		kind, session string
		pid           int
	}
	earliest := map[sessionKey]*domain.Run{}
	before := func(a, b *domain.Run) bool {
		switch {
		case a.StartedAt == nil || b.StartedAt == nil:
			if (a.StartedAt == nil) != (b.StartedAt == nil) {
				return a.StartedAt != nil
			}
		case !a.StartedAt.Equal(*b.StartedAt):
			return a.StartedAt.Before(*b.StartedAt)
		}
		return a.ID.String() < b.ID.String()
	}
	key := func(run *domain.Run) (sessionKey, bool) {
		if run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeAttached || run.Status.IsTerminal() ||
			run.RunnerPID <= 1 || strings.TrimSpace(run.HarnessKind) == "" || strings.TrimSpace(run.HarnessSessionID) == "" {
			return sessionKey{}, false
		}
		return sessionKey{run.HarnessKind, run.HarnessSessionID, run.RunnerPID}, true
	}
	for _, run := range runs {
		if k, ok := key(run); ok && (earliest[k] == nil || before(run, earliest[k])) {
			earliest[k] = run
		}
	}
	out := map[string]bool{}
	for _, run := range runs {
		if k, ok := key(run); ok && earliest[k] != run {
			out[run.ID.String()] = true
		}
	}
	return out
}

func validateExecutorScopeVerdict(item struct {
	RunID   string   `json:"runId"`
	State   string   `json:"state"`
	PIDs    []int    `json:"pids"`
	Reasons []string `json:"reasons"`
}) error {
	switch item.State {
	case "present":
		if len(item.PIDs) == 0 {
			return fmt.Errorf("present executor has no physical evidence")
		}
		for _, pid := range item.PIDs {
			if pid <= 0 {
				return fmt.Errorf("invalid executor PID evidence")
			}
		}
	case "absent":
		if len(item.PIDs) > 0 || len(item.Reasons) > 0 {
			return fmt.Errorf("contradictory executor exclusion evidence")
		}
	case "unknown":
	default:
		return fmt.Errorf("unknown executor-scope verdict")
	}
	return nil
}

type boundedScopeOutput struct{ bytes.Buffer }

func (b *boundedScopeOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > executorScopeMaxBytes {
		return 0, fmt.Errorf("executor-scope output exceeds evidence limit")
	}
	return b.Buffer.Write(p)
}

type boundedScopeErrorOutput struct{ bytes.Buffer }

func (b *boundedScopeErrorOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := executorScopeMaxErrorBytes - b.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	// stderr is diagnostic only; consume excess bytes so the child keeps its
	// normal exit behavior while retaining a strictly bounded prefix.
	return n, nil
}

func safeExecutorScopeError(data []byte) string {
	var payload struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &payload) != nil {
		return "root CLI returned a non-JSON error"
	}
	parts := make([]string, 0, 2)
	for _, value := range []string{payload.Code, payload.Error} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return ' '
			}
			return r
		}, value)
		if len(value) > 256 {
			value = value[:256] + " (truncated)"
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "root CLI returned no structured error detail"
	}
	return strings.Join(parts, ": ")
}

func readExecutorScope(parent context.Context, input []byte) ([]byte, error) {
	// The control-plane command has its own 20s proof budget. Leave time to
	// marshal and return its bounded report before the inventory deadline.
	ctx, cancel := context.WithTimeout(parent, 22*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "vrooli", "runtime", "executor-scope", "--json")
	cmd.Stdin = bytes.NewReader(input)
	var out boundedScopeOutput
	var stderr boundedScopeErrorOutput
	cmd.Stdout, cmd.Stderr = &out, &stderr
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil {
		if out.Len() == 0 {
			detail := safeExecutorScopeError(stderr.Bytes())
			wrapped := fmt.Errorf("%w: %s", err, detail)
			if strings.Contains(detail, "executor-scope requires 1..65536 executors") {
				return nil, errors.Join(errExecutorScopeReferenceLimit, wrapped)
			}
			return nil, wrapped
		}
		// The owner emits a structured incomplete report before returning a
		// non-zero status. Preserve that per-reference coverage so one unknown
		// attachment does not erase valid ordinary executor evidence.
		return out.Bytes(), err
	}
	return out.Bytes(), nil
}
