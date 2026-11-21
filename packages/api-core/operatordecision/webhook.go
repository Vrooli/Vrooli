package operatordecision

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/serviceprincipal"
)

// SignatureHeader is the header vrooli-events signs its webhook bodies with.
const SignatureHeader = "X-Vrooli-Events-Signature"

// maxWebhookBody caps the resolution webhook body (1 MiB). A body at the cap is
// accepted; an over-cap body is rejected with 413 rather than silently truncated
// (which would fail the HMAC check and look like a permanent bad signature).
const maxWebhookBody = 1 << 20

// ValidSignature reports whether value is the HMAC-SHA256 of body under secret,
// in constant time. value may carry the "sha256=" prefix. An empty secret or an
// undecodable value is invalid. It ports validProgramEventSignature so every
// scenario's resolution webhook verifies identically.
func ValidSignature(body []byte, value, secret string) bool {
	if strings.TrimSpace(secret) == "" {
		return false
	}
	value = strings.TrimPrefix(strings.TrimSpace(value), "sha256=")
	provided, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare(provided, expected) == 1
}

// DomainEffect applies a resolved decision to one scenario domain. Handles
// reports whether the effect owns resolutions for a given producer event type
// (the resolution's SourceEventType); Apply acts on a resolution it owns. The
// Webhook fans a resolution to its effects in order and the FIRST that reports
// handled=true wins (so overlapping effects are ordered by registration, exactly
// like agent-manager's run-approval-then-authorization fall-through).
type DomainEffect interface {
	// Handles reports whether this effect owns the producer event type. The webhook
	// skips Apply for effects that do not.
	Handles(sourceEventType string) bool
	// Apply acts on a resolution. handled=true stops fan-out; handled=false lets a
	// later effect try (used when Handles is coarse). An error is surfaced as 500 so
	// vrooli-events retries, EXCEPT the effect should return a benign handled=true
	// nil for an unknown/already-decided ask to avoid a poison-message retry loop.
	Apply(ctx context.Context, resolution DecisionResolution) (handled bool, err error)
}

// SecretSource supplies the webhook signing secret.
type SecretSource func() string

// StaticSecret is a SecretSource for a secret known at start.
func StaticSecret(secret string) SecretSource { return func() string { return secret } }

// AskReader reads an ask back from Notification Hub as the scenario that raised
// it. *askreader.Reader is the production implementation.
type AskReader interface {
	ReadAsk(ctx context.Context, askID string) (askreader.Ask, error)
}

// WebhookConfig configures a resolution webhook.
type WebhookConfig struct {
	// Secret supplies the signing secret vrooli-events signs webhook bodies with,
	// read on every request so a secret that becomes resolvable after start is used
	// without a restart. The signature proves the bus delivered the body, not who
	// published the event. An empty secret refuses every request.
	Secret SecretSource
	// Source is the scenario whose asks this webhook applies: the Hub's Ask.Source,
	// which is the name the scenario publishes its decision requests under. Events
	// about any other scenario's asks are acknowledged without a read. Required
	// when Reader is set.
	Source string
	// Reader reads the ask back from the Hub. With it, nothing is applied that the
	// Hub does not itself say (see Webhook). Without it, or while the scenario has
	// no ask-reader credential, resolutions are applied from the body alone and
	// each one is counted and logged as unverified.
	Reader AskReader
	// Effects apply a resolved decision, in order.
	Effects []DomainEffect
	// OnRefused is told of each resolution the Hub's own record contradicts, once
	// per delivery and before the 409 is sent: the body names a closed ask the Hub
	// still has open, or differs from it. It is the way a refusal reaches the
	// operator (see RefusalNotifier). It is not called for an unreadable Hub, an ask
	// the Hub does not hold, or a context request, and it must not block for long.
	OnRefused func(ctx context.Context, refusal Refusal)
	// Logf receives refusals and unverified applications. log.Printf by default.
	Logf func(format string, args ...any)
}

// Refusal is one resolution the Hub's own record contradicts.
type Refusal struct {
	// Source is the scenario whose webhook refused it.
	Source string
	AskID  string
	// Reason is ReasonMismatch or ReasonAskOpen.
	Reason string
	// Detail says what differed. It is for logs and may vary between deliveries
	// of the same refusal.
	Detail string
	// Question is the Hub's own text of the ask, never the event body's.
	Question string
}

