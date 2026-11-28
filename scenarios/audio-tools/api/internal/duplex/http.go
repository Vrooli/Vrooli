package duplex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"audio-tools/internal/httporigin"
	"audio-tools/internal/session"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/vrooli/api-core/identity"
)

const (
	maxClientFrame   = 1 << 20
	maxClientText    = 16 << 10
	maxClientMessage = 2 << 20
)

// HTTPModule exposes the bounded browser transport for an already-composed
// coordinator. Production should mount it only when its admission dependencies
// are available; admission still runs before provider contact on every open.
func HTTPModule(coordinator *Coordinator) func(*mux.Router) {
	return func(r *mux.Router) {
		r.HandleFunc("/api/v1/duplex/sessions", func(w http.ResponseWriter, req *http.Request) {
			serveWebSocket(coordinator, w, req)
		}).Methods(http.MethodGet)
	}
}

type openMessage struct {
	Type           string `json:"type"`
	Voice          string `json:"voice"`
	Language       string `json:"language"`
	Model          string `json:"model"`
	ProviderID     string `json:"provider_id"`
	RemoteAllowed  bool   `json:"remote_allowed"`
	Consent        bool   `json:"consent"`
	ConsentReceipt string `json:"consent_receipt"`
	BudgetMicros   uint64 `json:"budget_micros"`
}

