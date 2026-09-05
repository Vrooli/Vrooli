package cleanup

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operationcoord"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
)

type service struct {
	repo     Repository
	nodes    NodeReader
	presence Presence
	pusher   CommandPusher
	audit    AuditSink
	clock    schedule.Clock
	coord    *operationcoord.Coordinator[Event]
}

func NewService(repo Repository, nodes NodeReader, presence Presence, pusher CommandPusher, audit AuditSink, clock schedule.Clock) Service {
	return &service{repo: repo, nodes: nodes, presence: presence, pusher: pusher, audit: audit, clock: clock, coord: operationcoord.New[Event]()}
}

func (s *service) Prepare(ctx context.Context, in StartInput, actor string) (Target, error) {
	in = normalizeStartInput(in)
	if err := validateStartInput(in); err != nil {
		return Target{}, err
	}
	node, selection, err := s.resolveTarget(ctx, in)
	if err != nil {
		return Target{}, err
	}
	return Target{
		MachineID:        in.MachineID,
		NodeID:           in.NodeID,
		Target:           in.Target,
		Scope:            in.Scope,
		Transport:        string(selection.Transport),
		TransportReason:  selection.Reason,
		OperatorID:       strings.TrimSpace(actor),
		OperationID:      uuid.NewString(),
		SealingPublicKey: append([]byte(nil), node.SealingPublicKey...),
		Capabilities:     append([]string(nil), node.Capabilities...),
		ApprovedScopes:   append([]string(nil), node.Scopes...),
	}, nil
}

func (s *service) ProvisionBreakGlass(ctx context.Context, in ProvisionInput) (Operation, error) {
	if strings.TrimSpace(in.OperationID) == "" {
		return Operation{}, ErrInvalid{Field: "operation_id", Reason: "required"}
	}
	if len(in.SealedPassphrase) == 0 {
		return Operation{}, ErrInvalid{Field: "sealed_passphrase", Reason: "required"}
	}
	operatorID := strings.TrimSpace(in.OperatorID)
	if operatorID == "" {
		return Operation{}, ErrInvalid{Field: "operator_id", Reason: "required"}
	}
	start := normalizeStartInput(StartInput{MachineID: in.MachineID, NodeID: in.NodeID, Target: in.Target, Scope: in.Scope})
	if err := validateStartInput(start); err != nil {
		return Operation{}, err
	}
	node, selection, err := s.resolveTarget(ctx, start)
	if err != nil {
		return Operation{}, err
	}
	op, err := s.repo.Create(ctx, Operation{ID: in.OperationID, MachineID: start.MachineID, NodeID: start.NodeID, Target: start.Target, Scope: start.Scope, Status: StatusQueued, Transport: string(selection.Transport), TransportReason: selection.Reason, SealingPublicKey: append([]byte(nil), node.SealingPublicKey...), OperatorID: operatorID})
	if err != nil {
		if existing, getErr := s.repo.Get(ctx, in.OperationID); getErr == nil {
			if existing.MachineID != start.MachineID || existing.NodeID != start.NodeID || existing.Target != start.Target || existing.Scope != start.Scope || existing.OperatorID != operatorID {
				return Operation{}, ErrConflict{Field: "operation_id", Reason: "operation id is already bound to another cleanup target"}
			}
			return existing, nil
		}
		if active, lookupErr := s.findActive(ctx, start.MachineID); lookupErr == nil {
			return Operation{}, ErrInFlight{MachineID: start.MachineID, OperationID: active.ID}
		}
		return Operation{}, err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{Actor: operatorID, NodeID: start.NodeID, OperationID: op.ID, Verb: "cleanup.provision_break_glass", Outcome: "accepted", Detail: "target-bound protection provisioning requested"}); err != nil {
			return s.fail(ctx, op, "audit write failed")
		}
	}
	if op, err = s.push(ctx, op, "provision_break_glass", true, in.SealedPassphrase); err != nil {
		return s.fail(ctx, op, err.Error())
	}
	return op, nil
}

