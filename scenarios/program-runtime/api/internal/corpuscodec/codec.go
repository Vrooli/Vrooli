// Package corpuscodec encodes protected program text without confusing literal
// ciphertext markers with encrypted resume receipts.
package corpuscodec

import (
	"encoding/json"
	"errors"
	"strings"

	"program-runtime/internal/receipts"
)

type segment struct {
	Text      string `json:"text,omitempty"`
	Protected string `json:"protected,omitempty"`
}

// Encode keeps literal text distinct from encrypted receipt segments.
func Encode(s receipts.Sealer, value string) (string, error) {
	parts := make([]segment, 0)
	last := 0
	for _, loc := range receipts.PlaintextPattern.FindAllStringIndex(value, -1) {
		if loc[0] > last {
			parts = append(parts, segment{Text: value[last:loc[0]]})
		}
		sealed, err := s.Seal(value[loc[0]:loc[1]])
		if err != nil {
			return "", err
		}
		parts = append(parts, segment{Protected: sealed})
		last = loc[1]
	}
	if last < len(value) {
		parts = append(parts, segment{Text: value[last:]})
	}
	data, err := json.Marshal(parts)
	return string(data), err
}

// Decode opens only explicitly tagged segments. A nil sealer returns the
// protected representation and reports that protected authority is present.
func Decode(s *receipts.Sealer, value string, format int) (string, bool, error) {
	switch format {
	case 0:
		// Historical text has no way to distinguish literal sealed markers from
		// ciphertext. Preserve the old owner behavior; projections treat markers
		// conservatively as protected.
		if s != nil {
			return s.Open(value), false, nil
		}
		return value, strings.Contains(value, "prt_sealed_v1_") || receipts.PlaintextPattern.MatchString(value), nil
	case 1:
		var parts []segment
		if err := json.Unmarshal([]byte(value), &parts); err != nil || parts == nil {
			return "", false, errors.New("invalid corpus receipt encoding")
		}
		var out strings.Builder
		protected := false
		for _, part := range parts {
			if part.Protected != "" {
				if part.Text != "" || !strings.HasPrefix(part.Protected, "prt_sealed_v1_") {
					return "", false, errors.New("invalid corpus receipt segment")
				}
				protected = true
				if s == nil {
					out.WriteString(part.Protected)
				} else {
					out.WriteString(s.Open(part.Protected))
				}
			} else {
				out.WriteString(part.Text)
			}
		}
		return out.String(), protected, nil
	default:
		return "", false, errors.New("unknown corpus receipt format")
	}
}
