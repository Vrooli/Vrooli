package duplex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/session"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	openAILiveURL     = "wss://api.openai.com/v1/live/sessions"
	maxLiveMessage    = 4 << 20
	maxLiveAudioFrame = 1 << 20
)

var (
	ErrLiveFixtureActive    = errors.New("duplex: a GPT-Live output fixture is already active")
	ErrTextInputUnsupported = errors.New("duplex: GPT-Live client delegation does not document user text input")
	ErrUnsupportedLiveModel = errors.New("duplex: selected model has no sourced OpenAI GPT-Live pricing")
	liveCatalogOnce         sync.Once
	liveCatalog             *modelcatalog.Catalog
)

var ErrLiveInputDropped = errors.New("duplex: input arrived during the Live output interruption fixture and was dropped")

// KeyResolver is the sole credential boundary. Implementations should resolve
// the provider's key from the approved credential store.
type KeyResolver interface {
	ResolveOpenAIKey(context.Context) (string, error)
}

// OpenAILiveProvider speaks the documented primary Live WebSocket protocol.
// It contains no environment or credential-store access of its own.
type OpenAILiveProvider struct {
	Keys   KeyResolver
	URL    string // injectable only for deterministic contract tests
	Dialer *websocket.Dialer
}

func (p *OpenAILiveProvider) Open(ctx context.Context, req Request, sink Sink) (Connection, error) {
	if ctx == nil || sink == nil || p == nil || p.Keys == nil {
		return nil, errors.New("duplex: invalid OpenAI Live provider configuration")
	}
	// Model choice belongs to the model registry and the caller. A silent
	// provider-local default can bypass future catalog policy and make a stale
	// model ID look like an intentional selection.
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return nil, ErrUnsupportedLiveModel
	}
	if !validLiveModel(model) {
		return nil, ErrUnsupportedLiveModel
	}
	key, err := p.Keys.ResolveOpenAIKey(ctx)
	if err != nil {
		return nil, errors.New("duplex: OpenAI credential resolution failed")
	}
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("duplex: OpenAI credential is empty")
	}
	url := p.URL
	if url == "" {
		url = openAILiveURL
	}
	d := p.Dialer
	if d == nil {
		d = websocket.DefaultDialer
	}
	h := http.Header{}
	h.Set("Authorization", "Bearer "+key)
	c, _, err := d.DialContext(ctx, url, h)
	if err != nil {
		return nil, errors.New("duplex: OpenAI Live WebSocket connection failed")
	}
	c.SetReadLimit(maxLiveMessage)
	conn := &openAILiveConn{ws: c, sink: sink, model: model, done: make(chan struct{}), closeSignal: make(chan struct{}), closeComplete: make(chan struct{})}
	if err := startOpenAILiveSession(ctx, conn, req.Voice); err != nil {
		return nil, err
	}
	go conn.readLoop(ctx)
	return conn, nil
}

// startOpenAILiveSession owns the ordered start command and startup handshake.
// It must consume pre-session events in provider order before the read loop
// starts, and close the socket on every unsuccessful handshake.
func startOpenAILiveSession(ctx context.Context, conn *openAILiveConn, voice string) error {
	config := map[string]any{"type": "session.start", "event_id": uuid.NewString(), "session": map[string]any{
		"model": conn.model, "audio": map[string]any{"format": map[string]any{"type": "audio/pcm", "rate": 24000}, "output": map[string]any{"voice": defaultVoice(voice)}},
		"delegation": map[string]any{"type": "client"},
	}}
	if err := conn.write(ctx, config); err != nil {
		_ = conn.ws.Close()
		return errors.New("duplex: OpenAI Live start command failed")
	}
	startupCtx, startupCancel := context.WithTimeout(ctx, 15*time.Second)
	defer startupCancel()
	for {
		var ev liveEvent
		if err := conn.read(startupCtx, &ev); err != nil {
			_ = conn.ws.Close()
			return errors.New("duplex: OpenAI Live startup handshake failed")
		}
		if ev.Type == "session.started" {
			break
		}
		if ev.Type == "error" {
			_ = conn.ws.Close()
			return errors.New("duplex: OpenAI Live rejected session start")
		}
	}
	return nil
}

func defaultVoice(s string) string {
	if s == "" {
		return "marin"
	}
	return s
}

