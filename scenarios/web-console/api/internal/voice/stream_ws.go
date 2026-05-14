// DOC: docs/internal/SEAMS.md#voice-stream-websocket-seam
package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// VadLookbackMs is the audio lookback margin applied when the client signals
// speech onset. The client-side VAD runs at ~15 Hz with a ~250ms MediaRecorder
// chunk interval, so speech-start messages arrive ~300-500ms after the actual
// onset. We rewind the segment/partial offsets by this amount to avoid clipping.
const VadLookbackMs = 600

// AudioBitrateBps is the expected audio bitrate (matches the frontend
// AUDIO_BITRATE constant of 48 kbps Opus).
const AudioBitrateBps = 48_000

// Voice streaming WebSocket message types.
const (
	MsgDone            = "done"
	MsgSegmentBoundary = "segment-boundary"
	MsgVadSpeechStart  = "vad-speech-start"
	MsgVadSpeechEnd    = "vad-speech-end"
	MsgPartial         = "partial"
	MsgFinal           = "final"
	MsgSegmentFinal    = "segment-final"
	MsgSegmentAccepted = "segment-accepted"
	MsgSegmentRejected = "segment-rejected"
	MsgSpeakerStatus   = "speaker-status"
	MsgError           = "error"
)

// StreamMessage is the JSON message format for voice streaming.
type StreamMessage struct {
	Type              string  `json:"type"`
	Text              string  `json:"text,omitempty"`
	SegmentIndex      int     `json:"segmentIndex,omitempty"`
	Score             float64 `json:"score,omitempty"`
	Threshold         float64 `json:"threshold,omitempty"`
	Enabled           bool    `json:"enabled,omitempty"`
	ProfileConfigured bool    `json:"profileConfigured,omitempty"`
	ProfileID         string  `json:"profileId,omitempty"`
	Extracted         bool    `json:"extracted,omitempty"`
	ExtractionEnabled bool    `json:"extractionEnabled,omitempty"`
}

// FindWebMInitEnd locates the end of the WebM initialization segment by
// scanning for the first Cluster element ID (0x1F43B675). Returns the byte
// offset of the Cluster start, or 0 if not found.
func FindWebMInitEnd(buf []byte) int {
	clusterID := []byte{0x1F, 0x43, 0xB6, 0x75}
	idx := bytes.Index(buf, clusterID)
	if idx < 0 {
		return 0
	}
	return idx
}

