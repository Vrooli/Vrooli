// serial_episode.go owns durable conservative handoffs between exact native episodes.
package effortauthority

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"
)

// SerialEdge is part of the human-consented policy, not a request override.
// Both endpoints name existing member/profile identities from that same policy.
type SerialEdge struct {
	FromMember  string `json:"fromMember"`
	FromProfile string `json:"fromProfile"`
	ToMember    string `json:"toMember"`
	ToProfile   string `json:"toProfile"`
}

// SerialHandoff captures the next exact native input before native terminal settlement.
// A handoff alone consumes no execution slot and authorizes no native effect.
type SerialHandoff struct {
	Payload     json.RawMessage `json:"payload,omitempty"`
	SourceKey   string          `json:"sourceKey"`
	SourceRunID string          `json:"sourceRunId"`
	Next        Intent          `json:"next"`
}

func validateSerialEdges(p Policy) error {
	if len(p.SerialEdges) == 0 {
		return nil
	}
	if len(p.SerialEdges) != 2 || len(p.Profiles) != 2 || (len(p.Members) != 1 && len(p.Members) != 2) || (len(p.Members) == 2 && p.Members[0] == p.Members[1]) || !p.AllowChildren || !slices.Contains(p.Effects, "run.child") || p.MaxConcurrent != 1 || p.MaxStarts > 8 || p.MaxTurns > 32 || p.MaxToolCalls > 100 || p.MaxRunSeconds > 3600 || p.TotalTurns > 256 || p.TotalToolCalls > 800 || p.TotalRunSeconds > 28800 || p.Deadline.Sub(p.NotBefore) > 8*time.Hour {
		return ErrRefused
	}
	seenProfiles := map[string]bool{}
	for _, d := range p.Profiles {
		if seenProfiles[d] {
			return ErrRefused
		}
		seenProfiles[d] = true
	}
	a, b := p.SerialEdges[0], p.SerialEdges[1]
	for _, e := range p.SerialEdges {
		if e.FromProfile == e.ToProfile || !slices.Contains(p.Members, e.FromMember) || !slices.Contains(p.Members, e.ToMember) || p.Profiles[e.FromProfile] == "" || p.Profiles[e.ToProfile] == "" {
			return ErrRefused
		}
	}
	if a.FromMember != b.ToMember || a.ToMember != b.FromMember || a.FromProfile != b.ToProfile || a.ToProfile != b.FromProfile {
		return ErrRefused
	}
	for _, member := range p.Members {
		if member != a.FromMember && member != a.ToMember {
			return ErrRefused
		}
	}
	return nil
}
func allowedSerialEdge(p Policy, source, next Intent) bool {
	for _, e := range p.SerialEdges {
		if source.Member == e.FromMember && source.Profile == e.FromProfile && next.Member == e.ToMember && next.Profile == e.ToProfile {
			return true
		}
	}
	return false
}
func checkSerialHandoff(r Record, key string, next Intent, terminal bool) error {
	h, ok := r.SerialHandoffs[key]
	if !ok || key != next.IdempotencyKey || Digest(h.Next) != Digest(next) {
		return ErrRefused
	}
	if len(h.Payload) > 24576 || (len(h.Payload) > 0 && next.SerialPayloadDigest != Digest(string(h.Payload))) || (len(h.Payload) == 0 && next.SerialPayloadDigest != "") {
		return ErrRefused
	}
	source, ok := r.Reservations[h.SourceKey]
	if !ok || !source.NativeBound || source.RunID == "" || source.RunID != h.SourceRunID || source.Intent.IdempotencyKey != h.SourceKey || source.IntentDigest != Digest(source.Intent) || (terminal && !source.Terminal) || next.ParentRunID != source.RunID || next.SourceRunID != "" || !allowedSerialEdge(r.Policy, source.Intent, next) || checkIntent(r.Policy, source.Intent) != nil || checkIntent(r.Policy, next) != nil {
		return ErrRefused
	}
	return nil
}

