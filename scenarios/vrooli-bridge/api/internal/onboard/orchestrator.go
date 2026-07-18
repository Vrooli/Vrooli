package onboard

import (
	"context"
	"fmt"
	"os"
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
		Host: in.Host, Port: in.Port, User: in.User, Password: in.Password, KeyName: in.SSHKeyName,
		ProvisionSudo: in.ProvisionSudo,
	})
	if err != nil {
		if s.cancelled(ctx) {
			s.finishCancelled(ctx, opID, &seq)
			return
		}
		s.emit(ctx, opID, &seq, StepSSHSetup, StepStatusFailed, err.Error())
		s.finishFailed(ctx, opID, &seq, FailureSSHSetup, 0, err.Error(), "")
		return
	}
	if s.linker != nil {
		if err := s.linker.RecordCorrelatedTrust(ctx, correlationForOp(ctx, s.repo, opID), conn); err != nil {
			s.finishFailed(ctx, opID, &seq, FailureSSHSetup, 0, "could not persist machine trust: "+err.Error(), "")
			return
		}
	}
	s.emit(ctx, opID, &seq, StepSSHSetup, StepStatusOK, sshSetupDetail(conn.SudoState))
	if s.cancelled(ctx) {
		s.finishCancelled(ctx, opID, &seq)
		return
	}

	// ---- Phase: CANDIDATE_ADMISSION ----
	// This must remain before every expensive or security-sensitive action: no
	// tree/script transfer, setup, pairing-code issue, or redeem is allowed until
	// the target itself proves it can reach the selected Bridge endpoint.
	s.emit(ctx, opID, &seq, StepAdmission, StepStatusStarted, "probing Bridge endpoint from candidate node")
	admission := s.admitCandidate(ctx, conn, in.ControlPlaneURL)
	if candidateIP := automaticFirewallCandidate(in.ReachabilityMode, admission); candidateIP != "" && s.firewallAdmitter != nil {
		s.emit(ctx, opID, &seq, StepAdmission, StepStatusStarted, "candidate source "+candidateIP+" is blocked; ensuring scoped Bridge firewall admission")
		firewall, firewallErr := s.firewallAdmitter.AllowCandidate(ctx, candidateIP)
		if firewallErr != nil || (firewall.Status != "changed" && firewall.Status != "already_present") || !firewall.Managed {
			detail := "automatic scoped firewall admission failed"
			if firewallErr != nil {
				detail += ": " + firewallErr.Error()
			} else if firewall.Code != "" {
				detail += ": " + firewall.Code
			} else if firewall.Status != "" {
				detail += ": " + firewall.Status
			}
			s.emit(ctx, opID, &seq, StepAdmission, StepStatusFailed, detail)
			s.finishFailed(ctx, opID, &seq, FailureControlPlaneUnreachable, 0, detail, "")
			return
		}
		s.emit(ctx, opID, &seq, StepAdmission, StepStatusStarted, "scoped Bridge firewall admission ready; retrying candidate probe")
		admission = s.admitCandidate(ctx, conn, in.ControlPlaneURL)
	}
	if admission.Category != AdmissionPassed {
		detail := admissionDetail(admission)
		s.emit(ctx, opID, &seq, StepAdmission, StepStatusFailed, detail)
		s.finishFailed(ctx, opID, &seq, admissionFailureReason(admission.Category), 0, detail, "")
		return
	}
	s.emit(ctx, opID, &seq, StepAdmission, StepStatusOK, admissionDetail(admission))

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
		s.finishFailed(ctx, opID, &seq, FailureScriptPush, 0, err.Error(), "")
		return
	}
	s.emit(ctx, opID, &seq, StepPushScript, StepStatusOK, "bootstrap script staged on node")

	// ---- Phase: SYNC_TREE (working-tree source mode only) ----
	// Ship the control plane's LOCAL tree to the node over the established SSH
	// channel so uncommitted work onboards without a commit or push. The bootstrap
	// then verifies the pre-synced tree instead of cloning. Pinned mode skips this
	// entirely (the node clones the pushed commit itself).
	var wtDigest, wtSourceDir string
	var remoteArtifacts RemoteArtifacts
	if in.WorkingTree() {
		s.emit(ctx, opID, &seq, StepSyncTree, StepStatusStarted, "shipping control-plane working tree to node")
		snap, snErr := s.worktree.Snapshot(ctx)
		if snErr != nil {
			if s.cancelled(ctx) {
				s.finishCancelled(ctx, opID, &seq)
				return
			}
			s.emit(ctx, opID, &seq, StepSyncTree, StepStatusFailed, snErr.Error())
			s.finishFailed(ctx, opID, &seq, FailureWorkingTreeSync, 0, "working-tree snapshot failed: "+snErr.Error(), "")
			return
		}
		syncRes, syErr := s.driver.SyncTree(ctx, SyncParams{
			Conn: conn, RepoDir: snap.RepoDir, Files: snap.Files, DestDir: trimField(in.CheckoutDir),
		})
		if syErr != nil {
			if s.cancelled(ctx) {
				s.finishCancelled(ctx, opID, &seq)
				return
			}
			s.emit(ctx, opID, &seq, StepSyncTree, StepStatusFailed, syErr.Error())
			s.finishFailed(ctx, opID, &seq, FailureWorkingTreeSync, 0, "working-tree ship failed: "+syErr.Error(), "")
			return
		}
		wtDigest = snap.Digest
		wtSourceDir = syncRes.ResolvedDestDir
		s.recordDigest(ctx, opID, wtDigest)
		s.emit(ctx, opID, &seq, StepSyncTree, StepStatusOK, fmt.Sprintf(
			"shipped %d file(s), %s, digest %s → %s", len(snap.Files), humanBytes(syncRes.BytesTransferred), shortDigest(wtDigest), wtSourceDir))
		if s.cancelled(ctx) {
			s.finishCancelled(ctx, opID, &seq)
			return
		}

		// The node needs no local Go toolchain: discover its one target, build all
		// executable inputs from the exact RepoDir just shipped, then transfer them
		// and their sidecars before bootstrap starts.
		s.emit(ctx, opID, &seq, StepPrebuiltArtifacts, StepStatusStarted, "cross-building prebuilt binaries for node")
		platform, pErr := s.driver.DetectPlatform(ctx, conn)
		if pErr != nil {
			s.emit(ctx, opID, &seq, StepPrebuiltArtifacts, StepStatusFailed, pErr.Error())
			s.finishFailed(ctx, opID, &seq, FailurePrebuiltArtifacts, 0, "node platform detection failed: "+pErr.Error(), "")
			return
		}
		built, bErr := s.artifacts.Build(ctx, ArtifactBuildParams{RepoDir: snap.RepoDir, Target: platform})
		if built.Directory != "" {
			defer os.RemoveAll(built.Directory)
		}
		if bErr != nil {
			s.emit(ctx, opID, &seq, StepPrebuiltArtifacts, StepStatusFailed, bErr.Error())
			s.finishFailed(ctx, opID, &seq, FailurePrebuiltArtifacts, 0, "control-plane cross-build failed: "+bErr.Error(), "")
			return
		}
		remoteArtifacts, bErr = s.driver.PushArtifacts(ctx, ArtifactPushParams{Conn: conn, Artifacts: built})
		if bErr != nil {
			s.emit(ctx, opID, &seq, StepPrebuiltArtifacts, StepStatusFailed, bErr.Error())
			s.finishFailed(ctx, opID, &seq, FailurePrebuiltArtifacts, 0, "prebuilt artifact transfer failed: "+bErr.Error(), "")
			return
		}
		s.emit(ctx, opID, &seq, StepPrebuiltArtifacts, StepStatusOK, fmt.Sprintf(
			"received prebuilt binaries for %s/%s (fingerprint %s)", platform.OS, platform.Arch, shortDigest(built.Fingerprint)))
	}

	// ---- Issue the server-side pairing code (operator never handles one) ----
	correlationID := opID
	if op, getErr := s.repo.Get(ctx, opID); getErr == nil && op.CorrelationID != "" {
		correlationID = op.CorrelationID
	}
	code, err := s.issuer.Issue(ctx, IssueParams{NodeName: in.NodeName, Scopes: nil, CorrelationID: correlationID})
	if err != nil {
		s.finishFailed(ctx, opID, &seq, FailurePairingIssue, 0, "could not issue pairing code", "")
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
		s.handleMarker(ctx, opID, &seq, m)
	}
	res, runErr := s.driver.RunBootstrap(ctx, RunParams{
		Conn: conn, RemotePath: remotePath, Args: buildBootstrapArgs(in, wtSourceDir, wtDigest, remoteArtifacts), PairingCode: code,
	}, onMarker)
	// The code has served its one purpose — destroy our copy immediately.
	zeroBytes(code)
	// Pairing may have completed before a later bootstrap/service-install failure.
	// Resolve the durable correlation before interpreting that terminal process
	// result so the recovery record never depends on free-text stdout markers.
	if s.resolver == nil {
		s.finishFailed(ctx, opID, &seq, FailurePairing, int32(res.ExitCode), "durable pairing result resolver is not configured", res.Diagnostics)
		return
	}
	resolved, paired, resolveErr := s.resolver.ResolveEnrollment(ctx, correlationID)
	if resolveErr != nil {
		s.finishFailed(ctx, opID, &seq, FailurePairing, int32(res.ExitCode), "could not resolve durable pairing result: "+resolveErr.Error(), res.Diagnostics)
		return
	}
	if paired {
		nodeID = resolved
		s.recordNodeID(ctx, opID, nodeID)
		if s.linker != nil {
			if err := s.linker.LinkCorrelatedNode(ctx, correlationID, nodeID); err != nil {
				s.finishFailed(ctx, opID, &seq, FailurePairing, int32(res.ExitCode), "could not persist machine node lineage: "+err.Error(), res.Diagnostics)
				return
			}
		}
	}

	if runErr != nil {
		if s.cancelled(ctx) {
			s.finishCancelled(ctx, opID, &seq)
			return
		}
		s.finishFailed(ctx, opID, &seq, FailureBootstrap, int32(res.ExitCode), "bootstrap transport error: "+runErr.Error(), res.Diagnostics)
		return
	}
	if reason, ok := failureForExit(res.ExitCode); ok {
		s.finishFailed(ctx, opID, &seq, reason, int32(res.ExitCode), fmt.Sprintf("bootstrap exited %d", res.ExitCode), res.Diagnostics)
		return
	}
	if s.cancelled(ctx) {
		s.finishCancelled(ctx, opID, &seq)
		return
	}

	// ---- Phase: VERIFYING ----
	if nodeID == "" {
		s.finishFailed(ctx, opID, &seq, FailureVerifyOnline, int32(res.ExitCode), "bootstrap succeeded without a durable pairing result", res.Diagnostics)
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
		s.finishFailed(ctx, opID, &seq, FailureVerifyOnline, int32(res.ExitCode), detail, res.Diagnostics)
		return
	}
	s.emit(ctx, opID, &seq, StepVerifyOnline, StepStatusOK, "node is online with control-plane key pinned")
	s.recordNodeRevision(ctx, opID, &seq, nodeID, in)
	s.finishSucceeded(ctx, opID, nodeID, int32(res.ExitCode))
}

