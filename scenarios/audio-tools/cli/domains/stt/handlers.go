package stt

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client sttconnect.STTServiceClient
	stream sttconnect.STTServiceClient
	admin  sttconnect.STTAdminServiceClient
}

type streamingHTTPClient struct {
	core    *cliapp.ScenarioApp
	client  *http.Client
	baseURL string
}

func (c streamingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("streaming HTTP request is nil")
	}
	// Like cli-core's Connect client, generated requests capture the API base
	// when the client is constructed. CLI flags are parsed later, so rebase
	// those requests against the invocation's selected API before sending.
	if err := c.rebaseRequest(req); err != nil {
		return nil, err
	}
	if c.core != nil {
		if c.core.HTTPClient != nil {
			c.core.HTTPClient.ApplyRequestHeaders(req)
		}
		if c.core.APIClient != nil {
			for k, v := range c.core.APIClient.AuthHeaders() {
				req.Header.Set(k, v)
			}
		}
	}
	client := c.client
	if client == nil {
		return nil, errors.New("streaming HTTP client is not configured")
	}
	if req.URL == nil || req.URL.Scheme == "" || req.URL.Host == "" {
		return nil, errors.New("streaming HTTP request URL must be absolute")
	}
	// The CLI deliberately connects to the operator-selected scenario API base;
	// the shared cli-core preflight validates that base before commands run.
	// #nosec G704 -- this is the configured scenario endpoint, not a request URL derived from input.
	return client.Do(req)
}

func (c streamingHTTPClient) rebaseRequest(req *http.Request) error {
	if req.URL == nil {
		return errors.New("streaming HTTP request URL is nil")
	}
	if initial, err := url.Parse(c.baseURL); err == nil && initial.Scheme != "" && initial.Host != "" && req.URL.Scheme == initial.Scheme && req.URL.Host == initial.Host {
		prefix := strings.TrimRight(initial.Path, "/") + "/"
		if strings.HasPrefix(req.URL.Path, prefix) {
			rebased := *req.URL
			rebased.Scheme, rebased.Host, rebased.User = "", "", nil
			rebased.Path = "/" + strings.TrimPrefix(rebased.Path, prefix)
			rebased.RawPath = ""
			req.URL = &rebased
			req.Host = ""
		}
	}
	if req.URL.Scheme != "" && req.URL.Host != "" {
		return nil // preserve explicit absolute URLs unrelated to this client
	}
	if c.core == nil {
		return errors.New("streaming HTTP client has no API base")
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(c.core.APIRootBase()), "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return errors.New("streaming HTTP client API base URL is invalid")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(req.URL.Path, "/")
	base.RawPath = ""
	base.RawQuery = req.URL.RawQuery
	base.Fragment = req.URL.Fragment
	req.URL = base
	return nil
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sttconnect.NewSTTServiceClient(httpClient, baseURL),
		stream: NewStreamingSTTClient(core, baseURL),
		admin:  sttconnect.NewSTTAdminServiceClient(httpClient, baseURL),
	}
}

func applyBoolFlag(ctx cliapp.OperationContext, flag, path string, mask *fieldmaskpb.FieldMask, set func(bool)) error {
	raw := ctx.Flag(flag)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("--%s must be true|false: %q", flag, raw)
	}
	set(value)
	mask.Paths = append(mask.Paths, path)
	return nil
}

// NewStreamingSTTClient builds the HTTP/2-capable STT client used by commands
// that send a long-lived request stream. Keeping this transport seam here lets
// validation and normal STT commands share the same auth, transport, and clear
// missing-client behavior.
func NewStreamingSTTClient(core *cliapp.ScenarioApp, baseURL string) sttconnect.STTServiceClient {
	client := newStreamingHTTPClient()
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Scheme == "http" {
		client.Transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		}
	}
	return sttconnect.NewSTTServiceClient(streamingHTTPClient{core: core, client: client, baseURL: baseURL}, baseURL, connect.WithGRPC())
}

func newStreamingHTTPClient() *http.Client {
	// Bidi streams can legitimately outlive the ordinary RPC timeout (the lab
	// qualification stream runs for an hour). Their context owns cancellation;
	// http.Client.Timeout would abort an active stream at a fixed wall-clock age.
	return &http.Client{}
}

func audioFormatFromFlag(s string) commonv1.AudioFormat {
	switch s {
	case "", "wav":
		return commonv1.AudioFormat_AUDIO_FORMAT_WAV
	case "mp3":
		return commonv1.AudioFormat_AUDIO_FORMAT_MP3
	case "flac":
		return commonv1.AudioFormat_AUDIO_FORMAT_FLAC
	case "ogg":
		return commonv1.AudioFormat_AUDIO_FORMAT_OGG
	case "webm":
		return commonv1.AudioFormat_AUDIO_FORMAT_WEBM
	case "opus":
		return commonv1.AudioFormat_AUDIO_FORMAT_OPUS
	default:
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
	}
}

// audioFormatLabel renders a common.AudioFormat as its lower-case wire
// name for human output. It is the display inverse of audioFormatFromFlag.
func audioFormatLabel(f commonv1.AudioFormat) string {
	switch f {
	case commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE:
		return "pcm_s16le"
	case commonv1.AudioFormat_AUDIO_FORMAT_WAV:
		return "wav"
	case commonv1.AudioFormat_AUDIO_FORMAT_MP3:
		return "mp3"
	case commonv1.AudioFormat_AUDIO_FORMAT_FLAC:
		return "flac"
	case commonv1.AudioFormat_AUDIO_FORMAT_OGG:
		return "ogg"
	case commonv1.AudioFormat_AUDIO_FORMAT_WEBM:
		return "webm"
	case commonv1.AudioFormat_AUDIO_FORMAT_OPUS:
		return "opus"
	case commonv1.AudioFormat_AUDIO_FORMAT_AAC:
		return "aac"
	default:
		return "unspecified"
	}
}

func availabilityLabel(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
}

func streamingModeLabel(m sttv1.StreamingMode) string {
	switch m {
	case sttv1.StreamingMode_STREAMING_MODE_OFF:
		return "off"
	case sttv1.StreamingMode_STREAMING_MODE_AUTO:
		return "auto"
	default:
		return "auto"
	}
}

func strategyPreferenceLabel(p sttv1.StrategyPreference) string {
	switch p {
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD:
		return "vad"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP:
		return "overlap"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH:
		return "passthrough"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO:
		return "auto"
	default:
		return "auto"
	}
}

func streamingModeFromFlag(s string) (sttv1.StreamingMode, error) {
	switch s {
	case "auto":
		return sttv1.StreamingMode_STREAMING_MODE_AUTO, nil
	case "off":
		return sttv1.StreamingMode_STREAMING_MODE_OFF, nil
	default:
		return sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED, fmt.Errorf("--streaming-mode must be auto|off: %q", s)
	}
}

func strategyPreferenceFromFlag(s string) (sttv1.StrategyPreference, error) {
	switch s {
	case "auto":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO, nil
	case "vad":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD, nil
	case "overlap":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP, nil
	case "passthrough":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH, nil
	default:
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_UNSPECIFIED, fmt.Errorf("--strategy-preference must be auto|vad|overlap|passthrough: %q", s)
	}
}
