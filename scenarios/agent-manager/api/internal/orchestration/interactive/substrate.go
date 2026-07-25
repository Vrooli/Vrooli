package interactive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

const (
	// defaultDiscoveryTimeout bounds how long Launch waits for the agent's first
	// transcript to appear after the session is created. It must cover a real
	// interactive CLI's cold boot (node startup, sandbox/bubblewrap init, TUI
	// render) plus the first prompt landing and starting a turn — measured at up
	// to ~30s on a busy host, so 60s leaves margin (and, with the re-delivery
	// interval below, more attempts for a paste to land on the ready input).
	defaultDiscoveryTimeout = 60 * time.Second
	defaultPollInterval     = 250 * time.Millisecond
	defaultStopGrace        = 750 * time.Millisecond
	// defaultPromptBootDelay is the beat we wait after the session is created
	// (which launches the CLI) before pasting the run's initial prompt. A paste
	// into a not-yet-rendered TUI can be silently dropped, so we let claude/codex
	// draw their first frame first. Bounded and best-effort — the resend below is
	// the real safety net.
	defaultPromptBootDelay = 2 * time.Second
	// defaultPromptResendAfter is the interval discovery waits, with no transcript
	// yet, before re-delivering the prompt. A freshly launched CLI writes no
	// transcript until its first turn starts, so transcript appearance is the
	// natural ack that the prompt landed. Re-delivery (bounded by the discovery
	// timeout) recovers a paste dropped by a slow boot AND clears a one-shot
	// in-band gate that swallows a delivery — notably claude's first-run
	// "trust this folder?" dialog: the earlier paste accepts the dialog, a later
	// one lands the prompt. Kept comfortably above turn-start latency so a
	// delivered prompt is detected before the next re-delivery.
	defaultPromptResendAfter = 5 * time.Second
)

// LaunchInfoResolver resolves the per-agent launch facts (tag env key, binary
// path) for a runner type. A registry-backed resolver ([RegistryLaunchInfo]) is
// used in production; tests supply a trivial one.
type LaunchInfoResolver func(domain.RunnerType) (runner.AgentLaunchInfo, error)

// RegistryLaunchInfo adapts a runner.Registry to a LaunchInfoResolver, resolving
// the runner and asserting it exposes runner.AgentLaunchInfo — the same
// resolve-then-assert pattern transcript recovery uses for TranscriptParser.
func RegistryLaunchInfo(reg runner.Registry) LaunchInfoResolver {
	return func(rt domain.RunnerType) (runner.AgentLaunchInfo, error) {
		if reg == nil {
			return nil, fmt.Errorf("runner registry is not configured")
		}
		r, err := reg.Get(rt)
		if err != nil {
			return nil, err
		}
		info, ok := r.(runner.AgentLaunchInfo)
		if !ok {
			return nil, fmt.Errorf("runner %s does not expose interactive launch info", rt)
		}
		return info, nil
	}
}

// Substrate creates, launches, and stops interactive web-console sessions for a
// run. It owns the session lifecycle and the launch-command + transcript-
// discovery contract; it does not tail the transcript or interpret events —
// that is Phase 3/4, which build on the TranscriptPath this substrate resolves.
type Substrate struct {
	sessions   webconsole.SessionController
	launchInfo LaunchInfoResolver

	// homeDir overrides the user home for claude transcript discovery (tests).
	homeDir string

	discoveryTimeout  time.Duration
	pollInterval      time.Duration
	stopGrace         time.Duration
	promptBootDelay   time.Duration
	promptResendAfter time.Duration
	now               func() time.Time
}

// Option configures a Substrate.
type Option func(*Substrate)

// WithHomeDir overrides the home directory used for claude transcript discovery.
func WithHomeDir(dir string) Option { return func(s *Substrate) { s.homeDir = dir } }

// WithDiscoveryTimeout bounds how long Launch waits for the transcript to appear.
func WithDiscoveryTimeout(d time.Duration) Option {
	return func(s *Substrate) {
		if d > 0 {
			s.discoveryTimeout = d
		}
	}
}

// WithPollInterval sets the transcript-discovery poll cadence.
func WithPollInterval(d time.Duration) Option {
	return func(s *Substrate) {
		if d > 0 {
			s.pollInterval = d
		}
	}
}

// WithStopGrace sets the delay between the interrupt sequence and the delete
// fallback in Stop.
func WithStopGrace(d time.Duration) Option {
	return func(s *Substrate) {
		if d >= 0 {
			s.stopGrace = d
		}
	}
}

// WithPromptBootDelay sets the delay between session creation and pasting the
// run's initial prompt (giving the CLI's TUI a beat to render). Zero disables
// the delay (used by tests and by the no-prompt path).
func WithPromptBootDelay(d time.Duration) Option {
	return func(s *Substrate) {
		if d >= 0 {
			s.promptBootDelay = d
		}
	}
}

// WithPromptResendAfter sets how long discovery waits with no transcript before
// resending the initial prompt once.
func WithPromptResendAfter(d time.Duration) Option {
	return func(s *Substrate) {
		if d > 0 {
			s.promptResendAfter = d
		}
	}
}

