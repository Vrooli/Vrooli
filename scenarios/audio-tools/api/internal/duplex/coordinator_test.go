package duplex

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"

	"audio-tools/internal/providercatalog"
	"audio-tools/internal/session"
)

type fakeProvider struct {
	mu       sync.Mutex
	conn     *fakeConnection
	openErr  error
	request  Request
	openSeen bool
}

type fakeAdmission struct {
	err      error
	requests []Request
}

type authorizeOnlyAdmission struct{}

func (authorizeOnlyAdmission) Authorize(context.Context, Request) error { return nil }

type missingConsentRevalidatorAdmission struct{ fakeAdmission }

func (missingConsentRevalidatorAdmission) RequiresActiveConsentRevalidation(Request) bool {
	return true
}

func (a *fakeAdmission) Authorize(_ context.Context, request Request) error {
	a.requests = append(a.requests, request)
	return a.err
}

func (a *fakeAdmission) Reserve(ctx context.Context, request Request) (Reservation, error) {
	if err := a.Authorize(ctx, request); err != nil {
		return nil, err
	}
	return &testReservation{}, nil
}

type testReservation struct{}

func (*testReservation) Settle(uint64) error { return nil }
func (*testReservation) Release()            {}

func newTestCoordinator(t *testing.T, provider Provider) (*Coordinator, *fakeAdmission) {
	t.Helper()
	admission := &fakeAdmission{}
	coordinator, err := NewCoordinator(provider, admission)
	require.NoError(t, err)
	return coordinator, admission
}

func (p *fakeProvider) Open(_ context.Context, request Request, sink Sink) (Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.request = request
	p.openSeen = true
	if p.openErr != nil {
		return nil, p.openErr
	}
	p.conn = &fakeConnection{sink: sink}
	return p.conn, nil
}

type fakeConnection struct {
	mu           sync.Mutex
	sink         Sink
	audio        []session.AudioFrame
	texts        []string
	cancels      []cancelCall
	closeCalls   int
	audioErr     error
	emitError    error
	textErr      error
	cancelErr    error
	closed       bool
	closeStarted chan struct{}
	closeRelease <-chan struct{}
}

type cancelCall struct {
	reason  session.BargeInReason
	eventID string
}

func (c *fakeConnection) SendAudio(_ context.Context, frame session.AudioFrame) error {
	c.mu.Lock()
	c.audio = append(c.audio, cloneAudio(frame))
	err := c.audioErr
	emitError := c.emitError
	c.mu.Unlock()
	if err == nil {
		if emitError != nil {
			c.sink.Emit(Event{Err: emitError})
		} else {
			c.sink.Emit(Event{Value: session.SessionEvent{Type: session.EventTranscriptDelta, TranscriptDelta: &session.TranscriptDelta{Text: "heard"}}})
			c.sink.Emit(Event{Value: session.SessionEvent{Type: session.EventAudioOutput, AudioOutput: &session.AudioFrame{Bytes: []byte{4, 5}, Format: "pcm_s16le", SampleRate: 24000}}})
		}
	}
	return err
}

func (c *fakeConnection) SendText(_ context.Context, text string) error {
	c.mu.Lock()
	c.texts = append(c.texts, text)
	err := c.textErr
	c.mu.Unlock()
	if err == nil {
		c.sink.Emit(Event{Value: session.SessionEvent{EventID: "assistant-1", Type: session.EventAssistantDelta, AssistantDelta: &session.AssistantDelta{Text: "response"}}})
	}
	return err
}

func (c *fakeConnection) Cancel(_ context.Context, reason session.BargeInReason, eventID string) error {
	c.mu.Lock()
	c.cancels = append(c.cancels, cancelCall{reason: reason, eventID: eventID})
	err := c.cancelErr
	c.mu.Unlock()
	return err
}

func (c *fakeConnection) Close() error {
	c.mu.Lock()
	c.closeCalls++
	c.closed = true
	if c.closeStarted != nil {
		close(c.closeStarted)
		c.closeStarted = nil
	}
	release := c.closeRelease
	c.mu.Unlock()
	if release != nil {
		<-release
	}
	return nil
}