func (s *service) ResetBreakGlass(ctx context.Context, in ResetInput, actor string) (Operation, error) {
	start := normalizeStartInput(StartInput{MachineID: in.MachineID, NodeID: in.NodeID, Target: in.Target, Scope: in.Scope})
	if err := validateStartInput(start); err != nil {
		return Operation{}, err
	}
	if active, err := s.findActive(ctx, start.MachineID); err == nil {
		return Operation{}, ErrInFlight{MachineID: start.MachineID, OperationID: active.ID}
	}
	node, selection, err := s.resolveTarget(ctx, start)
	if err != nil {
		return Operation{}, err
	}
	op, err := s.repo.Create(ctx, Operation{
		ID: uuid.NewString(), MachineID: start.MachineID, NodeID: start.NodeID, Target: start.Target,
		Scope: start.Scope, Status: StatusQueued, Transport: string(selection.Transport),
		TransportReason: selection.Reason, OperatorID: strings.TrimSpace(actor),
		SealingPublicKey: append([]byte(nil), node.SealingPublicKey...),
	})
	if err != nil {
		if active, lookupErr := s.findActive(ctx, start.MachineID); lookupErr == nil {
			return Operation{}, ErrInFlight{MachineID: start.MachineID, OperationID: active.ID}
		}
		return Operation{}, err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{Actor: op.OperatorID, NodeID: op.NodeID, OperationID: op.ID, Verb: "cleanup.reset_break_glass", Outcome: "accepted", Detail: "target-bound break-glass material retirement requested"}); err != nil {
			return s.fail(ctx, op, "audit write failed")
		}
	}
	if op, err = s.push(ctx, op, privilegedops.ResetBreakGlass, true); err != nil {
		return s.fail(ctx, op, err.Error())
	}
	return op, nil
}

func (s *service) Start(ctx context.Context, in StartInput, actor string) (Operation, error) {
	in = normalizeStartInput(in)
	if err := validateStartInput(in); err != nil {
		return Operation{}, err
	}
	node, selection, err := s.resolveTarget(ctx, in)
	if err != nil {
		return Operation{}, err
	}
	op, err := s.repo.Create(ctx, Operation{ID: uuid.NewString(), MachineID: in.MachineID, NodeID: in.NodeID, Target: in.Target, Scope: in.Scope, Status: StatusQueued, Transport: string(selection.Transport), TransportReason: selection.Reason, SealingPublicKey: append([]byte(nil), node.SealingPublicKey...), OperatorID: actor})
	if err != nil {
		// The partial unique index is the concurrency guard. Resolve its typed
		// result instead of leaking a SQLite constraint to the operator.
		if active, lookupErr := s.findActive(ctx, in.MachineID); lookupErr == nil {
			return Operation{}, ErrInFlight{MachineID: in.MachineID, OperationID: active.ID}
		}
		return Operation{}, err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{Actor: actor, NodeID: in.NodeID, OperationID: op.ID, Verb: "cleanup.start", Outcome: "accepted", Detail: "cleanup inventory requested"}); err != nil {
			_, _ = s.fail(ctx, op, "audit write failed")
			return Operation{}, err
		}
	}
	if op, err = s.push(ctx, op, "plan_uninstall", false); err != nil {
		return s.fail(ctx, op, err.Error())
	}
	return op, nil
}

func normalizeStartInput(in StartInput) StartInput {
	in.MachineID = strings.TrimSpace(in.MachineID)
	in.NodeID = strings.TrimSpace(in.NodeID)
	in.Target = strings.TrimSpace(in.Target)
	in.Scope = strings.TrimSpace(in.Scope)
	return in
}

func validateStartInput(in StartInput) error {
	if in.MachineID == "" {
		return ErrInvalid{Field: "machine_id", Reason: "required"}
	}
	if in.NodeID == "" {
		return ErrInvalid{Field: "node_id", Reason: "required"}
	}
	if in.Target == "" {
		return ErrInvalid{Field: "target", Reason: "required"}
	}
	return nil
}

