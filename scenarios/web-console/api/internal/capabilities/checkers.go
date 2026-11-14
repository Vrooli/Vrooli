package capabilities

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	healthstatusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
	"github.com/vrooli/vrooli/scenarios/audio-tools/clients/go/audiotools"
)

type ResourceChecker struct {
	URL    string
	Client *http.Client
}

// AudioToolsChecker preserves scenario reachability while enriching the row
// with audio-tools' authoritative per-feature liveness rollup.
type AudioToolsChecker struct {
	Scenario       ScenarioChecker
	ProviderHealth func(context.Context) (*healthstatusv1.GetProviderHealthResponse, error)
	Features       []string
	Timeout        time.Duration
}

func (c *AudioToolsChecker) Check(ctx context.Context) (Status, string) {
	r := c.CheckResult(ctx)
	return r.Status, r.Message
}

func (c *AudioToolsChecker) CheckResult(ctx context.Context) CheckResult {
	base := c.Scenario.CheckResult(ctx)
	// A reachable but degraded scenario still has authoritative provider
	// health. Only stopped/unreachable rows should skip the RPC entirely.
	if (base.Status != StatusAvailable && base.ReasonCode != "scenario_degraded") || c.ProviderHealth == nil {
		return base
	}
	featureStatus := make(map[string]string, len(c.Features))
	featureReason := make(map[string]string)
	featureOperatorCommand := make(map[string]string)
	providerStatus := make(map[string]string)
	providerFeatures := make(map[string]string)
	for _, feature := range c.Features {
		featureStatus[feature] = string(StatusUnknown)
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := c.ProviderHealth(callCtx)
	if err != nil {
		base.FeatureStatus = featureStatus
		return base
	}
	for _, capHealth := range resp.GetCapabilities() {
		providers := capHealth.GetProviders()
		if len(providers) == 0 {
			status := providerStateStatus(capHealth.GetEffectiveState())
			for _, feature := range featuresForCapability(capHealth.GetCapability()) {
				featureStatus[feature] = status
			}
			continue
		}
		for _, provider := range providers {
			status := providerStateStatus(provider.GetState())
			features := featuresForProvider(provider.GetProviderId())
			if len(features) > 0 {
				providerFeatures[provider.GetProviderId()] = strings.Join(features, ",")
			}
			// Browser speech is client-owned. Keep it in the feature rollup so
			// consumers can describe the last-resort path, but do not publish it
			// as a server provider that a server-TTS gate could select.
			if provider.GetProviderId() != "browser-stt" && provider.GetProviderId() != "browser-tts" {
				providerStatus[provider.GetProviderId()] = status
			}
			for _, feature := range featuresForProvider(provider.GetProviderId()) {
				featureStatus[feature] = status
				if status == string(StatusUnavailable) {
					reason := provider.GetErrorMessage()
					if reason == "" {
						reason = "provider " + provider.GetProviderId() + " is unavailable"
					}
					featureReason[feature] = reason
					featureOperatorCommand[feature] = operatorCommandForProvider(provider.GetProviderId())
				} else if status == string(StatusAvailable) {
					delete(featureReason, feature)
					delete(featureOperatorCommand, feature)
				}
			}
		}
	}
	for feature := range featureReason {
		if featureStatus[feature] != string(StatusUnavailable) {
			delete(featureReason, feature)
			delete(featureOperatorCommand, feature)
		}
	}
	base.FeatureStatus = featureStatus
	base.ProviderStatus = providerStatus
	base.ProviderFeatures = providerFeatures
	if len(featureReason) > 0 {
		base.FeatureReason = featureReason
		base.FeatureOperatorCommand = featureOperatorCommand
	}
	return base
}

func providerStateStatus(state sharedv1.ProviderState) string {
	switch state {
	case sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE:
		return string(StatusAvailable)
	case sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE:
		return string(StatusUnavailable)
	default:
		return string(StatusUnknown)
	}
}

func featuresForProvider(providerID string) []string {
	features := map[string][]string{
		"whisper-stt":          {"voice-input"},
		"openai-whisper":       {"voice-input"},
		"browser-stt":          {"voice-input", "voice-streaming"},
		"kyutai-stt":           {"voice-streaming"},
		"deepgram":             {"voice-streaming"},
		"kokoro-tts":           {"voice-output"},
		"openai-tts":           {"voice-output"},
		"elevenlabs":           {"voice-output"},
		"browser-tts":          {"voice-output"},
		"speaker-verification": {"voice-speaker-verification", "voice-enrollment"},
	}
	return features[providerID]
}

func operatorCommandForProvider(providerID string) string {
	commands := map[string]string{
		"whisper-stt":          "vrooli resource start whisper",
		"kyutai-stt":           "vrooli resource start kyutai-stt",
		"kokoro-tts":           "vrooli resource start kokoro",
		"speaker-verification": "vrooli resource status sherpa-onnx --json",
	}
	if command, ok := commands[providerID]; ok {
		return command
	}
	return "audio-tools settings providers"
}

func featuresForCapability(capability diagv1.Capability) []string {
	features := map[diagv1.Capability][]commonv1.AudioToolsFeature{
		diagv1.Capability_CAPABILITY_STT:       {commonv1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_INPUT, commonv1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_STREAMING, commonv1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_SPEAKER_VERIFICATION, commonv1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_ENROLLMENT},
		diagv1.Capability_CAPABILITY_TTS:       {commonv1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_OUTPUT},
		diagv1.Capability_CAPABILITY_SUMMARIZE: {commonv1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_SUMMARIZATION},
	}
	out := make([]string, 0, len(features[capability]))
	for _, feature := range features[capability] {
		out = append(out, audiotools.FeatureSlug(feature))
	}
	return out
}

// StaticChecker adapts a host probe to the capability registry without
// duplicating the registry's status vocabulary in a handler.
type StaticChecker struct {
	Available func() (bool, string)
}

// HostCapabilityChecker adapts the shared ai-cli probe to the integrations
// registry, preserving observed local versions separately from health prose.
// A discovered executable does not identify the selected request handler.
type HostCapabilityChecker struct {
	Definition capabilityprobe.Definition
}

func (c HostCapabilityChecker) Check(ctx context.Context) (Status, string) {
	result := c.CheckResult(ctx)
	return result.Status, result.Message
}

func (c HostCapabilityChecker) CheckResult(ctx context.Context) CheckResult {
	observations := capabilityprobe.Probe(ctx, []capabilityprobe.Definition{c.Definition})
	if len(observations) == 0 {
		return CheckResult{Status: StatusUnknown, Message: "host capability probe returned no observation"}
	}
	observation := observations[0]
	if observation.State == capabilityprobe.Ready {
		return CheckResult{Status: StatusAvailable, Message: observation.Detail, Installation: &capabilityregistry.Installation{Location: "machine", Version: observation.Version}}
	}
	return CheckResult{Status: StatusUnavailable, Message: observation.Detail}
}

func (c *StaticChecker) Check(context.Context) (Status, string) {
	if c == nil || c.Available == nil {
		return StatusUnavailable, "host capability probe is not configured"
	}
	ok, reason := c.Available()
	if ok {
		return StatusAvailable, "available"
	}
	return StatusUnavailable, reason
}

// BridgeChecker probes the Bridge control plane without exposing its owner
// credentials to the browser. Configuration failures are typed so the
// capability surface can explain the recovery path instead of collapsing all
// remote-terminal failures into a generic unavailable state.
type BridgeChecker struct {
	BaseURL     string
	ResolveURL  func(context.Context) (string, error)
	OwnerToken  string
	ReauthToken string
	// Credentials, when set, supplies the owner and re-auth credentials for
	// each check instead of OwnerToken/ReauthToken. Locally minted owner
	// sessions live 15 minutes, so a checker built with the startup value
	// fails Bridge authentication for the rest of the process's life.
	Credentials func() (owner, reauth string)
	Client      *http.Client
	Probe       bool
}

func (c *BridgeChecker) Check(ctx context.Context) (Status, string) {
	result := c.CheckResult(ctx)
	return result.Status, result.Message
}

func (c *BridgeChecker) CheckResult(ctx context.Context) CheckResult {
	start := CheckResult{
		Status:          StatusUnavailable,
		ActionKind:      ActionKindScenarioStart,
		ActionLabel:     "Start Bridge",
		OperatorCommand: "vrooli scenario start vrooli-bridge --json",
	}
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" && c.ResolveURL != nil {
		resolved, resolveErr := c.ResolveURL(ctx)
		if resolveErr != nil {
			start.Message = "Bridge could not be discovered"
			start.ReasonCode = "bridge_unreachable"
			return start
		}
		base = strings.TrimRight(strings.TrimSpace(resolved), "/")
	}
	if base == "" {
		start.Message = "Bridge URL is not configured"
		start.ReasonCode = "bridge_url_missing"
		return start
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		start.Message = "Bridge URL is invalid"
		start.ReasonCode = "bridge_url_invalid"
		return start
	}
	ownerToken, reauthToken := c.OwnerToken, c.ReauthToken
	if c.Credentials != nil {
		ownerToken, reauthToken = c.Credentials()
	}
	if strings.TrimSpace(ownerToken) == "" || (strings.TrimSpace(reauthToken) == "" && !strings.HasPrefix(strings.TrimSpace(ownerToken), "LocalSession ")) {
		start.Message = "Bridge credentials are not configured"
		start.ReasonCode = "bridge_credentials_missing"
		return start
	}
	if !c.Probe {
		return CheckResult{Status: StatusAvailable, Message: "Bridge is configured"}
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, base+"/health", nil)
	if err != nil {
		start.Message = "Bridge health request could not be created"
		start.ReasonCode = "bridge_unreachable"
		return start
	}
	req.Header.Set("Authorization", ownerToken)
	if strings.TrimSpace(reauthToken) != "" {
		req.Header.Set("X-Bridge-Owner-Reauth", reauthToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		start.Message = "Bridge registry is unreachable"
		start.ReasonCode = "bridge_unreachable"
		return start
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		start.Message = "Bridge registry is unreachable"
		start.ReasonCode = "bridge_unreachable"
		return start
	}
	return CheckResult{Status: StatusAvailable, Message: "Bridge is reachable and ready"}
}

func (c *ResourceChecker) Check(ctx context.Context) (Status, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "resource is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTemporaryRedirect {
		return StatusAvailable, "resource is healthy"
	}

	return StatusUnavailable, "resource returned unexpected status"
}

