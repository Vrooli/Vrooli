// Package clients hosts LPBS audio-gateway clients per capability.
//
// Costs are hard-coded constants per the provider-routing contract:
// they are not env-overridable. Operators who want different pricing run
// their own LPBS instance with different rates.
//
// Status: implementations stubbed pending `execute/lpbs-audio-gateway-endpoints`
// landing on the LPBS side. Today AUDIO_AI_ENABLE_VROOLI defaults to false;
// the chains skip the Vrooli tier entirely.
package clients

// Operation cost constants in credits.
const (
	CostTranscribePerSecond = 1
	CostSynthesizePerKChars = 5
	CostSummarizePerKTokens = 2
)

// AppBundleKey is the LPBS app-bundle key audio-tools reports usage under.
const AppBundleKey = "audio-tools"
