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
	VoiceMsgDone    = "done"    // client → server: recording finished
	VoiceMsgPartial = "partial" // server → client: partial transcript
	VoiceMsgFinal   = "final"   // server → client: definitive transcript
	VoiceMsgError   = "error"   // server → client: error
)

// VoiceStreamMessage is the JSON message format for voice streaming.
type VoiceStreamMessage struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
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

	// Audio buffer protected by mutex. Binary frames append here.
	var bufMu sync.Mutex
	var audioBuffer bytes.Buffer
	var lastPartialOffset int
	done := make(chan struct{})
	var tickerDone sync.WaitGroup
	tickerDone.Add(1)

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
			case <-ticker.C:
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
				if overlapStart < 0 {
					overlapStart = 0
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
				if len(webmInitSegment) > 0 && lastPartialOffset > 0 {
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

				if text == "" {
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
		}
	}

	close(done)
	partialCancel()   // cancel any in-flight partial HTTP request to Whisper
	tickerDone.Wait() // wait for goroutine to fully exit before final retranscribe

	log.Printf("voice-ws: recording done, audioBytes=%d, partials=%d", audioBuffer.Len(), partialCount)

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

	t0 := time.Now()
	text, err := transcribeBytes(finalCtx, finalAudio, language, true, "")
	if err != nil {
		log.Printf("voice-ws: final transcribe failed (%v): %v", time.Since(t0), err)
		_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgError, Text: "Final transcription failed"})
		return
	}

	log.Printf("voice-ws: full retranscribe took %v, text=%q", time.Since(t0), strings.TrimSpace(text))
	log.Printf("voice-ws: session closed, duration=%v, audioBytes=%d, partials=%d", time.Since(sessionStart), len(finalAudio), partialCount)
	_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgFinal, Text: strings.TrimSpace(text)})
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