// WithClock overrides the clock (tests).
func WithClock(now func() time.Time) Option {
	return func(s *Substrate) {
		if now != nil {
			s.now = now
		}
	}
}

// NewSubstrate builds a Substrate from a web-console session controller and a
// launch-info resolver.
func NewSubstrate(sessions webconsole.SessionController, launchInfo LaunchInfoResolver, opts ...Option) *Substrate {
	s := &Substrate{
		sessions:          sessions,
		launchInfo:        launchInfo,
		discoveryTimeout:  defaultDiscoveryTimeout,
		pollInterval:      defaultPollInterval,
		stopGrace:         defaultStopGrace,
		promptBootDelay:   defaultPromptBootDelay,
		promptResendAfter: defaultPromptResendAfter,
		now:               time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// LaunchParams is the input to Launch.
type LaunchParams struct {
	RunID      uuid.UUID
	RunnerType domain.RunnerType
	Tag        string
	WorkingDir string
	// RunDir is the agent-manager-owned per-run directory. codex/grok session
	// homes are created under it; claude ignores it.
	RunDir string
	// DisplayLabel is the web-console sidebar label; defaults to the tag.
	DisplayLabel string
	// Prompt is the run's initial task prompt, typed into the freshly launched
	// CLI to start the first turn (locked decision 4 — the prompt is typed into
	// the session, not passed on the command line). Empty means no prompt is
	// delivered (the launch just creates the session and discovers the transcript
	// — used by the fake-CLI integration harnesses that self-start a turn).
	Prompt string
	Model  string
	Effort domain.Effort
}

// LaunchResult is the durable outcome of a launch.
type LaunchResult struct {
	SessionID      string
	TranscriptPath string
	LaunchCommand  string
	ExecutionMode  domain.ExecutionMode
}

// Launch creates a web-console session running the real interactive agent CLI,
// delivers the run's initial task prompt into it, and resolves the agent-owned
// transcript path. The flow (design §1 + §4 addendum) is: resolve launch info →
// build env-scoped launch command → create session with execute_launch_command
// → paste the initial prompt (after a boot beat) → discover the transcript the
// first turn produces.
//
// Prompt delivery MUST precede discovery: a freshly launched claude/codex TUI
// sits idle and writes no transcript until its first turn starts, so the paste
// is what makes the transcript appear. Transcript appearance is therefore the
// natural ack — while none shows, discovery re-delivers the prompt every
// promptResendAfter (bounded by the discovery timeout), recovering a paste lost
// to a slow boot and clearing a one-shot in-band gate that swallows a delivery
// (claude's first-run trust dialog), before failing with a clear error.
//
// The returned LaunchResult always carries the created SessionID (once the
// session exists) even when prompt delivery or transcript discovery fails, so
// the caller can persist the session id and still tear the session down.
// ExecutionMode is set to interactive on the happy path.
func (s *Substrate) Launch(ctx context.Context, p LaunchParams) (LaunchResult, error) {
	spec, ok := specFor(p.RunnerType)
	if !ok {
		return LaunchResult{}, fmt.Errorf("interactive mode is not supported for runner %q (descoped from v1)", p.RunnerType)
	}

	info, err := s.launchInfo(p.RunnerType)
	if err != nil {
		return LaunchResult{}, fmt.Errorf("resolve interactive launch info for %s: %w", p.RunnerType, err)
	}

	// Pre-create the relocated session home and seed it with the shared home's
	// credentials so the launched CLI stays authenticated (design §4) — a fresh
	// home with no auth drops the CLI to a sign-in wall and it never starts a turn.
	if spec.relocatesHome() {
		home := homeDirFor(spec, p.RunDir)
		if err := os.MkdirAll(home, 0o755); err != nil {
			return LaunchResult{}, fmt.Errorf("create %s for %s: %w", spec.homeEnvVar, p.RunnerType, err)
		}
		if err := seedRelocatedHome(spec, home, p.WorkingDir, s.homeDir); err != nil {
			return LaunchResult{}, err
		}
	}

	launchCmd, err := BuildLaunchCommand(LaunchCommandParams{
		RunnerType:    p.RunnerType,
		BinaryPath:    info.BinaryPath(),
		TagEnvKey:     info.TagEnvKey(),
		Tag:           p.Tag,
		WorkingDir:    p.WorkingDir,
		RunDir:        p.RunDir,
		Model:         p.Model,
		Effort:        p.Effort,
		InitialPrompt: p.Prompt,
	})
	if err != nil {
		return LaunchResult{}, err
	}

	label := p.DisplayLabel
	if label == "" {
		label = p.Tag
	}

	launchedAt := s.now()
	sessionID, err := s.sessions.CreateSession(ctx, webconsole.CreateSessionParams{
		LaunchCommand: launchCmd,
		Execute:       true,
		DisplayLabel:  label,
		Backend:       "persistent",
	})
	if err != nil {
		return LaunchResult{}, fmt.Errorf("create interactive session for %s: %w", p.RunnerType, err)
	}

	result := LaunchResult{
		SessionID:     sessionID,
		LaunchCommand: launchCmd,
		ExecutionMode: domain.ExecutionModeInteractive,
	}
	// Give the just-created interactive process its normal boot window before
	// transcript discovery. The initial prompt is already on its command line;
	// this is not a second prompt delivery.
	if p.Prompt != "" {
		if err := s.awaitBoot(ctx); err != nil {
			return result, err
		}
	}

	// BuildLaunchCommand passes the initial prompt as a safely quoted trailing
	// argument. Do not paste it into the session as well: duplicating the first
	// turn can produce two independent agent actions with the same run identity.
	// Interactive follow-up messages still use SendPrompt through the coordinator.
	var resend func() error

	transcriptPath, err := s.discoverTranscript(ctx, DiscoverParams{
		RunnerType: p.RunnerType,
		WorkingDir: p.WorkingDir,
		RunDir:     p.RunDir,
		LaunchedAt: launchedAt,
		HomeDir:    s.homeDir,
	}, resend)
	if err != nil {
		// Session is live; return it so the caller can persist + tear down.
		return result, fmt.Errorf("discover transcript for %s (session %s): %w", p.RunnerType, sessionID, err)
	}
	result.TranscriptPath = transcriptPath
	return result, nil
}

// awaitBoot waits promptBootDelay (or returns immediately when disabled),
// honouring context cancellation so a stopped launch does not block.
func (s *Substrate) awaitBoot(ctx context.Context) error {
	if s.promptBootDelay <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.promptBootDelay):
		return nil
	}
}