type liveEvent struct {
	Type          string `json:"type"`
	EventID       string `json:"event_id"`
	ClientEventID string `json:"client_event_id"`
	Reason        string `json:"reason"`
	Delta         string `json:"delta"`
	Text          string `json:"text"`
	Audio         string `json:"audio"`
	Response      struct {
		ID string `json:"id"`
	} `json:"response"`
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
	Usage *struct {
		Seconds uint64  `json:"seconds"`
		Active  *uint64 `json:"active_seconds"`
		Paused  *uint64 `json:"paused_seconds"`
	} `json:"usage"`
}

type openAILiveConn struct {
	ws             *websocket.Conn
	sink           Sink
	model          string
	writeMu        sync.Mutex
	opMu           sync.RWMutex
	outputMu       sync.RWMutex
	mu             sync.Mutex
	closing        bool
	terminal       bool
	fixtureActive  bool
	fixtureEpoch   uint64
	fixtureID      string
	fixtureWaiters map[string]chan struct{}
	closeErr       error
	finalUsage     ProviderUsage
	confirmed      bool
	done           chan struct{}
	closeSignal    chan struct{}
	closeComplete  chan struct{}
}

func (c *openAILiveConn) write(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(data) > maxLiveMessage {
		return errors.New("duplex: Live command too large")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

func (c *openAILiveConn) read(ctx context.Context, out *liveEvent) error {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.ws.SetReadDeadline(deadline)
	} else if ctx.Done() != nil {
		_ = c.ws.SetReadDeadline(time.Now().Add(15 * time.Second))
	} else {
		_ = c.ws.SetReadDeadline(time.Time{})
	}
	_, data, err := c.ws.ReadMessage()
	if err != nil {
		return err
	}
	if len(data) > maxLiveMessage {
		return errors.New("duplex: Live message too large")
	}
	return json.Unmarshal(data, out)
}

func (c *openAILiveConn) readLoop(ctx context.Context) {
	defer close(c.done)
	stopCloseOnCancel := context.AfterFunc(ctx, func() { _ = c.ws.Close() })
	defer stopCloseOnCancel()
	for {
		var ev liveEvent
		if err := c.read(ctx, &ev); err != nil {
			if ctx.Err() != nil {
				return
			}
			c.sink.Emit(Event{Err: errors.New("OpenAI Live disconnected before confirmed close")})
			return
		}
		if ev.Type == "session.closed" {
			c.mu.Lock()
			requested := c.closing
			c.terminal = true
			c.fixtureActive = false
			c.fixtureID = ""
			c.fixtureWaiters = nil
			if ev.Usage != nil && strings.TrimSpace(ev.Session.ID) != "" {
				active, paused := ev.Usage.Seconds, uint64(0)
				if ev.Usage.Active != nil {
					active = *ev.Usage.Active
				}
				if ev.Usage.Paused != nil {
					paused = *ev.Usage.Paused
				}
				if cost, ok := liveCostMicros(c.model, active, paused); ok {
					c.confirmed = true
					c.finalUsage = ProviderUsage{
						ProviderReceiptID: strings.TrimSpace(ev.Session.ID),
						Model:             c.model,
						Provider:          openAILiveProviderID,
						ActiveSeconds:     active,
						PausedSeconds:     paused,
						CostMicros:        cost,
					}
				}
			}
			c.mu.Unlock()
			close(c.closeSignal)
			_ = c.ws.Close()
			if !requested {
				c.sink.Emit(Event{Err: fmt.Errorf("GPT-Live session ended remotely (%s)", safeReason(ev.Reason))})
			}
			return
		}
		c.mapEvent(ev)
	}
}

func safeReason(reason string) string {
	switch reason {
	case "close_requested", "expired", "content", "remote_hangup", "connection_lost":
		return reason
	default:
		return "unknown reason"
	}
}

func (c *openAILiveConn) mapEvent(e liveEvent) {
	id := e.EventID
	if e.Type == "session.output_audio.fixture.played" {
		c.mu.Lock()
		clientID := e.ClientEventID
		if clientID == "" {
			clientID = c.fixtureID
		}
		if clientID != "" && clientID == c.fixtureID {
			c.fixtureActive = false
			c.fixtureID = ""
		}
		if waiter := c.fixtureWaiters[clientID]; waiter != nil {
			delete(c.fixtureWaiters, clientID)
			close(waiter)
		}
		c.mu.Unlock()
		return
	}
	// Serialize output delivery with Cancel's interruption boundary. Checking
	// fixtureActive and then unlocking before Emit allowed a delta to pass the
	// check, lose the race to Cancel, and arrive at the client after the
	// interruption had started. Cancel holds outputMu exclusively only while
	// changing interruption state; holding a read lock through the check and
	// delivery makes each delta wholly precede or follow that boundary without
	// preventing the read loop from receiving the fixture acknowledgment.
	isAssistantOutput := e.Type == "session.output_transcript.delta" || e.Type == "session.output_audio.delta"
	if isAssistantOutput {
		c.outputMu.RLock()
		defer c.outputMu.RUnlock()
		c.mu.Lock()
		interrupted := c.fixtureActive
		c.mu.Unlock()
		if interrupted {
			return
		}
	}
	switch e.Type {
	case "session.output_audio.delta":
		b, err := base64.StdEncoding.DecodeString(e.Delta)
		if err != nil {
			c.sink.Emit(Event{Err: errors.New("invalid audio payload from OpenAI Live")})
			return
		}
		c.sink.Emit(Event{Value: session.SessionEvent{EventID: id, Type: session.EventAudioOutput, AudioOutput: &session.AudioFrame{Bytes: b, Format: "pcm_s16le", SampleRate: 24000}}})
	case "session.output_transcript.delta":
		c.sink.Emit(Event{Value: session.SessionEvent{EventID: id, Type: session.EventAssistantDelta, AssistantDelta: &session.AssistantDelta{Text: e.Delta}}})
	case "session.input_transcript.delta":
		c.sink.Emit(Event{Value: session.SessionEvent{EventID: id, Type: session.EventTranscriptDelta, TranscriptDelta: &session.TranscriptDelta{Text: e.Delta}}})
	case "error":
		c.sink.Emit(Event{Err: errors.New("OpenAI Live returned an error")})
	}
}

func (c *openAILiveConn) SendAudio(ctx context.Context, f session.AudioFrame) error {
	if ctx == nil {
		return ErrNilContext
	}
	if len(f.Bytes) == 0 || len(f.Bytes) > maxLiveAudioFrame || len(f.Bytes)%2 != 0 || f.Format != "pcm_s16le" || f.SampleRate != 24000 {
		return errors.New("duplex: expected bounded mono PCM16LE 24kHz audio frame")
	}
	c.mu.Lock()
	epoch, active := c.fixtureEpoch, c.fixtureActive
	c.mu.Unlock()
	c.opMu.RLock()
	defer c.opMu.RUnlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	dropped := active || epoch != c.fixtureEpoch || c.fixtureActive
	terminal := c.terminal || c.closing
	c.mu.Unlock()
	if dropped {
		return ErrLiveInputDropped
	}
	if terminal {
		return ErrClosed
	}
	return c.write(ctx, map[string]any{"type": "session.input_audio.append", "event_id": uuid.NewString(), "audio": base64.StdEncoding.EncodeToString(f.Bytes)})
}

func (c *openAILiveConn) SendText(ctx context.Context, text string) error {
	if ctx == nil {
		return ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrTextInputUnsupported
}

func (c *openAILiveConn) Cancel(ctx context.Context, _ session.BargeInReason, _ string) error {
	if ctx == nil {
		return ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.outputMu.Lock()
	c.mu.Lock()
	if c.closing || c.terminal {
		c.mu.Unlock()
		c.outputMu.Unlock()
		return ErrClosed
	}
	if c.fixtureActive {
		c.mu.Unlock()
		c.outputMu.Unlock()
		return ErrLiveFixtureActive
	}
	c.fixtureEpoch++
	c.fixtureActive = true
	if c.fixtureWaiters == nil {
		c.fixtureWaiters = make(map[string]chan struct{})
	}
	eventID := uuid.NewString()
	c.fixtureID = eventID
	played := make(chan struct{})
	c.fixtureWaiters[eventID] = played
	c.mu.Unlock()
	// Do not hold outputMu during the provider acknowledgment wait: the
	// read loop must remain free to process session.output_audio.fixture.played.
	c.outputMu.Unlock()
	// 10 ms raw mono PCM16LE at 24 kHz. The documented fixture operation
	// interrupts current model speech and mutes input until fixture.played.
	const silenceSamples = 240
	if err := c.write(ctx, map[string]any{"type": "session.output_audio.fixture.play", "event_id": eventID, "audio": base64.StdEncoding.EncodeToString(make([]byte, silenceSamples*2))}); err != nil {
		c.mu.Lock()
		delete(c.fixtureWaiters, eventID)
		if c.fixtureID == eventID {
			c.fixtureActive = false
			c.fixtureID = ""
		}
		c.mu.Unlock()
		return err
	}
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case <-played:
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.fixtureWaiters, eventID)
		c.mu.Unlock()
		return ctx.Err()
	case <-c.closeSignal:
		c.mu.Lock()
		delete(c.fixtureWaiters, eventID)
		c.mu.Unlock()
		return ErrClosed
	case <-timer.C:
		c.mu.Lock()
		delete(c.fixtureWaiters, eventID)
		c.mu.Unlock()
		return errors.New("duplex: Live interruption fixture was not acknowledged")
	case <-c.done:
		c.mu.Lock()
		delete(c.fixtureWaiters, eventID)
		c.mu.Unlock()
		return ErrClosed
	}
}

func (c *openAILiveConn) Close() error {
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		<-c.closeComplete
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.closeErr
	}
	c.closing = true
	terminal := c.terminal
	c.mu.Unlock()
	defer close(c.closeComplete)
	if terminal {
		<-c.done
		var err error
		if _, confirmed := c.FinalProviderUsage(); !confirmed {
			err = errors.New("duplex: Live session closed without final usage")
		}
		c.mu.Lock()
		c.closeErr = err
		c.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := c.write(ctx, map[string]any{"type": "session.close", "event_id": uuid.NewString()})
	if err == nil {
		select {
		case <-c.closeSignal:
			if _, confirmed := c.FinalProviderUsage(); !confirmed {
				err = errors.New("duplex: Live session closed without final usage")
			}
			<-c.done
		case <-ctx.Done():
			err = fmt.Errorf("duplex: Live close was not confirmed")
		}
	}
	if err != nil {
		_ = c.ws.Close()
		<-c.done
	}
	c.mu.Lock()
	c.closeErr = err
	c.mu.Unlock()
	return err
}

func (c *openAILiveConn) FinalProviderUsage() (ProviderUsage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.finalUsage, c.confirmed
}

// OpenAI's published per-second prices are fractional in micros. Ceiling the
// combined exact numerator avoids undercharging due to integer truncation.
func liveCostMicros(modelID string, active, paused uint64) (uint64, bool) {
	liveCatalogOnce.Do(func() { liveCatalog = modelcatalog.Default() })
	model, ok := liveCatalog.ByID(modelID)
	if !ok || !validLiveModelRecord(model) {
		return 0, false
	}
	if *model.Pricing.ActiveMicrosPerMinute < 0 || *model.Pricing.StandbyMicrosPerMinute < 0 {
		return 0, false
	}
	activeRate, standbyRate := uint64(*model.Pricing.ActiveMicrosPerMinute), uint64(*model.Pricing.StandbyMicrosPerMinute)
	max := ^uint64(0)
	if activeRate > 0 && active > max/activeRate || standbyRate > 0 && paused > max/standbyRate {
		return max, true
	}
	activeCost, standbyCost := activeRate*active, standbyRate*paused
	if standbyCost > max-activeCost || activeCost+standbyCost > max-59 {
		return max, true
	}
	return (activeCost + standbyCost + 59) / 60, true
}

func validLiveModel(modelID string) bool {
	liveCatalogOnce.Do(func() { liveCatalog = modelcatalog.Default() })
	model, ok := liveCatalog.ByID(modelID)
	return ok && validLiveModelRecord(model)
}

func validLiveModelRecord(model modelcatalog.Model) bool {
	if model.Capability != modelcatalog.CapabilityDuplex || model.Pricing == nil || model.Pricing.Currency != "USD" || model.Pricing.Unit != "minute" || model.Pricing.ActiveMicrosPerMinute == nil || model.Pricing.StandbyMicrosPerMinute == nil || model.Pricing.BillingIncrementSeconds != 1 || strings.TrimSpace(model.Pricing.SourceURL) == "" {
		return false
	}
	if *model.Pricing.ActiveMicrosPerMinute < 0 || *model.Pricing.StandbyMicrosPerMinute < 0 {
		return false
	}
	owned := false
	for _, providerID := range model.ProviderIDs {
		if providerID == "openai-gpt-live" {
			owned = true
			break
		}
	}
	return owned
}