func TestCoordinatorOpenBidirectionalEventsBargeInAndClose(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, admission := newTestCoordinator(t, provider)
	handle, err := coordinator.Open(context.Background(), Request{Voice: "warm", Language: "en", Model: "fixture"})
	require.NoError(t, err)
	require.Len(t, admission.requests, 1)
	require.True(t, provider.openSeen)
	require.Equal(t, Request{Voice: "warm", Language: "en", Model: "fixture"}, provider.request)
	require.Equal(t, "duplex", handle.Events().Transport())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, events, err := handle.Subscribe(ctx, 16)
	require.NoError(t, err)

	inbound := session.AudioFrame{Bytes: []byte{1, 2, 3}, Format: "pcm_s16le", SampleRate: 16000, Sequence: 7}
	require.NoError(t, handle.SendAudio(context.Background(), inbound))
	require.NoError(t, handle.SendText(context.Background(), "hello"))
	require.Eventually(t, func() bool {
		provider.conn.mu.Lock()
		defer provider.conn.mu.Unlock()
		return len(provider.conn.audio) == 1 && len(provider.conn.texts) == 1
	}, time.Second, time.Millisecond)
	// The provider owns the input buffer after SendAudio returns, so the
	// coordinator passes a stable copy.
	inbound.Bytes[0] = 99
	provider.conn.mu.Lock()
	require.Equal(t, byte(1), provider.conn.audio[0].Bytes[0])
	provider.conn.mu.Unlock()

	seenTranscript, seenOutput, seenAssistant := false, false, false
	for i := 0; i < 3; i++ {
		select {
		case event := <-events:
			require.Equal(t, handle.ID(), event.SessionID)
			switch event.Type {
			case session.EventTranscriptDelta:
				seenTranscript = event.TranscriptDelta.Text == "heard"
			case session.EventAudioOutput:
				seenOutput = string(event.AudioOutput.Bytes) == string([]byte{4, 5}) && event.AudioOutput.SampleRate == 24000
			case session.EventAssistantDelta:
				seenAssistant = event.AssistantDelta.Text == "response"
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for provider events")
		}
	}
	require.True(t, seenTranscript)
	require.True(t, seenOutput)
	require.True(t, seenAssistant)

	handle.BargeIn(session.BargeInVAD)
	select {
	case event := <-events:
		require.Equal(t, session.EventBargeInCancel, event.Type)
		require.Equal(t, "assistant-1", event.BargeInCancel.CanceledEventID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for barge-in event")
	}
	provider.conn.mu.Lock()
	require.Equal(t, []cancelCall{{reason: session.BargeInVAD, eventID: "assistant-1"}}, provider.conn.cancels)
	provider.conn.mu.Unlock()

	require.NoError(t, handle.Close("test-close"))
	require.NoError(t, handle.Close("duplicate-close"))
	select {
	case event, ok := <-events:
		require.True(t, ok)
		require.Equal(t, session.EventClosed, event.Type)
		require.Equal(t, "test-close", event.Closed.Reason)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal close event")
	}
	_, ok := <-events
	require.False(t, ok, "event stream should close after terminal event")
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	provider.conn.mu.Unlock()
}

func TestCoordinatorRequiresAdmissionAndDenialPrecedesProviderContact(t *testing.T) {
	provider := &fakeProvider{}
	_, err := NewCoordinator(provider, nil)
	require.ErrorIs(t, err, ErrNilAdmission)

	denied := errors.New("remote audio is not permitted by this privacy policy")
	admission := &fakeAdmission{err: denied}
	coordinator, err := NewCoordinator(provider, admission)
	require.NoError(t, err)
	request := Request{ProviderID: "openai-realtime", Model: "realtime-model", Consent: true, RemoteAllowed: false, BudgetMicros: 1000}
	handle, err := coordinator.Open(context.Background(), request)
	require.Nil(t, handle)
	require.ErrorIs(t, err, denied)
	require.Equal(t, []Request{request}, admission.requests)
	require.False(t, provider.openSeen, "denied requests must not contact a provider")
}

func TestCoordinatorRequiresBudgetReservationBeforeProviderContact(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, err := NewCoordinator(provider, authorizeOnlyAdmission{})
	require.NoError(t, err)
	handle, err := coordinator.Open(context.Background(), Request{BudgetMicros: 100})
	require.Nil(t, handle)
	require.ErrorIs(t, err, ErrReservationRequired)
	require.False(t, provider.openSeen, "authorization without a budget hold must not reach provider")
}

func TestCoordinatorProviderEventErrorClosesSession(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	var err error
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	_, events, err := handle.Subscribe(context.Background(), 4)
	require.NoError(t, err)

	handle.Emit(Event{Err: errors.New("upstream disconnected")})
	first := <-events
	require.Equal(t, session.EventProviderError, first.Type)
	require.Contains(t, first.ProviderError.Message, "upstream disconnected")
	second := <-events
	require.Equal(t, session.EventClosed, second.Type)
	require.Equal(t, "provider-error", second.Closed.Reason)
	_, ok := <-events
	require.False(t, ok)
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	provider.conn.mu.Unlock()
}

func TestCoordinatorCancellationErrorAndParentCancellationCleanUp(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	var err error
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	_, events, err := handle.Subscribe(context.Background(), 4)
	require.NoError(t, err)
	provider.conn.cancelErr = errors.New("cancel rejected")
	require.NoError(t, handle.SendText(context.Background(), "hello"))
	assistant := <-events
	require.Equal(t, session.EventAssistantDelta, assistant.Type)
	handle.BargeIn(session.BargeInExplicit)
	require.Equal(t, session.EventProviderError, (<-events).Type)
	require.Equal(t, session.EventClosed, (<-events).Type)
	_, ok := <-events
	require.False(t, ok)
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	provider.conn.mu.Unlock()

	parent, cancel := context.WithCancel(context.Background())
	provider = &fakeProvider{}
	coordinator, _ = newTestCoordinator(t, provider)
	handle, err = coordinator.Open(parent, Request{})
	require.NoError(t, err)
	_, events, err = handle.Subscribe(context.Background(), 2)
	require.NoError(t, err)
	cancel()
	select {
	case event := <-events:
		require.Equal(t, session.EventClosed, event.Type)
		require.Equal(t, "context-canceled", event.Closed.Reason)
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not close provider session")
	}
	_, ok = <-events
	require.False(t, ok)
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	provider.conn.mu.Unlock()
}

func TestCoordinatorSynchronousProviderErrorDoesNotDeadlockSend(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	var err error
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	_, events, err := handle.Subscribe(context.Background(), 4)
	require.NoError(t, err)
	provider.conn.emitError = errors.New("provider callback failed")

	sendDone := make(chan error, 1)
	go func() { sendDone <- handle.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{1}}) }()
	select {
	case err := <-sendDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("SendAudio deadlocked during synchronous provider error callback")
	}
	require.Equal(t, session.EventProviderError, (<-events).Type)
	require.Equal(t, session.EventClosed, (<-events).Type)
	_, ok := <-events
	require.False(t, ok)
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	provider.conn.mu.Unlock()
}

