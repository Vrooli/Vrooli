package aisearch

import (
	"hash/fnv"
	"strings"
)

// bm25SparseEncoder is the local, model-free SparseEncoder (Phase 0 decision
// D2). It tokenizes text, hashes each term to a uint32 dimension, and emits
// term-frequency weights; Qdrant applies IDF server-side via the collection's
// "idf" modifier. No model, no GPU — computed inline during reconcile.
type bm25SparseEncoder struct{}

// NewBM25SparseEncoder returns the default local sparse encoder.
func NewBM25SparseEncoder() SparseEncoder { return bm25SparseEncoder{} }

// Encode tokenizes text and returns parallel index/value slices. Repeated terms
// accumulate weight (raw term frequency); the dimension is the FNV-1a hash of
// the lowercased term. Empty/whitespace input yields an empty vector.
func (bm25SparseEncoder) Encode(text string) SparseVector {
	counts := make(map[uint32]float32)
	for _, term := range sparseTokenize(text) {
		h := fnv.New32a()
		_, _ = h.Write([]byte(term))
		counts[h.Sum32()]++
	}
	if len(counts) == 0 {
		return SparseVector{}
	}
	out := SparseVector{
		Indices: make([]uint32, 0, len(counts)),
		Values:  make([]float32, 0, len(counts)),
	}
	for idx, w := range counts {
		out.Indices = append(out.Indices, idx)
		out.Values = append(out.Values, w)
	}
	return out
}

// sparseTokenize lowercases and splits on non-alphanumeric runes, dropping
// single-character tokens (mirrors the keyword tokenizer used by the text leg).
func sparseTokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 2 {
			continue
		}
		out = append(out, f)
	}
	return out
}
