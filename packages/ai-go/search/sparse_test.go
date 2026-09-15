package aisearch

import "testing"

func TestBM25EncodeDeterministic(t *testing.T) {
	enc := NewBM25SparseEncoder()
	a := enc.Encode("restart a scenario lifecycle")
	b := enc.Encode("restart a scenario lifecycle")
	if len(a.Indices) != len(b.Indices) || len(a.Indices) == 0 {
		t.Fatalf("encoding not deterministic or empty: %d vs %d", len(a.Indices), len(b.Indices))
	}
}

func TestBM25EncodeTermFrequency(t *testing.T) {
	enc := NewBM25SparseEncoder()
	v := enc.Encode("restart restart restart scenario")
	if len(v.Indices) != 2 {
		t.Fatalf("expected 2 distinct terms (restart, scenario), got %d", len(v.Indices))
	}
	// The repeated term must carry a higher weight than the singleton.
	var maxW float32
	for _, w := range v.Values {
		if w > maxW {
			maxW = w
		}
	}
	if maxW < 3 {
		t.Fatalf("expected term frequency >= 3 for repeated term, got %v", maxW)
	}
}

func TestBM25EncodeEmpty(t *testing.T) {
	v := NewBM25SparseEncoder().Encode("  a , ! ")
	if len(v.Indices) != 0 || len(v.Values) != 0 {
		t.Fatalf("single-char/empty input should yield empty vector, got %d terms", len(v.Indices))
	}
}
