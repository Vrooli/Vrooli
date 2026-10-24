package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFakeRunner_Smoke(t *testing.T) {
	f := NewFakeRunner([]byte("out"), nil)
	out, err := f.Run(context.Background(), "ffmpeg", []byte("in"), "-y")
	require.NoError(t, err)
	require.Equal(t, "out", string(out))
	require.Len(t, f.Calls, 1)
	require.Equal(t, "ffmpeg", f.Calls[0].Name)

	// Respond override
	f.Respond = func(name string, args []string) ([]byte, error) {
		return []byte(name), nil
	}
	out, _ = f.Run(context.Background(), "ffprobe", nil)
	require.Equal(t, "ffprobe", string(out))

	// Err pass-through
	f2 := NewFakeRunner(nil, errors.New("boom"))
	_, err = f2.Run(context.Background(), "ffmpeg", nil)
	require.Error(t, err)
}
