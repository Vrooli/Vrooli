package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
)

func TestFakeBYOK_Smoke(t *testing.T) {
	f := &FakeBYOK{IDStr: "x", Available: true}
	require.Equal(t, "x", f.ID())
	require.True(t, f.IsAvailable(context.Background(), "k"))
	require.Equal(t, "fake-model", f.Model())
	res, _ := f.Summarize(context.Background(), "k", summarizechain.Request{})
	require.Equal(t, "byok-summary", res.Text)
	f.Err = errors.New("e")
	_, err := f.Summarize(context.Background(), "k", summarizechain.Request{})
	require.Error(t, err)
	f.Err = nil
	f.SummarizeFn = func(context.Context, string, summarizechain.Request) (*summarizechain.Result, error) {
		return &summarizechain.Result{Text: "fn"}, nil
	}
	res, _ = f.Summarize(context.Background(), "k", summarizechain.Request{})
	require.Equal(t, "fn", res.Text)
}

func TestFakeVrooliClient_Smoke(t *testing.T) {
	c := &FakeVrooliClient{Available: true}
	require.True(t, c.IsAvailable(context.Background()))
	require.Equal(t, "lpbs-model", c.Model())
	res, _ := c.Summarize(context.Background(), "t", "u", summarizechain.Request{})
	require.Equal(t, "vrooli-summary", res.Text)
	c.Err = errors.New("e")
	_, err := c.Summarize(context.Background(), "t", "u", summarizechain.Request{})
	require.Error(t, err)
	c.Err = nil
	c.SummarizeFn = func(context.Context, string, string, summarizechain.Request) (*summarizechain.Result, error) {
		return &summarizechain.Result{Text: "fn"}, nil
	}
	res, _ = c.Summarize(context.Background(), "t", "u", summarizechain.Request{})
	require.Equal(t, "fn", res.Text)
}