// WhisperChecker / KokoroChecker removed in the audio-tools adoption —
// Whisper + Kokoro are owned end-to-end by the audio-tools scenario, and
// web-console talks to audio-tools (not the raw resources). The
// `audio-tools` capability entry above is checked via ScenarioChecker.

// OllamaChecker verifies that Ollama is running by hitting its /api/tags
// endpoint.
type OllamaChecker struct {
	BaseURL string
	Client  *http.Client
}

func (c *OllamaChecker) Check(ctx context.Context) (Status, string) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "Ollama is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "Ollama is running"
	}

	return StatusUnavailable, "Ollama returned unexpected status"
}

// ScenarioChecker probes the selected peer's health endpoint. Discovery is
// shared with feature calls, so bundled dependencies need no control-plane CLI.
type ScenarioChecker struct {
	Slug       string
	ResolveURL func(context.Context, string) (string, error)
	Client     *http.Client
	Timeout    time.Duration
}

func (c *ScenarioChecker) Check(ctx context.Context) (Status, string) {
	result := c.CheckResult(ctx)
	return result.Status, result.Message
}

func (c *ScenarioChecker) CheckResult(ctx context.Context) CheckResult {
	if c.Slug == "" {
		return CheckResult{Status: StatusUnavailable, Message: "scenario slug not configured", ReasonCode: "scenario_slug_missing"}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resolve := c.ResolveURL
	if resolve == nil {
		resolver := discovery.NewResolver(discovery.ResolverConfig{})
		resolve = resolver.ResolveScenarioURLDefault
	}
	base, err := resolve(probeCtx, c.Slug)
	if err != nil {
		return CheckResult{Status: StatusUnavailable, Message: "scenario is not available: " + c.Slug, ReasonCode: "scenario_not_reachable"}
	}
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, strings.TrimRight(base, "/")+"/health", nil)
	if err != nil {
		return CheckResult{Status: StatusUnknown, Message: "scenario health address is invalid", ReasonCode: "scenario_health_address_invalid"}
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{Status: StatusUnavailable, Message: "scenario health is not reachable", ReasonCode: "scenario_not_reachable"}
	}
	defer resp.Body.Close()
	var health struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&health); err != nil {
		return CheckResult{Status: StatusUnknown, Message: "scenario health response was not valid JSON", ReasonCode: "scenario_health_malformed_json"}
	}
	if health.Status == "degraded" {
		if health.Message == "" {
			health.Message = "scenario reported degraded without a reason"
		}
		return CheckResult{Status: StatusUnavailable, Message: joinStatusMessage("scenario is degraded", health.Message), ReasonCode: "scenario_degraded"}
	}
	if resp.StatusCode == http.StatusOK && (health.Status == "healthy" || health.Status == "ok") {
		return CheckResult{Status: StatusAvailable, Message: "scenario is healthy"}
	}
	return CheckResult{Status: StatusUnavailable, Message: "scenario health is not ready", ReasonCode: "scenario_health_not_ready"}
}

func joinStatusMessage(prefix, detail string) string {
	if detail == "" {
		return prefix
	}
	return prefix + ": " + detail
}

// OpenRouterChecker verifies that OpenRouter is configured and reachable.
type OpenRouterChecker struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func (c *OpenRouterChecker) Check(ctx context.Context) (Status, string) {
	if c.APIKey == "" {
		return StatusUnavailable, "OPENROUTER_API_KEY not configured"
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/models", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "OpenRouter is not reachable"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "OpenRouter is configured and reachable"
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return StatusUnavailable, "OpenRouter API key is invalid"
	}

	return StatusUnavailable, "OpenRouter returned unexpected status"
}
