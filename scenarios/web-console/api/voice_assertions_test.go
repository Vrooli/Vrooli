package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// readUISource reads a UI source file from the sibling `ui/src` tree.
// Fails the test loudly if the file is missing — that would indicate a
// rename/move that needs the assertion updated in lockstep.
func readUISource(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join("..", "ui", "src", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestGreenfield_SkipVerificationIsStrictTrue enforces the greenfield
// contract that the `skip_speaker_verification` query parameter is matched
// exclusively against the literal string "true". Loose parses like
// `strconv.ParseBool` (which accepts "1", "TRUE", "yes") would silently
// broaden the bypass surface and weaken the typo-safety guarantee.
//
func TestGreenfield_SkipVerificationIsStrictTrue(t *testing.T) {
	data, err := os.ReadFile("voice_transcribe.go")
	if err != nil {
		t.Fatalf("read voice_transcribe.go: %v", err)
	}
	src := string(data)

	// The handler must reference the query parameter.
	if !strings.Contains(src, `"skip_speaker_verification"`) {
		t.Fatal("voice_transcribe.go does not reference skip_speaker_verification — handler regressed")
	}

	// The comparison must be equality against the literal "true". Reject any
	// ParseBool or case-insensitive matching that could broaden the accepted
	// values.
	strict := regexp.MustCompile(`Query\(\)\.Get\("skip_speaker_verification"\)\s*==\s*"true"`)
	if !strict.MatchString(src) {
		t.Errorf("voice_transcribe.go: skip_speaker_verification must be matched with `== \"true\"`; other matching forms are forbidden")
	}

	banned := []string{
		"strconv.ParseBool", // would accept "1", "t", "T", "TRUE", "True"
		"EqualFold",         // would accept "TRUE", "True"
		"ToLower",           // implies case-insensitive matching
	}
	for _, b := range banned {
		if strings.Contains(src, b) {
			t.Errorf("voice_transcribe.go: banned construct %q detected — skip_speaker_verification must use strict equality", b)
		}
	}
}

// TestGreenfield_VoiceSkipVerificationCounterExposed enforces that the
// bypass counter is surfaced on the /metrics JSON so operators can monitor
// bypass volume without reading logs.
func TestGreenfield_VoiceSkipVerificationCounterExposed(t *testing.T) {
	data, err := os.ReadFile("metrics.go")
	if err != nil {
		t.Fatalf("read metrics.go: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "VoiceSkipVerificationTotal") {
		t.Error("metrics.go: VoiceSkipVerificationTotal field missing from Metrics struct")
	}
	if !strings.Contains(src, `"voice_skip_verification_total"`) {
		t.Error("metrics.go: voice_skip_verification_total JSON tag missing from MetricsResponse")
	}
}

// TestGreenfield_UIRejectionContractIntact locks in the UI-side contract
// for the STT voice-filter-retry feature. The legacy `speakerNotice` field
// and its auto-dismiss timer must stay removed; `rejectedAudio` and the
// `VoiceProvider` retention seams must stay present.
func TestGreenfield_UIRejectionContractIntact(t *testing.T) {
	hook := readUISource(t, "hooks/useVoiceInput.ts")
	if strings.Contains(hook, "speakerNoticeTimerRef") {
		t.Error("useVoiceInput.ts: speakerNoticeTimerRef reintroduced — auto-dismiss timer must stay removed")
	}
	if regexp.MustCompile(`\bspeakerNotice\b`).MatchString(hook) {
		t.Error("useVoiceInput.ts: speakerNotice reintroduced — field removed in favor of rejectedAudio")
	}

	types := readUISource(t, "hooks/voice/types.ts")
	if !regexp.MustCompile(`getLastTurnAudio\s*\(\s*\)\s*:\s*LastTurnAudio\s*\|\s*null`).MatchString(types) {
		t.Error("types.ts: VoiceProvider must declare getLastTurnAudio(): LastTurnAudio | null")
	}
	if !regexp.MustCompile(`disposeLastTurn\s*\(\s*\)\s*:\s*void`).MatchString(types) {
		t.Error("types.ts: VoiceProvider must declare disposeLastTurn(): void")
	}
	if !regexp.MustCompile(`rejectedAudio:\s*VoiceRejection\s*\|\s*null`).MatchString(types) {
		t.Error("types.ts: VoiceInputState must include rejectedAudio: VoiceRejection | null")
	}
	if regexp.MustCompile(`speakerNotice:\s*string\s*\|\s*null`).MatchString(types) {
		t.Error("types.ts: speakerNotice field reintroduced — replaced by rejectedAudio")
	}

	workspace := readUISource(t, "components/Workspace.tsx")
	if strings.Contains(workspace, "speakerNotice") {
		t.Error("Workspace.tsx: speakerNotice reference reintroduced")
	}
	if !strings.Contains(workspace, "VoiceRejectionBanner") {
		t.Error("Workspace.tsx: VoiceRejectionBanner render missing")
	}

	api := readUISource(t, "lib/api.ts")
	if !strings.Contains(api, "export async function transcribeAudioBypassFilter") {
		t.Error("lib/api.ts: transcribeAudioBypassFilter function missing")
	}
	// Client must send the exact literal "true" on the query param; anything
	// else would risk divergence with the server's strict matching.
	if !regexp.MustCompile(`skip_speaker_verification[^"']*["']true["']`).MatchString(api) {
		t.Error("lib/api.ts: transcribeAudioBypassFilter must send skip_speaker_verification=\"true\"")
	}

	for _, path := range []string{
		"hooks/voice/WhisperProvider.ts",
		"hooks/voice/VoiceStreamProvider.ts",
		"hooks/voice/WebSpeechProvider.ts",
	} {
		src := readUISource(t, path)
		if !regexp.MustCompile(`getLastTurnAudio\s*\(`).MatchString(src) {
			t.Errorf("%s: must implement getLastTurnAudio()", path)
		}
		if !regexp.MustCompile(`disposeLastTurn\s*\(`).MatchString(src) {
			t.Errorf("%s: must implement disposeLastTurn()", path)
		}
	}
}