func (s *service) resolveTarget(ctx context.Context, in StartInput) (TargetNode, TransportSelection, error) {
	node, err := s.nodes.GetTarget(ctx, in.NodeID)
	if err != nil {
		return TargetNode{}, TransportSelection{}, err
	}
	if node.Revoked {
		return TargetNode{}, TransportSelection{}, ErrBlocked{Field: "node", Reason: "node is revoked"}
	}
	if len(node.SealingPublicKey) == 0 {
		return TargetNode{}, TransportSelection{}, ErrBlocked{Field: "sealing_key", Reason: "paired node has no published sealing key"}
	}
	selection, selectErr := SelectTransport(TransportFacts{AgentOnline: s.presence.IsOnline(in.NodeID), SSHManagement: contains(node.Capabilities, privilegedops.CapabilitySSHManagement), SSHScopeApproved: contains(node.Scopes, privilegedops.CapabilitySSHManagement), TargetReachable: strings.TrimSpace(node.Endpoint) != ""})
	if selectErr != nil {
		return TargetNode{}, TransportSelection{}, selectErr
	}
	if selection.Transport == TransportSSH {
		if _, ok := s.pusher.(SSHCommandPusher); !ok {
			return TargetNode{}, TransportSelection{}, ErrBlocked{Field: "ssh.management", Reason: "verified SSH transport is not configured for typed cleanup"}
		}
	}
	return node, selection, nil
}

func (s *service) Get(ctx context.Context, id string) (Operation, []Event, error) {
	op, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Operation{}, nil, err
	}
	events, err := s.repo.ListEvents(ctx, op.ID)
	if err != nil {
		return Operation{}, nil, err
	}
	return op, events, nil
}

func (s *service) Plan(ctx context.Context, id string) (Operation, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if len(op.PlanJSON) == 0 {
		return Operation{}, ErrConflict{Field: "plan", Reason: "inventory has not been returned"}
	}
	if _, err := ParseFrozenPlan(op.PlanJSON); err != nil {
		return Operation{}, ErrConflict{Field: "plan", Reason: err.Error()}
	}
	return op, nil
}

func (s *service) Confirm(ctx context.Context, in ConfirmInput) (Operation, error) {
	op, err := s.repo.Get(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return Operation{}, err
	}
	if err := s.ensureOperationTarget(ctx, op); err != nil {
		return Operation{}, err
	}
	if op.Status != StatusPlanned {
		return Operation{}, ErrConflict{Field: "status", Reason: "operation is not planned"}
	}
	if strings.TrimSpace(in.Target) == "" || in.Target != op.Target {
		return Operation{}, ErrConflict{Field: "target", Reason: "confirmation target does not match operation target"}
	}
	if strings.TrimSpace(in.PlanHash) == "" || in.PlanHash != op.PlanHash {
		return Operation{}, ErrConflict{Field: "plan_hash", Reason: "confirmation hash does not match frozen plan"}
	}
	if len(in.SealedPassphrase) == 0 {
		return Operation{}, ErrInvalid{Field: "sealed_passphrase", Reason: "required"}
	}
	operatorID := strings.TrimSpace(in.OperatorID)
	if operatorID == "" {
		operatorID = strings.TrimSpace(op.OperatorID)
	}
	if operatorID == "" {
		return Operation{}, ErrInvalid{Field: "operator_id", Reason: "required"}
	}
	if op.OperatorID != "" && operatorID != op.OperatorID {
		return Operation{}, ErrConflict{Field: "operator_id", Reason: "confirmation operator does not match the operation owner"}
	}
	op.OperatorID = operatorID
	op.SealedPassphrase = append([]byte(nil), in.SealedPassphrase...)
	op.Capability = append([]byte(nil), in.Capability...)
	op.Status = StatusConfirmed
	op.UpdatedAt = s.clock.Now().UTC()
	if _, err := s.repo.Update(ctx, op); err != nil {
		return Operation{}, err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{Actor: op.OperatorID, NodeID: op.NodeID, OperationID: op.ID, Verb: "cleanup.confirm", Outcome: "accepted", Detail: "frozen cleanup plan confirmed"}); err != nil {
			return s.fail(ctx, op, "audit write failed")
		}
	}
	// The control plane stores only opaque, node-bound ciphertext; it cannot open
	// the envelope. The helper is the only component that receives plaintext.
	if op, err = s.push(ctx, op, "apply_frozen_plan", true, in.SealedPassphrase, in.Capability); err != nil {
		return s.fail(ctx, op, err.Error())
	}
	return op, nil
}

