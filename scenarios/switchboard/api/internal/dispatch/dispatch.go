package dispatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"switchboard/internal/channels"
	"switchboard/internal/ingress"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
)

type Runner interface {
	Run(context.Context, string, []string, string) (string, error)
}
type ConversationRunner interface {
	RunConversation(context.Context, string, []string, string, string, string) (string, error)
}
type Reply func(context.Context, channels.Outbound) error

// OwnerNotice is invoked once when a thread ceiling suppresses a turn and the
// durable budget transitions to owner-notification due.
type (
	OwnerNotice func(context.Context, channels.Envelope, string) error
	// GrantResolver returns the capability grant declared by an agent profile.
	// Switchboard holds the profile by reference; the resolver reads it live.
	GrantResolver interface {
		GrantFor(ctx context.Context, agentID string) trust.Grant
	}
)

type Processor struct {
	Ingress     *ingress.Store
	Threads     *threads.Store
	Runner      Runner
	Send        Reply
	NotifyOwner OwnerNotice
	// Grant is the fallback when Grants is nil (tests and single-agent wiring).
	Grant  trust.Grant
	Grants GrantResolver
}
type Outcome string

const (
	OutcomeAccepted   Outcome = "accepted"
	OutcomeDuplicate  Outcome = "duplicate"
	OutcomeSuppressed Outcome = "suppressed"
	OutcomeRefused    Outcome = "refused"
	// OutcomeFailed is a turn that was admitted but could not run: the agent
	// runner was unavailable or rejected the run. It is stated out loud on the
	// thread so the conversation is never silently dropped.
	OutcomeFailed Outcome = "failed"
)

type Result struct {
	Outcome Outcome `json:"outcome"`
	Reply   string  `json:"reply,omitempty"`
	Reason  string  `json:"reason,omitempty"`
}

func (p *Processor) record(ctx context.Context, e channels.Envelope, agentID string, result Result) (Result, error) {
	if p.Threads != nil {
		if err := p.Threads.RecordEvent(ctx, e, agentID, string(result.Outcome), result.Reason); err != nil {
			return Result{}, fmt.Errorf("record turn event: %w", err)
		}
	}
	return result, nil
}

// RefusalText states a refusal out loud: what was withheld and what would
// unblock it, in the words a person on the other end can act on.
func RefusalText(sender, ceiling trust.Tier) string {
	limit := sender
	if ceiling < limit {
		limit = ceiling
	}
	why := "your trust tier is " + sender.String()
	if ceiling < sender {
		why = "this room's ceiling is " + ceiling.String() + " because of who else is in it"
	}
	return fmt.Sprintf("I can't act on that from this conversation: %s, so no capability is available at the %s tier. The owner can raise it in Switchboard → Contacts.", why, limit.String())
}

