package audioports

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// RemoteTextToSpeech implements TextToSpeech against the audio-tools service.
//
// Cache lookup remains a web-console concern at the conversation-event level
// (event_id -> content_hash); the adapter forwards both shapes to audio-tools
// which owns the content-addressable byte cache.
type RemoteTextToSpeech struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

var _ TextToSpeech = (*RemoteTextToSpeech)(nil)

func (r *RemoteTextToSpeech) Synthesize(ctx context.Context, in TTSRequest) (TTSResult, error) {
	if r == nil || r.Client == nil {
		return TTSResult{}, audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return TTSResult{}, audiotools.ErrUnavailable
	}
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text:           in.Input,
		Voice:          in.Voice,
		Speed:          in.Speed,
		ResponseFormat: in.ResponseFormat,
		EventId:        in.EventID,
		Version:        in.Version,
	})
	if r.Credentials != nil {
		req = audiotools.AttachCredentials(req, r.Credentials(ctx))
	}
	resp, err := r.Client.TTS.Synthesize(ctx, req)
	if err != nil {
		if isTransportFailure(err) {
			r.Client.HandleTransportFailure()
		}
		return TTSResult{}, audiotools.NormalizeError(err)
	}
	if resp == nil || resp.Msg == nil {
		return TTSResult{}, errors.New("audiotools: empty synthesize response")
	}
	return TTSResult{Audio: resp.Msg.Audio, ContentType: resp.Msg.ContentType}, nil
}

func (r *RemoteTextToSpeech) ListVoices(ctx context.Context) ([]Voice, error) {
	if r == nil || r.Client == nil {
		return nil, audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return nil, audiotools.ErrUnavailable
	}
	resp, err := r.Client.TTS.ListVoices(ctx, connect.NewRequest(&ttsv1.ListVoicesRequest{}))
	if err != nil {
		if isTransportFailure(err) {
			r.Client.HandleTransportFailure()
		}
		return nil, audiotools.NormalizeError(err)
	}
	if resp == nil || resp.Msg == nil {
		return nil, nil
	}
	out := make([]Voice, 0, len(resp.Msg.Voices))
	for _, v := range resp.Msg.Voices {
		out = append(out, Voice{ID: v.Id, Name: v.Name})
	}
	return out, nil
}

func (r *RemoteTextToSpeech) GetCached(ctx context.Context, key CacheLookup) (TTSResult, bool) {
	if r == nil || r.Client == nil {
		return TTSResult{}, false
	}
	if err := r.Client.Ensure(); err != nil {
		return TTSResult{}, false
	}
	resp, err := r.Client.TTS.GetCache(ctx, connect.NewRequest(&ttsv1.GetCacheRequest{
		EventId: key.EventID,
		Voice:   key.Voice,
		Speed:   key.Speed,
		Version: key.Version,
	}))
	if err != nil || resp == nil || resp.Msg == nil || !resp.Msg.Hit {
		return TTSResult{}, false
	}
	return TTSResult{Audio: resp.Msg.Audio, ContentType: resp.Msg.ContentType}, true
}
