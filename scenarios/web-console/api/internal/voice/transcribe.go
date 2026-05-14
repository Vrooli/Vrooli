package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	MaxAudioSize                  = 10 << 20
	MaxSpeakerEnrollmentAudioSize = 20 << 20
)

// ResolveWhisperURL returns the Whisper ASR endpoint URL from WHISPER_URL env
// var with a sensible default for cross-platform portability.
func ResolveWhisperURL() string {
	base := "http://localhost:8090"
	if v := os.Getenv("WHISPER_URL"); v != "" {
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
			log.Printf("voice: transcode failed, sending raw: %v", tcErr)
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

	resp, err := http.DefaultClient.Do(req)
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