func correlationForOp(ctx context.Context, repo Repository, opID string) string {
	op, err := repo.Get(ctx, opID)
	if err == nil && op.CorrelationID != "" {
		return op.CorrelationID
	}
	return opID
}

// recordNodeRevision stamps the node record with the provenance the op brought it
// to — the pinned commit, or a "<base>+dirty" marker for a working-tree node — so
// `nodes list`, node detail, and the fleet UI show a dirty node as visibly not a
// pinned one. Best-effort: the node is already paired and ONLINE, so a recording
// failure is surfaced as a non-fatal note, never a failed op.
func (s *service) recordNodeRevision(ctx context.Context, opID string, seq *uint64, nodeID string, in StartInput) {
	if s.nodeRev == nil || nodeID == "" {
		return
	}
	rev := trimField(in.TargetRevision)
	if in.WorkingTree() {
		rev = workingTreeRevision(in.BaseRevision)
	}
	if rev == "" {
		return
	}
	if err := s.nodeRev.RecordRevision(ctx, nodeID, rev); err != nil {
		s.emit(ctx, opID, seq, StepVerifyOnline, StepStatusOK, "node online; could not stamp provenance revision ("+err.Error()+")")
	}
}

// sshSetupDetail renders the ssh-setup OK step detail, naming the passwordless-
// sudo provisioning outcome when one was reported (never the password). An empty
// state (sudo not requested for this driver) collapses to the plain message.
func sshSetupDetail(sudoState string) string {
	if sudoState == "" {
		return "passwordless SSH established"
	}
	return "passwordless SSH established; sudo: " + sudoState
}

