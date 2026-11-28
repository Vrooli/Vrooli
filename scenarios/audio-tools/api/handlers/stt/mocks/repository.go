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

type fakeRepository[T any] struct {
	Items     map[string]T
	UpsertErr error
	DeleteErr error
	GetErr    error
	ListErr   error
}

func (f *fakeRepository[T]) upsert(value T, id string) error {
	if f.UpsertErr != nil {
		return f.UpsertErr
	}
	if f.Items == nil {
		f.Items = map[string]T{}
	}
	f.Items[id] = value
	return nil
}

func (f *fakeRepository[T]) delete(id string) (bool, error) {
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	_, ok := f.Items[id]
	delete(f.Items, id)
	return ok, nil
}

func (f *fakeRepository[T]) get(id string) (T, bool, error) {
	if f.GetErr != nil {
		var zero T
		return zero, false, f.GetErr
	}
	value, ok := f.Items[id]
	return value, ok, nil
}

func (f *fakeRepository[T]) list() ([]T, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	out := make([]T, 0, len(f.Items))
	for _, v := range f.Items {
		out = append(out, v)
	}
	return out, nil
}

type FakeWakeword struct {
	fakeRepository[store.WakeWordTemplate]
}

func (f *FakeWakeword) Upsert(_ context.Context, value store.WakeWordTemplate) error {
	return f.upsert(value, value.ID)
}
func (f *FakeWakeword) Delete(_ context.Context, id string) (bool, error) { return f.delete(id) }
func (f *FakeWakeword) Get(_ context.Context, id string) (store.WakeWordTemplate, bool, error) {
	return f.get(id)
}
func (f *FakeWakeword) List(context.Context) ([]store.WakeWordTemplate, error) { return f.list() }

type FakeSpeaker struct {
	fakeRepository[store.SpeakerProfile]
}

func (f *FakeSpeaker) Upsert(_ context.Context, value store.SpeakerProfile) error {
	return f.upsert(value, value.ID)
}
func (f *FakeSpeaker) Delete(_ context.Context, id string) (bool, error) { return f.delete(id) }
func (f *FakeSpeaker) Get(_ context.Context, id string) (store.SpeakerProfile, bool, error) {
	return f.get(id)
}
func (f *FakeSpeaker) List(context.Context) ([]store.SpeakerProfile, error) { return f.list() }

var (
	_ sttH.STTStreamConfigRepository = (*FakeSTTStreamConfig)(nil)
	_ sttH.WakewordRepository        = (*FakeWakeword)(nil)
	_ sttH.SpeakerRepository         = (*FakeSpeaker)(nil)
)