func (s *service) Apply(ctx context.Context, id string) (Operation, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if err := s.ensureOperationTarget(ctx, op); err != nil {
		return Operation{}, err
	}
	if op.Status != StatusConfirmed && op.Status != StatusApplying && op.Status != StatusFailed && op.Status != StatusBlocked {
		return Operation{}, ErrConflict{Field: "status", Reason: "operation is not confirmed or resumable"}
	}
	if len(op.SealedPassphrase) == 0 || op.PlanHash == "" || len(op.PlanJSON) == 0 {
		return Operation{}, ErrConflict{Field: "authorization", Reason: "confirmed opaque authorization is unavailable for resume"}
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{Actor: op.OperatorID, NodeID: op.NodeID, OperationID: op.ID, Verb: "cleanup.apply", Outcome: "accepted", Detail: "frozen cleanup plan resume requested"}); err != nil {
			return s.fail(ctx, op, "audit write failed")
		}
	}
	if op, err = s.push(ctx, op, "apply_frozen_plan", true, op.SealedPassphrase, op.Capability); err != nil {
		return s.fail(ctx, op, err.Error())
	}
	return op, nil
}

func (s *service) Verify(ctx context.Context, id string) (Operation, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	// A terminal operation already has its durable outcome. The helper is
	// intentionally allowed to shut down after publishing a receipt, so a
	// later best-effort verification must never turn a completed cleanup into a
	// transport failure merely because the helper is no longer reachable.
	if op.Status.Terminal() {
		return op, nil
	}
	if op.PlanHash == "" {
		return Operation{}, ErrConflict{Field: "plan_hash", Reason: "no frozen plan"}
	}
	if _, err := s.push(ctx, op, "verify_result", false); err != nil {
		return s.fail(ctx, op, err.Error())
	}
	return op, nil
}

func (s *service) Cancel(ctx context.Context, id, reason string) (Operation, error) {
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	if op.Status.Terminal() {
		return op, nil
	}
	op.Status = StatusCanceled
	op.Reason = strings.TrimSpace(reason)
	op.UpdatedAt = s.clock.Now().UTC()
	op.FinishedAt = op.UpdatedAt
	updated, err := s.repo.Update(ctx, op)
	if err != nil {
		return Operation{}, err
	}
	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{Actor: op.OperatorID, NodeID: op.NodeID, OperationID: op.ID, Verb: "cleanup.cancel", Outcome: "accepted", Detail: op.Reason}); err != nil {
			return s.fail(ctx, updated, "audit write failed")
		}
	}
	return updated, nil
}

