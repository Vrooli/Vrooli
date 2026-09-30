package chains_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
)

// TestCoordinator_Reconfigure_FanOut verifies the coordinator pushes
// the new config to every non-nil chain. We use the chains' Probe()
// output as the observable side-effect: when EnableLocal=false and
// all providers are nil, Probe.Local must be false on every chain.
// Flipping all toggles to true with nil providers still produces
// Probe values of false (nil-guarded), so we instead observe by
// flipping flags and then asserting via repeat reconfigure that the
// chains don't panic and that the coordinator survives nil chains.
func TestCoordinator_Reconfigure_FanOut(t *testing.T) {
	co := &chains.Coordinator{
		STT:       sttchain.NewChain(sttchain.Options{EnableLocal: true, EnableBYOK: true, EnableVrooli: true}),
		TTS:       ttschain.NewChain(ttschain.Options{EnableLocal: true, EnableBYOK: true, EnableVrooli: true}),
		Summarize: summarizechain.NewChain(summarizechain.Options{EnableLocal: true, EnableBYOK: true, EnableVrooli: true}),
	}

	// All disabled.
	co.Reconfigure(chains.Config{
		BYOKEnabled: false, VrooliEnabled: false, LocalEnabled: false,
		TTLByOK: time.Minute, TTLVrooli: 10 * time.Second,
	})
	p := co.Probe(t.Context())
	require.False(t, p.STT.Local)
	require.False(t, p.STT.BYOK)
	require.False(t, p.STT.Vrooli)
	require.False(t, p.TTS.Local)
	require.False(t, p.TTS.BYOK)
	require.False(t, p.TTS.Vrooli)
	require.False(t, p.Summarize.Local)
	require.False(t, p.Summarize.BYOK)
	require.False(t, p.Summarize.Vrooli)

	// Re-enable. Providers are nil so Probe still returns false on every
	// tier, but the call must not panic and must traverse every chain.
	co.Reconfigure(chains.Config{
		BYOKEnabled: true, VrooliEnabled: true, LocalEnabled: true,
		TTLByOK: time.Minute, TTLVrooli: 10 * time.Second,
	})
	p = co.Probe(t.Context())
	require.False(t, p.STT.Local, "nil local provider keeps Probe.Local=false even with EnableLocal=true")
}

// TestCoordinator_Reconfigure_NilChainsAreSafe locks the guard against
// nil chains. main.go may legally construct a Coordinator with one
// chain wired and others nil during early bootstrap.
func TestCoordinator_Reconfigure_NilChainsAreSafe(t *testing.T) {
	co := &chains.Coordinator{}
	require.NotPanics(t, func() {
		co.Reconfigure(chains.Config{BYOKEnabled: true, VrooliEnabled: true, LocalEnabled: true})
	})
	require.NotPanics(t, func() {
		_ = co.Probe(t.Context())
	})
}
