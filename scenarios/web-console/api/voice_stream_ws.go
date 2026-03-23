// DOC: docs/internal/SEAMS.md#voice-stream-websocket-seam
package main

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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Voice streaming WebSocket message types.
const (
	VoiceMsgDone            = "done"             // client → server: recording finished
	VoiceMsgSegmentBoundary = "segment-boundary" // client → server: VAD detected silence gap
	VoiceMsgVadSpeechStart  = "vad-speech-start" // client → server: VAD detected speech
	VoiceMsgVadSpeechEnd    = "vad-speech-end"   // client → server: VAD detected silence
	VoiceMsgPartial         = "partial"          // server → client: partial transcript
	VoiceMsgFinal           = "final"            // server → client: definitive transcript
	VoiceMsgSegmentFinal    = "segment-final"    // server → client: high-quality segment transcription
	VoiceMsgSegmentAccepted = "segment-accepted" // server → client: speaker verification accepted a segment
	VoiceMsgSegmentRejected = "segment-rejected" // server → client: speaker verification rejected a segment
	VoiceMsgSpeakerStatus   = "speaker-status"   // server → client: speaker verification session status
	VoiceMsgError           = "error"            // server → client: error
)

// VoiceStreamMessage is the JSON message format for voice streaming.
type VoiceStreamMessage struct {
	Type              string  `json:"type"`
	Text              string  `json:"text,omitempty"`
	SegmentIndex      int     `json:"segmentIndex,omitempty"`
	Score             float64 `json:"score,omitempty"`
	Threshold         float64 `json:"threshold,omitempty"`
	Enabled           bool    `json:"enabled,omitempty"`
	ProfileConfigured bool    `json:"profileConfigured,omitempty"`
	ProfileID         string  `json:"profileId,omitempty"`
}

// findWebMInitEnd locates the end of the WebM initialization segment (EBML
// header + Segment + Tracks) by scanning for the first Cluster element ID
// (0x1F43B675). Returns the byte offset of the Cluster start, or 0 if not
// found (caller should treat the entire buffer as needing a header).
func findWebMInitEnd(buf []byte) int {
	// Cluster element ID in EBML: 0x1F 0x43 0xB6 0x75
	clusterID := []byte{0x1F, 0x43, 0xB6, 0x75}
	idx := bytes.Index(buf, clusterID)
	if idx < 0 {
		return 0
	}
	return idx
}