// discoverTranscript polls findTranscript until the transcript appears or the
// discovery timeout elapses. If resend is non-nil, it re-delivers the prompt
// every promptResendAfter while no transcript has appeared — recovering a paste
// dropped by a not-yet-rendered TUI and clearing a one-shot in-band launch gate
// that swallows a delivery (claude's first-run trust dialog). Re-delivery is
// bounded by the discovery deadline; a resend error is non-fatal because the
// deadline is the authoritative failure.
func (s *Substrate) discoverTranscript(ctx context.Context, p DiscoverParams, resend func() error) (string, error) {
	start := s.now()
	deadline := start.Add(s.discoveryTimeout)
	nextResend := start.Add(s.promptResendAfter)
	for {
		path, err := findTranscript(p)
		if err != nil {
			return "", err
		}
		if path != "" {
			return path, nil
		}
		now := s.now()
		if now.After(deadline) {
			return "", fmt.Errorf("transcript did not appear within %s", s.discoveryTimeout)
		}
		if resend != nil && !now.Before(nextResend) {
			_ = resend()
			nextResend = now.Add(s.promptResendAfter)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(s.pollInterval):
		}
	}
}

// Stop ends an interactive run's session. It sends the graceful interrupt
// sequence (Escape then Ctrl+C), waits a short grace so the CLI can react, then
// deletes the session as the hard-kill fallback (design decision 6, risk R5).
// Delete is idempotent, so Stop always finalizes the session even if the
// interrupt left the CLI mid-turn. An interrupt failure is non-fatal as long as
// the delete succeeds.
func (s *Substrate) Stop(ctx context.Context, sessionID, source string) error {
	if sessionID == "" {
		return fmt.Errorf("interactive stop: empty session id")
	}
	interruptErr := s.sessions.Interrupt(ctx, sessionID, source)

	if s.stopGrace > 0 {
		select {
		case <-ctx.Done():
			// fall through to the delete fallback even on cancellation so the
			// session is not leaked.
		case <-time.After(s.stopGrace):
		}
	}

	if delErr := s.sessions.DeleteSession(ctx, sessionID); delErr != nil {
		return errors.Join(interruptErr, delErr)
	}
	return nil
}

// CleanupCredentials removes the seeded credential/config files from a run's
// relocated home after its interactive session has been torn down. Callers pass
// the agent-manager-owned run dir (codex/grok relocate their home under it);
// it is a no-op for shared-home agents like claude. Call only once the CLI is
// dead (post-Stop or a failed launch), so a live session's authentication is
// never pulled out from under it.
func (s *Substrate) CleanupCredentials(runnerType domain.RunnerType, runDir string) error {
	spec, ok := specFor(runnerType)
	if !ok {
		return nil
	}
	return cleanupSeededHome(spec, homeDirFor(spec, runDir))
}

// ApplyToRun writes the durable interactive facts onto a run record: execution
// mode, web-console session id, and the resolved agent-owned transcript path
// (reusing Run.TranscriptPath — provenance differs from codec-pipe but the
// field is the same, per design §5). The caller persists the run afterwards.
func ApplyToRun(run *domain.Run, res LaunchResult) {
	if run == nil {
		return
	}
	if res.ExecutionMode != "" {
		run.ExecutionMode = res.ExecutionMode
	}
	if res.SessionID != "" {
		run.WebConsoleSessionID = res.SessionID
	}
	if res.TranscriptPath != "" {
		run.TranscriptPath = res.TranscriptPath
	}
}
