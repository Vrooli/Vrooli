package websocket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"test-genie/internal/structure/types"

	"github.com/gorilla/websocket"
)

// Validator validates WebSocket endpoints.
type Validator interface {
	// Validate performs all WebSocket validation checks.
	Validate(ctx context.Context) ValidationResult
}

// Dialer abstracts WebSocket dialing for testing.
type Dialer interface {
	DialContext(ctx context.Context, url string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

// ValidationResult contains the outcome of WebSocket validation.
type ValidationResult struct {
	types.Result

	// Endpoint is the WebSocket endpoint that was checked.
	Endpoint string

	// ConnectionTimeMs is the time to establish the connection in milliseconds.
	ConnectionTimeMs int64

	// MessageRoundTripMs is the time for a ping-pong round trip (if tested).
	MessageRoundTripMs int64
}

// Config holds configuration for WebSocket validation.
type Config struct {
	// URL is the WebSocket URL (e.g., "ws://localhost:8080/ws").
	URL string

	// MaxConnectionMs is the maximum acceptable connection time in milliseconds (default: 2000).
	MaxConnectionMs int64

	// PingMessage is the message to send for ping test (empty to skip).
	PingMessage string

	// ExpectedPongPattern is a substring expected in the pong response.
	ExpectedPongPattern string

	// ReadTimeout is the timeout for reading messages (default: 5s).
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for writing messages (default: 5s).
	WriteTimeout time.Duration
}

// validator implements the Validator interface.
type validator struct {
	config    Config
	dialer    Dialer
	logWriter io.Writer
}

// Option configures a validator.
type Option func(*validator)

// defaultDialer wraps gorilla/websocket.Dialer.
type defaultDialer struct {
	*websocket.Dialer
}

func (d *defaultDialer) DialContext(ctx context.Context, url string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	return d.Dialer.DialContext(ctx, url, requestHeader)
}

// New creates a new WebSocket validator.
func New(config Config, opts ...Option) Validator {
	// Apply defaults
	if config.MaxConnectionMs == 0 {
		config.MaxConnectionMs = 2000
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 5 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 5 * time.Second
	}

	v := &validator{
		config: config,
		dialer: &defaultDialer{
			Dialer: &websocket.Dialer{
				HandshakeTimeout: time.Duration(config.MaxConnectionMs) * time.Millisecond,
			},
		},
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// WithLogger sets the log writer.
func WithLogger(w io.Writer) Option {
	return func(v *validator) {
		v.logWriter = w
	}
}

// WithDialer sets a custom WebSocket dialer (for testing).
func WithDialer(dialer Dialer) Option {
	return func(v *validator) {
		v.dialer = dialer
	}
}

// Validate performs all WebSocket validation checks.
func (v *validator) Validate(ctx context.Context) ValidationResult {
	if err := ctx.Err(); err != nil {
		return ValidationResult{
			Result: types.FailSystem(err, "Context cancelled"),
		}
	}

	var observations []types.Observation
	observations = append(observations, types.NewSectionObservation("🔌", "Validating WebSocket connection..."))

	// Extract endpoint path from URL for display
	endpoint := v.config.URL
	if idx := strings.Index(endpoint, "://"); idx != -1 {
		if pathIdx := strings.Index(endpoint[idx+3:], "/"); pathIdx != -1 {
			endpoint = endpoint[idx+3+pathIdx:]
		}
	}

	v.logStep("Connecting to WebSocket: %s", v.config.URL)

	// Attempt connection
	start := time.Now()
	conn, resp, err := v.dialer.DialContext(ctx, v.config.URL, nil)
	connectionMs := time.Since(start).Milliseconds()

	if err != nil {
		statusInfo := ""
		if resp != nil {
			statusInfo = fmt.Sprintf(" (HTTP %d)", resp.StatusCode)
		}
		observations = append(observations, types.NewErrorObservation(fmt.Sprintf("connection failed%s: %v", statusInfo, err)))
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("WebSocket connection failed: %w", err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Ensure the scenario is running and WebSocket endpoint %s is accessible.", v.config.URL),
				Observations: observations,
			},
			Endpoint:         endpoint,
			ConnectionTimeMs: connectionMs,
		}
	}
	defer conn.Close()

	v.logStep("Connected in %dms", connectionMs)
	observations = append(observations, types.NewSuccessObservation(fmt.Sprintf("WebSocket connected in %dms", connectionMs)))

	// Check connection time
	if connectionMs > v.config.MaxConnectionMs {
		observations = append(observations, types.NewWarningObservation(fmt.Sprintf("connection time %dms exceeds threshold %dms", connectionMs, v.config.MaxConnectionMs)))
		return ValidationResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("connection time %dms exceeds threshold %dms", connectionMs, v.config.MaxConnectionMs),
				FailureClass: types.FailureClassSystem,
				Remediation:  "Investigate WebSocket server performance or increase the connection threshold.",
				Observations: observations,
			},
			Endpoint:         endpoint,
			ConnectionTimeMs: connectionMs,
		}
	}

	// Optionally test ping-pong
	var roundTripMs int64
	if v.config.PingMessage != "" {
		roundTripMs, err = v.testPingPong(conn, observations)
		if err != nil {
			observations = append(observations, types.NewErrorObservation(fmt.Sprintf("ping-pong failed: %v", err)))
			return ValidationResult{
				Result: types.Result{
					Success:      false,
					Error:        err,
					FailureClass: types.FailureClassSystem,
					Remediation:  "Ensure the WebSocket server handles messages correctly.",
					Observations: observations,
				},
				Endpoint:           endpoint,
				ConnectionTimeMs:   connectionMs,
				MessageRoundTripMs: roundTripMs,
			}
		}
		observations = append(observations, types.NewSuccessObservation(fmt.Sprintf("ping-pong round trip: %dms", roundTripMs)))
	} else {
		observations = append(observations, types.NewInfoObservation("ping-pong test skipped (no ping message configured)"))
	}

	// Test graceful close
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		v.logStep("Warning: graceful close failed: %v", err)
		observations = append(observations, types.NewWarningObservation("graceful close send failed"))
	} else {
		observations = append(observations, types.NewSuccessObservation("graceful close initiated"))
	}

	return ValidationResult{
		Result: types.Result{
			Success:      true,
			Observations: observations,
		},
		Endpoint:           endpoint,
		ConnectionTimeMs:   connectionMs,
		MessageRoundTripMs: roundTripMs,
	}
}

