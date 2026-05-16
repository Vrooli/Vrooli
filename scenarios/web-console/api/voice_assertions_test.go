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

// readEmbedSource reads a source file from the @audio-tools/embed package
// that ships the audio hooks/components web-console adopts. The hooks/voice
// and hooks/tts trees moved out of web-console as part of the audio-tools
// adoption — the contract assertions follow them.
func readEmbedSource(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join("..", "..", "audio-tools", "embed", "src", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// TestGreenfield_VoiceSkipVerificationCounterExposed enforces that the
// bypass counter is surfaced on the /metrics JSON so operators can monitor
// bypass volume without reading logs.
func TestGreenfield_VoiceSkipVerificationCounterExposed(t *testing.T) {
	data, err := os.ReadFile("internal/metrics/metrics.go")
	if err != nil {
		t.Fatalf("read internal/metrics/metrics.go: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "VoiceSkipVerificationTotal") {
		t.Error("internal/metrics/metrics.go: VoiceSkipVerificationTotal field missing from Metrics struct")
	}
	if !strings.Contains(src, `"voice_skip_verification_total"`) {
		t.Error("internal/metrics/metrics.go: voice_skip_verification_total JSON tag missing from Response")
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

	types := readEmbedSource(t, "hooks/voice/types.ts")
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

	api := readEmbedSource(t, "api/voice.ts")
	if !strings.Contains(api, "transcribeAudioBypassFilter") {
		t.Error("embed api/voice.ts: transcribeAudioBypassFilter function missing")
	}

	for _, path := range []string{
		"hooks/voice/WhisperProvider.ts",
		"hooks/voice/VoiceStreamProvider.ts",
		"hooks/voice/WebSpeechProvider.ts",
	} {
		src := readEmbedSource(t, path)
		if !regexp.MustCompile(`getLastTurnAudio\s*\(`).MatchString(src) {
			t.Errorf("%s: must implement getLastTurnAudio()", path)
		}
		if !regexp.MustCompile(`disposeLastTurn\s*\(`).MatchString(src) {
			t.Errorf("%s: must implement disposeLastTurn()", path)
		}
	}
}
