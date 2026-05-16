package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// VoiceLister is the testability seam for voice listing.
type VoiceLister interface {
	ListVoices(ctx context.Context) ([]Voice, error)
}

// KokoroVoiceLister fetches available voices from a Kokoro-FastAPI instance.
type KokoroVoiceLister struct {
	BaseURL string
	Client  *http.Client
}

func (k *KokoroVoiceLister) ListVoices(ctx context.Context) ([]Voice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.BaseURL+"/v1/audio/voices", nil)
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

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
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
