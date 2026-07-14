package onboard

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// runOnboarding is the server-owned orchestration for one op, driven on a
// detached, op-scoped context so it survives client disconnect. It is the SOLE
// writer of the op's state after Start, so every transition is a plain
// read-modify-write with no locking beyond the repository. Cancel cancels this
// context; the goroutine observes it at each phase boundary and drives the op to
// CANCELLED. Every phase and every bootstrap marker is persisted as an
// append-only step event, so a re-attaching client reads the exact progress.
//
// SECRETS: in.Password rides into ssh.FirstTouch (which zeroes it); the pairing
// code is issued here, injected over stdin by the driver, and zeroed the instant
// RunBootstrap returns. Neither is ever written to the op row, an event, or a
// log.
func (s *service) runOnboarding(ctx context.Context, opID string, in StartInput) {
	defer s.coord.releaseOp(opID)

	var seq uint64

	// ---- Phase: SSH_SETUP ----
	s.transition(ctx, opID, StateSSHSetup)
	s.emit(ctx, opID, &seq, StepSSHSetup, StepStatusStarted, "establishing passwordless SSH")
	conn, err := s.driver.FirstTouch(ctx, FirstTouchParams{
		Host: in.Host, Port: in.Port, User: in.User, Password: in.Password,
	})
	if err != nil {
		if s.cancelled(ctx) {
			s.finishCancelled(ctx, opID, &seq)
			return
		}
		s.emit(ctx, opID, &seq, StepSSHSetup, StepStatusFailed, err.Error())
		s.finishFailed(ctx, opID, &seq, FailureSSHSetup, 0, err.Error())
		return
	}
	s.emit(ctx, opID, &seq, StepSSHSetup, StepStatusOK, "passwordless SSH established")
	if s.cancelled(ctx) {
		s.finishCancelled(ctx, opID, &seq)
		return
	}

	// ---- Phase: PUSHING_SCRIPT ----
	s.transition(ctx, opID, StatePushingScript)
	s.emit(ctx, opID, &seq, StepPushScript, StepStatusStarted, "copying bootstrap script to node")
	remotePath, err := s.driver.PushScript(ctx, conn)
	if err != nil {
		if s.cancelled(ctx) {
			s.finishCancelled(ctx, opID, &seq)
			return
		}
		s.emit(ctx, opID, &seq, StepPushScript, StepStatusFailed, err.Error())
		s.finishFailed(ctx, opID, &seq, FailureScriptPush, 0, err.Error())
		return
	}
	s.emit(ctx, opID, &seq, StepPushScript, StepStatusOK, "bootstrap script staged on node")

	// ---- Issue the server-side pairing code (operator never handles one) ----
	code, err := s.issuer.Issue(ctx, IssueParams{NodeName: in.NodeName, Scopes: nil})
	if err != nil {
		s.finishFailed(ctx, opID, &seq, FailurePairingIssue, 0, "could not issue pairing code")
		return
	}
	if s.cancelled(ctx) {
		zeroBytes(code)
		s.finishCancelled(ctx, opID, &seq)
		return
	}

	// ---- Phase: BOOTSTRAPPING ----
	s.transition(ctx, opID, StateBootstrapping)
	var nodeID string
	onMarker := func(m Marker) {
		s.handleMarker(ctx, opID, &seq, m, &nodeID)
	}
	res, runErr := s.driver.RunBootstrap(ctx, RunParams{
		Conn: conn, RemotePath: remotePath, Args: buildBootstrapArgs(in), PairingCode: code,
	}, onMarker)
	// The code has served its one purpose — destroy our copy immediately.
	zeroBytes(code)

	if runErr != nil {
		if s.cancelled(ctx) {
			s.finishCancelled(ctx, opID, &seq)
			return
		}
		s.finishFailed(ctx, opID, &seq, FailureBootstrap, int32(res.ExitCode), "bootstrap transport error: "+runErr.Error())
		return
	}
	if reason, ok := failureForExit(res.ExitCode); ok {
		s.finishFailed(ctx, opID, &seq, reason, int32(res.ExitCode), fmt.Sprintf("bootstrap exited %d", res.ExitCode))
		return
	}
	if s.cancelled(ctx) {
		s.finishCancelled(ctx, opID, &seq)
		return
	}

	// ---- Phase: VERIFYING ----
	if nodeID == "" {
		s.finishFailed(ctx, opID, &seq, FailureVerifyOnline, int32(res.ExitCode), "bootstrap succeeded but no node id was observed in its markers")
		return
	}
	s.recordNodeID(ctx, opID, nodeID)
	s.transition(ctx, opID, StateVerifying)
	s.emit(ctx, opID, &seq, StepVerifyOnline, StepStatusStarted, "confirming node is online in the fleet")
	online, cErr := s.confirmer.ConfirmOnline(ctx, nodeID, verifyTimeout(in))
	if cErr != nil || !online {
		detail := "node did not come online within the verification budget"
		if cErr != nil {
			detail = "online confirmation failed: " + cErr.Error()
		}
		s.emit(ctx, opID, &seq, StepVerifyOnline, StepStatusFailed, detail)
		s.finishFailed(ctx, opID, &seq, FailureVerifyOnline, int32(res.ExitCode), detail)
		return
	}
	s.emit(ctx, opID, &seq, StepVerifyOnline, StepStatusOK, "node is online with control-plane key pinned")
	s.finishSucceeded(ctx, opID, nodeID, int32(res.ExitCode))
}

