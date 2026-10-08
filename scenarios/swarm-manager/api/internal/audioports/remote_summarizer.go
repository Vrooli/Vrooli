package audioports

import (
	"context"
	"time"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

// summarizeLevelFromString maps human level strings to proto enum.
func summarizeLevelFromString(s string) summv1.SummarizeLevel {
	switch s {
	case "light":
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT
	case "moderate":
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE
	case "heavy":
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY
	default:
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED
	}
}

func providerTierToString(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	default:
		return ""
	}
}

// Summarizer is the narrow port web-console talks to for long-message
// summarization on the TTS pipeline. The local implementation in
// internal/tts is the canonical owner today; the RemoteSummarizer here
// fronts audio-tools' SummarizeService.Summarize RPC so the same call site
// can swap to remote without rippling through callers.
type Summarizer interface {
	Summarize(ctx context.Context, in SummarizeInput) (SummarizeOutput, error)
}

// SummarizeInput mirrors what audio-tools' SummarizeRequest accepts today.
// Optional fields are zero-valued to mean "use provider default".
type SummarizeInput struct {
	Text           string
	Level          string // "light" | "moderate" | "heavy"
	Model          string // optional override
	TimeoutSeconds int
}

// SummarizeOutput is the response shape callers care about. Provider trace
// fields are surfaced so web-console can log which tier handled the call.
type SummarizeOutput struct {
	Text         string
	PromptTokens int
	OutputTokens int
	ProviderTier string
	ProviderID   string
	ModelID      string
	Latency      time.Duration
}

// RemoteSummarizer implements Summarizer by calling audio-tools' Connect
// SummarizeService through the integrations/audiotools adapter.
type RemoteSummarizer struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

// Compile-time interface check.
var _ Summarizer = (*RemoteSummarizer)(nil)

func (r *RemoteSummarizer) Summarize(ctx context.Context, in SummarizeInput) (SummarizeOutput, error) {
	if r == nil || r.Client == nil {
		return SummarizeOutput{}, audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return SummarizeOutput{}, audiotools.ErrUnavailable
	}
	req := connect.NewRequest(&summv1.SummarizeRequest{
		Text:           in.Text,
		Level:          summarizeLevelFromString(in.Level),
		Model:          in.Model,
		TimeoutSeconds: int32(in.TimeoutSeconds),
	})
	if r.Credentials != nil {
		req = audiotools.AttachCredentials(req, r.Credentials(ctx))
	}
	resp, err := r.Client.Summarize.Summarize(ctx, req)
	if err != nil {
		if isTransportFailure(err) {
			r.Client.HandleTransportFailure()
		}
		return SummarizeOutput{}, audiotools.NormalizeError(err)
	}
	if resp == nil || resp.Msg == nil {
		return SummarizeOutput{}, audiotools.ErrUnavailable
	}
	return SummarizeOutput{
		Text:         resp.Msg.Text,
		PromptTokens: int(resp.Msg.PromptTokens),
		OutputTokens: int(resp.Msg.OutputTokens),
		ProviderTier: providerTierToString(resp.Msg.ProviderTier),
		ProviderID:   resp.Msg.ProviderId,
		ModelID:      resp.Msg.ModelId,
		Latency:      time.Duration(resp.Msg.LatencyMs) * time.Millisecond,
	}, nil
}
