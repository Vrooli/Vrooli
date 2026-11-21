// Package presentationseed loads the repository's declarative presentation
// seed through the same strict codec used by the configuration owner.
package presentationseed

import (
	_ "embed"

	"landing-page-business-suite-api/internal/presentation"
)

//go:embed recommended-signal-studio.json
var recommendedJSON []byte

// Recommended returns a fresh validated document. All product copy remains in
// the one declarative JSON source, not Go strings or a parallel UI fixture.
func Recommended() (presentation.Document, error) { return DecodeAndValidate(recommendedJSON) }

// Decode performs strict version-1 document decoding. Unknown document or
// block-content fields are rejected by the presentation codec.
func Decode(data []byte) (presentation.Document, error) {
	return presentation.DecodeDocument(data)
}

// DecodeAndValidate decodes and validates a seed without adding storage,
// transport, or bootstrap behavior to the presentation domain.
func DecodeAndValidate(data []byte) (presentation.Document, error) {
	document, err := Decode(data)
	if err != nil {
		return presentation.Document{}, err
	}
	if err := document.Validate(); err != nil {
		return presentation.Document{}, err
	}
	return document, nil
}
