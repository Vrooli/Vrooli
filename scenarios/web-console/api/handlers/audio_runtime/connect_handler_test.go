package audio_runtime

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"web-console/internal/audioports"

	audioruntimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_runtime"
	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_common"
)

type fakeSTT struct {
	lastOpts audioports.STTOptions
	result   audioports.STTResult
	err      error
}

func (f *fakeSTT) Transcribe(_ context.Context, _ []byte, opts audioports.STTOptions) (audioports.STTResult, error) {
	f.lastOpts = opts
	return f.result, f.err
}

type fakePlayback struct{ last audioports.PlaybackEvent }

func (f *fakePlayback) RecordPlaybackEvent(_ context.Context, ev audioports.PlaybackEvent) error {
	f.last = ev
	return nil
}

func TestTranscribe_HappyPath(t *testing.T) {
	stt := &fakeSTT{result: audioports.STTResult{Text: "hello world"}}
	h := NewConnectHandler(Deps{STT: stt})
	resp, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{
		Audio:                   []byte{0x01},
		Format:                  audiocommonv1.AudioFormat_AUDIO_FORMAT_WEBM,
		SkipSpeakerVerification: true,
		Language:                "en",
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Msg.Text != "hello world" {
		t.Errorf("text: got %q, want %q", resp.Msg.Text, "hello world")
	}
	if !stt.lastOpts.SkipSpeakerVerification || stt.lastOpts.Language != "en" {
		t.Errorf("opts forwarded incorrectly: %+v", stt.lastOpts)
	}
}

func TestTranscribe_EmptyAudio_InvalidArgument(t *testing.T) {
	h := NewConnectHandler(Deps{STT: &fakeSTT{}})
	_, err := h.Transcribe(context.Background(), connect.NewRequest(&audioruntimev1.TranscribeRequest{}))
	if err == nil {
		t.Fatal("expected error")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("err: got %v, want CodeInvalidArgument", err)
	}
}

func TestRecordPlaybackEvent_PassesFields(t *testing.T) {
	pb := &fakePlayback{}
	h := NewConnectHandler(Deps{Playback: pb})
	_, err := h.RecordPlaybackEvent(context.Background(), connect.NewRequest(&audioruntimev1.RecordPlaybackEventRequest{
		Event: &audioruntimev1.PlaybackEvent{
			Source:    "ui",
			Stage:     "start",
			Backend:   "kokoro",
			SessionId: "sess-1",
			EventId:   "evt-1",
		},
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if pb.last.Source != "ui" || pb.last.Stage != "start" || pb.last.EventID != "evt-1" {
		t.Errorf("playback event forwarded incorrectly: %+v", pb.last)
	}
}