// testPingPong sends a ping message and waits for a response.
func (v *validator) testPingPong(conn *websocket.Conn, observations []types.Observation) (int64, error) {
	v.logStep("Sending ping message: %s", v.config.PingMessage)

	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(v.config.WriteTimeout)); err != nil {
		return 0, fmt.Errorf("failed to set write deadline: %w", err)
	}

	start := time.Now()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v.config.PingMessage)); err != nil {
		return 0, fmt.Errorf("failed to send ping: %w", err)
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(v.config.ReadTimeout)); err != nil {
		return 0, fmt.Errorf("failed to set read deadline: %w", err)
	}

	_, message, err := conn.ReadMessage()
	roundTripMs := time.Since(start).Milliseconds()

	if err != nil {
		return roundTripMs, fmt.Errorf("failed to read pong: %w", err)
	}

	v.logStep("Received response in %dms: %s", roundTripMs, string(message))

	// Optionally validate response content
	if v.config.ExpectedPongPattern != "" {
		if !strings.Contains(string(message), v.config.ExpectedPongPattern) {
			return roundTripMs, fmt.Errorf("pong message does not contain expected pattern %q", v.config.ExpectedPongPattern)
		}
	}

	return roundTripMs, nil
}

// logStep writes a step message to the log.
func (v *validator) logStep(format string, args ...interface{}) {
	if v.logWriter == nil {
		return
	}
	fmt.Fprintf(v.logWriter, format+"\n", args...)
}
