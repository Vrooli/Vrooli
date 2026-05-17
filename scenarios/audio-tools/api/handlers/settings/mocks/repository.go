// Package mocks holds the consumer-side seam fakes for the settings handler.
package mocks

import (
	"context"
	"sync/atomic"

	settingsH "audio-tools/handlers/settings"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/store"
)

// FakeProviderConfig satisfies handlers/settings.ProviderConfigRepository.
type FakeProviderConfig struct {
	Cfg       store.ProviderConfig
	GetErr    error
	UpdateErr error
	GetCalls  atomic.Int64
	UpdCalls  atomic.Int64
}

func (f *FakeProviderConfig) Get(context.Context) (store.ProviderConfig, error) {
	f.GetCalls.Add(1)
	return f.Cfg, f.GetErr
}
func (f *FakeProviderConfig) Update(_ context.Context, p store.ProviderConfigPatch) (store.ProviderConfig, error) {
	f.UpdCalls.Add(1)
	return f.Cfg, f.UpdateErr
}

// FakeBYOK satisfies handlers/settings.BYOKRepository.
type FakeBYOK struct {
	Creds      []byokstore.Credential
	UpsertErr  error
	DeleteErr  error
	ListErr    error
	ListCalls  atomic.Int64
	UpsCalls   atomic.Int64
	DelCalls   atomic.Int64
}

func (f *FakeBYOK) List(context.Context) ([]byokstore.Credential, error) {
	f.ListCalls.Add(1)
	return f.Creds, f.ListErr
}
func (f *FakeBYOK) Upsert(_ context.Context, providerID, capability, _ string) (byokstore.Credential, error) {
	f.UpsCalls.Add(1)
	return byokstore.Credential{ProviderID: providerID, Capability: capability}, f.UpsertErr
}
func (f *FakeBYOK) Delete(context.Context, string, string) (bool, error) {
	f.DelCalls.Add(1)
	return true, f.DeleteErr
}

// FakeVoiceOverrides satisfies handlers/settings.VoiceOverridesRepository.
type FakeVoiceOverrides struct {
	Rows   []store.VoiceOverride
	ListErr error
	SetErr  error
}

func (f *FakeVoiceOverrides) List(context.Context) ([]store.VoiceOverride, error) {
	return f.Rows, f.ListErr
}
func (f *FakeVoiceOverrides) Set(context.Context, store.VoiceOverride) error { return f.SetErr }

var (
	_ settingsH.ProviderConfigRepository = (*FakeProviderConfig)(nil)
	_ settingsH.BYOKRepository           = (*FakeBYOK)(nil)
	_ settingsH.VoiceOverridesRepository = (*FakeVoiceOverrides)(nil)
)