// WebhookStats counts what a webhook did, since it started.
type WebhookStats struct {
	// Verified resolutions and context requests were confirmed by the Hub.
	Verified uint64
	// UnverifiedApplied were applied from the event body alone because the scenario
	// has no ask-reader credential. Each is a gap serviceprincipal.IssueCommand closes.
	UnverifiedApplied uint64
	// Unavailable could not be checked because the Hub could not be read; the bus
	// redelivers them.
	Unavailable uint64
	// Refused by reason: the Hub holds no such ask, still has it open, or says
	// something other than the body does.
	Refused map[string]uint64
}

// Webhook is the http.Handler a scenario mounts to receive
// notification_hub.ask.resolved.v1 (and, for a producer that answers them,
// notification_hub.ask.context_requested.v1) through the signed vrooli-events
// webhook. It verifies the HMAC signature, decodes the standard envelope, gates
// on the two event types, and fans the payload to effects (first handled wins).
// A resolution goes to DomainEffect.Apply; a context request goes to the effects
// that implement ContextResponder.
//
// THE BODY IS A CLAIM, NOT EVIDENCE. The signature proves vrooli-events delivered
// it; the bus accepts an event from any caller with any source, so a body can be
// forged. With a Reader, the webhook reads the ask back from the Hub as the
// scenario's own ask-reader principal before any effect runs, and applies only
// what the Hub says: the ask must be the scenario's own (Hub source), closed in
// the state, with the answer, answerer, requester reference, subject key and
// correlation the body claims. The effect receives the Hub's values, so the
// correlation that locates the scenario's own record is the one the scenario
// raised, never one a forger chose. A body that disagrees with the Hub is refused
// (409, counted, logged) and, like any non-2xx, redelivered by the bus and
// dead-lettered after its attempts, where an operator can retry it. A Hub that
// cannot be read is 503, so a real approval is delayed and never lost to an
// outage. A scenario with no ask-reader credential yet applies the body as
// before, and every such application is counted and logged as unverified.
//
// Responses: 405 for non-POST, 413 for an over-cap body, 400 for an unreadable or
// undecodable body, 401 for a bad/absent signature, 200 {"ignored":true} for any
// other event, another scenario's ask, or an unclaimed resolution or context
// request, 409 for a claim the Hub does not confirm, 503 when the Hub cannot be
// read, 500 for an effect error (so the event is retried), and 200
// {"handled":true,...} when an effect claimed it.
//
// REPLAY SAFETY is delegated to each DomainEffect: the webhook itself does not
// deduplicate, so vrooli-events MAY redeliver the same resolution (e.g. after a
// lost ack, or a re-raise inside the dedup window). An effect MUST treat a
// resolution for an already-decided / unknown ask as a benign no-op and return
// (handled=true, nil) — never re-apply the effect and never return an error that
// would trigger a retry storm. See DomainEffect.Apply. The same holds for a
// context request already answered (ContextResponder.RespondToContext).
type Webhook struct {
	cfg WebhookConfig

	verified, unverified, unavailable atomic.Uint64
	mu                                sync.Mutex
	refused                           map[string]uint64
}

// NewWebhook builds the resolution webhook.
func NewWebhook(cfg WebhookConfig) *Webhook {
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	return &Webhook{cfg: cfg, refused: map[string]uint64{}}
}

// Stats is what the webhook has done since it started.
func (w *Webhook) Stats() WebhookStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	refused := make(map[string]uint64, len(w.refused))
	for reason, n := range w.refused {
		refused[reason] = n
	}
	return WebhookStats{Verified: w.verified.Load(), UnverifiedApplied: w.unverified.Load(), Unavailable: w.unavailable.Load(), Refused: refused}
}

func (w *Webhook) refuse(reason string) {
	w.mu.Lock()
	w.refused[reason]++
	w.mu.Unlock()
}

// check is how one claim fared against the Hub.
type check int

const (
	checkVerified   check = iota // the Hub confirmed the claim
	checkUnverified              // no ask-reader credential: apply the body, counted
	checkRefused                 // the Hub does not confirm the claim
	checkUnavailable
)