func TestCoordinatorStartFailureAndSendFailureCleanUp(t *testing.T) {
	startErr := errors.New("cannot connect")
	coordinator, err := NewCoordinator(&fakeProvider{openErr: startErr}, &fakeAdmission{})
	require.NoError(t, err)
	handle, err := coordinator.Open(context.Background(), Request{})
	require.Nil(t, handle)
	require.ErrorIs(t, err, startErr)

	provider := &fakeProvider{}
	coordinator, err = NewCoordinator(provider, &fakeAdmission{})
	require.NoError(t, err)
	handle, err = coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	provider.conn.audioErr = errors.New("write failed")
	err = handle.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{1}})
	require.ErrorContains(t, err, "write failed")
	require.ErrorIs(t, handle.SendText(context.Background(), "later"), ErrClosed)
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	require.True(t, provider.conn.closed)
	provider.conn.mu.Unlock()
}

func TestCoordinatorCloseSerializesWithSendAndProviderEvents(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	var err error
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	_, events, err := handle.Subscribe(context.Background(), 256)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 40; j++ {
				_ = handle.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{byte(i), byte(j)}})
				handle.Emit(Event{Value: session.SessionEvent{Type: session.EventTranscriptDelta, TranscriptDelta: &session.TranscriptDelta{Text: "race"}}})
			}
		}(i)
	}
	closeDone := make(chan struct{})
	go func() {
		defer close(closeDone)
		_ = handle.Close("raced-close")
	}()
	wg.Wait()
	<-closeDone
	for range events {
	}
	provider.conn.mu.Lock()
	require.Equal(t, 1, provider.conn.closeCalls)
	require.True(t, provider.conn.closed)
	provider.conn.mu.Unlock()
}

