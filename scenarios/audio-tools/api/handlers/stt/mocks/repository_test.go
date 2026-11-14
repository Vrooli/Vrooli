package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestFakes_Smoke(t *testing.T) {
	sc := &FakeSTTStreamConfig{Json: "{}", Ok: true}
	cfg, ok, err := sc.Get(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "{}", cfg)
	require.NoError(t, sc.Set(context.Background(), ""))

	w := &FakeWakeword{}
	require.NoError(t, w.Upsert(context.Background(), store.WakeWordTemplate{ID: "w1"}))
	_, ok, _ = w.Get(context.Background(), "w1")
	require.True(t, ok)
	out, _ := w.List(context.Background())
	require.Len(t, out, 1)
	deleted, _ := w.Delete(context.Background(), "w1")
	require.True(t, deleted)

	sp := &FakeSpeaker{}
	require.NoError(t, sp.Upsert(context.Background(), store.SpeakerProfile{ID: "s1"}))
	out2, _ := sp.List(context.Background())
	require.Len(t, out2, 1)
	deleted2, _ := sp.Delete(context.Background(), "s1")
	require.True(t, deleted2)
}