func (w *Webhook) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Read one byte past the cap so a body AT the cap still verifies but an
	// OVER-cap body is detected and rejected with an explicit 413 — otherwise a
	// truncated body fails the HMAC check and masquerades as a permanent 401.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	if err != nil {
		http.Error(rw, "event webhook body could not be read", http.StatusBadRequest)
		return
	}
	if len(body) > maxWebhookBody {
		http.Error(rw, "event webhook body exceeds the size limit", http.StatusRequestEntityTooLarge)
		return
	}
	secret := ""
	if w.cfg.Secret != nil {
		secret = w.cfg.Secret()
	}
	if !ValidSignature(body, r.Header.Get(SignatureHeader), secret) {
		http.Error(rw, "event webhook signature is invalid or unavailable", http.StatusUnauthorized)
		return
	}
	var envelope struct {
		EventID   string          `json:"event_id"`
		EventType string          `json:"event_type"`
		Source    string          `json:"source_scenario"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(rw, "invalid webhook body", http.StatusBadRequest)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	switch envelope.EventType {
	case AskResolvedEventType:
		w.serveResolution(rw, r, envelope.Payload)
	case AskContextRequestedEventType:
		w.serveContextRequest(rw, r, envelope.Payload)
	default:
		_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "not an ask resolution"})
	}
}

// ours reports whether a claimed ask belongs to this webhook's scenario. A claim
// about another scenario's ask is not read: ignoring a forged claim is safe, and
// every scenario receives every resolution on the bus.
func (w *Webhook) ours(source string) bool {
	return w.cfg.Source == "" || strings.TrimSpace(source) == w.cfg.Source
}

func (w *Webhook) serveResolution(rw http.ResponseWriter, r *http.Request, payload json.RawMessage) {
	var claimed DecisionResolution
	if err := json.Unmarshal(payload, &claimed); err != nil {
		http.Error(rw, "invalid webhook body", http.StatusBadRequest)
		return
	}
	if !w.ours(claimed.Source) {
		_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "not this scenario's ask"})
		return
	}
	resolution := claimed
	if !w.handlesAny(claimed.SourceEventType) {
		_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "no matching decision"})
		return
	}
	outcome, hubAsk, detail := w.readBack(r.Context(), claimed.AskID, "resolution")
	switch outcome {
	case checkVerified:
		verified, err := VerifiedResolution(claimed, hubAsk)
		if err != nil {
			w.reject(r.Context(), rw, "resolution", claimed.AskID, hubAsk.Question, err)
			return
		}
		w.verified.Add(1)
		resolution = verified
	case checkUnverified:
		w.unverifiedApplied(claimed.AskID, claimed.SourceEventType)
	case checkUnavailable:
		w.notRead(rw, claimed.AskID, detail)
		return
	default:
		w.reject(r.Context(), rw, "resolution", claimed.AskID, "", detail)
		return
	}
	for _, effect := range w.cfg.Effects {
		if effect == nil || !effect.Handles(resolution.SourceEventType) {
			continue
		}
		handled, rErr := effect.Apply(r.Context(), resolution)
		if rErr != nil {
			http.Error(rw, "apply decision resolution: "+rErr.Error(), http.StatusInternalServerError)
			return
		}
		if handled {
			_ = json.NewEncoder(rw).Encode(map[string]any{
				"handled": true,
				"ask_id":  resolution.AskID,
				"state":   resolution.State,
			})
			return
		}
	}
	_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "no matching decision"})
}

func (w *Webhook) handlesAny(sourceEventType string) bool {
	for _, effect := range w.cfg.Effects {
		if effect != nil && effect.Handles(sourceEventType) {
			return true
		}
	}
	return false
}

// serveContextRequest fans a context request to the effects that answer them.
func (w *Webhook) serveContextRequest(rw http.ResponseWriter, r *http.Request, payload json.RawMessage) {
	claimed, err := DecodeContextRequest(payload)
	if err != nil {
		http.Error(rw, "invalid context request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !w.ours(claimed.Source) {
		_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "not this scenario's ask"})
		return
	}
	request := claimed
	answers := false
	for _, effect := range w.cfg.Effects {
		if _, ok := effect.(ContextResponder); ok && effect.Handles(claimed.SourceEventType) {
			answers = true
		}
	}
	if !answers {
		_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "no producer answers this context request"})
		return
	}
	outcome, hubAsk, detail := w.readBack(r.Context(), claimed.AskID, "context request")
	switch outcome {
	case checkVerified:
		verified, err := VerifiedContextRequest(claimed, hubAsk)
		if err != nil {
			w.reject(r.Context(), rw, "context request", claimed.AskID, hubAsk.Question, err)
			return
		}
		w.verified.Add(1)
		request = verified
	case checkUnverified:
		w.unverifiedApplied(claimed.AskID, claimed.SourceEventType)
	case checkUnavailable:
		w.notRead(rw, claimed.AskID, detail)
		return
	default:
		w.reject(r.Context(), rw, "context request", claimed.AskID, "", detail)
		return
	}
	for _, effect := range w.cfg.Effects {
		responder, ok := effect.(ContextResponder)
		if !ok || !effect.Handles(request.SourceEventType) {
			continue
		}
		handled, err := responder.RespondToContext(r.Context(), request)
		if err != nil {
			http.Error(rw, "answer context request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if handled {
			_ = json.NewEncoder(rw).Encode(map[string]any{"handled": true, "ask_id": request.AskID, "request_id": request.RequestID})
			return
		}
	}
	_ = json.NewEncoder(rw).Encode(map[string]any{"ignored": true, "reason": "no producer answers this context request"})
}

// readBack reads the claimed ask at the Hub. detail is the reason for a refusal
// or for an unavailable Hub.
func (w *Webhook) readBack(ctx context.Context, askID, what string) (check, askreader.Ask, error) {
	if w.cfg.Reader == nil {
		return checkUnverified, askreader.Ask{}, nil
	}
	ask, err := w.cfg.Reader.ReadAsk(ctx, askID)
	switch {
	case err == nil:
	case errors.Is(err, askreader.ErrNotIssued):
		return checkUnverified, askreader.Ask{}, nil
	case errors.Is(err, askreader.ErrAskNotFound):
		return checkRefused, askreader.Ask{}, &RefusedError{Reason: ReasonNoSuchAsk, Detail: fmt.Sprintf("the Hub holds no ask %q for %s", askID, w.sourceName())}
	default:
		return checkUnavailable, askreader.Ask{}, err
	}
	if w.cfg.Source != "" && ask.Source != w.cfg.Source {
		return checkRefused, askreader.Ask{}, &RefusedError{Reason: ReasonOtherSource, Detail: fmt.Sprintf("the Hub says ask %q belongs to %q, not %q", askID, ask.Source, w.cfg.Source)}
	}
	return checkVerified, ask, nil
}

func (w *Webhook) sourceName() string {
	if w.cfg.Source == "" {
		return "this scenario"
	}
	return w.cfg.Source
}

func (w *Webhook) unverifiedApplied(askID, sourceEventType string) {
	w.unverified.Add(1)
	w.cfg.Logf("operatordecision: UNVERIFIED resolution applied for ask %s (%s): %s has no ask-reader credential, so the event body was trusted as delivered; the operator closes this with `%s`", askID, sourceEventType, w.sourceName(), serviceprincipal.IssueCommand)
}

// reject answers a claim the Hub does not confirm: 409, so the bus keeps it. A
// refused resolution that the Hub's own record contradicts is also reported to
// OnRefused, because it is someone applying an answer the operator did not give.
func (w *Webhook) reject(ctx context.Context, rw http.ResponseWriter, what, askID, question string, err error) {
	var refused *RefusedError
	if !errors.As(err, &refused) {
		refused = &RefusedError{Reason: ReasonUnconfirmed, Detail: err.Error()}
	}
	w.refuse(refused.Reason)
	w.cfg.Logf("operatordecision: REFUSED %s for ask %s: %s", what, askID, refused.Detail)
	if w.cfg.OnRefused != nil && what == "resolution" && (refused.Reason == ReasonMismatch || refused.Reason == ReasonAskOpen) {
		w.cfg.OnRefused(ctx, Refusal{Source: w.cfg.Source, AskID: askID, Reason: refused.Reason, Detail: refused.Detail, Question: question})
	}
	http.Error(rw, "the Hub does not confirm this "+what+": "+refused.Detail, http.StatusConflict)
}

// notRead answers a claim that could not be checked because the Hub could not be
// read: 503, so the bus redelivers it once the Hub is back.
func (w *Webhook) notRead(rw http.ResponseWriter, askID string, err error) {
	w.unavailable.Add(1)
	w.cfg.Logf("operatordecision: the Hub could not be read to verify ask %s; the bus will redeliver: %v", askID, err)
	http.Error(rw, "the Hub could not be read to verify this resolution; it will be redelivered", http.StatusServiceUnavailable)
}
