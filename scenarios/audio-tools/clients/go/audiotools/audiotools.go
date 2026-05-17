// Package audiotools is the canonical Go client for the audio-tools
// scenario API. Adopters import this package to talk to audio-tools
// instead of constructing the four generated Connect clients by hand.
//
// Wire-up:
//
//	client, err := audiotools.New(audiotools.DefaultResolver(), audiotools.Policy{Required: true})
//	if err != nil { return err }
//	ctx = audiotools.WithCredentials(ctx, audiotools.Credentials{LPBSToken: "tok"})
//	resp, err := client.STT.Transcribe(ctx, connect.NewRequest(&sttv1.TranscribeRequest{...}))
//
// The Client struct bundles the four generated Connect clients
// (STT/TTS/Summarize/Settings/Session/Usage/Audio/Health) so adopters
// don't reach into the proto-generated tree for each surface. The
// canonical credentials interceptor (see credentials.go) is
// installed by default; opt out by passing
// Policy{WithoutCredentialsInterceptor: true} only for tests or for
// pipelines that explicitly do not need BYOK/LPBS routing.
package audiotools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
	sessionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session/session_v1connect"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
	summarizeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"
)

// Policy tunes client construction. Zero value is "lazy resolve, no
// per-call timeout, install credentials interceptor with the default
// CredentialsGetter, no retry beyond Connect defaults."
type Policy struct {
	// Required, when true, forces URL resolution at New() time and
	// returns ErrUnavailable if the resolver fails. When false
	// (default), the resolver runs lazily at first call.
	Required bool

	// PerCallTimeout, when non-zero, wraps every unary call's context
	// with a deadline. Streaming RPCs ignore this and rely on caller
	// context.
	PerCallTimeout time.Duration

	// MaxTransportRetries controls how many times the client retries
	// a transport-level failure with a refreshed URL. Defaults to 3.
	MaxTransportRetries int

	// CredentialsGetter overrides the per-call credentials source.
	// Defaults to DefaultCredentialsGetter (reads from context via
	// FromContext).
	CredentialsGetter CredentialsGetter

	// WithoutCredentialsInterceptor disables the interceptor entirely.
	// Use only for tests or for pipelines that explicitly do not need
	// header credentials.
	WithoutCredentialsInterceptor bool

	// HTTPClient overrides the underlying *http.Client. Defaults to
	// http.DefaultClient.
	HTTPClient *http.Client
}

// Client bundles the seven generated Connect clients that make up
// the audio-tools API surface. Health is served as a plain HTTP
// endpoint (REST exception) and lives outside this struct; adopters
// call /health via Client.HealthURL() + http.Get.
type Client struct {
	STT       sttconnect.STTServiceClient
	TTS       ttsconnect.TTSServiceClient
	Summarize summarizeconnect.SummarizeServiceClient
	Audio     audioconnect.AudioProcessingServiceClient
	Session   sessionconnect.SessionServiceClient
	Settings  settingsconnect.SettingsServiceClient
	Usage     usageconnect.UsageServiceClient

	resolver URLResolver
	policy   Policy
	mu       sync.RWMutex
	baseURL  string
}

// New constructs a Client bound to the given URL resolver and policy.
// When policy.Required is true, the resolver is invoked at New time
// and any failure is normalized to ErrUnavailable.
func New(resolver URLResolver, policy Policy) (*Client, error) {
	if resolver == nil {
		return nil, fmt.Errorf("audiotools: resolver required")
	}
	if policy.MaxTransportRetries <= 0 {
		policy.MaxTransportRetries = 3
	}
	if policy.HTTPClient == nil {
		policy.HTTPClient = http.DefaultClient
	}
	c := &Client{resolver: resolver, policy: policy}
	if policy.Required {
		url, err := resolver.ResolveURL(context.Background())
		if err != nil {
			return nil, errors.Join(ErrUnavailable, err)
		}
		c.baseURL = url
		c.bindClients(url)
	}
	return c, nil
}

// Ensure resolves the base URL if it hasn't been resolved yet, then
// returns the live URL.
func (c *Client) Ensure(ctx context.Context) (string, error) {
	c.mu.RLock()
	url := c.baseURL
	c.mu.RUnlock()
	if url != "" {
		return url, nil
	}
	resolved, err := c.resolver.ResolveURL(ctx)
	if err != nil {
		return "", errors.Join(ErrUnavailable, err)
	}
	c.mu.Lock()
	c.baseURL = resolved
	c.bindClients(resolved)
	c.mu.Unlock()
	return resolved, nil
}

// BaseURL returns the currently-bound URL or empty string when the
// client has not resolved yet.
func (c *Client) BaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// HandleTransportFailure clears the bound URL so the next call
// triggers a fresh resolution. Adopters call this after a transport
// failure that suggests the audio-tools port moved (e.g., scenario
// restart).
func (c *Client) HandleTransportFailure() {
	c.mu.Lock()
	c.baseURL = ""
	c.mu.Unlock()
}

func (c *Client) bindClients(baseURL string) {
	var interceptors []connect.Interceptor
	if !c.policy.WithoutCredentialsInterceptor {
		getter := c.policy.CredentialsGetter
		if getter == nil {
			getter = DefaultCredentialsGetter
		}
		interceptors = append(interceptors, WithCredentialsInterceptor(getter))
	}
	opts := []connect.ClientOption{connect.WithInterceptors(interceptors...)}
	httpc := c.policy.HTTPClient
	c.STT = sttconnect.NewSTTServiceClient(httpc, baseURL, opts...)
	c.TTS = ttsconnect.NewTTSServiceClient(httpc, baseURL, opts...)
	c.Summarize = summarizeconnect.NewSummarizeServiceClient(httpc, baseURL, opts...)
	c.Audio = audioconnect.NewAudioProcessingServiceClient(httpc, baseURL, opts...)
	c.Session = sessionconnect.NewSessionServiceClient(httpc, baseURL, opts...)
	c.Settings = settingsconnect.NewSettingsServiceClient(httpc, baseURL, opts...)
	c.Usage = usageconnect.NewUsageServiceClient(httpc, baseURL, opts...)
}

// HealthURL returns the /health endpoint URL for the currently bound
// base URL. Empty string when the client has not resolved yet.
func (c *Client) HealthURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.baseURL == "" {
		return ""
	}
	return c.baseURL + "/health"
}
