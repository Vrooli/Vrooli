package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/byokstore"
	"audio-tools/internal/store"
)

func TestFakes_Smoke(t *testing.T) {
	pc := &FakeProviderConfig{Cfg: store.ProviderConfig{}}
	_, err := pc.Get(context.Background())
	require.NoError(t, err)
	_, err = pc.Update(context.Background(), store.ProviderConfigPatch{})
	require.NoError(t, err)
	pc.GetErr = errors.New("e")
	pc.UpdateErr = errors.New("e")
	_, err = pc.Get(context.Background())
	require.Error(t, err)
	_, err = pc.Update(context.Background(), store.ProviderConfigPatch{})
	require.Error(t, err)

	b := &FakeBYOK{Creds: []byokstore.Credential{{ProviderID: "p"}}}
	creds, _ := b.List(context.Background())
	require.Len(t, creds, 1)
	_, err = b.Upsert(context.Background(), "p", "c", "s")
	require.NoError(t, err)
	_, err = b.Delete(context.Background(), "p", "c")
	require.NoError(t, err)

	v := &FakeVoiceOverrides{Rows: []store.VoiceOverride{{}}}
	_, err = v.List(context.Background())
	require.NoError(t, err)
	require.NoError(t, v.Set(context.Background(), store.VoiceOverride{}))
}
