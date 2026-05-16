// Package audiotools is web-console's integration adapter for the
// scenarios/audio-tools scenario. Wraps the generated Connect-RPC clients
// with discovery (via api-core/discovery), bounded retry, envelope/status
// normalization, and credential injection.
//
// Wire boundary:
//
//	web-console handlers / orchestration
//	         |
//	         v
//	api/internal/audioports.Remote{STT,TTS,Processor}
//	         |
//	         v
//	api/integrations/audiotools.Client
//	         |
//	         v  (Connect-RPC; URL re-resolved on transport failure)
//	scenarios/audio-tools
package audiotools

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

// Client bundles every generated audio-tools Connect client web-console
// touches, plus the discovery + retry policy that owns URL freshness.
type Client struct {
	resolver  URLResolver
	policy    Policy
	http      *http.Client

	mu        sync.RWMutex
	baseURL   string
	resolved  atomic.Bool

	STT       sttconnect.STTServiceClient
	TTS       ttsconnect.TTSServiceClient
	Summarize summconnect.SummarizeServiceClient
	Audio     audioconnect.AudioProcessingServiceClient
}

// URLResolver returns the current audio-tools base URL. Implementations
// typically wrap api-core/discovery.ResolveScenarioURLDefault.
type URLResolver interface {
	Resolve() (string, error)
}

// Policy is the retry/timeout/required-vs-optional policy the integration
// adapter enforces.
type Policy struct {
	// PerCallTimeout is the upper bound on any single audio-tools call.
	PerCallTimeout time.Duration
	// MaxRetries is the bounded retry count on transport failure (default 3).
	MaxRetries int
	// Required gates fail-fast behavior. When false (rollout), audio-tools
	// unavailability surfaces as an audioports error consumers can degrade on.
	Required bool
}

// New constructs a Client. The clients are wired on first Resolve call;
// callers do not need to re-construct on URL change because the underlying
// httpx.Transport re-resolves on connection refused (interoperability-steer §12).
func New(resolver URLResolver, policy Policy) (*Client, error) {
	if policy.PerCallTimeout == 0 {
		policy.PerCallTimeout = 30 * time.Second
	}
	if policy.MaxRetries == 0 {
		policy.MaxRetries = 3
	}
	c := &Client{
		resolver: resolver,
		policy:   policy,
		http:     &http.Client{Timeout: policy.PerCallTimeout},
	}
	if err := c.refresh(); err != nil {
		// During rollout (Required=false) we allow lazy resolution.
		if policy.Required {
			return nil, err
		}
	}
	return c, nil
}

// refresh resolves the current URL and rebuilds the typed clients.
func (c *Client) refresh() error {
	base, err := c.resolver.Resolve()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = base
	c.STT = sttconnect.NewSTTServiceClient(c.http, base)
	c.TTS = ttsconnect.NewTTSServiceClient(c.http, base)
	c.Summarize = summconnect.NewSummarizeServiceClient(c.http, base)
	c.Audio = audioconnect.NewAudioProcessingServiceClient(c.http, base)
	c.resolved.Store(true)
	return nil
}

// Ensure the suite of generated client interfaces is wired before each call.
// In practice this is a no-op after the first successful refresh — but on
// transport failure we re-resolve.
func (c *Client) Ensure() error {
	if c.resolved.Load() {
		return nil
	}
	return c.refresh()
}

// HandleTransportFailure is called after a connection-refused-class error
// to schedule a re-resolve. interoperability-steer §12 — captured URLs are
// short-lived.
func (c *Client) HandleTransportFailure() {
	c.resolved.Store(false)
}

// BaseURL returns the currently resolved base URL (for diagnostics/tests).
func (c *Client) BaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// Resolved reports whether the client has successfully resolved its URL.
// Exported for cross-package tests (Phase I cross-scenario verification).
func (c *Client) Resolved() bool { return c.resolved.Load() }

// AttachCredentials decorates a Connect request with the per-call audio-tools
// credential headers. Consumer code calls this on every Connect request that
// might benefit from a BYOK or LPBS path.
func AttachCredentials[T any](req *connect.Request[T], creds Credentials) *connect.Request[T] {
	if creds.BYOKKey != "" {
		req.Header().Set("X-Audio-BYOK-Key", creds.BYOKKey)
		req.Header().Set("X-Audio-BYOK-Provider", creds.BYOKProvider)
	}
	if creds.LPBSToken != "" {
		req.Header().Set("X-Audio-LPBS-Token", creds.LPBSToken)
	}
	if creds.UserIdentity != "" {
		req.Header().Set("X-Audio-User-Identity", creds.UserIdentity)
	}
	return req
}

// Credentials bundles the per-call audio-tools credentials.
type Credentials struct {
	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string
}