func (s *service) AppendEvent(ctx context.Context, ev Event) (bool, error) {
	op, err := s.repo.Get(ctx, ev.OperationID)
	if err != nil {
		return false, err
	}
	if op.Status.Terminal() {
		return false, nil
	}
	if ev.Kind == EventPlan {
		plan, parseErr := ParseFrozenPlan(ev.PlanJSON)
		if parseErr != nil {
			return false, ErrConflict{Field: "plan", Reason: parseErr.Error()}
		}
		if op.PlanHash != "" && op.PlanHash != plan.PlanHash {
			return false, ErrConflict{Field: "plan_hash", Reason: "resolved artifact list changed after freeze"}
		}
		if len(op.PlanJSON) > 0 && !bytes.Equal(op.PlanJSON, ev.PlanJSON) {
			return false, ErrConflict{Field: "plan", Reason: "frozen plan cannot be replaced"}
		}
	}
	if ev.Kind == EventReceipt {
		if op.PlanHash == "" || len(op.PlanJSON) == 0 {
			return false, ErrConflict{Field: "receipt", Reason: "receipt arrived before the frozen plan"}
		}
		if receiptErr := ValidateReceipt(ev.ReceiptJSON, op); receiptErr != nil {
			return false, ErrConflict{Field: "receipt", Reason: receiptErr.Error()}
		}
	}
	// Agent commands have independent local event counters. Reconcile those
	// counters into one durable, operation-global sequence before persisting.
	// The retry loop also closes the small race where two transports observe the
	// same current maximum and choose the same next sequence.
	accepted := false
	for attempt := 0; attempt < 4; attempt++ {
		events, listErr := s.repo.ListEvents(ctx, ev.OperationID)
		if listErr != nil {
			return false, listErr
		}
		reconciled, shouldAppend, reconcileErr := reconcileEventSequence(events, ev)
		if reconcileErr != nil {
			return false, reconcileErr
		}
		if !shouldAppend {
			return false, nil
		}
		accepted, err = s.repo.AppendEvent(ctx, reconciled)
		if accepted {
			ev = reconciled
			break
		}
		if err != nil {
			return false, err
		}
	}
	if !accepted {
		return false, errors.New("cleanup event sequence could not be allocated")
	}
	changed := false
	if ev.Status != "" {
		if status, ok := parseStatus(ev.Status); ok {
			op.Status = status
			changed = true
		}
	}
	if len(ev.PlanJSON) > 0 {
		if len(op.PlanJSON) == 0 {
			op.PlanJSON = append([]byte(nil), ev.PlanJSON...)
			changed = true
		}
		if plan, parseErr := ParseFrozenPlan(ev.PlanJSON); parseErr == nil && op.PlanHash == "" {
			op.PlanHash = plan.PlanHash
			changed = true
		}
	}
	if len(ev.ReceiptJSON) > 0 {
		op.ReceiptJSON = append([]byte(nil), ev.ReceiptJSON...)
		changed = true
	}
	if ev.Reason != "" {
		op.Reason = ev.Reason
		changed = true
	}
	if ev.Kind == EventExit && !op.Status.Terminal() {
		// A successful inventory/plan operation emits a planned status before
		// its terminal exit event. The exit acknowledges the read-only request;
		// it must not turn a frozen plan into a completed operation, because
		// confirmation is intentionally the next legal state.
		if !(op.Status == StatusPlanned && ev.ExitCode == 0) {
			op.Status = statusFromExit(ev.ExitCode)
			changed = true
		}
	}
	if op.Status.Terminal() && op.FinishedAt.IsZero() {
		op.FinishedAt = s.clock.Now().UTC()
		changed = true
	}
	if changed {
		op.UpdatedAt = s.clock.Now().UTC()
		_, err = s.repo.Update(ctx, op)
	}
	s.coord.Publish(op.ID, ev)
	if op.Status.Terminal() {
		s.coord.SignalTerminal(op.ID)
	}
	return true, err
}

// reconcileEventSequence maps a transport-local sequence into the durable
// operation sequence. Identical replays are idempotent; a conflicting or
// stale local sequence is assigned after the current durable maximum.
func reconcileEventSequence(existing []Event, incoming Event) (Event, bool, error) {
	max := uint64(0)
	conflict := false
	for _, current := range existing {
		if current.Sequence > max {
			max = current.Sequence
		}
		if current.Sequence == incoming.Sequence {
			if equivalentEvent(current, incoming) {
				return incoming, false, nil
			}
			conflict = true
		}
	}
	if incoming.Sequence == 0 || incoming.Sequence <= max || conflict {
		if max == ^uint64(0) {
			return Event{}, false, errors.New("cleanup event sequence exhausted")
		}
		incoming.Sequence = max + 1
	}
	return incoming, true, nil
}

func equivalentEvent(a, b Event) bool {
	return a.OperationID == b.OperationID &&
		a.Kind == b.Kind &&
		a.Status == b.Status &&
		a.LogChunk == b.LogChunk &&
		bytes.Equal(a.PlanJSON, b.PlanJSON) &&
		bytes.Equal(a.ReceiptJSON, b.ReceiptJSON) &&
		a.Reason == b.Reason &&
		a.ExitCode == b.ExitCode
}

func (s *service) Wait(ctx context.Context, id string, timeout time.Duration) (Operation, bool, error) {
	if timeout <= 0 {
		timeout = 60 * time.Minute
	}
	wait, cancel := s.coord.RegisterWaiter(id)
	defer cancel()
	op, err := s.repo.Get(ctx, id)
	if err != nil {
		return Operation{}, false, err
	}
	if op.Status.Terminal() {
		return op, false, nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Operation{}, false, ctx.Err()
	case <-timer.C:
		op, err = s.repo.Get(ctx, id)
		if err != nil {
			return Operation{}, false, err
		}
		return op, !op.Status.Terminal(), nil
	case <-wait:
		op, err = s.repo.Get(ctx, id)
		if err != nil {
			return Operation{}, false, err
		}
		return op, false, nil
	}
}

