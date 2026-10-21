// Package clients hosts LPBS audio-gateway clients per capability.
//
// Costs are hard-coded constants per the provider-routing contract:
// they are not env-overridable. Operators who want different pricing run
// their own LPBS instance with different rates.
//
// Status: capability clients remain unavailable pending
// `execute/lpbs-audio-gateway-endpoints` landing on the LPBS side. The
// composition root wires the Vrooli tier, while AUDIO_AI_ENABLE_VROOLI
// defaults to false until hosted delivery is qualified.
package clients

// Operation cost constants in credits.
const (
	CostTranscribePerSecond = 1
	CostSynthesizePerKChars = 5
	CostSummarizePerKTokens = 2
)

// AppBundleKey is the LPBS app-bundle key audio-tools reports usage under.
const AppBundleKey = "audio-tools"
