package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TTSVoiceLister is the testability seam for voice listing.
// DOC: docs/internal/SEAMS.md#tts-voice-lister-seam
type TTSVoiceLister interface {
	ListVoices(ctx context.Context) ([]TTSVoice, error)
}

// TTSVoice represents an available TTS voice.
type TTSVoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// KokoroVoiceLister fetches available voices from a Kokoro-FastAPI instance.
type KokoroVoiceLister struct {
	BaseURL string
	Client  *http.Client
}

func (k *KokoroVoiceLister) ListVoices(ctx context.Context) ([]TTSVoice, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", k.BaseURL+"/v1/audio/voices", nil)
	if err != nil {
		return nil, err
	}

	client := k.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Kokoro-FastAPI returns a JSON object with voice data.
	// Parse it and convert to our TTSVoice slice.
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// Kokoro-FastAPI's /v1/audio/voices response format varies by version:
	//   v0.3.x: array of objects [{id, name}, ...]
	//   v0.2.x: object with voice IDs as keys {voice_id: {...}, ...}
	//   v0.1.x: array of strings ["voice_id", ...]
	// We try all three to maintain backward compatibility.
	var voices []TTSVoice
	if err := json.Unmarshal(raw, &voices); err == nil && len(voices) > 0 {
		return voices, nil
	}

	// Try parsing as object with voice IDs as keys
	var voiceMap map[string]interface{}
	if err := json.Unmarshal(raw, &voiceMap); err == nil {
		voices = make([]TTSVoice, 0, len(voiceMap))
		for id := range voiceMap {
			voices = append(voices, TTSVoice{ID: id, Name: id})
		}
		return voices, nil
	}

	// Try parsing as array of strings
	var voiceStrings []string
	if err := json.Unmarshal(raw, &voiceStrings); err == nil {
		voices = make([]TTSVoice, len(voiceStrings))
		for i, v := range voiceStrings {
			voices[i] = TTSVoice{ID: v, Name: v}
		}
		return voices, nil
	}

	return nil, fmt.Errorf("unexpected voice list format")
}

// HTTP handler for /api/v1/tts/voices moved to handlers/tts. The voice-list
// fetch/validation now lives in tts_adapter.go's ListVoices.