// Process is the single turn admission path. It records and deduplicates
// before scope resolution or execution, and it never lets an agent message
// trigger another run.
func (p *Processor) Process(ctx context.Context, e channels.Envelope, sender, ceiling trust.Tier, bindingAgent string, group, addressed bool) (Result, error) {
	group = group || e.Group
	seen, err := p.Ingress.Accept(e)
	if err != nil {
		return Result{}, err
	}
	if seen == ingress.AlreadySeen {
		return Result{Outcome: OutcomeDuplicate, Reason: "redelivery of a message already recorded"}, nil
	}
	if p.Threads != nil {
		thread, err := p.Threads.Upsert(ctx, e, group)
		if err != nil {
			return Result{}, fmt.Errorf("persist thread: %w", err)
		}
		if _, err := p.Threads.Append(ctx, thread, e); err != nil {
			return Result{}, fmt.Errorf("persist message: %w", err)
		}
	}
	if e.AuthorKind == channels.AuthorAgent {
		return p.record(ctx, e, bindingAgent, Result{Outcome: OutcomeSuppressed, Reason: "agent-authored message recorded; agents never trigger a turn"})
	}
	if !threads.ShouldRespond(e, group, addressed) {
		return p.record(ctx, e, bindingAgent, Result{Outcome: OutcomeSuppressed, Reason: "group message not addressed to the agent; recorded without a turn"})
	}
	if p.Threads != nil {
		thread, err := p.Threads.Upsert(ctx, e, group)
		if err != nil {
			return Result{}, fmt.Errorf("load thread budget: %w", err)
		}
		allowed, notifyOwner, err := p.Threads.AllowTurn(ctx, thread, time.Now(), 1)
		if err != nil {
			return Result{}, fmt.Errorf("admit thread turn: %w", err)
		}
		if !allowed {
			reason := "thread budget exhausted"
			if notifyOwner {
				reason += "; owner notification due"
				if p.NotifyOwner != nil {
					if err := p.NotifyOwner(ctx, e, reason); err != nil {
						return Result{}, fmt.Errorf("notify owner: %w", err)
					}
				}
			}
			return p.record(ctx, e, bindingAgent, Result{Outcome: OutcomeSuppressed, Reason: reason})
		}
	}
	grant := p.Grant
	if p.Grants != nil {
		grant = p.Grants.GrantFor(ctx, bindingAgent)
	}
	scope := trust.Resolve(sender, ceiling, grant)
	if len(scope.Scopes) == 0 {
		reason := fmt.Sprintf("no permitted scope remains: sender tier %s, room ceiling %s", sender, ceiling)
		refusal := RefusalText(sender, ceiling)
		if p.Send != nil {
			if err := p.Send(ctx, channels.Outbound{ChannelID: e.ChannelID, ThreadKey: e.ThreadKey, Text: refusal, ReplyToRemoteID: e.RemoteMessageID}); err != nil {
				reason += "; refusal could not be delivered: " + err.Error()
			}
		}
		return p.record(ctx, e, bindingAgent, Result{Outcome: OutcomeRefused, Reason: reason, Reply: refusal})
	}
	if p.Runner == nil {
		return Result{}, fmt.Errorf("agent runner unavailable")
	}
	var reply string
	var runErr error
	if conversational, ok := p.Runner.(ConversationRunner); ok {
		reply, runErr = conversational.RunConversation(ctx, bindingAgent, scope.Scopes, e.ChannelID, e.ThreadKey, e.Text)
	} else {
		reply, runErr = p.Runner.Run(ctx, bindingAgent, scope.Scopes, e.Text)
	}
	if runErr != nil {
		notice := "I couldn't start on that just now: " + humanRunError(runErr) + " Your message is kept on this thread; send it again in a moment."
		if p.Send != nil {
			_ = p.Send(ctx, channels.Outbound{ChannelID: e.ChannelID, ThreadKey: e.ThreadKey, Text: notice, ReplyToRemoteID: e.RemoteMessageID})
		}
		return p.record(ctx, e, bindingAgent, Result{Outcome: OutcomeFailed, Reason: runErr.Error(), Reply: notice})
	}
	if p.Send != nil && reply != "" {
		if err := p.Send(ctx, channels.Outbound{ChannelID: e.ChannelID, ThreadKey: e.ThreadKey, Text: reply, ReplyToRemoteID: e.RemoteMessageID}); err != nil {
			return Result{}, err
		}
	}
	return p.record(ctx, e, bindingAgent, Result{Outcome: OutcomeAccepted, Reply: reply, Reason: scope.Reason})
}

// humanRunError turns a runner error into the one line a person on the other
// end can act on, without leaking request internals.
func humanRunError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "CAPACITY") || strings.Contains(msg, "capacity"):
		return "the agent runtime is at capacity."
	case strings.Contains(msg, "unavailable") || strings.Contains(msg, "connection refused"):
		return "the agent runtime is unreachable."
	}
	return "the agent runtime rejected the run."
}

// PersistingReply wraps an adapter send so every agent-authored message is
// written to the thread before it leaves. Both the synchronous refusal path
// and the asynchronous agent-manager reply path go through it, which is what
// keeps a reloaded transcript complete.
func PersistingReply(store *threads.Store, send Reply, agentID func(channels.Outbound) string) Reply {
	return func(ctx context.Context, out channels.Outbound) error {
		if store != nil {
			author := ""
			if agentID != nil {
				author = agentID(out)
			}
			if err := store.AppendOutbound(ctx, out, author); err != nil {
				return fmt.Errorf("persist agent message: %w", err)
			}
		}
		if send == nil {
			return nil
		}
		return send(ctx, out)
	}
}