func (s *service) Subscribe(id string) (<-chan Event, func()) {
	return s.coord.Subscribe(id)
}

func (s *service) push(ctx context.Context, op Operation, operation string, confirmed bool, secret ...[]byte) (Operation, error) {
	cmd := Command{Operation: operation, OpID: op.ID, MachineID: op.MachineID, NodeID: op.NodeID, Target: op.Target, Scope: op.Scope, PlanID: op.ID, PlanHash: op.PlanHash, OperatorConfirmed: confirmed, OperatorID: op.OperatorID}
	if len(secret) > 0 {
		cmd.SealedPassphrase = append([]byte(nil), secret[0]...)
	}
	if len(secret) > 1 {
		cmd.Capability = append([]byte(nil), secret[1]...)
	}
	var pushErr error
	if op.Transport == string(TransportSSH) {
		pusher, ok := s.pusher.(SSHCommandPusher)
		if !ok {
			return op, ErrBlocked{Field: "ssh.management", Reason: "verified SSH transport is not configured for typed cleanup"}
		}
		_, pushErr = pusher.PushCleanupSSH(ctx, op.NodeID, cmd)
	} else {
		_, pushErr = s.pusher.PushCleanup(ctx, op.NodeID, cmd)
	}
	if pushErr != nil {
		return op, pushErr
	}
	if operation == "plan_uninstall" {
		op.Status = StatusPlanning
	} else if operation == "apply_frozen_plan" || operation == privilegedops.ResetBreakGlass {
		op.Status = StatusApplying
	}
	op.UpdatedAt = s.clock.Now().UTC()
	op, err := s.repo.Update(ctx, op)
	return op, err
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func (s *service) ensureOperationTarget(ctx context.Context, op Operation) error {
	node, err := s.nodes.GetTarget(ctx, op.NodeID)
	if err != nil {
		return err
	}
	if node.Revoked {
		return ErrBlocked{Field: "node", Reason: "node was revoked after the operation was created"}
	}
	if node.ID != "" && node.ID != op.NodeID {
		return ErrBlocked{Field: "node_id", Reason: "operation node identity no longer matches the paired node"}
	}
	if len(node.SealingPublicKey) == 0 {
		return ErrBlocked{Field: "sealing_key", Reason: "paired node no longer has a published sealing key"}
	}
	return nil
}

func (s *service) fail(ctx context.Context, op Operation, reason string) (Operation, error) {
	op.Status = StatusFailed
	op.Reason = reason
	op.UpdatedAt = s.clock.Now().UTC()
	op.FinishedAt = op.UpdatedAt
	updated, updateErr := s.repo.Update(ctx, op)
	if updateErr != nil {
		return op, updateErr
	}
	ev := Event{OperationID: op.ID, Kind: EventStatus, Sequence: s.coord.NextSyntheticSeq(op.ID), Status: "failed", Reason: reason, EmittedAt: op.UpdatedAt}
	_, _ = s.repo.AppendEvent(ctx, ev)
	s.coord.Publish(op.ID, ev)
	s.coord.SignalTerminal(op.ID)
	return updated, errors.New(reason)
}

func (s *service) findActive(ctx context.Context, machineID string) (Operation, error) {
	// Repository implementations are intentionally narrow; the unique-index
	// error is resolved by querying known statuses through a small optional seam.
	if q, ok := s.repo.(interface {
		FindActive(context.Context, string) (Operation, error)
	}); ok {
		return q.FindActive(ctx, machineID)
	}
	return Operation{}, errors.New("active operation lookup unavailable")
}

func parseStatus(v string) (Status, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "queued":
		return StatusQueued, true
	case "planning":
		return StatusPlanning, true
	case "planned":
		return StatusPlanned, true
	case "confirmed":
		return StatusConfirmed, true
	case "applying":
		return StatusApplying, true
	case "completed":
		return StatusCompleted, true
	case "failed":
		return StatusFailed, true
	case "blocked":
		return StatusBlocked, true
	case "canceled", "cancelled":
		return StatusCanceled, true
	default:
		return StatusUnspecified, false
	}
}

func statusFromExit(code int32) Status {
	if code == 0 {
		return StatusCompleted
	}
	return StatusFailed
}
