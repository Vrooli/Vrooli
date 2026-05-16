// Package sttchain implements the speech-to-text provider chain.
//
// Routing precedence (fixed): BYOK -> Vrooli/LPBS -> Local.
// ErrInsufficientCredits from the Vrooli tier short-circuits and does NOT
// fall through to Local — credit exhaustion is a billing failure, not a
// capability failure. Per-request credentials travel through Request.
//
// Mirrors the BAS chain pattern at scenarios/browser-automation-studio/api/services/ai/.
package sttchain

import (
	"context"
	"errors"
	"time"
)

// ProviderTier identifies which tier handled a request.
type ProviderTier string

const (
	TierBYOK   ProviderTier = "byok"
	TierVrooli ProviderTier = "vrooli"
	TierLocal  ProviderTier = "local"
)

// Request is the per-call input to the chain.
type Request struct {
	Audio                    []byte
	Format                   string
	Language                 string
	SkipSpeakerVerification  bool
	InitialPrompt            string

	// Per-request creds carried via headers and injected by the handler interceptor.
	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string

	// Optional override; chain default model applies when empty.
	Model string
}

// Result is the chain's response, including provider trace.
type Result struct {
	Text             string
	DetectedLanguage string
	DurationSeconds  float64
	Tier             ProviderTier
	ProviderID       string
	ModelID          string
	Latency          time.Duration
}

// Provider is the 4-method interface implemented by Local/BYOK/Vrooli tiers.
//
// Each concrete provider is short-lived (constructed per-request from a
// factory) so it can capture per-request creds without leaking them across
// callers. Availability is cached at the chain level with a per-tier TTL.
type Provider interface {
	Type() ProviderTier
	IsAvailable(ctx context.Context) bool
	Transcribe(ctx context.Context, req Request) (*Result, error)
	Model() string
}

// ErrInsufficientCredits is returned by the Vrooli provider when LPBS reports
// the user is out of audio credits. The chain MUST NOT fall through to Local
// on this error — it is a billing failure, not an availability failure.
var ErrInsufficientCredits = errors.New("audio-tools: insufficient credits for Vrooli tier")

// ErrAllProvidersFailed is returned when every enabled tier was tried and
// none succeeded.
var ErrAllProvidersFailed = errors.New("audio-tools: all providers failed")

// ErrUnknownBYOKProvider is returned when X-Audio-BYOK-Provider names a
// provider not in the registry.
var ErrUnknownBYOKProvider = errors.New("audio-tools: unknown BYOK provider")

// ErrMissingBYOKProvider is returned when X-Audio-BYOK-Key is set but
// X-Audio-BYOK-Provider is absent — silent provider selection is forbidden.
var ErrMissingBYOKProvider = errors.New("audio-tools: BYOK key set without BYOK provider")