// handleVoiceStreamWS upgrades to WebSocket and streams audio chunks to Whisper,
// returning partial and final transcripts.
func (s *Server) handleVoiceStreamWS(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if !s.capabilities.IsAvailable(ctx, "whisper-stt") {
		writeCatalogError(w, "voice_unavailable", "Voice transcription is currently unavailable")
		return
	}

	// Snapshot config for this session — changes via PUT take effect on the
	// next recording, not the one in progress.
	vcfg := s.getVoiceConfig()
	speakerCfg := s.getSpeakerVerificationConfig()

	language := r.URL.Query().Get("language")

	upgrader := websocket.Upgrader{
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 16 * 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("voice-ws: upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	sessionStart := time.Now()
	log.Printf("voice-ws: session opened, language=%q, config: flush=%dms delta=%d overlap=%d",
		language, vcfg.FlushIntervalMs, vcfg.MinDeltaBytes, vcfg.OverlapBytes)

	// Derived context for partial transcriptions — cancelled on recording stop
	// so in-flight Whisper HTTP calls abort immediately, freeing the (often
	// single-threaded) Whisper server for the final retranscribe.
	partialCtx, partialCancel := context.WithCancel(ctx)
	defer partialCancel()

	var writeMu sync.Mutex

	writeJSON := func(msg VoiceStreamMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(msg)
	}

	if speakerCfg.Enabled {
		_ = writeJSON(VoiceStreamMessage{
			Type:              VoiceMsgSpeakerStatus,
			Enabled:           speakerCfg.Enabled && speakerCfg.Mode != "off",
			ProfileConfigured: speakerCfg.ProfileID != "",
			ProfileID:         speakerCfg.ProfileID,
			Threshold:         speakerCfg.Threshold,
		})
	}

	// Audio buffer protected by mutex. Binary frames append here.
	var bufMu sync.Mutex
	var audioBuffer bytes.Buffer
	var lastPartialOffset int
	done := make(chan struct{})
	var tickerDone sync.WaitGroup
	tickerDone.Add(1)

	// Segment tracking: segmentStartOffset marks the beginning of the current
	// speech segment within the audio buffer. On segment-boundary, we snapshot
	// audio from segmentStartOffset..currentLen for high-quality retranscription.
	var segmentStartOffset int
	var segmentIndex int
	// segmentFinalWg tracks in-flight segment-final goroutines so we can wait
	// for them before closing the WebSocket.
	var segmentFinalWg sync.WaitGroup

	// Channel for segment-boundary requests from the input loop to the ticker goroutine.
	segmentBoundaryCh := make(chan struct{}, 4)

	// VAD speech state: the ticker skips partial transcription when the
	// client-side VAD reports silence, preventing Whisper hallucinations
	// (e.g. "Thank you" from ambient noise). VAD gating only activates
	// once the client sends its first vad-speech-start; until then,
	// partials flow freely (backward-compatible with non-VAD clients).
	var vadSignaled bool  // true after first vad-speech-start received
	var speechActive bool // current VAD speech state
	var speechMu sync.Mutex

	// WebM init segment: everything before the first Cluster element.
	// Detected lazily once enough data has arrived. Prepended to each
	// partial chunk so Whisper receives a valid container.
	var webmInitSegment []byte
	var webmInitDetected bool

	// Background ticker: periodically send new audio delta to Whisper
	// for partial transcripts, using initial_prompt for context continuity.
	var previousTranscript string
	var accumulatedTranscript string
	var partialCount int
	go func() {
		defer tickerDone.Done()
		ticker := time.NewTicker(time.Duration(vcfg.FlushIntervalMs) * time.Millisecond)
		defer ticker.Stop()
		firstTick := true
		for {
			select {
			case <-done:
				return
			case <-partialCtx.Done():
				return
			case <-segmentBoundaryCh:
				// Segment boundary: snapshot the current segment audio and
				// run a high-quality retranscription in a separate goroutine.
				bufMu.Lock()
				currentLen := audioBuffer.Len()
				segAudioLen := currentLen - segmentStartOffset
				if segAudioLen <= 0 {
					bufMu.Unlock()
					continue
				}
				segAudio := make([]byte, segAudioLen)
				copy(segAudio, audioBuffer.Bytes()[segmentStartOffset:currentLen])
				// Prepend WebM init segment for a valid container
				if len(webmInitSegment) > 0 && segmentStartOffset > 0 {
					segAudio = append(webmInitSegment, segAudio...)
				}
				thisSegIdx := segmentIndex
				segmentIndex++
				segmentStartOffset = currentLen
				lastPartialOffset = currentLen
				bufMu.Unlock()

				log.Printf("voice-ws: segment-boundary #%d: audioBytes=%d", thisSegIdx, segAudioLen)
				segmentFinalWg.Add(1)
				go func(audio []byte, idx int) {
					defer segmentFinalWg.Done()
					segCtx, segCancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer segCancel()

					// Skip segment-level speaker verification for short segments —
					// TitaNet embeddings are unreliable under ~2 seconds of audio.
					// The final full-recording verification still gates the output.
					const minSpeakerVerifyBytes = 12_000 // ~2s at 48kbps Opus
					var decision speakerVerificationGateDecision
					if len(audio) >= minSpeakerVerifyBytes {
						decision = s.evaluateSpeakerVerification(segCtx, audio)
					}
					if decision.Enabled {
						if decision.Applied {
							log.Printf(
								"voice-ws: speaker decision #%d matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s",
								idx,
								decision.Matched,
								decision.Allowed,
								decision.Score,
								decision.Threshold,
								decision.ProfileID,
								decision.Mode,
							)
						} else if decision.ErrorMessage != "" {
							log.Printf("voice-ws: segment #%d %s", idx, formatSpeakerDecisionError(decision))
						}
						if !decision.Allowed {
							_ = writeJSON(VoiceStreamMessage{
								Type:         VoiceMsgSegmentRejected,
								SegmentIndex: idx,
								Score:        decision.Score,
								Threshold:    decision.Threshold,
								ProfileID:    decision.ProfileID,
							})
							return
						}
					}

					t0 := time.Now()
					text, err := transcribeBytes(segCtx, audio, language, true, "")
					if err != nil {
						log.Printf("voice-ws: segment-final #%d failed (%v): %v", idx, time.Since(t0), err)
						return
					}
					text = strings.TrimSpace(text)
					log.Printf("voice-ws: segment-final #%d took %v, text=%q", idx, time.Since(t0), truncateForLog(text, 200))
					if isWhisperHallucination(text) {
						log.Printf("voice-ws: segment-final #%d filtered hallucination: %q", idx, text)
						return
					}
					if decision.Enabled && decision.Applied {
						_ = writeJSON(VoiceStreamMessage{
							Type:         VoiceMsgSegmentAccepted,
							SegmentIndex: idx,
							Score:        decision.Score,
							Threshold:    decision.Threshold,
							ProfileID:    decision.ProfileID,
						})
					}
					_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgSegmentFinal, Text: text, SegmentIndex: idx})
				}(segAudio, thisSegIdx)

				// Reset accumulated transcript for partial dedup in the next segment,
				// preserving context for the initial_prompt.
				previousTranscript = ""
				firstTick = true

			case <-ticker.C:
				// Skip partial transcription when VAD reports silence —
				// Whisper hallucinates on silent audio (e.g. "Thank you").
				// Only gate when client has opted in via vad-speech-start.
				speechMu.Lock()
				gated := vadSignaled && !speechActive
				speechMu.Unlock()
				if gated {
					continue
				}

				bufMu.Lock()
				currentLen := audioBuffer.Len()
				deltaSize := currentLen - lastPartialOffset
				// On the first tick, bypass vcfg.MinDeltaBytes to reduce perceived
				// latency for the initial partial transcript.
				if currentLen == 0 || (!firstTick && deltaSize < vcfg.MinDeltaBytes) {
					bufMu.Unlock()
					continue
				}

				// Lazily detect WebM init segment on first tick with data.
				if !webmInitDetected {
					initEnd := findWebMInitEnd(audioBuffer.Bytes()[:currentLen])
					if initEnd > 0 {
						webmInitSegment = make([]byte, initEnd)
						copy(webmInitSegment, audioBuffer.Bytes()[:initEnd])
						log.Printf("voice-ws: webm init segment detected, size=%d bytes", initEnd)
					}
					webmInitDetected = true
				}

				deltaCopy := make([]byte, deltaSize)
				copy(deltaCopy, audioBuffer.Bytes()[lastPartialOffset:currentLen])

				// Prepend trailing overlap from previous audio for Whisper context.
				prevOffset := currentLen - deltaSize // offset before this tick's delta
				overlapStart := prevOffset - vcfg.OverlapBytes
				if overlapStart < segmentStartOffset {
					overlapStart = segmentStartOffset
				}
				sendData := deltaCopy
				actualOverlap := 0
				if prevOffset > overlapStart {
					actualOverlap = prevOffset - overlapStart
					overlap := make([]byte, actualOverlap)
					copy(overlap, audioBuffer.Bytes()[overlapStart:prevOffset])
					sendData = append(overlap, deltaCopy...)
				}

				// Prepend WebM init segment so Whisper receives a valid
				// container even for mid-stream partial chunks.
				if len(webmInitSegment) > 0 && lastPartialOffset > segmentStartOffset {
					sendData = append(webmInitSegment, sendData...)
				} else if len(webmInitSegment) > 0 && lastPartialOffset == segmentStartOffset && segmentStartOffset > 0 {
					// First partial of a new segment after boundary — needs init header
					sendData = append(webmInitSegment, sendData...)
				}

				lastPartialOffset = currentLen
				bufMu.Unlock()
				isFirst := firstTick
				firstTick = false

				prompt := lastNWords(accumulatedTranscript, 10)
				t0 := time.Now()
				text, err := transcribeBytes(partialCtx, sendData, language, false, prompt)
				elapsed := time.Since(t0)
				if err != nil {
					if partialCtx.Err() != nil {
						return // cancelled during shutdown — expected
					}
					log.Printf("voice-ws: partial transcribe failed (%v): %v", elapsed, err)
					continue
				}

				partialCount++
				text = strings.TrimSpace(text)

				// Log per-tick details: partial index, sizes, timing, and Whisper output.
				// This is the primary signal for diagnosing duplication and slowness.
				log.Printf("voice-ws: partial #%d: delta=%d overlap=%d sent=%d bytes, took=%v, first=%t, text=%q",
					partialCount, deltaSize, actualOverlap, len(sendData), elapsed, isFirst, truncateForLog(text, 200))

				if text == "" || isWhisperHallucination(text) {
					if text != "" {
						log.Printf("voice-ws: partial filtered hallucination: %q", text)
					}
					continue
				}

				prevAccum := accumulatedTranscript
				accumulatedTranscript = deduplicateOverlap(accumulatedTranscript, text)

				// Log dedup result when overlap was detected (words were merged).
				if accumulatedTranscript != prevAccum+" "+text && prevAccum != "" {
					addedWords := len(strings.Fields(accumulatedTranscript)) - len(strings.Fields(prevAccum))
					whisperWords := len(strings.Fields(text))
					mergedWords := whisperWords - addedWords
					if mergedWords > 0 {
						log.Printf("voice-ws: dedup: merged %d overlapping word(s), added %d new", mergedWords, addedWords)
					}
				}

				if accumulatedTranscript != previousTranscript {
					previousTranscript = accumulatedTranscript
					if writeErr := writeJSON(VoiceStreamMessage{Type: VoiceMsgPartial, Text: accumulatedTranscript}); writeErr != nil {
						return
					}
				}
			}
		}
	}()

	// Input loop: read binary audio chunks and JSON control messages.
	firstChunk := true
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			if firstChunk {
				log.Printf("voice-ws: first audio chunk, size=%d bytes", len(data))
				firstChunk = false
			}
			bufMu.Lock()
			audioBuffer.Write(data)
			bufMu.Unlock()
			continue
		}

		if msgType == websocket.TextMessage {
			var msg VoiceStreamMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if msg.Type == VoiceMsgDone {
				break
			}
			if msg.Type == VoiceMsgVadSpeechStart {
				speechMu.Lock()
				vadSignaled = true
				speechActive = true
				speechMu.Unlock()
				// Advance both partial and segment offsets past accumulated
				// silence so the next transcription only covers speech onset,
				// not the preceding silence that Whisper would hallucinate on.
				bufMu.Lock()
				pos := audioBuffer.Len()
				lastPartialOffset = pos
				segmentStartOffset = pos
				bufMu.Unlock()
				continue
			}
			if msg.Type == VoiceMsgVadSpeechEnd {
				speechMu.Lock()
				speechActive = false
				speechMu.Unlock()
				continue
			}
			if msg.Type == VoiceMsgSegmentBoundary {
				// Non-blocking send to avoid deadlock if channel is full
				select {
				case segmentBoundaryCh <- struct{}{}:
				default:
					log.Printf("voice-ws: segment-boundary dropped (channel full)")
				}
			}
		}
	}

	close(done)
	partialCancel()   // cancel any in-flight partial HTTP request to Whisper
	tickerDone.Wait() // wait for ticker goroutine to fully exit

	// Wait for any in-flight segment-final transcriptions to complete
	segmentFinalWg.Wait()

	log.Printf("voice-ws: recording done, audioBytes=%d, partials=%d, segments=%d", audioBuffer.Len(), partialCount, segmentIndex)

	// Full retranscription of entire audio buffer. Partials above are for
	// real-time UI feedback only; the final result always comes from a single
	// Whisper pass over the complete audio with transcode=true (ffmpeg WAV)
	// for maximum accuracy.
	bufMu.Lock()
	finalAudio := make([]byte, audioBuffer.Len())
	copy(finalAudio, audioBuffer.Bytes())
	bufMu.Unlock()

	if len(finalAudio) == 0 {
		log.Printf("voice-ws: session closed, duration=%v, audioBytes=0, partials=%d, strategy=empty", time.Since(sessionStart), partialCount)
		_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgFinal, Text: ""})
		return
	}

	// Fresh context for final retranscribe — decoupled from the session
	// timeout so recording duration doesn't eat into transcription time.
	finalCtx, finalCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer finalCancel()

	decision := s.evaluateSpeakerVerification(finalCtx, finalAudio)
	if decision.Enabled {
		if decision.Applied {
			log.Printf(
				"voice-ws: final speaker decision matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s",
				decision.Matched,
				decision.Allowed,
				decision.Score,
				decision.Threshold,
				decision.ProfileID,
				decision.Mode,
			)
		} else if decision.ErrorMessage != "" {
			log.Printf("voice-ws: final %s", formatSpeakerDecisionError(decision))
		}
		if !decision.Allowed {
			_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgFinal, Text: ""})
			return
		}
	}

	t0 := time.Now()
	text, err := transcribeBytes(finalCtx, finalAudio, language, true, "")
	if err != nil {
		log.Printf("voice-ws: final transcribe failed (%v): %v", time.Since(t0), err)
		_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgError, Text: "Final transcription failed"})
		return
	}

	finalText := strings.TrimSpace(text)
	if isWhisperHallucination(finalText) {
		log.Printf("voice-ws: final filtered hallucination: %q", finalText)
		finalText = ""
	}
	log.Printf("voice-ws: full retranscribe took %v, text=%q", time.Since(t0), finalText)
	log.Printf("voice-ws: session closed, duration=%v, audioBytes=%d, partials=%d, segments=%d", time.Since(sessionStart), len(finalAudio), partialCount, segmentIndex)
	_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgFinal, Text: finalText})
}

