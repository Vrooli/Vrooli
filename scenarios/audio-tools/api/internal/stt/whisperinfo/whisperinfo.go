// Package whisperinfo is the model-truth seam for the local Whisper
// sidecar. It exists because the upstream onerahmet/openai-whisper-asr-
// webservice image (v1.9.1) exposes only /asr and /detect-language —
// there is no /models or /health endpoint to query, so the audio-tools
// process cannot ask the sidecar "what model are you running?" at
// runtime. Instead, the resource lifecycle passes the chosen model into
// the audio-tools process via the AUDIO_WHISPER_MODEL env var
// (mirroring the ASR_MODEL the sidecar received), and EnvClient reads
// it.
//
// This keeps LocalProvider.Model() honest without inventing the value
// or fetching it from the sidecar via an endpoint that does not exist.
// When the env var is unset (e.g. ad-hoc dev runs), Client returns
// ModelUnknown rather than fabricating a model name.
package whisperinfo

import (
	"audio-tools/internal/envx"
)

// ModelUnknown is the sentinel for "we don't know what the sidecar is
// running." Never fabricate a model name in its place.
const ModelUnknown = "whisper-unknown"

// envVar is the operator-visible knob. Production sets it from the
// whisper resource's resolved model at scenario start.
const envVar = "AUDIO_WHISPER_MODEL"

// Info is the model-truth snapshot returned by Client.
type Info struct {
	ModelID string
	// Engine is the sidecar engine (e.g. "faster_whisper"). Optional —
	// not all production setups surface it.
	Engine string
	// Source is human-readable provenance for diagnostics.
	Source string
}

// Client is the model-truth seam.
//
// seam: whisperinfo.Client is the local-Whisper model-truth seam
// (SEAMS.md row "whisperinfo.Client"). Production wires EnvClient;
// tests inject FakeClient from internal/stt/whisperinfo/mocks.
type Client interface {
	CurrentModel() Info
}

// EnvClient resolves the configured model from the process environment
// via an envx.Reader. Cheap and deterministic — no I/O, no caching
// needed (env values do not change mid-process).
type EnvClient struct {
	Env envx.Reader
}

// New constructs an EnvClient reading the real OS env.
func New() *EnvClient {
	return &EnvClient{Env: envx.OS{}}
}

// NewWith constructs an EnvClient with an injected reader.
func NewWith(env envx.Reader) *EnvClient {
	if env == nil {
		env = envx.OS{}
	}
	return &EnvClient{Env: env}
}

// CurrentModel returns the resolved Info.
func (c *EnvClient) CurrentModel() Info {
	env := c.Env
	if env == nil {
		env = envx.OS{}
	}
	model := env.Get(envVar)
	if model == "" {
		return Info{ModelID: ModelUnknown, Source: "default (env unset)"}
	}
	return Info{
		ModelID: "whisper-" + model,
		Engine:  env.Get("AUDIO_WHISPER_ENGINE"),
		Source:  envVar,
	}
}

var _ Client = (*EnvClient)(nil)
