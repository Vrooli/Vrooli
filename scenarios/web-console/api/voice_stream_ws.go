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

// voiceStreamFlushInterval is how often accumulated audio is sent to Whisper
// for partial transcription. Package-level var for testability.
var voiceStreamFlushInterval = 500 * time.Millisecond

// minDeltaSize is the minimum audio delta (in bytes) before a partial
// transcription is attempted. Very short clips produce poor Whisper results.
const minDeltaSize = 8 * 1024

// overlapBytes is the trailing audio overlap prepended to each delta for
// partial transcription. At 48 kbps (~6 KB/s), 0.8 s ≈ 5 KB. Combined
// with initial_prompt context, this is sufficient for Whisper continuity.
var overlapBytes = 5 * 1024

// skipFinalCoverageThreshold: when partial-transcribed bytes / total audio
// exceeds this ratio and accumulated transcript is non-empty, use the
// accumulated partials as the final result instead of re-transcribing.
var skipFinalCoverageThreshold = 0.70

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

	// WebM init segment: everything before the first Cluster element.
	// Detected lazily once enough data has arrived. Prepended to each
	// partial chunk so Whisper receives a valid container.
	var webmInitSegment []byte
	var webmInitDetected bool

	// Background ticker: periodically send new audio delta to Whisper
	// for partial transcripts, using initial_prompt for context continuity.
	var previousTranscript string
	var totalPartialBytes int
	var accumulatedTranscript string
	go func() {
		ticker := time.NewTicker(voiceStreamFlushInterval)
		defer ticker.Stop()
		firstTick := true
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				bufMu.Lock()
				currentLen := audioBuffer.Len()
				deltaSize := currentLen - lastPartialOffset
				// On the first tick, bypass minDeltaSize to reduce perceived
				// latency for the initial partial transcript.
				if currentLen == 0 || (!firstTick && deltaSize < minDeltaSize) {
					bufMu.Unlock()
					continue
				}

				// Lazily detect WebM init segment on first tick with data.
				if !webmInitDetected {
					initEnd := findWebMInitEnd(audioBuffer.Bytes()[:currentLen])
					if initEnd > 0 {
						webmInitSegment = make([]byte, initEnd)
						copy(webmInitSegment, audioBuffer.Bytes()[:initEnd])
					}
					webmInitDetected = true
				}

				deltaCopy := make([]byte, deltaSize)
				copy(deltaCopy, audioBuffer.Bytes()[lastPartialOffset:currentLen])

				// Prepend trailing overlap from previous audio for Whisper context.
				prevOffset := currentLen - deltaSize // offset before this tick's delta
				overlapStart := prevOffset - overlapBytes
				if overlapStart < 0 {
					overlapStart = 0
				}
				sendData := deltaCopy
				if prevOffset > overlapStart {
					overlap := make([]byte, prevOffset-overlapStart)
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
				firstTick = false

				prompt := lastNWords(accumulatedTranscript, 10)
				text, err := transcribeBytes(ctx, sendData, language, false, prompt)
				if err != nil {
					log.Printf("voice-ws: partial transcribe: %v", err)
					continue
				}
				totalPartialBytes += deltaSize
				text = strings.TrimSpace(text)
				if text == "" {
					continue
				}
				if accumulatedTranscript != "" {
					accumulatedTranscript += " " + text
				} else {
					accumulatedTranscript = text
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
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
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

	// Pipeline: transcribe any un-transcribed tail from the last partial
	// offset. This often pushes coverage above the threshold, avoiding the
	// expensive full-buffer re-transcription.
	bufMu.Lock()
	tailOffset := lastPartialOffset
	totalLen := audioBuffer.Len()
	tailLen := totalLen - tailOffset
	var tailCopy []byte
	if tailLen > 0 {
		tailCopy = make([]byte, tailLen)
		copy(tailCopy, audioBuffer.Bytes()[tailOffset:])
	}
	bufMu.Unlock()

	if len(tailCopy) > 0 {
		prompt := lastNWords(accumulatedTranscript, 10)
		if t, err := transcribeBytes(ctx, tailCopy, language, false, prompt); err == nil {
			t = strings.TrimSpace(t)
			totalPartialBytes += tailLen
			if t != "" {
				if accumulatedTranscript != "" {
					accumulatedTranscript += " " + t
				} else {
					accumulatedTranscript = t
				}
			}
		}
	}

	// Final transcription with the complete audio buffer.
	bufMu.Lock()
	finalAudio := make([]byte, audioBuffer.Len())
	copy(finalAudio, audioBuffer.Bytes())
	bufMu.Unlock()

	if len(finalAudio) == 0 {
		_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgFinal, Text: ""})
		return
	}

	// Use accumulated partials when coverage is sufficient.
	if accumulatedTranscript != "" {
		coverage := float64(totalPartialBytes) / float64(len(finalAudio))
		if coverage >= skipFinalCoverageThreshold {
			_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgFinal, Text: strings.TrimSpace(accumulatedTranscript)})
			return
		}
	}

	text, err := transcribeBytes(ctx, finalAudio, language, true, "")
	if err != nil {
		log.Printf("voice-ws: final transcribe: %v", err)
		_ = writeJSON(VoiceStreamMessage{Type: VoiceMsgError, Text: "Final transcription failed"})
		return
	}

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

// transcribeBytes sends audio bytes to the Whisper /asr endpoint and returns
// the transcribed text. When transcode is true, audio is transcoded to 16kHz
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
