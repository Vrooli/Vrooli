package feedback

import (
	"context"
	"errors"
	"strings"
	"testing"

	"swarm-manager/internal/proposals"
)

// fakePoller is a controllable AgentRunPoller used to drive
// EnsurePolledTurn through every relevant branch.
type fakePoller struct {
	enabled bool
	state   RunState
	err     error
	calls   int
}

func (p *fakePoller) IsEnabled() bool { return p.enabled }

func (p *fakePoller) GetRunState(_ context.Context, _ string) (RunState, error) {
	p.calls++
	if p.err != nil {
		return RunState{}, p.err
	}
	return p.state, nil
}

// withPoller rebuilds the service in env with the supplied poller, since
// newServiceEnv doesn't wire one by default. Uses a minimal state
// builder — EnsurePolledTurn doesn't traverse Apply, so the builder
// behavior doesn't matter for these tests.
func withPoller(t *testing.T, env *serviceEnv, p AgentRunPoller) *Service {
	t.Helper()
	stateBuilder := func(name string) (proposals.CurrentState, error) {
		return proposals.CurrentState{InitiativeName: name}, nil
	}
	svc, err := NewService(Config{
		Store:        env.store,
		Lock:         env.lock,
		Spawner:      env.spawner,
		Poller:       p,
		Apply:        env.applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestEnsurePolledTurn_NoOp_WhenRoundNotAgentThinking(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{enabled: true}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "just a note",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Note rounds land in dismissed status — not agent_thinking.
	if round.Status == RoundStatusAgentThinking {
		t.Fatal("expected note round not to be agent_thinking")
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatalf("EnsurePolledTurn: %v", err)
	}
	if poller.calls != 0 {
		t.Fatalf("poller should not be called for non-agent_thinking rounds, got %d calls", poller.calls)
	}
	if out.Status != round.Status {
		t.Fatal("status should not change on no-op")
	}
}

func TestEnsurePolledTurn_NoOp_WhenPollerDisabled(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{enabled: false}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", round.Status)
	}

	if _, err := svc.EnsurePolledTurn(context.Background(), round); err != nil {
		t.Fatal(err)
	}
	if poller.calls != 0 {
		t.Fatalf("poller should not be called when disabled, got %d calls", poller.calls)
	}
}

func TestEnsurePolledTurn_NoOp_WhenRunIDMissing(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{enabled: true}
	svc := withPoller(t, env, poller)

	r := Round{
		InitiativeName: "ui-rewrite",
		Number:         1,
		Status:         RoundStatusAgentThinking,
		RunID:          "",
	}
	out, err := svc.EnsurePolledTurn(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if poller.calls != 0 {
		t.Fatalf("poller should not be called with empty RunID, got %d calls", poller.calls)
	}
	if out.Status != RoundStatusAgentThinking {
		t.Fatal("status should not change when run id missing")
	}
}

func TestEnsurePolledTurn_NoOp_WhenRunNotTerminal(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state:   RunState{Status: "running", Summary: "still working"},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatal(err)
	}
	if poller.calls != 1 {
		t.Fatalf("poller should be consulted exactly once, got %d", poller.calls)
	}
	if out.Status != RoundStatusAgentThinking {
		t.Fatalf("expected status to stay agent_thinking, got %s", out.Status)
	}
	// Reload from disk to confirm no turn was persisted.
	loaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	// Only the user submission should be in the thread.
	if len(loaded.Thread) != 1 {
		t.Fatalf("expected 1 message in thread, got %d", len(loaded.Thread))
	}
}

func TestEnsurePolledTurn_PollErrorIsNonFatal(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		err:     errors.New("agent-manager unreachable"),
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatalf("poll error must be swallowed, got %v", err)
	}
	if out.Status != RoundStatusAgentThinking {
		t.Fatalf("expected status to remain agent_thinking on poll error, got %s", out.Status)
	}
}

func TestEnsurePolledTurn_TerminalCompletion_RecordsAgentTurn(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state: RunState{
			Status: "complete",
			Summary: "Looks good.\n```json\n" +
				`{"form":"mutation_list","mutations":[{"id":"m1","op":"change_priority","target":"execute/foo","priority":7}]}` +
				"\n```",
		},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatalf("EnsurePolledTurn: %v", err)
	}
	if out.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user after terminal poll, got %s", out.Status)
	}
	if poller.calls != 1 {
		t.Fatalf("poller should be called exactly once, got %d", poller.calls)
	}
	if len(out.Proposals) != 1 {
		t.Fatalf("expected 1 proposal extracted from agent summary, got %d", len(out.Proposals))
	}
	if out.CurrentProposalID == "" {
		t.Fatal("CurrentProposalID should be set after extracting a proposal")
	}
}

func TestEnsurePolledTurn_TerminalFailure_RecordsFailureMessage(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state: RunState{
			Status:   "failed",
			Summary:  "",
			ErrorMsg: "sandbox crashed",
		},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatalf("EnsurePolledTurn: %v", err)
	}
	if out.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user after terminal failure, got %s", out.Status)
	}
	// The agent message should describe the failure so the user can act.
	last := out.Thread[len(out.Thread)-1]
	if last.Role != "agent" {
		t.Fatalf("expected agent message at tail, got role=%q", last.Role)
	}
	if !strings.Contains(last.Content, "agent run failed") {
		t.Fatalf("expected failure summary, got %q", last.Content)
	}
	if !strings.Contains(last.Content, "sandbox crashed") {
		t.Fatalf("expected error message included, got %q", last.Content)
	}
	if len(out.Proposals) != 0 {
		t.Fatalf("failure shouldn't yield a proposal, got %d", len(out.Proposals))
	}
}

