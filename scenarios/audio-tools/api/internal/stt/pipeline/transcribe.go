package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"audio-tools/internal/envx"
	"audio-tools/internal/logx"
)

// PackageLogger is the package-level logx.Logger used by stateless
// helpers like TranscribeBytes. Production leaves it at logx.Std{};
// tests swap in mocks.FakeLogger via SetPackageLogger.
var packageLogger logx.Logger = logx.Std{}

// SetPackageLogger overrides the package-level logger. Returns the
// previous logger so tests can restore via t.Cleanup.
func SetPackageLogger(l logx.Logger) logx.Logger {
	prev := packageLogger
	packageLogger = l
	return prev
}

const (
	MaxAudioSize                  = 10 << 20
	MaxSpeakerEnrollmentAudioSize = 20 << 20
)

// seam: HTTPDoer sends outbound transcription requests. Production wires an
// *http.Client from main.go; tests can provide a fake or httptest client.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

var _ HTTPDoer = (*http.Client)(nil)

// ResolveWhisperURL returns the Whisper ASR endpoint URL from WHISPER_URL env
// var with a sensible default for cross-platform portability. Reads the env
// via the canonical envx.OS seam in production; tests pass envx.Reader
// directly via ResolveWhisperURLWith.
func ResolveWhisperURL() string {
	return ResolveWhisperURLWith(envx.OS{})
}

// ResolveWhisperURLWith is the env-reader-injected variant. Tests pass a
// mocks.FakeEnv to assert the WHISPER_URL read without t.Setenv.
func ResolveWhisperURLWith(env envx.Reader) string {
	if env == nil {
		env = envx.OS{}
	}
	base := "http://localhost:8090"
	if v := env.Get("WHISPER_URL"); v != "" {
		base = v
	}
	return base + "/asr?output=json"
}

// TranscribeBytes sends audio bytes to the Whisper /asr endpoint and returns
// the transcribed text.
//
// When transcode is non-nil and used, audio is transcoded to 16kHz mono WAV
// via ffmpeg for best accuracy. The language parameter is an ISO-639-1 code
// (e.g. "en"); when empty, Whisper auto-detects. initialPrompt provides
// Whisper with context from previous transcription segments.
func TranscribeBytes(
	ctx context.Context,
	whisperURL string,
	httpClient HTTPDoer,
	transcode func(context.Context, []byte) ([]byte, error),
	audio []byte,
	language string,
	doTranscode bool,
	initialPrompt string,
) (string, error) {
	filename := "recording.webm"
	transcoded := audio
	if doTranscode && transcode != nil {
		out, tcErr := transcode(ctx, audio)
		if tcErr != nil {
			packageLogger.Printf("voice: transcode failed, sending raw: %v", tcErr)
			out = audio
		}
		transcoded = out
		if len(transcoded) > 0 && len(audio) > 0 && &transcoded[0] != &audio[0] {
			filename = "recording.wav"
		}
	}

	targetURL := whisperURL
	if language != "" {
		targetURL += "&language=" + language
	}
	if initialPrompt != "" {
		targetURL += "&initial_prompt=" + url.QueryEscape(initialPrompt)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("audio_file", filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, bytes.NewReader(transcoded)); err != nil {
			pw.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if httpClient == nil {
		return "", fmt.Errorf("voice transcription HTTP client is not configured")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper returned status %d", resp.StatusCode)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Text, nil
}

// whisperHallucinations contains phrases Whisper commonly produces when
// transcribing silent or near-silent audio. Stored WITHOUT trailing
// punctuation — IsWhisperHallucination strips trailing punctuation before
// lookup so "Thanks for watching!" matches "thanks for watching".
var whisperHallucinations = map[string]struct{}{
	"":                       {},
	"...":                    {},
	"bye":                    {},
	"goodbye":                {},
	"like and subscribe":     {},
	"please subscribe":       {},
	"so":                     {},
	"subscribe":              {},
	"the end":                {},
	"thank you":              {},
	"thank you for watching": {},
	"thank you very much":    {},
	"thanks":                 {},
	"thanks for watching":    {},
	"you":                    {},
}

func IsWhisperHallucination(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.TrimRight(normalized, ".,;:!?")
	_, found := whisperHallucinations[normalized]
	return found
}

// LastNWords returns the last n whitespace-delimited words of s.
func LastNWords(s string, n int) string {
	if n <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= n {
		return s
	}
	return strings.Join(words[len(words)-n:], " ")
}

func TruncateForLog(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

func stripTrailingPunct(w string) string {
	return strings.TrimRight(w, ".,;:!?\"')")
}

// DeduplicateOverlap merges newText into accumulated by detecting the longest
// suffix of accumulated (in words) that matches a prefix of newText.
// Comparison is case-insensitive and ignores trailing punctuation.
func DeduplicateOverlap(accumulated, newText string) string {
	if accumulated == "" {
		return newText
	}
	if newText == "" {
		return accumulated
	}

	accWords := strings.Fields(accumulated)
	newWords := strings.Fields(newText)
	maxCheck := min(len(accWords), len(newWords))

	bestOverlap := 0
	for k := maxCheck; k >= 1; k-- {
		match := true
		for i := range k {
			if !strings.EqualFold(stripTrailingPunct(accWords[len(accWords)-k+i]), stripTrailingPunct(newWords[i])) {
				match = false
				break
			}
		}
		if match {
			bestOverlap = k
			break
		}
	}

	if bestOverlap > 0 && bestOverlap < len(newWords) {
		return accumulated + " " + strings.Join(newWords[bestOverlap:], " ")
	}
	if bestOverlap == len(newWords) {
		return accumulated
	}
	return accumulated + " " + newText
}