type clientMessage struct {
	Type        string `json:"type"`
	AudioBase64 string `json:"audio_base64,omitempty"`
	Format      string `json:"format,omitempty"`
	SampleRate  int32  `json:"sample_rate,omitempty"`
	Sequence    uint64 `json:"sequence,omitempty"`
	Text        string `json:"text,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

func serveWebSocket(coordinator *Coordinator, w http.ResponseWriter, req *http.Request) {
	ctx, err := verifiedWebSocketContext(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if coordinator == nil {
		http.Error(w, "duplex service unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := upgradeWebSocket(w, req)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(maxClientMessage)
	request, err := readOpenRequest(conn)
	if err != nil {
		writeSocketError(conn, "invalid session request")
		return
	}
	handle, err := coordinator.Open(ctx, request)
	if err != nil {
		writeSocketError(conn, "session admission or startup failed")
		return
	}
	// The browser must not send captured audio until provider startup and all
	// admission checks have succeeded. Keep this acknowledgment ahead of the
	// session event pump so its meaning is unambiguous to clients.
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteJSON(map[string]string{"type": "ready"}); err != nil {
		_ = handle.Close("client-disconnected")
		return
	}
	serveAdmittedSession(ctx, conn, handle)
}

func verifiedWebSocketContext(req *http.Request) (context.Context, error) {
	principal, ok := identity.PrincipalFromContext(req.Context())
	if !ok || !principal.Verified || principal.Kind != identity.ActorHuman || principal.Source != identity.SourceConsumerAccessToken || principal.Subject == "" {
		return nil, errors.New("verified human identity required")
	}
	// Preserve the exact verified LPBS bearer (header or supported cookie) for
	// the wallet call. The helper rejects all other identity sources.
	ctx, err := WithVerifiedHumanRequest(req.Context(), req)
	if err != nil {
		return nil, errors.New("verified human bearer required")
	}
	return ctx, nil
}

func upgradeWebSocket(w http.ResponseWriter, req *http.Request) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{HandshakeTimeout: 10 * time.Second, ReadBufferSize: 4096, WriteBufferSize: 4096, CheckOrigin: func(r *http.Request) bool {
		// Cross-origin browser access is not authorized by the websocket
		// handshake. Same-origin requests and non-browser clients are allowed.
		origin := r.Header.Get("Origin")
		return origin == "" || (origin != "null" && sameOrigin(r, origin))
	}}
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func readOpenRequest(conn *websocket.Conn) (Request, error) {
	kind, payload, err := readOpenEnvelope(conn)
	if err != nil || kind != websocket.TextMessage || len(payload) > maxClientText {
		return Request{}, errors.New("invalid session request")
	}
	var start openMessage
	if json.Unmarshal(payload, &start) != nil || start.Type != "open" {
		return Request{}, errors.New("first message must open a session")
	}
	return Request{Voice: start.Voice, Language: start.Language, Model: start.Model, ProviderID: start.ProviderID, RemoteAllowed: start.RemoteAllowed, Consent: start.Consent, ConsentReceipt: start.ConsentReceipt, BudgetMicros: start.BudgetMicros}, nil
}

func serveAdmittedSession(ctx context.Context, conn *websocket.Conn, handle *Handle) {
	_, events, err := handle.Subscribe(ctx, 32)
	if err != nil {
		_ = handle.Close("client-disconnected")
		writeSocketError(conn, "unable to subscribe to session")
		return
	}
	writeDone := make(chan struct{})
	var writeMu sync.Mutex
	writeError := socketErrorWriter(conn, &writeMu)
	// Every return after the event pump starts must close the handle (which
	// terminates the event stream) and join the writer before the socket closes.
	defer func() {
		_ = handle.Close("client-disconnected")
		<-writeDone
	}()
	startSessionEventWriter(ctx, conn, events, writeDone, &writeMu)
	readClientMessages(ctx, conn, handle, writeError)
}

func socketErrorWriter(conn *websocket.Conn, mu *sync.Mutex) func(string) {
	return func(message string) {
		mu.Lock()
		defer mu.Unlock()
		writeSocketError(conn, message)
	}
}

func startSessionEventWriter(ctx context.Context, conn *websocket.Conn, events <-chan session.SessionEvent, done chan<- struct{}, mu *sync.Mutex) {
	go func() {
		defer close(done)
		// The request goroutine can be blocked in ReadMessage while the event
		// stream ends (provider failure, revocation, or parent cancellation).
		// Closing the socket here interrupts that read and lets the handler join
		// its reader/writer cleanup deterministically.
		defer conn.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event, open := <-events:
				if !open {
					return
				}
				mu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				writeErr := conn.WriteJSON(event)
				mu.Unlock()
				if writeErr != nil {
					return
				}
				if event.Type == session.EventClosed {
					return
				}
			}
		}
	}()
}

func readClientMessages(ctx context.Context, conn *websocket.Conn, handle *Handle, writeError func(string)) {
	for {
		kind, data, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}
		if kind != websocket.TextMessage || len(data) > maxClientMessage {
			writeError("client messages must be bounded JSON text")
			return
		}
		var message clientMessage
		if json.Unmarshal(data, &message) != nil {
			writeError("invalid client message")
			return
		}
		if !handleClientMessage(ctx, handle, message, writeError) {
			return
		}
	}
}

func handleClientMessage(ctx context.Context, handle *Handle, message clientMessage, writeError func(string)) bool {
	switch message.Type {
	case "audio":
		return sendAudioMessage(ctx, handle, message, writeError)
	case "text":
		if message.Text == "" || len(message.Text) > maxClientText {
			writeError("invalid text input")
			return false
		}
		if handle.SendText(ctx, message.Text) != nil {
			writeError("text input unavailable")
			return false
		}
		return true
	case "barge_in":
		reason := session.BargeInExplicit
		if message.Reason == string(session.BargeInVAD) {
			reason = session.BargeInVAD
		}
		handle.BargeIn(reason)
		return true
	case "close":
		return false
	default:
		writeError("unsupported client message type")
		return false
	}
}

func sendAudioMessage(ctx context.Context, handle *Handle, message clientMessage, writeError func(string)) bool {
	decoded, err := base64.StdEncoding.DecodeString(message.AudioBase64)
	if err != nil || len(decoded) == 0 || len(decoded) > maxClientFrame {
		writeError("invalid audio frame")
		return false
	}
	return handle.SendAudio(ctx, session.AudioFrame{Bytes: decoded, Format: message.Format, SampleRate: message.SampleRate, Sequence: message.Sequence}) == nil
}

func readOpenEnvelope(conn *websocket.Conn) (int, []byte, error) {
	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	kind, payload, err := conn.ReadMessage()
	// The timeout protects only the initial request envelope. Leaving it on
	// the connection would silently cap every admitted conversation at 15s.
	_ = conn.SetReadDeadline(time.Time{})
	return kind, payload, err
}

func sameOrigin(req *http.Request, origin string) bool {
	return httporigin.Matches(req, origin)
}

func writeSocketError(conn *websocket.Conn, message string) {
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_ = conn.WriteJSON(map[string]string{"type": "error", "message": message})
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, message), time.Now().Add(time.Second))
}
