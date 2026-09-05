package capture

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// defaultInlineDomMaxBytes caps CaptureResponse.dom_html so a pathological
// page cannot balloon the RPC response. Truncation is silent (documented
// on the proto field); readable-text consumers tolerate a cut-off tail by
// design.
const defaultInlineDomMaxBytes = 2 << 20

// defaultInlineDomExpression is evaluated in-page to read the rendered DOM.
// An EVALUATE node is used (not EXTRACT) because the Playwright driver's
// extract handler only does textContent today, while evaluate returns the
// raw expression result through extracted_data.
const defaultInlineDomExpression = "document.documentElement.outerHTML"

// InlineDomConfig holds the tunables for inline-DOM capture. Zero values
// fall back to the package defaults via withDefaults so callers can wire
// only the fields they want to override.
type InlineDomConfig struct {
	// Expression is the in-page JS evaluated to read the rendered DOM.
	Expression string
	// MaxBytes caps the returned dom_html length (silent truncation).
	MaxBytes int
}

// withDefaults returns a copy with any unset field filled from the
// package defaults.
func (c InlineDomConfig) withDefaults() InlineDomConfig {
	if c.Expression == "" {
		c.Expression = defaultInlineDomExpression
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = defaultInlineDomMaxBytes
	}
	return c
}

// readInlineDom pulls the evaluate node's expression result out of the
// exported timeline.json. The driver's evaluate handler returns the raw
// result under the "result" key of extracted_data, which the execution
// writer persists verbatim as the frame's extracted_data_preview.
func (c InlineDomConfig) readInlineDom(outDir, nodeID string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(outDir, "timeline.json"))
	if err != nil {
		return "", fmt.Errorf("read timeline.json: %w", err)
	}
	var timeline struct {
		Frames []struct {
			NodeID               string         `json:"node_id"`
			ExtractedDataPreview map[string]any `json:"extracted_data_preview"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(raw, &timeline); err != nil {
		return "", fmt.Errorf("decode timeline.json: %w", err)
	}
	for _, frame := range timeline.Frames {
		if frame.NodeID != nodeID {
			continue
		}
		if value, ok := frame.ExtractedDataPreview["result"].(string); ok && value != "" {
			if len(value) > c.MaxBytes {
				value = value[:c.MaxBytes]
			}
			return value, nil
		}
	}
	return "", errors.New("timeline has no DOM evaluate result")
}
