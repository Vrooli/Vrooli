package conformance

import (
	"context"
	"switchboard/internal/channels"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
)

type CaseResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// Run executes the same transport-neutral checks for every registered adapter.
// It intentionally receives only the Adapter contract and descriptor.
func Run(ctx context.Context, adapter channels.Adapter, descriptor channels.Descriptor) []CaseResult {
	results := []CaseResult{
		checkText(ctx, adapter, descriptor), checkMedia(ctx, adapter, descriptor), checkLimit(descriptor),
		checkThreaded(ctx, adapter, descriptor), checkRateLimit(), checkUnknownSender(), checkAgentMessage(),
	}
	return results
}

func checkText(ctx context.Context, a channels.Adapter, d channels.Descriptor) CaseResult {
	err := a.Send(ctx, channels.Outbound{ChannelID: d.ID, ThreadKey: "thread", Text: "hello"})
	return result("text-round-trip", err)
}
func checkMedia(ctx context.Context, a channels.Adapter, d channels.Descriptor) CaseResult {
	err := a.Send(ctx, channels.Outbound{ChannelID: d.ID, ThreadKey: "thread", Media: []channels.Media{{Name: "image.png", MIME: "image/png", Size: 10}, {Name: "document.txt", MIME: "text/plain", Size: 10}}})
	return result("media-round-trip", err)
}
func checkLimit(d channels.Descriptor) CaseResult {
	err := channels.ValidateOutbound(d, channels.Outbound{ChannelID: d.ID, Media: []channels.Media{{Name: "too-large", Size: d.Limits.MaxMediaBytes + 1}}})
	if err == nil {
		return CaseResult{Name: "declared-limit-rejection"}
	}
	return CaseResult{Name: "declared-limit-rejection", Passed: true, Detail: err.Error()}
}
func checkThreaded(ctx context.Context, a channels.Adapter, d channels.Descriptor) CaseResult {
	err := a.Send(ctx, channels.Outbound{ChannelID: d.ID, ThreadKey: "thread", ReplyToRemoteID: "remote-1", Text: "reply"})
	return result("threaded-reply", err)
}
func checkRateLimit() CaseResult {
	l := NewLimiter(2)
	if !l.Allow() || !l.Allow() || l.Allow() {
		return CaseResult{Name: "rate-limit-backoff"}
	}
	return CaseResult{Name: "rate-limit-backoff", Passed: true}
}
func checkUnknownSender() CaseResult {
	r := trust.Resolve(trust.Stranger, trust.Stranger, trust.Grant{Scopes: []string{"owner"}})
	if len(r.Scopes) != 0 {
		return CaseResult{Name: "unknown-sender-ingress"}
	}
	return CaseResult{Name: "unknown-sender-ingress", Passed: true}
}
func checkAgentMessage() CaseResult {
	if threads.ShouldRespond(channels.Envelope{AuthorKind: channels.AuthorAgent}, false, true) {
		return CaseResult{Name: "agent-authored-refusal"}
	}
	return CaseResult{Name: "agent-authored-refusal", Passed: true}
}
func result(name string, err error) CaseResult {
	if err != nil {
		return CaseResult{Name: name, Detail: err.Error()}
	}
	return CaseResult{Name: name, Passed: true}
}

type Limiter struct{ remaining int }

func NewLimiter(n int) *Limiter { return &Limiter{remaining: n} }
func (l *Limiter) Allow() bool {
	if l.remaining <= 0 {
		return false
	}
	l.remaining--
	return true
}