// HandleStreamWS upgrades to WebSocket and streams audio chunks to Whisper,
// returning partial and final transcripts.
func (s *Service) HandleStreamWS(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if !s.WhisperAvailable(ctx) {
		s.writeUnavailable(w)
		return
	}

	vcfg := s.GetConfig()
	speakerCfg := s.SpeakerConfigSnapshot()

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

	partialCtx, partialCancel := context.WithCancel(ctx)
	defer partialCancel()

	var writeMu sync.Mutex

	writeJSON := func(msg StreamMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(msg)
	}

	if speakerCfg.Enabled {
		_ = writeJSON(StreamMessage{
			Type:              MsgSpeakerStatus,
			Enabled:           speakerCfg.Enabled && speakerCfg.Mode != "off",
			ProfileConfigured: len(speakerCfg.ProfileIDs) > 0,
			Threshold:         speakerCfg.Threshold,
			ExtractionEnabled: speakerCfg.ExtractionEnabled,
		})
	}

	var bufMu sync.Mutex
	var audioBuffer bytes.Buffer
	var lastPartialOffset int
	done := make(chan struct{})
	var tickerDone sync.WaitGroup
	tickerDone.Add(1)

	var segmentStartOffset int
	var segmentIndex int
	var segmentOffsetSet bool
	var segmentFinalWg sync.WaitGroup

	segmentBoundaryCh := make(chan struct{}, 4)

	var vadSignaled bool
	var speechActive bool
	var speechMu sync.Mutex

	var vadSpeechStartCount int
	var vadSpeechEndCount int
	var partialsGatedBySilence int
	var hallucinationsFiltered int
	var totalLookbackBytes int

	var webmInitSegment []byte
	var webmInitDetected bool

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
				bufMu.Lock()
				currentLen := audioBuffer.Len()
				segAudioLen := currentLen - segmentStartOffset
				if segAudioLen <= 0 {
					bufMu.Unlock()
					continue
				}
				segAudio := make([]byte, segAudioLen)
				copy(segAudio, audioBuffer.Bytes()[segmentStartOffset:currentLen])
				if len(webmInitSegment) > 0 && segmentStartOffset > 0 {
					segAudio = append(webmInitSegment, segAudio...)
				}
				thisSegIdx := segmentIndex
				segmentIndex++
				segmentStartOffset = currentLen
				lastPartialOffset = currentLen
				segmentOffsetSet = false
				bufMu.Unlock()

				log.Printf("voice-ws: segment-boundary #%d: audioBytes=%d", thisSegIdx, segAudioLen)
				segmentFinalWg.Add(1)
				go func(audio []byte, idx int) {
					defer segmentFinalWg.Done()
					segCtx, segCancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer segCancel()

					const minSpeakerVerifyBytes = 12_000
					var decision SpeakerDecision
					transcribeAudio := audio
					if len(audio) >= minSpeakerVerifyBytes {
						if speakerCfg.ExtractionEnabled {
							transcribeAudio, decision = ExtractTargetSpeaker(segCtx, speakerCfg, s.speakerClient, audio)
						} else {
							decision = EvaluateSpeaker(segCtx, speakerCfg, s.speakerClient, audio)
						}
					}
					if decision.Enabled {
						if decision.Applied {
							log.Printf(
								"voice-ws: speaker decision #%d matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s extracted=%v",
								idx, decision.Matched, decision.Allowed, decision.Score, decision.Threshold,
								decision.ProfileID, decision.Mode, decision.Extracted,
							)
						} else if decision.ErrorMessage != "" {
							log.Printf("voice-ws: segment #%d %s", idx, FormatSpeakerDecisionError(decision))
						}
						if !decision.Allowed {
							_ = writeJSON(StreamMessage{
								Type:         MsgSegmentRejected,
								SegmentIndex: idx,
								Score:        decision.Score,
								Threshold:    decision.Threshold,
								ProfileID:    decision.ProfileID,
							})
							return
						}
					}

					t0 := time.Now()
					text, err := s.transcribe(segCtx, transcribeAudio, language, true, "")
					if err != nil {
						log.Printf("voice-ws: segment-final #%d failed (%v): %v", idx, time.Since(t0), err)
						return
					}
					text = strings.TrimSpace(text)
					log.Printf("voice-ws: segment-final #%d took %v, text=%q", idx, time.Since(t0), TruncateForLog(text, 200))
					if IsWhisperHallucination(text) {
						log.Printf("voice-ws: segment-final #%d filtered hallucination: %q", idx, text)
						return
					}
					if decision.Enabled && decision.Applied {
						_ = writeJSON(StreamMessage{
							Type:         MsgSegmentAccepted,
							SegmentIndex: idx,
							Score:        decision.Score,
							Threshold:    decision.Threshold,
							ProfileID:    decision.ProfileID,
							Extracted:    decision.Extracted,
						})
					}
					_ = writeJSON(StreamMessage{Type: MsgSegmentFinal, Text: text, SegmentIndex: idx})
				}(segAudio, thisSegIdx)

				previousTranscript = ""
				firstTick = true

			case <-ticker.C:
				speechMu.Lock()
				gated := vadSignaled && !speechActive
				speechMu.Unlock()
				if gated {
					partialsGatedBySilence++
					continue
				}

				bufMu.Lock()
				currentLen := audioBuffer.Len()
				deltaSize := currentLen - lastPartialOffset
				if currentLen == 0 || (!firstTick && deltaSize < vcfg.MinDeltaBytes) {
					bufMu.Unlock()
					continue
				}

				if !webmInitDetected {
					initEnd := FindWebMInitEnd(audioBuffer.Bytes()[:currentLen])
					if initEnd > 0 {
						webmInitSegment = make([]byte, initEnd)
						copy(webmInitSegment, audioBuffer.Bytes()[:initEnd])
						log.Printf("voice-ws: webm init segment detected, size=%d bytes", initEnd)
					}
					webmInitDetected = true
				}

				deltaCopy := make([]byte, deltaSize)
				copy(deltaCopy, audioBuffer.Bytes()[lastPartialOffset:currentLen])

				prevOffset := currentLen - deltaSize
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

				if len(webmInitSegment) > 0 && lastPartialOffset > segmentStartOffset {
					sendData = append(webmInitSegment, sendData...)
				} else if len(webmInitSegment) > 0 && lastPartialOffset == segmentStartOffset && segmentStartOffset > 0 {
					sendData = append(webmInitSegment, sendData...)
				}

				lastPartialOffset = currentLen
				bufMu.Unlock()
				isFirst := firstTick
				firstTick = false

				prompt := LastNWords(accumulatedTranscript, 10)
				t0 := time.Now()
				text, err := s.transcribe(partialCtx, sendData, language, false, prompt)
				elapsed := time.Since(t0)
				if err != nil {
					if partialCtx.Err() != nil {
						return
					}
					log.Printf("voice-ws: partial transcribe failed (%v): %v", elapsed, err)
					continue
				}

				partialCount++
				text = strings.TrimSpace(text)
				log.Printf("voice-ws: partial #%d: delta=%d overlap=%d sent=%d bytes, took=%v, first=%t, text=%q",
					partialCount, deltaSize, actualOverlap, len(sendData), elapsed, isFirst, TruncateForLog(text, 200))

				if text == "" || IsWhisperHallucination(text) {
					if text != "" {
						hallucinationsFiltered++
						log.Printf("voice-ws: partial filtered hallucination: %q", text)
					}
					continue
				}

				prevAccum := accumulatedTranscript
				accumulatedTranscript = DeduplicateOverlap(accumulatedTranscript, text)

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
					if writeErr := writeJSON(StreamMessage{Type: MsgPartial, Text: accumulatedTranscript}); writeErr != nil {
						return
					}
				}
			}
		}
	}()

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
			var msg StreamMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if msg.Type == MsgDone {
				break
			}
			if msg.Type == MsgVadSpeechStart {
				vadSpeechStartCount++
				speechMu.Lock()
				vadSignaled = true
				speechActive = true
				speechMu.Unlock()
				bufMu.Lock()
				if !segmentOffsetSet {
					segmentOffsetSet = true
					lookbackBytes := AudioBitrateBps / 8 * VadLookbackMs / 1000
					pos := audioBuffer.Len()
					rewindPos := pos - lookbackBytes
					if rewindPos < segmentStartOffset {
						rewindPos = segmentStartOffset
					}
					if rewindPos < 0 {
						rewindPos = 0
					}
					actualLookback := pos - rewindPos
					totalLookbackBytes += actualLookback
					lastPartialOffset = rewindPos
					segmentStartOffset = rewindPos
					log.Printf("voice-ws: vad-speech-start (first in segment): bufLen=%d rewindTo=%d lookback=%d bytes",
						pos, rewindPos, actualLookback)
				} else {
					log.Printf("voice-ws: vad-speech-start (resuming in segment): bufLen=%d segStart=%d",
						audioBuffer.Len(), segmentStartOffset)
				}
				bufMu.Unlock()
				continue
			}
			if msg.Type == MsgVadSpeechEnd {
				vadSpeechEndCount++
				speechMu.Lock()
				speechActive = false
				speechMu.Unlock()
				continue
			}
			if msg.Type == MsgSegmentBoundary {
				select {
				case segmentBoundaryCh <- struct{}{}:
				default:
					log.Printf("voice-ws: segment-boundary dropped (channel full)")
				}
			}
		}
	}

	close(done)
	partialCancel()
	tickerDone.Wait()
	segmentFinalWg.Wait()

	log.Printf("voice-ws: recording done, audioBytes=%d, partials=%d, segments=%d, vadStarts=%d, vadEnds=%d, gatedPartials=%d, hallucinations=%d, lookbackTotal=%d bytes",
		audioBuffer.Len(), partialCount, segmentIndex, vadSpeechStartCount, vadSpeechEndCount, partialsGatedBySilence, hallucinationsFiltered, totalLookbackBytes)

	bufMu.Lock()
	var finalAudio []byte
	if segmentIndex > 0 && segmentStartOffset < audioBuffer.Len() {
		tailLen := audioBuffer.Len() - segmentStartOffset
		finalAudio = make([]byte, tailLen)
		copy(finalAudio, audioBuffer.Bytes()[segmentStartOffset:])
		if len(webmInitSegment) > 0 && segmentStartOffset > 0 {
			finalAudio = append(webmInitSegment, finalAudio...)
		}
		log.Printf("voice-ws: final strategy=tail, tailOffset=%d, tailBytes=%d", segmentStartOffset, tailLen)
	} else if segmentIndex > 0 {
		log.Printf("voice-ws: final strategy=empty-tail, all audio covered by %d segment(s)", segmentIndex)
		finalAudio = nil
	} else {
		finalAudio = make([]byte, audioBuffer.Len())
		copy(finalAudio, audioBuffer.Bytes())
		log.Printf("voice-ws: final strategy=full, audioBytes=%d", len(finalAudio))
	}
	bufMu.Unlock()

	if len(finalAudio) == 0 {
		log.Printf("voice-ws: session closed, duration=%v, audioBytes=0, partials=%d, strategy=empty", time.Since(sessionStart), partialCount)
		_ = writeJSON(StreamMessage{Type: MsgFinal, Text: ""})
		return
	}

	finalCtx, finalCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer finalCancel()

	var decision SpeakerDecision
	transcribeAudio := finalAudio
	if speakerCfg.ExtractionEnabled {
		transcribeAudio, decision = ExtractTargetSpeaker(finalCtx, speakerCfg, s.speakerClient, finalAudio)
	} else {
		decision = EvaluateSpeaker(finalCtx, speakerCfg, s.speakerClient, finalAudio)
	}
	if decision.Enabled {
		if decision.Applied {
			log.Printf(
				"voice-ws: final speaker decision matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s extracted=%v",
				decision.Matched, decision.Allowed, decision.Score, decision.Threshold,
				decision.ProfileID, decision.Mode, decision.Extracted,
			)
		} else if decision.ErrorMessage != "" {
			log.Printf("voice-ws: final %s", FormatSpeakerDecisionError(decision))
		}
		if !decision.Allowed {
			_ = writeJSON(StreamMessage{
				Type:      MsgSegmentRejected,
				Score:     decision.Score,
				Threshold: decision.Threshold,
				ProfileID: decision.ProfileID,
			})
			_ = writeJSON(StreamMessage{Type: MsgFinal, Text: ""})
			return
		}
	}

	t0 := time.Now()
	text, err := s.transcribe(finalCtx, transcribeAudio, language, true, "")
	if err != nil {
		log.Printf("voice-ws: final transcribe failed (%v): %v", time.Since(t0), err)
		_ = writeJSON(StreamMessage{Type: MsgError, Text: "Final transcription failed"})
		return
	}

	finalText := strings.TrimSpace(text)
	if IsWhisperHallucination(finalText) {
		log.Printf("voice-ws: final filtered hallucination: %q", finalText)
		finalText = ""
	}
	log.Printf("voice-ws: full retranscribe took %v, text=%q", time.Since(t0), finalText)
	log.Printf("voice-ws: session closed, duration=%v, audioBytes=%d, partials=%d, segments=%d", time.Since(sessionStart), len(finalAudio), partialCount, segmentIndex)
	_ = writeJSON(StreamMessage{Type: MsgFinal, Text: finalText})
}

// writeUnavailable mirrors the package-main writeCatalogError("voice_unavailable").
func (s *Service) writeUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":     "voice_unavailable",
		"category": "dependency",
		"message":  "Voice transcription is currently unavailable",
		"recovery": "Ensure Whisper is running (resource whisper on port 8090)",
		"retry":    true,
	})
}
