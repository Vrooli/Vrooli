package stt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/logx"

	"github.com/vrooli/api-core/schedule"
)

// nilSpeakerRepo is a concrete pointer type that satisfies
// SpeakerConfigRepository. A typed nil of this type assigned to the interface is
// non-nil as an interface value, which used to reach repo.Get during
// construction and panic.
type nilSpeakerRepo struct{}

func (*nilSpeakerRepo) Get(context.Context) (string, bool, error) { return "", false, nil }
func (*nilSpeakerRepo) Set(context.Context, string) error         { return nil }

func TestNewConnectHandlerToleratesTypedNilSpeakerConfig(t *testing.T) {
	require.NotPanics(t, func() {
		handler := NewConnectHandler(Deps{
			Logger:        logx.Std{},
			Clock:         schedule.System(),
			SpeakerConfig: (*nilSpeakerRepo)(nil),
		})
		require.NotNil(t, handler)
	})
}
