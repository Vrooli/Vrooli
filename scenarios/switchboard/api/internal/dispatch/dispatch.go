package dispatch

import (
	"context"
	"fmt"
	"switchboard/internal/channels"
	"switchboard/internal/ingress"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
	"time"
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
type OwnerNotice func(context.Context, channels.Envelope, string) error
type Processor struct {
	Ingress     *ingress.Store
	Threads     *threads.Store
	Runner      Runner
	Send        Reply
	NotifyOwner OwnerNotice
	Grant       trust.Grant
}
type Outcome string

const (
	OutcomeAccepted   Outcome = "accepted"
	OutcomeDuplicate  Outcome = "duplicate"
	OutcomeSuppressed Outcome = "suppressed"
	OutcomeRefused    Outcome = "refused"
)

type Result struct {
	Outcome Outcome
	Reply   string
	Reason  string
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
		return Result{Outcome: OutcomeDuplicate}, nil
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
	if e.AuthorKind == channels.AuthorAgent || !threads.ShouldRespond(e, group, addressed) {
		return Result{Outcome: OutcomeSuppressed, Reason: "message recorded without starting a turn"}, nil
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
			return Result{Outcome: OutcomeSuppressed, Reason: reason}, nil
		}
	}
	scope := trust.Resolve(sender, ceiling, p.Grant)
	if len(scope.Scopes) == 0 {
		return Result{Outcome: OutcomeRefused, Reason: "no permitted scope remains"}, nil
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
		return Result{}, runErr
	}
	if p.Send != nil && reply != "" {
		if err := p.Send(ctx, channels.Outbound{ChannelID: e.ChannelID, ThreadKey: e.ThreadKey, Text: reply, ReplyToRemoteID: e.RemoteMessageID}); err != nil {
			return Result{}, err
		}
	}
	return Result{Outcome: OutcomeAccepted, Reply: reply}, nil
}
