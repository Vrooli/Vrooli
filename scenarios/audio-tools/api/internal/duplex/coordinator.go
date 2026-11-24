package duplex

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"audio-tools/internal/session"
)

var (
	ErrNilProvider                = errors.New("audio-tools/duplex: provider is nil")
	ErrNilProviderRegistry        = errors.New("audio-tools/duplex: provider registry is nil")
	ErrNilContext                 = errors.New("audio-tools/duplex: context is nil")
	ErrNilConnection              = errors.New("audio-tools/duplex: provider returned a nil connection")
	ErrNilAdmission               = errors.New("audio-tools/duplex: admission policy is nil")
	ErrReservationRequired        = errors.New("audio-tools/duplex: admission must reserve session budget")
	ErrConsentRevalidatorRequired = errors.New("audio-tools/duplex: provider requires active consent revalidation")
	ErrClosed                     = errors.New("audio-tools/duplex: session is closed")
)

// Coordinator starts provider-backed sessions. It deliberately accepts one
// already-selected provider and does not decide whether a route is permitted.
type Coordinator struct {
	provider               Provider
	providers              *ProviderRegistry
	admission              Admission
	consentRecheckInterval time.Duration
}

const defaultConsentRecheckInterval = 30 * time.Second

func NewCoordinator(provider Provider, admission Admission) (*Coordinator, error) {
	if provider == nil {
		return nil, ErrNilProvider
	}
	if admission == nil {
		return nil, ErrNilAdmission
	}
	return &Coordinator{provider: provider, admission: admission, consentRecheckInterval: defaultConsentRecheckInterval}, nil
}

// NewRoutedCoordinator constructs a coordinator that selects exactly the
// explicitly requested provider ID before admission. It never silently falls
// back to another provider, and admission still gates all provider contact.
func NewRoutedCoordinator(providers *ProviderRegistry, admission Admission) (*Coordinator, error) {
	if providers == nil {
		return nil, ErrNilProviderRegistry
	}
	if admission == nil {
		return nil, ErrNilAdmission
	}
	return &Coordinator{providers: providers, admission: admission, consentRecheckInterval: defaultConsentRecheckInterval}, nil
}

// Open starts one provider session. A provider startup error always closes the
// partially-created local session; it is never converted into an implicit
// fallback to another provider.
func (c *Coordinator) Open(ctx context.Context, request Request) (*Handle, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider := c.provider
	if c.providers != nil {
		var err error
		provider, err = c.providers.Resolve(request.ProviderID, ProviderKindRealtimeDuplex)
		if err != nil {
			return nil, err
		}
	}
	revalidateConsent := false
	if requirement, ok := c.admission.(ActiveConsentRequirement); ok && requirement.RequiresActiveConsentRevalidation(request) {
		if _, available := c.admission.(ActiveConsentAdmission); !available {
			return nil, ErrConsentRevalidatorRequired
		}
		revalidateConsent = true
	}
	reservation, err := c.admit(ctx, request)
	if err != nil {
		return nil, err
	}
	if err := authorizeReservationDispatch(ctx, reservation); err != nil {
		return nil, err
	}
	// Reservation and dispatch authorization can involve multiple remote
	// round-trips. Recheck consent after those calls and immediately before
	// provider contact so revocation during admission cannot start a new
	// metered session. Once LPBS has authorized dispatch, retain the hold for
	// reconciliation even though this process has not yet contacted the model.
	if revalidateConsent {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := c.admission.(ActiveConsentAdmission).RevalidateConsent(checkCtx, request)
		cancel()
		if err != nil {
			if durable, ok := reservation.(DispatchReservation); ok {
				_ = durable.RetainUncertain(context.WithoutCancel(ctx))
			} else if reservation != nil {
				reservation.Release()
			}
			return nil, ErrConsentInvalid
		}
	}
	if err := ctx.Err(); err != nil {
		if durable, ok := reservation.(DispatchReservation); ok {
			_ = durable.RetainUncertain(context.WithoutCancel(ctx))
		} else if reservation != nil {
			reservation.Release()
		}
		return nil, err
	}
	providerCtx, cancel := context.WithCancel(ctx)
	h := &Handle{
		ctx:         providerCtx,
		cancel:      cancel,
		done:        make(chan struct{}),
		closeDone:   make(chan struct{}),
		reservation: reservation,
	}
	h.voice = request.Voice
	h.language = request.Language
	h.session = session.New(session.Options{
		Transport: "duplex",
		Voice:     request.Voice,
		Language:  request.Language,
		CancelHook: func(reason session.BargeInReason, eventID string) {
			h.cancelAssistant(reason, eventID)
		},
	})
	go func() {
		select {
		case <-providerCtx.Done():
			_ = h.Close("context-canceled")
		case <-h.done:
		}
	}()

	conn, err := provider.Open(providerCtx, request, h)
	if err != nil {
		_ = h.fail("provider-start-failed", err)
		return nil, err
	}
	if conn == nil {
		err = ErrNilConnection
		_ = h.fail("provider-start-failed", err)
		return nil, err
	}
	h.mu.Lock()
	if h.closed || providerCtx.Err() != nil {
		h.mu.Unlock()
		_ = conn.Close()
		_ = h.Close("context-canceled")
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrClosed
	}
	h.connection = conn
	h.mu.Unlock()
	if revalidateConsent {
		revalidator := c.admission.(ActiveConsentAdmission)
		go monitorActiveConsent(providerCtx, h, revalidator, request, c.consentRecheckInterval)
	}
	return h, nil
}

