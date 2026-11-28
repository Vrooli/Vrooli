package capabilities

import (
	"testing"

	"github.com/stretchr/testify/require"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

func TestCapabilityForFeatureSpeakerVerification(t *testing.T) {
	for _, feature := range []string{"voice-speaker-verification", "voice-enrollment"} {
		capability, ok := CapabilityForFeature(feature)
		require.True(t, ok, feature)
		require.Equal(t, diagv1.Capability_CAPABILITY_STT, capability, feature)
	}
}
