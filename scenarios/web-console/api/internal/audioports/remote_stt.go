package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// RemoteSpeechToText implements SpeechToText by calling the audio-tools
// scenario via the integrations/audiotools adapter.
//
// Replaces LocalSpeechToText after Phase H adoption. Per-call BYOK/LPBS
// credentials should be plumbed through context (future expansion); today
// transcribe runs with the audio-tools instance's default provider chain.
type RemoteSpeechToText struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

// Compile-time interface check.
var _ SpeechToText = (*RemoteSpeechToText)(nil)

func (r *RemoteSpeechToText) Transcribe(ctx context.Context, audio []byte, opts STTOptions) (STTResult, error) {
	if r == nil || r.Client == nil {
		return STTResult{}, audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return STTResult{}, audiotools.ErrUnavailable
	}
	req := connect.NewRequest(&sttv1.TranscribeRequest{
		Audio:                   audio,
		Format:                  commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		Language:                opts.Language,
		SkipSpeakerVerification: opts.SkipSpeakerVerification,
		InitialPrompt:           opts.InitialPrompt,
	})
	if opts.ModelID != "" {
		req.Header().Set("X-Audio-Model-ID", opts.ModelID)
	} else if r.Credentials != nil {
		req = audiotools.AttachCredentials(req, r.Credentials(ctx))
	}
	if opts.ProviderTier != "" {
		req.Header().Set("X-Audio-Provider-Tier", opts.ProviderTier)
	}
	if opts.ProviderID != "" {
		req.Header().Set("X-Audio-Provider-ID", opts.ProviderID)
	}
	resp, err := r.Client.Clients().STT.Transcribe(ctx, req)
	if err != nil {
		if isTransportFailure(err) {
			r.Client.HandleTransportFailure()
		}
		return STTResult{}, audiotools.NormalizeError(err)
	}
	if resp == nil || resp.Msg == nil {
		return STTResult{}, errors.New("audiotools: empty transcribe response")
	}
	return STTResult{Text: resp.Msg.Text}, nil
}

func isTransportFailure(err error) bool {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr.Code() == connect.CodeUnavailable || connectErr.Code() == connect.CodeDeadlineExceeded
	}
	return false
}
