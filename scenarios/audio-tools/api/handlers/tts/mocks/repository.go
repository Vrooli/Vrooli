package mocks

import (
	"context"

	ttsH "audio-tools/handlers/tts"
	"audio-tools/internal/store"
)

type FakeTTSConfig struct {
	CfgJSON  string
	SummJSON string
	Ok       bool
	GetErr   error
	SetErr   error
}

func (f *FakeTTSConfig) Get(context.Context) (string, string, bool, error) {
	return f.CfgJSON, f.SummJSON, f.Ok, f.GetErr
}
func (f *FakeTTSConfig) Set(context.Context, string, string) error { return f.SetErr }

type FakePlayback struct {
	Events    []store.PlaybackEvent
	InsertErr error
	ListErr   error
}

func (f *FakePlayback) Insert(_ context.Context, e store.PlaybackEvent) error {
	if f.InsertErr != nil {
		return f.InsertErr
	}
	f.Events = append(f.Events, e)
	return nil
}

func (f *FakePlayback) List(_ context.Context, _ int) ([]store.PlaybackEvent, error) {
	return f.Events, f.ListErr
}

var (
	_ ttsH.TTSConfigRepository = (*FakeTTSConfig)(nil)
	_ ttsH.PlaybackRepository  = (*FakePlayback)(nil)
)