// handleMarker persists one parsed bootstrap marker as a step event and captures
// the node id the first time it appears in a marker detail.
func (s *service) handleMarker(ctx context.Context, opID string, seq *uint64, m Marker, nodeID *string) {
	if *nodeID == "" {
		if id := extractNodeID(m.Detail); id != "" {
			*nodeID = id
		}
	}
	switch m.Event {
	case eventRunStart:
		s.emit(ctx, opID, seq, StepRun, StepStatusStarted, m.Detail)
	case eventRunOK:
		s.emit(ctx, opID, seq, StepRun, StepStatusOK, m.Detail)
	case eventRunFail:
		s.emit(ctx, opID, seq, StepRun, StepStatusFailed, m.Detail)
	default:
		if st := stepStatusForEvent(m.Event); st != StepStatusUnspecified {
			s.emit(ctx, opID, seq, m.Step, st, m.Detail)
		}
	}
}

// buildBootstrapArgs assembles the bootstrap flags from a validated input. The
// pairing code is deliberately NOT among them — it rides stdin (env-only).
func buildBootstrapArgs(in StartInput) []string {
	args := []string{"--control-plane-url", in.ControlPlaneURL, "--revision", in.TargetRevision}
	if nn := trimField(in.NodeName); nn != "" {
		args = append(args, "--node-name", nn)
	}
	if ru := trimField(in.RepoURL); ru != "" {
		args = append(args, "--repo-url", ru)
	}
	if cd := trimField(in.CheckoutDir); cd != "" {
		args = append(args, "--checkout-dir", cd)
	}
	if in.VerifyTimeoutSeconds > 0 {
		args = append(args, "--verify-timeout", strconv.Itoa(int(in.VerifyTimeoutSeconds)))
	}
	if caps := joinCapabilities(in.Capabilities); caps != "" {
		args = append(args, "--capabilities", caps)
	}
	if in.SkipSetup {
		args = append(args, "--skip-setup")
	}
	if in.SkipPrereqs {
		args = append(args, "--skip-prereqs")
	}
	return args
}

func joinCapabilities(caps []string) string {
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if v := trimField(c); v != "" {
			out = append(out, v)
		}
	}
	return strings.Join(out, ",")
}

// failureForExit maps a bootstrap exit code to a terminal failure reason. ok is
// false for exit 0 (success — no failure).
func failureForExit(code int) (FailureReason, bool) {
	switch code {
	case 0:
		return "", false
	case 2:
		return FailureBootstrapUsage, true
	case 3:
		return FailureUnsupportedPlatform, true
	case 4:
		return FailurePairing, true
	default:
		return FailureBootstrap, true
	}
}

func verifyTimeout(in StartInput) time.Duration {
	if in.VerifyTimeoutSeconds > 0 {
		return time.Duration(in.VerifyTimeoutSeconds) * time.Second
	}
	return DefaultVerifyTimeout
}