// handleMarker persists human/bootstrap diagnostics only. Pairing identity is
// resolved through the durable correlation, never parsed from marker text.
func (s *service) handleMarker(ctx context.Context, opID string, seq *uint64, m Marker) {
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
// pairing code is deliberately NOT among them — it rides stdin (env-only). In
// working-tree mode wtSourceDir/wtDigest are set (the node-side tree the
// orchestrator pre-shipped and its content digest); the script then verifies that
// tree instead of cloning, and keys its setup sentinel on the digest.
func buildBootstrapArgs(in StartInput, wtSourceDir, wtDigest string, artifacts RemoteArtifacts) []string {
	args := []string{"--control-plane-url", in.ControlPlaneURL, "--revision", in.TargetRevision}
	if wtSourceDir != "" {
		// Working-tree mode: point the script at the pre-synced tree and give it the
		// content digest so a re-ship of changed work re-runs node-side setup.
		args = append(args, "--source-dir", wtSourceDir)
		if wtDigest != "" {
			args = append(args, "--source-digest", wtDigest)
		}
	}
	if artifacts.Vrooli != "" {
		args = append(args, "--vrooli-bin", artifacts.Vrooli)
	}
	if artifacts.BridgeCLI != "" {
		args = append(args, "--bridge-cli", artifacts.BridgeCLI)
	}
	if artifacts.Agent != "" {
		args = append(args, "--agent-bin", artifacts.Agent)
	}
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
	// Setup profile — the values are metachar-validated at Start (validateSetupProfile),
	// so they are safe to pass as bootstrap flags; the script quotes them defensively
	// and splices them into `make setup SETUP_ARGS=…`.
	if env := trimField(in.SetupEnvironment); env != "" {
		args = append(args, "--setup-environment", env)
	}
	if res := trimField(in.SetupResources); res != "" {
		args = append(args, "--setup-resources", res)
	}
	if scn := trimField(in.SetupScenarios); scn != "" {
		args = append(args, "--setup-scenarios", scn)
	}
	if in.IncludeOptional {
		args = append(args, "--include-optional")
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

// recordDigest persists the working-tree content digest on the op (working-tree
// mode) so the op's provenance is complete before the bootstrap runs.
func (s *service) recordDigest(ctx context.Context, opID, digest string) {
	op, err := s.repo.Get(ctx, opID)
	if err != nil {
		return
	}
	op.WorkingTreeDigest = digest
	_, _ = s.repo.Update(ctx, op)
}

// shortDigest renders the first 12 hex chars of a content digest for a step
// detail (the full digest lives on the op record).
func shortDigest(d string) string {
	if len(d) <= 12 {
		return d
	}
	return d[:12]
}

// humanBytes renders a transfer size in the largest sensible unit for a step
// detail, e.g. "4.2 MiB". It is display-only.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
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
	s.finish(ctx, opID, StateSucceeded, "", exitCode, nodeID, "")
}

// finishFailed drives an op to FAILED. detail is a single-line step-event note
// (the taxonomy in human words); diagnostics is the bounded, multi-line node-side
// output tail (empty for control-plane-side failures that never ran the remote
// script) persisted on the op so the operator sees the concrete cause, not just
// the reason code.
func (s *service) finishFailed(ctx context.Context, opID string, seq *uint64, reason FailureReason, exitCode int32, detail, diagnostics string) {
	if detail != "" {
		s.emit(ctx, opID, seq, StepRun, StepStatusFailed, detail)
	}
	s.finish(ctx, opID, StateFailed, reason, exitCode, "", diagnostics)
}

func (s *service) finishCancelled(ctx context.Context, opID string, seq *uint64) {
	s.emit(ctx, opID, seq, StepRun, StepStatusFailed, "onboarding cancelled by operator")
	s.finish(ctx, opID, StateCancelled, "", 0, "", "")
}

// finish drives an op to a terminal state and wakes block-once waiters. It never
// re-terminalises an already-terminal op.
func (s *service) finish(ctx context.Context, opID string, state State, reason FailureReason, exitCode int32, nodeID, diagnostics string) {
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
	// Only a FAILED op carries diagnostics; a late/empty tail never clobbers a
	// reason already recorded.
	if diagnostics != "" {
		op.FailureDetail = diagnostics
	}
	if _, err := s.repo.Update(ctx, op); err != nil {
		return
	}
	if op.CorrelationID != "" {
		if attempts, ok := s.repo.(AttemptStore); ok {
			if attempt, lookupErr := attempts.GetAttemptByCorrelation(ctx, op.CorrelationID); lookupErr == nil {
				attemptState := AttemptFailed
				if state == StateSucceeded {
					attemptState = AttemptSucceeded
				} else if state == StateCancelled {
					attemptState = AttemptInterrupted
				}
				result := string(reason)
				if state == StateSucceeded {
					result = "enrolled"
				}
				_, _ = attempts.CompleteAttempt(ctx, attempt.ID, attemptState, result, diagnostics)
			}
		}
	}
	s.coord.signalTerminal(opID)
}