// lastNWords returns the last n whitespace-delimited words of s.
func lastNWords(s string, n int) string {
	if n <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= n {
		return s
	}
	return strings.Join(words[len(words)-n:], " ")
}

// truncateForLog returns s truncated to maxLen runes with "…" appended if truncated.
// Used for logging Whisper output without flooding log lines.
func truncateForLog(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}

// stripTrailingPunct removes trailing punctuation from a word for comparison.
// This allows "world," to match "world" during overlap detection.
func stripTrailingPunct(w string) string {
	return strings.TrimRight(w, ".,;:!?\"')")
}

// deduplicateOverlap merges newText into accumulated by detecting the longest
// suffix of accumulated (in words) that matches a prefix of newText. Comparison
// is case-insensitive and ignores trailing punctuation, so "Hello," matches
// "hello". Original casing and punctuation from accumulated are preserved.
//
// Example: deduplicateOverlap("the quick brown", "brown fox") → "the quick brown fox"
func deduplicateOverlap(accumulated, newText string) string {
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
		// newText is entirely contained as a suffix of accumulated — nothing new to add.
		return accumulated
	}
	return accumulated + " " + newText
}

// whisperHallucinations contains phrases Whisper commonly produces when
// transcribing silent or near-silent audio. These are filtered as a safety
// net in addition to the primary VAD-gated partial suppression. Entries are
// stored WITHOUT trailing punctuation — isWhisperHallucination strips
// trailing punctuation before lookup so "Thanks for watching!" matches
// "thanks for watching".
var whisperHallucinations = map[string]struct{}{
	"":                        {},
	"...":                     {},
	"bye":                     {},
	"goodbye":                 {},
	"like and subscribe":      {},
	"please subscribe":        {},
	"so":                      {},
	"subscribe":               {},
	"the end":                 {},
	"thank you":               {},
	"thank you for watching":  {},
	"thank you very much":     {},
	"thanks":                  {},
	"thanks for watching":     {},
	"you":                     {},
}

