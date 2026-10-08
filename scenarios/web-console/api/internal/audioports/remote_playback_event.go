package audioports

import (
	"context"

	"connectrpc.com/connect"

	"web-console/integrations/audiotools"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// RemotePlaybackEventRecorder implements PlaybackEventRecorder against
// audio-tools' TTSService.RecordPlaybackEvent.
type RemotePlaybackEventRecorder struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

var _ PlaybackEventRecorder = (*RemotePlaybackEventRecorder)(nil)

func (r *RemotePlaybackEventRecorder) RecordPlaybackEvent(ctx context.Context, ev PlaybackEvent) error {
	if r == nil || r.Client == nil {
		return audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return audiotools.ErrUnavailable
	}
	req := connect.NewRequest(&ttsv1.RecordPlaybackEventRequest{
		Event: &ttsv1.PlaybackEvent{
			Source:    ev.Source,
			Stage:     ev.Stage,
			Backend:   ev.Backend,
			SessionId: ev.SessionID,
			Message:   ev.Message,
			EventId:   ev.EventID,
		},
	})
	if r.Credentials != nil {
		req = audiotools.AttachCredentials(req, r.Credentials(ctx))
	}
	_, err := r.Client.TTS.RecordPlaybackEvent(ctx, req)
	if err != nil {
		if isTransportFailure(err) {
			r.Client.HandleTransportFailure()
		}
		return audiotools.NormalizeError(err)
	}
	return nil
}