// monitorActiveConsent makes revocation effective during a live remote
// session. An unavailable verifier is treated as invalid consent; the normal
// close path then settles confirmed usage or retains the durable hold as
// uncertain. No provider is contacted after a failed recheck.
func monitorActiveConsent(ctx context.Context, handle *Handle, admission ActiveConsentAdmission, request Request, interval time.Duration) {
	if interval <= 0 {
		interval = defaultConsentRecheckInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := admission.RevalidateConsent(checkCtx, request)
		cancel()
		if err != nil {
			_ = handle.fail("consent-revoked-or-unavailable", ErrConsentInvalid)
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-handle.done:
			return
		case <-ticker.C:
		}
	}
}

// admit runs policy checks and any durable budget reservation before session
// creation or provider contact. A failed reservation is released if an
// implementation returns one alongside its error.
func (c *Coordinator) admit(ctx context.Context, request Request) (Reservation, error) {
	reserving, ok := c.admission.(ReservingAdmission)
	if !ok {
		return nil, ErrReservationRequired
	}
	reservation, err := reserving.Reserve(ctx, request)
	if err != nil {
		if reservation != nil {
			reservation.Release()
		}
		return nil, err
	}
	if reservation == nil {
		return nil, ErrNilReservation
	}
	return reservation, nil
}

// authorizeReservationDispatch is a pre-provider fence for durable holds.
// An ambiguous result retains the hold for owner reconciliation.
func authorizeReservationDispatch(ctx context.Context, reservation Reservation) error {
	dispatch, ok := reservation.(DispatchReservation)
	if !ok {
		return nil
	}
	authorized, err := dispatch.BeginDispatch(ctx)
	if err != nil {
		// The owner may have committed dispatch even if its response was lost.
		_ = dispatch.RetainUncertain(context.WithoutCancel(ctx))
		return fmt.Errorf("duplex: durable dispatch authorization failed: %w", err)
	}
	if !authorized {
		_ = dispatch.RetainUncertain(context.WithoutCancel(ctx))
		return errors.New("duplex: durable reservation did not authorize dispatch")
	}
	return nil
}

// Handle owns one live provider connection and its internal session event bus.
type Handle struct {
	mu          sync.Mutex
	ioMu        sync.RWMutex
	closed      bool
	connection  Connection
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	closeDone   chan struct{}
	closeErr    error
	reservation Reservation
	session     *session.Session
	activeID    string
	voice       string
	language    string
}

func (h *Handle) ID() string       { return h.session.ID() }
func (h *Handle) Voice() string    { return h.voice }
func (h *Handle) Language() string { return h.language }

func (h *Handle) Events() *session.Session { return h.session }
func (h *Handle) Subscribe(ctx context.Context, buffer int) (string, <-chan session.SessionEvent, error) {
	return h.session.Subscribe(ctx, buffer)
}

// SendAudio forwards captured input to the provider. A send error closes the
// session so callers cannot continue on a partially failed conversation.
func (h *Handle) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	if ctx == nil {
		return ErrNilContext
	}
	conn, err := h.connectionForIO()
	if err != nil {
		return err
	}
	err = conn.SendAudio(ctx, cloneAudio(frame))
	h.ioMu.RUnlock()
	if err != nil {
		if errors.Is(err, ErrLiveInputDropped) {
			return err
		}
		_ = h.fail("audio-send-failed", err)
		return err
	}
	return nil
}

// SendText forwards user text to the provider without synthesizing a local
// echo. Provider events remain the authority for assistant output.
func (h *Handle) SendText(ctx context.Context, text string) error {
	if ctx == nil {
		return ErrNilContext
	}
	if text == "" {
		return errors.New("audio-tools/duplex: text is empty")
	}
	conn, err := h.connectionForIO()
	if err != nil {
		return err
	}
	err = conn.SendText(ctx, text)
	h.ioMu.RUnlock()
	if err != nil {
		if errors.Is(err, ErrTextInputUnsupported) {
			return err
		}
		_ = h.fail("text-send-failed", err)
		return err
	}
	return nil
}

func (h *Handle) connectionForIO() (Connection, error) {
	h.ioMu.RLock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.connection == nil {
		h.ioMu.RUnlock()
		return nil, ErrClosed
	}
	return h.connection, nil
}

