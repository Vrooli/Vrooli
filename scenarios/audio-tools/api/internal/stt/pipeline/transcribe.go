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

	"audio-tools/internal/audioformat"
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

	// DefaultWhisperConcurrency is the resource-documented ceiling on
	// concurrent Whisper /asr requests (resources/whisper/docs/API.md).
	// The Service bounds local STT calls to this so N>5 concurrent
	// sessions queue instead of overrunning the sidecar.
	DefaultWhisperConcurrency = 5
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

// TranscriptionResult is the full output of a Whisper /asr call: the
// transcribed text plus the optional per-segment confidence signals
// faster-whisper reports (no_speech_prob / avg_logprob). HasConfidence is
// false when the response carried no segment array (e.g. a non-Whisper
// backend or an empty result), in which case the signal-domain egress
// stage is skipped gracefully.
//
// NoSpeechProb / AvgLogProb are the arithmetic means across the returned
// segments. They are the robust hallucination signals: silence-only audio
// produces a high no_speech_prob and a low (very negative) avg_logprob.
//
// Words carries flattened per-word timing (start/end in seconds, relative
// to the input audio) when the request enabled word_timestamps and the
// backend returned them. Empty for non-Whisper tiers or for engines that
// don't report words. The OverlapAgree streaming strategy uses Words to
// advance committedAudioBytes to the exact audio offset corresponding to
// committed text, eliminating re-emission on subsequent transcriptions.
type TranscriptionResult struct {
	Text          string
	HasConfidence bool
	NoSpeechProb  float64
	AvgLogProb    float64
	Words         []TimedWord
}

// TimedWord is a single word with its audio-relative start/end seconds and
// the model's word-level probability, as reported by faster-whisper when
// word_timestamps=true.
type TimedWord struct {
	Word  string
	Start float64
	End   float64
	Prob  float64
}

// TranscribeBytes sends audio bytes to the Whisper /asr endpoint and returns
// the transcribed text plus per-segment confidence signals.
//
// The audioformat engine owns container handling: canonical PCM is wrapped
// in a WAV header (ffmpeg-free) and real containers pass straight to
// Whisper's own decoder. format is the audioformat codec vocabulary
// describing the bytes; empty triggers a magic-byte sniff. The language
// parameter is an ISO-639-1 code (e.g. "en"); when empty, Whisper
// auto-detects. initialPrompt provides Whisper with context from previous
// transcription segments. When vadFilter is true the request enables
// faster-whisper's built-in voice-activity filter, which strips silence
// before decoding and removes the dominant source of "thank you for
// watching"-style hallucinations at the source.
func TranscribeBytes(
	ctx context.Context,
	whisperURL string,
	httpClient HTTPDoer,
	engine *audioformat.Engine,
	audio []byte,
	format string,
	language string,
	initialPrompt string,
	vadFilter bool,
) (TranscriptionResult, error) {
	if engine == nil {
		engine = audioformat.New()
	}
	codec := audioformat.CodecFromString(format)
	if codec == audioformat.CodecUnknown {
		// Undeclared: sniff the leading bytes. Detect returns ErrUnknownFormat
		// if the stream is neither declared nor recognizable.
		head := audio
		if len(head) > 64 {
			head = head[:64]
		}
		sniffed, derr := audioformat.Detect(audioformat.CodecUnknown, head)
		if derr != nil {
			return TranscriptionResult{}, derr
		}
		codec = sniffed
	}
	transcoded, filename, err := engine.PrepareForWhisper(codec, audio)
	if err != nil {
		return TranscriptionResult{}, err
	}

	targetURL := whisperURL
	if language != "" {
		targetURL += "&language=" + language
	}
	if initialPrompt != "" {
		targetURL += "&initial_prompt=" + url.QueryEscape(initialPrompt)
	}
	if vadFilter {
		// faster-whisper's built-in VAD filter (the resource runs
		// ASR_ENGINE=faster_whisper). Stripping silence before decode is the
		// most effective hallucination fix — Whisper never sees the empty
		// audio it would otherwise narrate as "thank you for watching".
		targetURL += "&vad_filter=true"
	}
	// word_timestamps=true asks faster-whisper to attach per-word
	// start/end/probability to each returned segment. OverlapAgree uses
	// these to advance committedAudioBytes to a real word boundary;
	// VADSegment ignores them. Always-on: the response cost is negligible
	// (one extra array of floats per segment) and the field is required by
	// the streaming-commit algorithm.
	targetURL += "&word_timestamps=true"

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
		return TranscriptionResult{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if httpClient == nil {
		return TranscriptionResult{}, fmt.Errorf("voice transcription HTTP client is not configured")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return TranscriptionResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TranscriptionResult{}, fmt.Errorf("whisper returned status %d", resp.StatusCode)
	}

	// The /asr?output=json response carries the full text plus a segments
	// array; each segment reports the per-segment confidence signals
	// faster-whisper computes. We keep the text (unchanged behaviour) and
	// fold the segment signals into a single TranscriptionResult so the
	// signal-domain egress stage can gate on them.
	var result struct {
		Text     string `json:"text"`
		Segments []struct {
			NoSpeechProb float64 `json:"no_speech_prob"`
			AvgLogprob   float64 `json:"avg_logprob"`
			Words        []struct {
				Word        string  `json:"word"`
				Start       float64 `json:"start"`
				End         float64 `json:"end"`
				Probability float64 `json:"probability"`
			} `json:"words"`
		} `json:"segments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TranscriptionResult{}, err
	}

	out := TranscriptionResult{Text: result.Text}
	if n := len(result.Segments); n > 0 {
		var sumNoSpeech, sumLogprob float64
		for _, seg := range result.Segments {
			sumNoSpeech += seg.NoSpeechProb
			sumLogprob += seg.AvgLogprob
			for _, w := range seg.Words {
				out.Words = append(out.Words, TimedWord{
					Word:  w.Word,
					Start: w.Start,
					End:   w.End,
					Prob:  w.Probability,
				})
			}
		}
		out.HasConfidence = true
		out.NoSpeechProb = sumNoSpeech / float64(n)
		out.AvgLogProb = sumLogprob / float64(n)
	}
	return out, nil
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

// NormalizeToken returns the case- and trailing-punctuation-normalized
// form of a single whitespace-delimited token. Used by both the
// DeduplicateOverlap defense and the OverlapAgree strategy's
// longestAgreedPrefix gate so consecutive Whisper hypotheses that
// differ only in capitalization or punctuation jitter ("Hello world"
// vs "hello world.") still register as agreeing.
func NormalizeToken(w string) string {
	return strings.ToLower(stripTrailingPunct(w))
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
