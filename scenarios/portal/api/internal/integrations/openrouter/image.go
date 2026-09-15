package openrouter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/png"
)

// MarshalJSON preserves text-only content and encodes private image bytes as
// ordered multimodal parts. No remote image URL or local path is accepted.
func (m Message) MarshalJSON() ([]byte, error) {
	if len(m.Images) == 0 {
		return json.Marshal(struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{m.Role, m.Content})
	}
	if m.Role != "user" || len(m.Images) > 4 {
		return nil, errors.New("invalid image message")
	}
	type imageURL struct {
		URL string `json:"url"`
	}
	type part struct {
		Type  string    `json:"type"`
		Text  string    `json:"text,omitempty"`
		Image *imageURL `json:"image_url,omitempty"`
	}
	parts := []part{{Type: "text", Text: m.Content}}
	total := 0
	for _, pixels := range m.Images {
		total += len(pixels)
		if len(pixels) == 0 || total > 32*1024*1024 {
			return nil, errors.New("image message exceeds bound")
		}
		config, err := png.DecodeConfig(bytes.NewReader(pixels))
		if err != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 16*1024*1024 {
			return nil, errors.New("invalid PNG image")
		}
		if _, err = png.Decode(bytes.NewReader(pixels)); err != nil {
			return nil, errors.New("invalid PNG image")
		}
		parts = append(parts, part{Type: "image_url", Image: &imageURL{URL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(pixels)}})
	}
	return json.Marshal(struct {
		Role    string `json:"role"`
		Content []part `json:"content"`
	}{m.Role, parts})
}
