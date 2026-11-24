package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"audio-tools/internal/httpc"
)

// seam: VoiceLister is the voice-catalog seam (SEAMS.md row
// "tts.VoiceLister"). Production wires the Kokoro HTTP voice lister;
// tests wire fakes.
//
// VoiceLister is the testability seam for voice listing.
type VoiceLister interface {
	ListVoices(ctx context.Context) ([]Voice, error)
}

// KokoroVoiceLister fetches available voices from a Kokoro-FastAPI instance.
type KokoroVoiceLister struct {
	BaseURL string
	Doer    httpc.Doer
}

const maxVoiceCatalogResponseBytes = 1 << 20

func (k *KokoroVoiceLister) ListVoices(ctx context.Context) ([]Voice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.BaseURL+"/v1/audio/voices", nil)
	if err != nil {
		return nil, err
	}

	resp, err := k.Doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("TTS voice endpoint returned status %d", resp.StatusCode)
	}

	raw, err := readBoundedResponse(resp.Body, maxVoiceCatalogResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("read TTS voice list: %w", err)
	}
	// Kale's compatible endpoint wraps its voice array with model identity:
	// {"model":"kale-tts-1-nano", "voices":[...]}. Unwrap that shape before
	// trying the legacy Kokoro array/map formats, or it would appear as two
	// bogus voices named "model" and "voices".
	var envelope struct {
		Voices []Voice `json:"voices"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Voices) > 0 {
		return envelope.Voices, nil
	}

	var voices []Voice
	if err := json.Unmarshal(raw, &voices); err == nil && len(voices) > 0 {
		return voices, nil
	}

	var voiceMap map[string]interface{}
	if err := json.Unmarshal(raw, &voiceMap); err == nil {
		voices = make([]Voice, 0, len(voiceMap))
		for id := range voiceMap {
			voices = append(voices, Voice{ID: id, Name: id})
		}
		return voices, nil
	}

	var voiceStrings []string
	if err := json.Unmarshal(raw, &voiceStrings); err == nil {
		voices = make([]Voice, len(voiceStrings))
		for i, v := range voiceStrings {
			voices[i] = Voice{ID: v, Name: v}
		}
		return voices, nil
	}

	return nil, fmt.Errorf("unexpected voice list format")
}
