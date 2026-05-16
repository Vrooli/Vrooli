package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"web-console/internal/audioports"
	inttts "web-console/internal/tts"
)

// invalidateTTSCacheForEvent removes every cached audio variant (voice, speed,
// version) for a given event. Used after summarization replaces an event's
// speech paragraphs, so the next playback regenerates audio from the new text.
func (s *Server) invalidateTTSCacheForEvent(eventID string) {
	if s.ttsCache == nil || eventID == "" {
		return
	}
	s.ttsCache.Evict(eventID)
}

// preSynthesizeTTS asynchronously synthesizes TTS audio for an assistant event
// and stores it in the cache for instant playback on tab switch.
func (s *Server) preSynthesizeTTS(event ConversationEvent, sessionID string) {
	if s.ttsPort == nil || s.ttsCache == nil {
		return
	}
	if event.Role != ConversationRoleAssistant {
		return
	}
	if len(event.SpeechParagraphs) == 0 {
		return
	}

	cfg := s.getTTSConfig()
	voice := cfg.KokoroVoice
	if voice == "" {
		voice = "af_heart"
	}
	speed := cfg.KokoroSpeed
	if speed <= 0 {
		speed = 1.0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	audio, contentType, err := s.synthesizeParagraphs(ctx, event.SpeechParagraphs, voice, speed)
	if err != nil {
		log.Printf("tts-precache: synthesis failed for event %s: %v", event.ID, err)
		return
	}

	key := inttts.CacheKey{
		EventID: event.ID,
		Voice:   voice,
		Speed:   speed,
		Version: "active",
	}
	s.ttsCache.Put(key, audio, contentType)
	log.Printf("tts-precache: cached %d bytes for event %s (voice=%s speed=%.1f)",
		len(audio), event.ID, voice, speed)
}

// synthesizeParagraphs synthesizes multiple paragraphs and concatenates the
// resulting audio into a single MP3 blob. This reuses the same Kokoro backend
// as the on-demand handleTTSSynthesize handler.
func (s *Server) synthesizeParagraphs(ctx context.Context, paragraphs []string, voice string, speed float64) ([]byte, string, error) {
	var combined []byte
	var contentType string

	for _, p := range paragraphs {
		if len(p) == 0 {
			continue
		}
		const maxSynthesizeInputLength = 5000 // matches internal/tts.maxSynthesizeInputLength
		if len(p) > maxSynthesizeInputLength {
			p = p[:maxSynthesizeInputLength]
		}

		res, err := s.ttsPort.Synthesize(ctx, audioports.TTSRequest{
			Input:          p,
			Voice:          voice,
			ResponseFormat: "mp3",
			Speed:          speed,
		})
		if err != nil {
			return nil, "", fmt.Errorf("synthesize paragraph: %w", err)
		}

		if len(res.Audio) > 0 {
			combined = append(combined, res.Audio...)
			if contentType == "" {
				contentType = res.ContentType
			}
		}
	}

	if len(combined) == 0 {
		return nil, "", fmt.Errorf("all paragraphs produced empty audio")
	}

	return combined, contentType, nil
}
