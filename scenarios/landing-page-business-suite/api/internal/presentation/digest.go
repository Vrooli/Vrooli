package presentation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// PreviewIdentity is the immutable content identity for an authenticated
// document preview. It is deliberately derived from the complete document,
// not from storage state or an editor generation.
func PreviewIdentity(document Document) (string, error) {
	data, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ContentDigest hashes normalized content only. Route identity, locale
// selection, diagnostics, and navigation context are intentionally excluded,
// allowing root single-app and /apps/:slug to prove composition parity.
func ContentDigest(page ResolvedPage) (string, error) {
	canonical := struct {
		Title       string          `json:"title"`
		Description string          `json:"description"`
		Theme       Theme           `json:"theme"`
		Blocks      []ResolvedBlock `json:"blocks"`
		Footer      Footer          `json:"footer"`
		Display     PageDisplay     `json:"display"`
	}{
		Title: page.Title, Description: page.Description, Theme: page.Theme,
		Blocks: page.Blocks, Footer: page.Footer,
		Display: page.Display,
	}
	b, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
