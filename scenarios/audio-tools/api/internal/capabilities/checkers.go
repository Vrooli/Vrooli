package capabilities

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/httpc"
)

// ResourceChecker is the generic HTTP-probe Checker used for resources whose
// liveness can be determined by a single GET against a known URL. It is the
// base predicate that more specialised checkers (Whisper, Kokoro, Ollama,
// OpenRouter, ScenarioChecker) extend.
type ResourceChecker struct {
	URL  string
	Doer httpc.Doer
}

// BYOKCredentialChecker exposes only redacted credential presence to health.
// It intentionally never reads or returns the stored secret.
type BYOKCredentialChecker struct {
	ProviderID string
	Capability string
	List       func(context.Context) ([]byokstore.Credential, error)
	Get        func(context.Context, string, string) (string, bool, error)
	Probe      func(context.Context, string) error
}

// StaticChecker is used for client-owned capabilities such as browser speech.
type StaticChecker struct {
	Available func() (bool, string)
}

func (c *StaticChecker) Check(context.Context) (Status, string) {
	if c == nil || c.Available == nil {
		return StatusUnknown, "capability probe is not configured"
	}
	ok, reason := c.Available()
	if ok {
		return StatusAvailable, reason
	}
	return StatusUnavailable, reason
}

func (c *BYOKCredentialChecker) Check(ctx context.Context) (Status, string) {
	if c == nil || c.List == nil {
		return StatusUnknown, "BYOK credential store is not configured"
	}
	credentials, err := c.List(ctx)
	if err != nil {
		return StatusUnknown, "BYOK credential status could not be read"
	}
	for _, credential := range credentials {
		if credential.ProviderID == c.ProviderID && credential.Capability == c.Capability {
			if c.Get != nil && c.Probe != nil {
				secret, ok, err := c.Get(ctx, c.ProviderID, c.Capability)
				if err != nil || !ok || secret == "" {
					return StatusUnavailable, "BYOK credential could not be read; use audio-tools settings providers"
				}
				if err := c.Probe(ctx, secret); err != nil {
					return StatusUnavailable, "BYOK endpoint is unavailable; verify the provider credential in audio-tools settings providers"
				}
			}
			return StatusAvailable, "BYOK credential configured"
		}
	}
	return StatusUnavailable, "no BYOK credential configured; use audio-tools settings providers"
}

// ProbeBYOKEndpoint performs a cheap authenticated provider check. It accepts
// a plaintext key only at the request boundary and never includes it in an
// error or response.
func ProbeBYOKEndpoint(providerID string) func(context.Context, string) error {
	return func(ctx context.Context, key string) error {
		endpoint, header, valuePrefix := byokProbeEndpoint(providerID)
		if endpoint == "" {
			return fmt.Errorf("provider endpoint is not configured")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("create provider probe: %w", err)
		}
		req.Header.Set(header, valuePrefix+key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("provider probe failed")
		}
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("provider probe returned status %d", resp.StatusCode)
		}
		return nil
	}
}

func byokProbeEndpoint(providerID string) (string, string, string) {
	switch providerID {
	case "openai-whisper", "openai-tts":
		return "https://api.openai.com/v1/models", "Authorization", "Bearer "
	case "deepgram":
		return "https://api.deepgram.com/v1/projects", "Authorization", "Token "
	case "elevenlabs":
		return "https://api.elevenlabs.io/v1/user", "xi-api-key", ""
	default:
		return "", "", ""
	}
}

// TranscodeChecker verifies the in-process audio conversion dependency. The
// runner is injectable so registry tests remain hermetic and can model a host
// where ffmpeg is absent.
type TranscodeChecker struct {
	LookPath func(string) (string, error)
}

func (c *TranscodeChecker) Check(context.Context) (Status, string) {
	lookPath := c.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	if _, err := lookPath("ffmpeg"); err != nil {
		return StatusUnavailable, "ffmpeg is not installed (install ffmpeg to enable audio transcoding)"
	}
	return StatusAvailable, "ffmpeg is available for audio transcoding"
}

func (c *ResourceChecker) Check(ctx context.Context) (Status, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	resp, err := c.Doer.Do(req)
	if err != nil {
		return StatusUnavailable, "resource is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTemporaryRedirect {
		return StatusAvailable, "resource is healthy"
	}

	return StatusUnavailable, "resource returned unexpected status"
}