// isWhisperHallucination returns true if the text matches a known Whisper
// hallucination pattern — short, generic phrases it produces from silence.
// Trailing punctuation (.,;:!?) is stripped before lookup so variants like
// "Thanks for watching!" are caught.
func isWhisperHallucination(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.TrimRight(normalized, ".,;:!?")
	_, found := whisperHallucinations[normalized]
	return found
}

// transcribeBytes sends audio bytes to the Whisper /asr endpoint and returns
// the transcribed text.
//
// Performance note: The default `medium` model on GPU takes ~0.3s per call.
// On CPU, expect ~2-4s. For lower VRAM usage at the cost of accuracy, use
// `small` (~0.2s GPU, 1.9 GB VRAM) or `base` (~0.2s GPU, 1.1 GB VRAM).
// For best accuracy, use `large` (~0.5s GPU, 8.2 GB VRAM).
// Since the final result always comes from a single full-buffer retranscribe,
// model speed directly impacts perceived finalization latency.
// When transcode is true, audio is transcoded to 16kHz
// mono WAV via ffmpeg for best accuracy. The language parameter is an ISO-639-1
// code (e.g. "en"); when empty, Whisper auto-detects. initialPrompt provides
// Whisper with context from previous transcription segments.
func transcribeBytes(ctx context.Context, audio []byte, language string, transcode bool, initialPrompt string) (string, error) {
	filename := "recording.webm"
	transcoded := audio
	if transcode {
		var tcErr error
		transcoded, tcErr = transcodeAudio(ctx, audio)
		if tcErr != nil {
			log.Printf("voice: transcode failed, sending raw: %v", tcErr)
			transcoded = audio
		}
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
