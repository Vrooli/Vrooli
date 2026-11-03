package routing

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ai-gateway/internal/inference"
)

// The executor bounds bytes before Write. Feature output is withheld until the
// complete answer passes its published schema; usage still gets its receipt.
type featureOutput struct {
	bytes.Buffer
	header http.Header
}

func (o *featureOutput) Header() http.Header { return o.header }
func (o *featureOutput) WriteHeader(int)     {}
func (o *featureOutput) content(rawSchema json.RawMessage) (string, error) {
	var content strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(o.Bytes()))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "data:") {
			continue
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content   string          `json:"content"`
					ToolCalls json.RawMessage `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "data:"))), &event) != nil {
			return "", errors.New("invalid feature output event")
		}
		for _, choice := range event.Choices {
			if len(choice.Delta.ToolCalls) > 0 {
				return "", errors.New("feature returned tools")
			}
			content.WriteString(choice.Delta.Content)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	schema, err := (inference.SchemaGate{}).Parse(string(rawSchema))
	if err != nil {
		return "", err
	}
	answer := content.String()
	value := []byte(answer)
	// A string schema describes a plain text answer; structured schemas describe JSON.
	if schema["type"] == "string" {
		value, _ = json.Marshal(answer)
	}
	if err := inference.ValidateJSON(schema, value); err != nil {
		return "", err
	}
	return answer, nil
}

// streamChat is the single boundary between a declared feature and free chat.
func (s *PrivatePolicyService) streamChat(featureID, model string, input *PrivateFeatureInput, raw json.RawMessage) (json.RawMessage, error) {
	feature := s.features[featureID]
	if feature.PinnedInstructionDigest == "" {
		if input != nil {
			return nil, errors.New("feature is not declared as bounded")
		}
		return raw, nil
	}
	if input == nil || (len(raw) > 0 && string(raw) != "null") {
		return nil, errors.New("bounded feature envelope is required")
	}
	if err := s.validateFeatureInput(featureID, *input); err != nil {
		return nil, err
	}
	if _, err := (inference.SchemaGate{}).Parse(string(feature.Schema)); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"model": model, "messages": []PrivateTurn{{Role: "system", Content: input.Instruction}, input.Turn}})
}