// PrepareSerialHandoff is a concrete native-owner operation, deliberately absent
// from base Engine and public HTTP. Caller identity must already be verified by its owning
// adapter; this method never accepts a public run ID as proof of ownership.
func (a *Authority) PrepareSerialHandoff(ctx context.Context, b Binding, sourceKey string, next Intent) (SerialHandoff, bool, error) {
	return a.prepareSerialHandoff(ctx, b, sourceKey, next, nil)
}
func (a *Authority) PrepareSerialHandoffWithPayload(ctx context.Context, b Binding, sourceKey string, next Intent, payload json.RawMessage) (SerialHandoff, bool, error) {
	canonical, err := json.Marshal(json.RawMessage(payload))
	if err != nil || !bytes.Equal(canonical, payload) {
		return SerialHandoff{}, false, ErrRefused
	}
	if len(payload) == 0 || len(payload) > 24576 || !json.Valid(payload) || next.SerialPayloadDigest != Digest(string(payload)) {
		return SerialHandoff{}, false, ErrRefused
	}
	return a.prepareSerialHandoff(ctx, b, sourceKey, next, append(json.RawMessage(nil), payload...))
}
func (a *Authority) prepareSerialHandoff(ctx context.Context, b Binding, sourceKey string, next Intent, payload json.RawMessage) (SerialHandoff, bool, error) {
	if (len(payload) == 0 && next.SerialPayloadDigest != "") || (len(payload) > 0 && next.SerialPayloadDigest != Digest(string(payload))) {
		return SerialHandoff{}, false, ErrRefused
	}

	p, err := a.CheckBinding(ctx, b)
	if err != nil || ctx.Err() != nil || len(p.SerialEdges) == 0 || next.Effect != "run.child" || next.Endpoint != "/api/v1/runs" || checkIntent(p, next) != nil || next.IdempotencyKey == sourceKey || next.Turns != p.MaxTurns || next.ToolCalls != p.MaxToolCalls || next.RunSeconds != p.MaxRunSeconds {
		return SerialHandoff{}, false, ErrRefused
	}
	var out SerialHandoff
	var replay bool
	err = a.Store.Update(ctx, b.PolicyID, func(r *Record) error {
		if ctx.Err() != nil || r.Revoked || binding(r.Policy) != b || !r.Policy.Deadline.After(a.now()) {
			return ErrRefused
		}
		if old, ok := r.SerialHandoffs[next.IdempotencyKey]; ok {
			if old.SourceKey != sourceKey || Digest(old.Next) != Digest(next) || string(old.Payload) != string(payload) || checkSerialHandoff(*r, next.IdempotencyKey, next, false) != nil {
				return ErrRefused
			}
			out, replay = old, true
			return nil
		}
		source, ok := r.Reservations[sourceKey]
		if !ok || source.Terminal || !source.NativeBound || source.RunID == "" || source.IntentDigest != Digest(source.Intent) || source.Intent.IdempotencyKey != sourceKey || checkIntent(r.Policy, source.Intent) != nil || next.ParentRunID != source.RunID || !allowedSerialEdge(r.Policy, source.Intent, next) {
			return ErrRefused
		}
		if _, ok := r.Reservations[next.IdempotencyKey]; ok {
			return ErrRefused
		}
		for _, h := range r.SerialHandoffs {
			if h.SourceKey == sourceKey {
				return ErrRefused
			}
		}
		if r.SerialHandoffs == nil {
			r.SerialHandoffs = map[string]SerialHandoff{}
		}
		out = SerialHandoff{SourceKey: sourceKey, SourceRunID: source.RunID, Next: next, Payload: append(json.RawMessage(nil), payload...)}
		r.SerialHandoffs[next.IdempotencyKey] = out
		return nil
	})
	if err != nil {
		return SerialHandoff{}, false, ErrRefused
	}
	out.Payload = append(json.RawMessage(nil), out.Payload...)
	return out, replay, nil
}

// ReserveSerialEpisode uses the existing durable reservation/budget transaction.
// It requires the prior accepted source reservation to have terminal evidence
// committed by its native owner. Native launch remains a separate source-qualified adapter; installation is held.
func (a *Authority) ReserveSerialEpisode(ctx context.Context, b Binding, handoffKey string) (Reservation, bool, error) {
	if ctx.Err() != nil {
		return Reservation{}, false, ErrRefused
	}
	rec, err := a.live(ctx, b.PolicyID)
	if err != nil || binding(rec.Policy) != b {
		return Reservation{}, false, ErrRefused
	}
	h, ok := rec.SerialHandoffs[handoffKey]
	if !ok {
		return Reservation{}, false, ErrRefused
	}
	return a.reserve(ctx, b, h.Next, nil, handoffKey)
}

func (a *Authority) ReadSerialHandoff(ctx context.Context, b Binding, key string) (SerialHandoff, error) {
	rec, err := a.live(ctx, b.PolicyID)
	if err != nil || binding(rec.Policy) != b || len(rec.Policy.SerialEdges) == 0 {
		return SerialHandoff{}, ErrRefused
	}
	h, ok := rec.SerialHandoffs[key]
	if !ok || checkSerialHandoff(rec, key, h.Next, false) != nil {
		return SerialHandoff{}, ErrRefused
	}
	h.Payload = append(json.RawMessage(nil), h.Payload...)
	return h, nil
}
