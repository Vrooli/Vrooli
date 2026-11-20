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
	cases := []struct {
		name      string
		configure func(*FakeVrooliClient)
		wantText  string
		wantErr   bool
	}{
		{name: "default", configure: func(*FakeVrooliClient) {}, wantText: "vrooli-summary"},
		{name: "error", configure: func(c *FakeVrooliClient) { c.Err = errors.New("e") }, wantErr: true},
		{name: "result override", configure: func(c *FakeVrooliClient) {
			c.Result = &summarizechain.Result{Text: "explicit"}
		}, wantText: "explicit"},
		{name: "function override wins over error", configure: func(c *FakeVrooliClient) {
			c.Err = errors.New("ignored")
			c.SummarizeFn = func(context.Context, string, string, summarizechain.Request) (*summarizechain.Result, error) {
				return &summarizechain.Result{Text: "fn"}, nil
			}
		}, wantText: "fn"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &FakeVrooliClient{Available: true}
			tc.configure(c)
			require.True(t, c.IsAvailable(context.Background()))
			require.Equal(t, "lpbs-model", c.Model())
			res, err := c.Summarize(context.Background(), "t", "u", summarizechain.Request{})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantText, res.Text)
		})
	}
}
