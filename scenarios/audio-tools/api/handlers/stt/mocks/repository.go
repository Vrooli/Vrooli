package mocks

import (
	"context"

	sttH "audio-tools/handlers/stt"
	"audio-tools/internal/store"
)

type FakeSTTStreamConfig struct {
	Json   string
	Ok     bool
	GetErr error
	SetErr error
}

func (f *FakeSTTStreamConfig) Get(context.Context) (string, bool, error) {
	return f.Json, f.Ok, f.GetErr
}
func (f *FakeSTTStreamConfig) Set(context.Context, string) error { return f.SetErr }

type FakeWakeword struct {
	Items     map[string]store.WakeWordTemplate
	UpsertErr error
	DeleteErr error
	GetErr    error
	ListErr   error
}

func (f *FakeWakeword) Upsert(_ context.Context, t store.WakeWordTemplate) error {
	if f.UpsertErr != nil {
		return f.UpsertErr
	}
	if f.Items == nil {
		f.Items = map[string]store.WakeWordTemplate{}
	}
	f.Items[t.ID] = t
	return nil
}

func (f *FakeWakeword) Delete(_ context.Context, id string) (bool, error) {
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	_, ok := f.Items[id]
	delete(f.Items, id)
	return ok, nil
}

func (f *FakeWakeword) Get(_ context.Context, id string) (store.WakeWordTemplate, bool, error) {
	if f.GetErr != nil {
		return store.WakeWordTemplate{}, false, f.GetErr
	}
	t, ok := f.Items[id]
	return t, ok, nil
}

func (f *FakeWakeword) List(context.Context) ([]store.WakeWordTemplate, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	out := make([]store.WakeWordTemplate, 0, len(f.Items))
	for _, v := range f.Items {
		out = append(out, v)
	}
	return out, nil
}

type FakeSpeaker struct {
	Items     map[string]store.SpeakerProfile
	UpsertErr error
	DeleteErr error
	ListErr   error
}

func (f *FakeSpeaker) Upsert(_ context.Context, p store.SpeakerProfile) error {
	if f.UpsertErr != nil {
		return f.UpsertErr
	}
	if f.Items == nil {
		f.Items = map[string]store.SpeakerProfile{}
	}
	f.Items[p.ID] = p
	return nil
}

func (f *FakeSpeaker) Delete(_ context.Context, id string) (bool, error) {
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	_, ok := f.Items[id]
	delete(f.Items, id)
	return ok, nil
}

func (f *FakeSpeaker) List(context.Context) ([]store.SpeakerProfile, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	out := make([]store.SpeakerProfile, 0, len(f.Items))
	for _, v := range f.Items {
		out = append(out, v)
	}
	return out, nil
}

var (
	_ sttH.STTStreamConfigRepository = (*FakeSTTStreamConfig)(nil)
	_ sttH.WakewordRepository        = (*FakeWakeword)(nil)
	_ sttH.SpeakerRepository         = (*FakeSpeaker)(nil)
)
