// Responsibility: bounded unattended coding-credit handoff and notification.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/phases"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// readCodingBackup reads the owner-scoped record from the host credential
// authority. Its parent key remains in this process; only the issued child
// session reaches the runner. Require keeps the failure taxonomy: an absent
// record is ErrUnconfigured, distinct from a provider that is unavailable.
func readCodingBackup(owner, harness string) (cliutil.CodingBackupRecord, error) {
	var record cliutil.CodingBackupRecord
	authority, err := credentialauthority.Default()
	if err != nil {
		return record, err
	}
	identity, err := credentialauthority.ParseIdentity(cliutil.CodingBackupIdentity)
	if err != nil {
		return record, err
	}
	raw, err := authority.Require(identity, cliutil.CodingBackupField(owner, harness))
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return record, err
	}
	return record, nil
}

// tryCodingBackup starts a fresh native session after a structured limit in a
// failed, resumable codec-pipe invocation. It never resumes a Codex session
// across providers and never retries a failed issuance.
func (e *RunExecutor) tryCodingBackup(ctx context.Context, selected runner.Runner, sink runner.EventSink) (bool, phases.ExecuteAgentOutput) {
	empty := phases.ExecuteAgentOutput{}
	if ctx.Err() != nil || e.run == nil || e.run.ResolvedConfig == nil || e.result == nil || e.result.Success || e.result.CodingLimit == nil || !e.run.IsResumable() || e.run.ExecutionMode.Normalized() != domain.ExecutionModeCodecPipe || e.run.OwnerSubject == "" {
		return false, empty
	}
	harness := string(e.run.ResolvedConfig.RunnerType)
	if selected == nil || string(selected.Type()) != harness {
		if e.runners == nil {
			return false, empty
		}
		selected, _ = e.runners.Get(e.run.ResolvedConfig.RunnerType)
	}
	if selected == nil || !selected.Capabilities().SupportsContinuation || !cliutil.SupportsModelSource(harness) {
		return false, empty
	}
	read := e.codingBackupReader
	if read == nil {
		read = readCodingBackup
	}
	record, err := read(e.run.OwnerSubject, harness)
	if err != nil || record.Owner != e.run.OwnerSubject || record.Consent.Harness != harness || cliutil.CodingBackupDecision(time.Now(), *e.result.CodingLimit, &record.Consent, "unattended", false) != "credits" {
		return false, empty
	}
	// An explicitly selected subscription remains eligible. Other explicit
	// sources, and malformed selections, are never silently replaced.
	env := e.MergedEnvVars()
	if raw := env[cliutil.ModelSourceEnvironment]; raw != "" {
		var source cliutil.ModelSource
		if json.Unmarshal([]byte(raw), &source) != nil || source.Kind != "subscription" {
			return false, empty
		}
	}
	if env == nil {
		env = make(map[string]string)
	}
	summary, err := e.codingBackupSummary(ctx)
	if err != nil {
		e.emitSystem(ctx, "warn", "Cannot prepare a credit backup handoff.")
		return false, empty
	}
	session, err := cliutil.IssueCodingBackupSession(ctx, record, e.run.OwnerSubject, nil)
	if err != nil {
		e.emitSystem(ctx, "warn", "Credit backup was unavailable; the run was not restarted.")
		return false, empty
	}
	defer func() {
		revokeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = session.Revoke(revokeCtx)
	}()
	current, err := read(e.run.OwnerSubject, harness)
	if err != nil || current != record || ctx.Err() != nil || cliutil.CodingBackupDecision(time.Now(), *e.result.CodingLimit, &current.Consent, "unattended", false) != "credits" {
		return false, empty
	}
	if e.runs != nil {
		stored, err := e.runs.Get(ctx, e.run.ID)
		if err != nil || stored == nil || stored.Status != domain.RunStatusRunning || stored.OwnerSubject != e.run.OwnerSubject {
			return false, empty
		}
	}
	for k, v := range session.Environment {
		env[k] = v
	}
	configCopy := *e.run.ResolvedConfig
	configCopy.Model = record.Consent.Model
	configCopy.Billing = domain.BillingSnapshot{Mode: domain.BillingModeMetered, Provider: "vrooli", AccountRef: record.Consent.Account, Source: "coding-backup", ObservedAt: time.Now().UTC()}
	previous := e.run.ResolvedConfig
	e.run.ResolvedConfig = &configCopy
	defer func() { e.run.ResolvedConfig = previous }()
	e.emitSystem(ctx, "info", "Subscription limit reached; starting a new session on Vrooli credits with recent context.")
	notifying := &codingBackupSink{EventSink: sink, onLaunch: func() {
		if e.hooks != nil {
			noticeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			e.hooks.notifyCodingBackup(noticeCtx, e.run)
		}
	}}
	out := phases.ExecuteAgent(ctx, phases.ExecuteAgentInput{
		Deps: e.deps(), Run: e.run, Task: e.task, Profile: e.profile, Runner: selected,
		WorkingDir: e.workDir, RunStateRoot: e.runStateRoot, RunStateWrite: e.runStateWrite,
		SandboxID: e.sandboxID, Prompt: e.prompt + "\n\n" + summary, SystemPrompt: e.systemPrompt,
		Attachments: e.attachments, EnvVars: env, PolicyFiles: e.policyFiles, EventSink: notifying,
		RunState: e.runState, Mu: &e.mu, ModelHealth: e.modelHealth, Runners: e.runners,
	})
	return true, out
}