func TestConcurrentCloseWaitsForProviderCleanup(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	started := make(chan struct{})
	release := make(chan struct{})
	provider.conn.mu.Lock()
	provider.conn.closeStarted = started
	provider.conn.closeRelease = release
	provider.conn.mu.Unlock()

	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- handle.Close("first") }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("provider cleanup did not start")
	}
	go func() { second <- handle.Close("second") }()
	select {
	case err := <-second:
		t.Fatalf("concurrent Close returned before cleanup completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-first)
	require.NoError(t, <-second)
}

func TestUnsupportedTextInputDoesNotCloseAudioSession(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	provider.conn.mu.Lock()
	provider.conn.textErr = ErrTextInputUnsupported
	provider.conn.mu.Unlock()
	require.ErrorIs(t, handle.SendText(context.Background(), "hello"), ErrTextInputUnsupported)
	require.NoError(t, handle.SendAudio(context.Background(), session.AudioFrame{Bytes: []byte{1, 0}, Format: "pcm_s16le", SampleRate: 24000}))
	provider.conn.mu.Lock()
	require.False(t, provider.conn.closed)
	provider.conn.mu.Unlock()
	require.NoError(t, handle.Close("test"))
}

func TestFixtureSuppressedAudioDoesNotCloseAudioSession(t *testing.T) {
	provider := &fakeProvider{}
	coordinator, _ := newTestCoordinator(t, provider)
	handle, err := coordinator.Open(context.Background(), Request{})
	require.NoError(t, err)
	frame := session.AudioFrame{Bytes: []byte{1, 0}, Format: "pcm_s16le", SampleRate: 24000}
	provider.conn.mu.Lock()
	provider.conn.audioErr = ErrLiveInputDropped
	provider.conn.mu.Unlock()
	require.ErrorIs(t, handle.SendAudio(context.Background(), frame), ErrLiveInputDropped)
	provider.conn.mu.Lock()
	provider.conn.audioErr = nil
	provider.conn.mu.Unlock()
	require.NoError(t, handle.SendAudio(context.Background(), frame))
	provider.conn.mu.Lock()
	require.False(t, provider.conn.closed)
	provider.conn.mu.Unlock()
	require.NoError(t, handle.Close("test"))
}

func TestCoordinatorRejectsMissingDeclaredConsentRevalidatorBeforeProviderContact(t *testing.T) {
	provider := &fakeProvider{}
	admission := &missingConsentRevalidatorAdmission{}
	coordinator, err := NewCoordinator(provider, admission)
	require.NoError(t, err)
	_, err = coordinator.Open(context.Background(), Request{ProviderID: "fixture-remote-voice"})
	require.ErrorIs(t, err, ErrConsentRevalidatorRequired)
	require.False(t, provider.openSeen, "provider must not open when active consent cannot be revalidated")
	require.Empty(t, admission.requests, "a spend hold/admission must not be attempted before revalidation availability is known")
}

type revocableConsentVerifier struct {
	mu      sync.Mutex
	revoked bool
	source  string
}

func (v *revocableConsentVerifier) Verify(_ context.Context, subject, receipt, source string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.revoked || subject != "human" || receipt != "opaque" || source != v.source {
		return ErrConsentInvalid
	}
	return nil
}

func TestCoordinatorRevalidatesDeclaredConsentForNonOpenAIProvider(t *testing.T) {
	const providerID, modelID, consentSource = "fixture-remote-voice", "fixture-live-model", "fixture-consent-source"
	provider := &fakeProvider{}
	verifier := &revocableConsentVerifier{source: consentSource}
	policy := NewAdmissionPolicy(NewBudget(5000), 4000, func(provider, model string) bool {
		return provider == providerID && model == modelID
	}).WithConsentVerifier(verifier)
	policy.consentRequirement = func(id string) (providercatalog.ConsentRequirement, bool) {
		if id != providerID {
			return providercatalog.ConsentRequirement{}, false
		}
		return providercatalog.ConsentRequirement{Source: providercatalog.ConsentSource(consentSource), RevalidateActiveSessions: true}, true
	}
	coordinator, err := NewCoordinator(provider, policy)
	require.NoError(t, err)
	coordinator.consentRecheckInterval = 5 * time.Millisecond
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	request := validAdmissionRequest()
	request.ProviderID, request.Model = providerID, modelID
	handle, err := coordinator.Open(ctx, request)
	require.NoError(t, err)
	verifier.mu.Lock()
	verifier.revoked = true
	verifier.mu.Unlock()
	select {
	case <-handle.done:
	case <-time.After(time.Second):
		t.Fatal("revoked source-bound consent did not close the active session")
	}
	provider.conn.mu.Lock()
	require.True(t, provider.conn.closed, "provider connection must close on consent revocation")
	provider.conn.mu.Unlock()
}
