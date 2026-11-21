//go:build pii_onnx

package main

import (
	"context"
	"os"
	"testing"
)

// TestSplitBIO locks the BIO tag parsing that decodeBIO depends on.
func TestSplitBIO(t *testing.T) {
	cases := []struct {
		tag, prefix, label string
	}{
		{"O", "", "O"},
		{"", "", "O"},
		{"B-EMAIL_ADDRESS", "B", "EMAIL_ADDRESS"},
		{"I-US_SSN", "I", "US_SSN"},
		{"B-PERSON", "B", "PERSON"},
		{"PERSON", "", "PERSON"}, // no BIO prefix -> treated as a bare label
	}
	for _, c := range cases {
		p, l := splitBIO(c.tag)
		if p != c.prefix || l != c.label {
			t.Errorf("splitBIO(%q) = (%q,%q), want (%q,%q)", c.tag, p, l, c.prefix, c.label)
		}
	}
}

// TestDecodeBIO verifies that a BIO label sequence over offset-tracked pieces
// produces byte-correct spans, including multi-piece spans and UTF-8 offsets.
func TestDecodeBIO(t *testing.T) {
	// "café john@x.io" — a multibyte prefix so byte offsets != rune offsets.
	// bytes: c(0)a(1)f(2)é(3,4)space(5) j(6)o(7)h(8)n(9)@(10)x(11).(12)i(13)o(14)
	text := "café john@x.io"
	// Pieces (contrived but byte-accurate): "café"[0:5], "john"[6:10], "@"[10:11],
	// "x"[11:12], "."[12:13], "io"[13:15].
	pieces := []piece{
		{start: 0, end: 5},
		{start: 6, end: 10},
		{start: 10, end: 11},
		{start: 11, end: 12},
		{start: 12, end: 13},
		{start: 13, end: 15},
	}
	labels := []string{"O", "B-EMAIL_ADDRESS", "I-EMAIL_ADDRESS", "I-EMAIL_ADDRESS", "I-EMAIL_ADDRESS", "I-EMAIL_ADDRESS"}
	scores := []float64{0.9, 0.99, 0.98, 0.98, 0.97, 0.99}

	spans := decodeBIO(text, pieces, labels, scores)
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d: %+v", len(spans), spans)
	}
	s := spans[0]
	if s.Category != "EMAIL_ADDRESS" {
		t.Errorf("category = %q", s.Category)
	}
	if s.Start != 6 || s.End != 15 {
		t.Errorf("offsets = [%d:%d], want [6:15]", s.Start, s.End)
	}
	if s.Text != "john@x.io" {
		t.Errorf("text = %q, want john@x.io", s.Text)
	}
	if text[s.Start:s.End] != s.Text {
		t.Errorf("span text does not index into source: %q vs %q", text[s.Start:s.End], s.Text)
	}
	if s.Source != "model" {
		t.Errorf("source = %q, want model", s.Source)
	}
}

// TestWordPieceTokenizerOffsets checks that the tokenizer records byte offsets
// that index back into the original (non-lowercased) text.
func TestWordPieceTokenizerOffsets(t *testing.T) {
	vocab := map[string]int64{
		"[UNK]": 0, "[CLS]": 1, "[SEP]": 2,
		"john": 10, "##@": 11, "x": 12, "##.": 13, "##io": 14,
		"hello": 20,
	}
	tok := newWordPieceTokenizer(vocab, true)
	text := "Hello JOHN"
	pieces := tok.tokenize(text)
	// Expect two words: "Hello"->hello, "JOHN"->john (lowercased lookup, original
	// offsets). Both are single vocab hits.
	if len(pieces) != 2 {
		t.Fatalf("want 2 pieces, got %d", len(pieces))
	}
	for _, p := range pieces {
		if p.start < 0 || p.end > len(text) || p.start >= p.end {
			t.Errorf("bad offsets %+v for text %q", p, text)
		}
	}
	// "Hello" is [0:5], "JOHN" is [6:10].
	if pieces[0].start != 0 || pieces[0].end != 5 {
		t.Errorf("piece0 offsets = [%d:%d], want [0:5]", pieces[0].start, pieces[0].end)
	}
	if pieces[1].start != 6 || pieces[1].end != 10 {
		t.Errorf("piece1 offsets = [%d:%d], want [6:10]", pieces[1].start, pieces[1].end)
	}
	if text[pieces[1].start:pieces[1].end] != "JOHN" {
		t.Errorf("piece1 maps to %q, want JOHN", text[pieces[1].start:pieces[1].end])
	}
}