// cancelled reports whether the op-scoped context has been cancelled (operator
// CancelOnboarding).
func (s *service) cancelled(ctx context.Context) bool { return ctx.Err() != nil }

// ---- persistence helpers (single-writer, plain read-modify-write) ----

// transition sets the op's state, stamping StartedAt on the first move off
// PENDING. It never regresses a terminal op.
func (s *service) transition(ctx context.Context, opID string, state State) {
	op, err := s.repo.Get(ctx, opID)
	if err != nil {
		return
	}
	if op.State.Terminal() {
		return
	}
	if op.StartedAt.IsZero() {
		op.StartedAt = s.clock.Now().UTC()
	}
	op.State = state
	_, _ = s.repo.Update(ctx, op)
}

// recordNodeID persists the learned fleet node id on the op.
func (s *service) recordNodeID(ctx context.Context, opID, nodeID string) {
	op, err := s.repo.Get(ctx, opID)
	if err != nil {
		return
	}
	op.NodeID = nodeID
	_, _ = s.repo.Update(ctx, op)
}

// emit appends one step event (assigning the next per-op sequence) and fans it
// out to live subscribers.
func (s *service) emit(ctx context.Context, opID string, seq *uint64, stepID string, status StepStatus, detail string) {
	*seq++
	ev := StepEvent{
		OpID:      opID,
		Sequence:  *seq,
		StepID:    stepID,
		Status:    status,
		Detail:    detail,
		EmittedAt: s.clock.Now().UTC(),
	}
	_ = s.repo.AppendEvent(ctx, ev)
	s.coord.publish(ev)
}

// appendEvent appends one step event computing the next sequence from the
// persisted history. Used off the orchestration path (ResumeInterrupted), where
// no in-memory sequence counter exists.
func (s *service) appendEvent(ctx context.Context, opID, stepID string, status StepStatus, detail string) {
	var next uint64 = 1
	if existing, err := s.repo.ListEvents(ctx, opID); err == nil {
		for _, ev := range existing {
			if ev.Sequence >= next {
				next = ev.Sequence + 1
			}
		}
	}
	ev := StepEvent{
		OpID:      opID,
		Sequence:  next,
		StepID:    stepID,
		Status:    status,
		Detail:    detail,
		EmittedAt: s.clock.Now().UTC(),
	}
	_ = s.repo.AppendEvent(ctx, ev)
	s.coord.publish(ev)
}

func (s *service) finishSucceeded(ctx context.Context, opID, nodeID string, exitCode int32) {
	s.finish(ctx, opID, StateSucceeded, "", exitCode, nodeID)
}

func (s *service) finishFailed(ctx context.Context, opID string, seq *uint64, reason FailureReason, exitCode int32, detail string) {
	if detail != "" {
		s.emit(ctx, opID, seq, StepRun, StepStatusFailed, detail)
	}
	s.finish(ctx, opID, StateFailed, reason, exitCode, "")
}

func (s *service) finishCancelled(ctx context.Context, opID string, seq *uint64) {
	s.emit(ctx, opID, seq, StepRun, StepStatusFailed, "onboarding cancelled by operator")
	s.finish(ctx, opID, StateCancelled, "", 0, "")
}

// finish drives an op to a terminal state and wakes block-once waiters. It never
// re-terminalises an already-terminal op.
func (s *service) finish(ctx context.Context, opID string, state State, reason FailureReason, exitCode int32, nodeID string) {
	op, err := s.repo.Get(ctx, opID)
	if err != nil {
		return
	}
	if op.State.Terminal() {
		return
	}
	now := s.clock.Now().UTC()
	if op.StartedAt.IsZero() {
		op.StartedAt = now
	}
	op.State = state
	op.FailureReason = reason
	if exitCode != 0 {
		op.ExitCode = exitCode
	}
	if nodeID != "" {
		op.NodeID = nodeID
	}
	op.FinishedAt = now
	if _, err := s.repo.Update(ctx, op); err != nil {
		return
	}
	s.coord.signalTerminal(opID)
}
