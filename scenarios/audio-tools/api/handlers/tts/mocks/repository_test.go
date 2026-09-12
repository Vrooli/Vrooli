package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestFakes_Smoke(t *testing.T) {
	c := &FakeTTSConfig{CfgJSON: "{}", SummJSON: "{}", Ok: true}
	cfg, summ, ok, err := c.Get(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "{}", cfg)
	require.Equal(t, "{}", summ)
	require.NoError(t, c.Set(context.Background(), "", ""))
	c.GetErr = errors.New("e")
	_, _, _, err = c.Get(context.Background())
	require.Error(t, err)

	p := &FakePlayback{}
	require.NoError(t, p.Insert(context.Background(), store.PlaybackEvent{}))
	out, err := p.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, out, 1)
}