func TestEnsurePolledTurn_TerminalNoOutput_RecordsPlaceholder(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state:   RunState{Status: "complete", Summary: ""},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatal(err)
	}
	last := out.Thread[len(out.Thread)-1]
	if !strings.Contains(last.Content, "completed without producing output") {
		t.Fatalf("expected placeholder summary, got %q", last.Content)
	}
}

func TestEnsurePolledTurn_PollError_RecordsFailureFields(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		err:     errors.New("agent-manager unreachable"),
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatalf("EnsurePolledTurn: %v", err)
	}
	if out.Status != RoundStatusAgentThinking {
		t.Fatalf("expected status to remain agent_thinking on first poll error, got %s", out.Status)
	}
	if out.PollFailureCount != 1 {
		t.Fatalf("expected PollFailureCount=1, got %d", out.PollFailureCount)
	}
	if !strings.Contains(out.LastPollError, "agent-manager unreachable") {
		t.Fatalf("expected LastPollError to record error, got %q", out.LastPollError)
	}
	if out.LastPolledAt == "" {
		t.Fatal("expected LastPolledAt to be set after a poll attempt")
	}
	// Persisted to disk as well.
	loaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PollFailureCount != 1 {
		t.Fatalf("disk PollFailureCount=%d", loaded.PollFailureCount)
	}
}

func TestEnsurePolledTurn_ConsecutiveFailures_SynthesizeTerminal(t *testing.T) {
	t.Setenv("SWARM_MANAGER_FEEDBACK_POLL_FAILURE_THRESHOLD", "3")
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		err:     errors.New("run not found"),
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	// First two failures: round stays in agent_thinking.
	for i := 1; i <= 2; i++ {
		latest, err := env.store.LoadRound("ui-rewrite", round.Number)
		if err != nil {
			t.Fatal(err)
		}
		out, err := svc.EnsurePolledTurn(context.Background(), latest)
		if err != nil {
			t.Fatalf("poll %d: %v", i, err)
		}
		if out.Status != RoundStatusAgentThinking {
			t.Fatalf("poll %d: expected agent_thinking, got %s", i, out.Status)
		}
		if out.PollFailureCount != i {
			t.Fatalf("poll %d: PollFailureCount=%d, want %d", i, out.PollFailureCount, i)
		}
	}

	// Third failure: synthesizes a terminal failure turn.
	latest, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.EnsurePolledTurn(context.Background(), latest)
	if err != nil {
		t.Fatalf("poll 3: %v", err)
	}
	if out.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user after threshold reached, got %s", out.Status)
	}
	last := out.Thread[len(out.Thread)-1]
	if last.Role != "agent" {
		t.Fatalf("expected agent message at tail, got role=%q", last.Role)
	}
	if !strings.Contains(last.Content, "no longer reachable") {
		t.Fatalf("expected unreachable summary, got %q", last.Content)
	}
}

func TestEnsurePolledTurn_RecoveryClearsFailureCounter(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		err:     errors.New("transient"),
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	// One failure, then recovery (poller starts returning a non-terminal
	// status). Counter should reset.
	if _, err := svc.EnsurePolledTurn(context.Background(), round); err != nil {
		t.Fatal(err)
	}
	loaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PollFailureCount != 1 {
		t.Fatalf("expected PollFailureCount=1 after first error, got %d", loaded.PollFailureCount)
	}
	// Poller recovers — non-terminal status this time.
	poller.err = nil
	poller.state = RunState{Status: "running"}
	out, err := svc.EnsurePolledTurn(context.Background(), loaded)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking after recovery, got %s", out.Status)
	}
	if out.PollFailureCount != 0 {
		t.Fatalf("expected PollFailureCount=0 after recovery, got %d", out.PollFailureCount)
	}
	if out.LastPollError != "" {
		t.Fatalf("expected LastPollError cleared after recovery, got %q", out.LastPollError)
	}
}

func TestEnsurePolledTurn_NotFoundStatus_TreatedAsTerminalFailure(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state: RunState{
			Status:   "not_found",
			ErrorMsg: "run gone",
		},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatalf("EnsurePolledTurn: %v", err)
	}
	if out.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user, got %s", out.Status)
	}
	last := out.Thread[len(out.Thread)-1]
	if !strings.Contains(last.Content, "agent run failed") {
		t.Fatalf("expected failure summary, got %q", last.Content)
	}
}

func TestEnsurePolledTurn_Idempotent_AfterTransition(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)
	poller := &fakePoller{
		enabled: true,
		state:   RunState{Status: "complete", Summary: "done"},
	}
	svc := withPoller(t, env, poller)

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate",
	})
	if err != nil {
		t.Fatal(err)
	}
	out1, err := svc.EnsurePolledTurn(context.Background(), round)
	if err != nil {
		t.Fatal(err)
	}
	if out1.Status != RoundStatusAwaitingUser {
		t.Fatalf("first poll should advance to awaiting_user, got %s", out1.Status)
	}
	// A second call with the same round (now no longer in agent_thinking)
	// must not double-record nor re-poll.
	calls := poller.calls
	out2, err := svc.EnsurePolledTurn(context.Background(), out1)
	if err != nil {
		t.Fatalf("second EnsurePolledTurn: %v", err)
	}
	if poller.calls != calls {
		t.Fatalf("poller should not be re-consulted after transition, got %d new calls",
			poller.calls-calls)
	}
	if len(out2.Thread) != len(out1.Thread) {
		t.Fatalf("thread length must not change on no-op call (got %d, want %d)",
			len(out2.Thread), len(out1.Thread))
	}
}
