// Package duplex defines an internal seam for providers that own a complete
// real-time audio conversation. It intentionally does not implement routing,
// authorization, billing, or a public transport.
package duplex

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"audio-tools/internal/session"
)

// ProviderKind is the protocol/session contract a provider implements. It is
// deliberately separate from capability names such as stt and tts: a duplex
// provider owns one interactive conversation and its terminal usage receipt.
type ProviderKind string

const ProviderKindRealtimeDuplex ProviderKind = "realtime-duplex"

var ErrRegisteredProviderUnavailable = errors.New("audio-tools/duplex: registered provider is unavailable")

// ProviderRegistration binds a stable catalog provider ID and kind to its
// runtime adapter. Registrations are immutable after construction.
type ProviderRegistration struct {
	ID       string
	Kind     ProviderKind
	Provider Provider
}

// ProviderRegistry resolves explicit provider choices without fallback. This
// is intentionally a narrow session router: privacy, eligibility, consent and
// spend admission remain owned by Coordinator's Admission implementation.
type ProviderRegistry struct {
	providers map[string]ProviderRegistration
}

func NewProviderRegistry(registrations ...ProviderRegistration) (*ProviderRegistry, error) {
	providers := make(map[string]ProviderRegistration, len(registrations))
	for _, registration := range registrations {
		registration.ID = strings.TrimSpace(registration.ID)
		if registration.ID == "" || registration.Kind == "" || registration.Provider == nil {
			return nil, errors.New("audio-tools/duplex: provider registration requires ID, kind, and adapter")
		}
		if registration.Kind != ProviderKindRealtimeDuplex {
			return nil, fmt.Errorf("audio-tools/duplex: unsupported provider kind %q", registration.Kind)
		}
		if _, exists := providers[registration.ID]; exists {
			return nil, fmt.Errorf("audio-tools/duplex: duplicate provider ID %q", registration.ID)
		}
		providers[registration.ID] = registration
	}
	return &ProviderRegistry{providers: providers}, nil
}

func (r *ProviderRegistry) Resolve(id string, kind ProviderKind) (Provider, error) {
	if r == nil {
		return nil, ErrRegisteredProviderUnavailable
	}
	registration, ok := r.providers[strings.TrimSpace(id)]
	if !ok || registration.Kind != kind {
		return nil, ErrRegisteredProviderUnavailable
	}
	return registration.Provider, nil
}

// Request describes the user-selected session settings passed to a duplex
// provider. Route and model selection remain the responsibility of the caller.
type Request struct {
	Voice          string
	Language       string
	Model          string
	ProviderID     string
	RemoteAllowed  bool
	Consent        bool
	ConsentReceipt string
	BudgetMicros   uint64
}

// Admission is the privacy and cost boundary in front of a duplex provider.
// Implementations resolve ProviderID against the live provider catalog, check
// the session's privacy posture and explicit consent, and reserve/validate its
// maximum spend against BudgetMicros before any provider connection is opened.
// It must fail closed when pricing or provider capability is unknown.
type Admission interface {
	Authorize(ctx context.Context, request Request) error
}

// Reservation is a bounded, provider-neutral hold against a shared spend
// ceiling. Settle records measured usage; Release returns a hold known to be
// unused. When provider usage is ambiguous after dispatch, callers must not
// Release: non-durable implementations retain the hold for their process
// lifetime, while durable implementations use DispatchReservation's explicit
// uncertain/reconciliation state.
type Reservation interface {
	Settle(actualMicros uint64) error
	Release()
}

// ReservingAdmission combines policy authorization and atomic reservation.
// Reserve must do both as one admission operation and must not contact a provider.
type ReservingAdmission interface {
	Admission
	Reserve(ctx context.Context, request Request) (Reservation, error)
}

// Event is emitted by a provider as its conversation progresses. A provider
// error terminates the session; otherwise Value carries one typed session
// event. The coordinator fills in missing event/session IDs.
type Event struct {
	Value session.SessionEvent
	Err   error
}

// Sink receives provider events. Implementations must be safe for concurrent
// calls and must not retain mutable event buffers after Emit returns.
type Sink interface {
	Emit(Event)
}

// Provider opens a single full-duplex conversation. Implementations are
// expected to honor ctx cancellation and return promptly from Close.
type Provider interface {
	Open(ctx context.Context, request Request, sink Sink) (Connection, error)
}

// Connection is the bidirectional live channel returned by Provider.Open.
// Implementations must allow concurrent send/cancel calls. The coordinator
// serializes Close against those calls.
type Connection interface {
	SendAudio(ctx context.Context, frame session.AudioFrame) error
	SendText(ctx context.Context, text string) error
	Cancel(ctx context.Context, reason session.BargeInReason, eventID string) error
	Close() error
}

// ProviderUsage is a provider-measured terminal usage record. ProviderReceiptID
// identifies the provider session/usage record for idempotency; for GPT-Live
// this is the session ID, not a financial invoice or an authenticated receipt.
type ProviderUsage struct {
	ProviderReceiptID string
	Model             string
	Provider          string
	ActiveSeconds     uint64
	PausedSeconds     uint64
	CostMicros        uint64
}

// FinalUsage is implemented only by connections whose protocol close handshake
// confirms a terminal provider usage record. Unknown usage must not be settled.
type FinalUsage interface{ FinalProviderUsage() (ProviderUsage, bool) }

// DispatchReservation is a durable reservation whose owner must authorize
// provider dispatch before the connection is opened. After authorization,
// failures retain the hold for reconciliation instead of releasing it.
type DispatchReservation interface {
	Reservation
	BeginDispatch(context.Context) (bool, error)
	SettleProviderUsage(context.Context, ProviderUsage) error
	RetainUncertain(context.Context) error
}