// Emit satisfies Sink. Provider output is mapped onto the shared session
// event vocabulary. Provider errors are terminal and trigger cleanup.
func (h *Handle) Emit(event Event) {
	if event.Err != nil {
		h.emitProviderError(event.Err)
		return
	}
	ev := event.Value
	if ev.Type == "" || ev.Type == session.EventClosed || ev.Type == session.EventBargeInCancel || ev.Type == session.EventProviderError {
		h.emitProviderError(fmt.Errorf("unsupported provider event type %q", ev.Type))
		return
	}
	if ev.EventID == "" {
		ev.EventID = uuid.NewString()
	}
	if ev.Type == session.EventAssistantDelta {
		h.mu.Lock()
		h.activeID = ev.EventID
		h.mu.Unlock()
		h.session.MarkInflight(ev.EventID)
	}
	if ev.Type == session.EventAudioOutput && ev.AudioOutput != nil {
		frame := cloneAudio(*ev.AudioOutput)
		ev.AudioOutput = &frame
	}
	if ev.Type == session.EventAssistantFinal {
		h.mu.Lock()
		activeID := h.activeID
		h.activeID = ""
		h.mu.Unlock()
		if activeID != "" {
			h.session.ClearInflight(activeID)
		}
	}
	h.session.EmitEvent(ev)
}

// Provider callbacks may be synchronous with SendAudio/SendText. Publish the
// terminal error immediately, then close asynchronously so cleanup does not
// try to acquire the connection's exclusive close gate from inside one of its
// own in-flight callbacks.
func (h *Handle) emitProviderError(err error) {
	h.session.EmitEvent(session.SessionEvent{
		Type:          session.EventProviderError,
		ProviderError: &session.ProviderError{Message: err.Error()},
	})
	go func() { _ = h.Close("provider-error") }()
}

func (h *Handle) cancelAssistant(reason session.BargeInReason, eventID string) {
	conn, err := h.connectionForIO()
	if err != nil {
		return
	}
	err = conn.Cancel(h.ctx, reason, eventID)
	h.ioMu.RUnlock()
	if err != nil {
		_ = h.fail("cancel-failed", err)
	}
}

// BargeIn cancels the current assistant event through the provider and emits
// the normal session BargeInCancel event.
func (h *Handle) BargeIn(reason session.BargeInReason) { h.session.BargeIn(reason) }

func (h *Handle) fail(reason string, cause error) error {
	if cause != nil {
		h.session.EmitEvent(session.SessionEvent{
			Type:          session.EventProviderError,
			ProviderError: &session.ProviderError{Message: cause.Error()},
		})
	}
	closeErr := h.Close(reason)
	return errors.Join(cause, closeErr)
}

// Close is idempotent and releases the provider connection before closing the
// observer bus. It can safely race provider callbacks and context cancellation.
func (h *Handle) Close(reason string) error {
	h.mu.Lock()
	if h.closed {
		completed := h.closeDone
		h.mu.Unlock()
		<-completed
		h.mu.Lock()
		err := h.closeErr
		h.mu.Unlock()
		return err
	}
	h.closed = true
	conn := h.connection
	h.connection = nil
	close(h.done)
	h.mu.Unlock()

	h.cancel()
	h.ioMu.Lock()
	var err error
	if conn != nil {
		err = conn.Close()
		if h.reservation != nil {
			if durable, ok := h.reservation.(DispatchReservation); ok {
				if reporter, reports := conn.(FinalUsage); reports {
					if usage, confirmed := reporter.FinalProviderUsage(); confirmed && usage.ProviderReceiptID != "" {
						settleCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						settleErr := durable.SettleProviderUsage(settleCtx, usage)
						cancel()
						if settleErr != nil {
							retainCtx, retainCancel := context.WithTimeout(context.Background(), 10*time.Second)
							retainErr := durable.RetainUncertain(retainCtx)
							retainCancel()
							err = errors.Join(err, settleErr, retainErr)
						}
					} else {
						retainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						err = errors.Join(err, durable.RetainUncertain(retainCtx))
						cancel()
					}
				} else {
					retainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					err = errors.Join(err, durable.RetainUncertain(retainCtx))
					cancel()
				}
			} else if reporter, reports := conn.(FinalUsage); reports {
				if usage, confirmed := reporter.FinalProviderUsage(); confirmed {
					if settleErr := h.reservation.Settle(usage.CostMicros); settleErr != nil {
						// A failed settlement does not prove that the provider incurred
						// no charge. Keep the process-local hold unavailable instead
						// of silently returning ambiguous spend capacity.
						err = errors.Join(err, settleErr)
					}
				}
				// Missing or unconfirmed final usage is uncertain. A non-durable
				// reservation has no reconciliation owner, so retain its hold for
				// this process lifetime by leaving it reserved.
			}
		}
	} else if h.reservation != nil {
		if durable, ok := h.reservation.(DispatchReservation); ok {
			retainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err = durable.RetainUncertain(retainCtx)
			cancel()
		} else {
			h.reservation.Release()
		}
	}
	h.ioMu.Unlock()
	h.session.Close(reason)
	h.mu.Lock()
	h.closeErr = err
	close(h.closeDone)
	h.mu.Unlock()
	return err
}

func cloneAudio(frame session.AudioFrame) session.AudioFrame {
	frame.Bytes = append([]byte(nil), frame.Bytes...)
	return frame
}