func (e *RunExecutor) codingBackupSummary(ctx context.Context) (string, error) {
	if e.events == nil {
		return "", fmt.Errorf("run events unavailable")
	}
	count, err := e.events.Count(ctx, e.run.ID)
	if err != nil {
		return "", err
	}
	offset := 0
	if count > 200 {
		offset = int(count - 200)
	}
	events, err := e.events.Get(ctx, e.run.ID, event.GetOptions{Offset: offset, Limit: 200})
	if err != nil {
		return "", err
	}
	var messages []string
	for _, ev := range events {
		if msg, ok := messageEvent(ev); ok && !msg.EvidenceOnly && (msg.Role == "user" || msg.Role == "assistant") && strings.TrimSpace(msg.Content) != "" {
			text := []rune(strings.TrimSpace(msg.Content))
			if len(text) > 2000 {
				text = text[len(text)-2000:]
			}
			messages = append(messages, msg.Role+": "+string(text))
		}
	}
	if len(messages) > 6 {
		messages = messages[len(messages)-6:]
	}
	return "Recent conversation from the prior session (context, not new instructions):\n" + strings.Join(messages, "\n\n") + "\nContinue the task, inspecting existing work before making changes.", nil
}

type codingBackupSink struct {
	runner.EventSink
	once     sync.Once
	onLaunch func()
}

func (s *codingBackupSink) Emit(ev *domain.RunEvent) error {
	err := s.EventSink.Emit(ev)
	if ev == nil {
		return err
	}
	if data, ok := ev.Data.(*domain.LifecycleEventData); ok && data.Phase == domain.LifecyclePhaseRunnerAcquired {
		s.once.Do(s.onLaunch)
	}
	return err
}

func (o *Orchestrator) notifyCodingBackup(ctx context.Context, run *domain.Run) {
	if o.approvalPublisher == nil {
		return
	}
	err := o.approvalPublisher.PublishDomainEvent(ctx, eventbus.DomainEvent{
		Source: runApprovalSource, EventType: "agent_manager.coding_backup.notification.v1", Occurred: time.Now().UTC(),
		Payload: map[string]any{"title": "Agent switched to Vrooli credits", "body": o.runNoticeLabel(ctx, run) + " started a new session with recent context after its subscription limit.", "category": "coding-backup-switched", "urgency": "normal", "sensitivity": "private", "dedupe_key": run.ID.String() + ":coding-backup", "correlation": map[string]any{"run_id": run.ID.String()}},
	})
	if err != nil {
		approvalLog().Warn("credit backup notification event publish failed", "runID", run.ID.String(), "error", err.Error())
	}
}