// TestModelLoadDisabledFailsClosed proves the capacity ladder's off rung makes
// the real engine fail closed (regex-only), not load a session.
func TestModelLoadDisabledFailsClosed(t *testing.T) {
	t.Setenv("PII_DETECTOR_LOAD_MODEL", "off")
	m, err := newNERModelFromEnv()
	if err != nil {
		t.Fatalf("newNERModelFromEnv: %v", err)
	}
	if _, ok := m.(UnavailableModel); !ok {
		t.Fatalf("want UnavailableModel when PII_DETECTOR_LOAD_MODEL=off, got %T", m)
	}
}

// TestRealModelDetectsPII is the end-to-end proof: it loads the staged ONNX
// model and asserts the NER pass detects known PII categories at correct byte
// offsets. It skips (with a reason) when the model/runtime are not staged, so
// the suite stays green on hosts without the artifacts while still exercising
// the real path wherever they exist.
//
// Stage locally with:
//
//	PII_DETECTOR_MODEL_DIR=<dir with model.quant.onnx,vocab.txt,config.json>
//	PII_DETECTOR_ONNXRUNTIME_LIB=<path to libonnxruntime.so (ORT 1.23.x)>
func TestRealModelDetectsPII(t *testing.T) {
	dir := os.Getenv("PII_DETECTOR_MODEL_DIR")
	lib := os.Getenv("PII_DETECTOR_ONNXRUNTIME_LIB")
	if dir == "" || lib == "" {
		t.Skip("skip live NER test: set PII_DETECTOR_MODEL_DIR and PII_DETECTOR_ONNXRUNTIME_LIB to the staged model + ONNX Runtime shared library")
	}
	if !fileReadable(lib) {
		t.Skipf("skip live NER test: ONNX Runtime lib not readable at %s", lib)
	}

	model, err := newNERModelFromEnv()
	if err != nil {
		t.Fatalf("newNERModelFromEnv: %v", err)
	}
	if u, ok := model.(UnavailableModel); ok {
		t.Fatalf("model unavailable despite staged artifacts: %s", u.reason)
	}
	defer model.Close()

	text := "My name is John Smith and my email is john.smith@acme.com, SSN 123-45-6789."
	spans, err := model.Detect(context.Background(), text)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(spans) == 0 {
		t.Fatal("real model returned no spans for text containing a name, email, and SSN")
	}

	byCat := map[string]Span{}
	for _, s := range spans {
		byCat[s.Category] = s
		if s.Source != "model" {
			t.Errorf("span %+v source = %q, want model", s, s.Source)
		}
		if s.Start < 0 || s.End > len(text) || s.Start >= s.End {
			t.Errorf("span %+v has bad byte offsets", s)
		}
		if text[s.Start:s.End] != s.Text {
			t.Errorf("span text %q does not index source %q", s.Text, text[s.Start:s.End])
		}
	}
	// The model must at least find the person name — the one category regex
	// cannot cover (the whole point of the NER tier).
	if _, ok := byCat["PERSON"]; !ok {
		t.Errorf("model did not detect PERSON in %q; categories found: %v", text, keysOf(byCat))
	}
	t.Logf("detected %d model spans: %+v", len(spans), spans)
}

func keysOf(m map[string]Span) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
