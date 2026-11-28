//go:build pii_onnx

package main

// engine_onnx.go is the REAL NER inference path, selected by the `pii_onnx`
// build tag. It implements the NERModel seam (engine.go) against a BERT-family
// token-classification model exported to ONNX, run through the CGO-free
// onnxruntime_go binding (approved dependency go/github.com/yalue/onnxruntime_go).
//
// Binding model (mirrors resources/sherpa-onnx's "bundle the native lib beside
// the server" discipline, but without build-time CGO linking): the binding loads
// libonnxruntime.so at runtime via dlopen. The path is resolved from
// PII_DETECTOR_ONNXRUNTIME_LIB, else conventional locations next to the binary
// (../lib/libonnxruntime.so) and the model cache. No `-lonnxruntime` link flag,
// so the default (stub) build never needs the native lib to compile.
//
// Model layout (staged into PII_DETECTOR_MODEL_CACHE / RESOURCE_DATA_DIR by the
// resource.json acquisition block, checksum-pinned in model-policy.json):
//
//	<cache>/<model-dir>/model.quant.onnx   INT8 token-classification graph
//	<cache>/<model-dir>/vocab.txt          WordPiece vocabulary (one token/line)
//	<cache>/<model-dir>/config.json        has id2label (the BIO label vocabulary)
//
// The small tier (gravitee-io/bert-small-pii-detection) is a
// BertForTokenClassification with 51 BIO labels, do_lower_case=true, inputs
// input_ids+attention_mask (int64), output logits [1,seq,num_labels].
//
// The capacity ladder's PII_DETECTOR_LOAD_MODEL=off rung is honored here: when
// set off, newNERModelFromEnv returns the fail-closed UnavailableModel so the
// real session (and its RAM) is never created, exactly as engine.go documents.
//
// SCOPE: this file serves EVERY token-classification tier uniformly — it is
// model-agnostic. The graph I/O names, the id2label label vocabulary, the model
// input arity, and the tokenizer casing (do_lower_case) are all read from the
// staged artifacts at load time, so the same code serves:
//   - the small tier (gravitee-io/bert-small-pii, uncased BertForTokenClassification)
//   - the large tier (ab-ai/pii_model_based_on_distilbert, uncased
//     DistilBertForTokenClassification, 65 labels — model-policy.json pii.large)
// Adding the large tier was a pure model swap: no new code path, just a staged
// model + the do_lower_case generalization below (so a cased classifier is
// tokenized correctly too). PII_DETECTOR_MODEL / PII_DETECTOR_MODEL_DIR /
// PII_DETECTOR_ONNX_FILE select which model this engine loads.
//
// NOT served here: the GLiNER zero-shot tiers from the original plan
// (knowledgator/gliner-pii-edge, urchade/gliner_multi_pii) were deliberately
// NOT shipped — GLiNER is a different protocol (SentencePiece tokenizer,
// request-time label prompts, span-score decode rather than per-token BIO) that
// would not reuse this machinery cleanly. See docs/ZERO_SHOT_DECISION.md for the
// clean-over-complete rationale and the recommended sidecar path if zero-shot
// PII is needed.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	ort "github.com/yalue/onnxruntime_go"
)

// maxSeqLen caps the token sequence fed to the model. BERT's positional
// embeddings cap at 512; we reserve two slots for [CLS]/[SEP]. Text longer than
// this is truncated for the model pass only — the regex pass still covers the
// whole input, and truncation is reported via model_available semantics upstream.
const maxSeqLen = 512

// onnxModel is the real NERModel: a loaded ONNX session plus the tokenizer and
// label vocabulary needed to turn logits back into byte-offset spans.
type onnxModel struct {
	mu         sync.Mutex // onnxruntime sessions are not guaranteed goroutine-safe
	session    *ort.DynamicAdvancedSession
	tok        *wordPieceTokenizer
	id2label   []string
	modelID    string
	inputArity int // number of model input tensors (2: ids+mask, 3: +token_type)
}

// ortInitOnce guards process-wide ONNX Runtime environment initialization so
// multiple model loads (e.g. a reload) do not double-initialize.
var (
	ortInitOnce sync.Once
	ortInitErr  error
)

func newNERModelFromEnv() (NERModel, error) {
	if modelLoadDisabled() {
		return UnavailableModel{reason: "PII_DETECTOR_LOAD_MODEL=off: capacity ladder pinned the server to the regex-only rung"}, nil
	}

	modelID := envOr("PII_DETECTOR_MODEL", "gravitee-io/bert-small-pii-detection")
	modelDir, err := resolveModelDir(modelID)
	if err != nil {
		return UnavailableModel{reason: err.Error()}, nil
	}

	onnxPath := filepath.Join(modelDir, envOr("PII_DETECTOR_ONNX_FILE", "model.quant.onnx"))
	vocabPath := filepath.Join(modelDir, "vocab.txt")
	configPath := filepath.Join(modelDir, "config.json")
	for _, p := range []string{onnxPath, vocabPath, configPath} {
		if !fileReadable(p) {
			return UnavailableModel{reason: fmt.Sprintf("model artifact missing: %s (stage the ONNX model + tokenizer into %s; see model-policy.json acquisition)", p, modelDir)}, nil
		}
	}

	vocab, err := loadVocab(vocabPath)
	if err != nil {
		return UnavailableModel{reason: fmt.Sprintf("load vocab: %v", err)}, nil
	}
	modelCfg, err := loadModelConfig(configPath)
	if err != nil {
		return UnavailableModel{reason: fmt.Sprintf("load labels: %v", err)}, nil
	}
	id2label := modelCfg.id2label

	if err := ensureORTEnvironment(); err != nil {
		return UnavailableModel{reason: fmt.Sprintf("initialize ONNX Runtime: %v", err)}, nil
	}

	inputs, outputs, err := graphIONames(onnxPath)
	if err != nil {
		return UnavailableModel{reason: fmt.Sprintf("inspect ONNX graph: %v", err)}, nil
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		return UnavailableModel{reason: fmt.Sprintf("session options: %v", err)}, nil
	}
	defer opts.Destroy()

	session, err := ort.NewDynamicAdvancedSession(onnxPath, inputs, outputs, opts)
	if err != nil {
		return UnavailableModel{reason: fmt.Sprintf("load ONNX session %q: %v", onnxPath, err)}, nil
	}

	return &onnxModel{
		session:    session,
		tok:        newWordPieceTokenizer(vocab, modelCfg.lowercase),
		id2label:   id2label,
		modelID:    modelID,
		inputArity: len(inputs),
	}, nil
}

// ensureORTEnvironment resolves the shared library path and initializes the ORT
// environment exactly once per process.
func ensureORTEnvironment() error {
	ortInitOnce.Do(func() {
		if lib := resolveSharedLibrary(); lib != "" {
			ort.SetSharedLibraryPath(lib)
		}
		ortInitErr = ort.InitializeEnvironment()
	})
	return ortInitErr
}

// resolveSharedLibrary finds libonnxruntime.so without hard-coding a single
// location: an explicit env override wins, then the bundled lib beside the
// binary (the sherpa-onnx bundle layout: <bundle>/server/<bin>, <bundle>/lib/…),
// then the model cache. Returns "" to let the binding use its default search.
func resolveSharedLibrary() string {
	if v := strings.TrimSpace(os.Getenv("PII_DETECTOR_ONNXRUNTIME_LIB")); v != "" {
		return v
	}
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "..", "lib", sharedLibName()),
			filepath.Join(dir, sharedLibName()),
		)
	}
	if cache := os.Getenv("PII_DETECTOR_MODEL_CACHE"); cache != "" {
		candidates = append(candidates, filepath.Join(cache, "lib", sharedLibName()))
	}
	for _, c := range candidates {
		if fileReadable(c) {
			return c
		}
	}
	return ""
}

func sharedLibName() string {
	// The binding understands platform-specific names; we only ship linux today.
	return "libonnxruntime.so"
}

// resolveModelDir locates the directory holding the model's ONNX + tokenizer.
// Preference: explicit PII_DETECTOR_MODEL_DIR, then <cache>/<sanitized modelID>,
// then <cache> itself (flat layout), then <RESOURCE_DATA_DIR>/model-cache/<id>.
func resolveModelDir(modelID string) (string, error) {
	if v := strings.TrimSpace(os.Getenv("PII_DETECTOR_MODEL_DIR")); v != "" {
		return v, nil
	}
	sanitized := strings.ReplaceAll(modelID, "/", "__")
	var roots []string
	if cache := os.Getenv("PII_DETECTOR_MODEL_CACHE"); cache != "" {
		roots = append(roots, cache)
	}
	if data := os.Getenv("RESOURCE_DATA_DIR"); data != "" {
		roots = append(roots, filepath.Join(data, "model-cache"))
	}
	for _, root := range roots {
		for _, dir := range []string{filepath.Join(root, sanitized), filepath.Join(root, modelID), root} {
			if fileReadable(filepath.Join(dir, "config.json")) {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("no model directory found (set PII_DETECTOR_MODEL_DIR or stage the model under PII_DETECTOR_MODEL_CACHE)")
}

// graphIONames returns the input and output tensor names for the model. The
// small tier uses input_ids+attention_mask -> logits; we read the actual graph
// so a different export (e.g. one that also takes token_type_ids) still works.
func graphIONames(onnxPath string) (inputs, outputs []string, err error) {
	ins, outs, err := ort.GetInputOutputInfo(onnxPath)
	if err != nil {
		return nil, nil, err
	}
	for _, in := range ins {
		inputs = append(inputs, in.Name)
	}
	for _, out := range outs {
		outputs = append(outputs, out.Name)
	}
	if len(inputs) == 0 || len(outputs) == 0 {
		return nil, nil, fmt.Errorf("model has no inputs/outputs")
	}
	return inputs, outputs, nil
}

func (m *onnxModel) ModelID() string { return m.modelID }

func (m *onnxModel) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		m.session.Destroy()
		m.session = nil
	}
	// The ORT environment is process-wide and shared; leave it for process exit.
}

// Detect tokenizes text, runs the ONNX token classifier, and decodes the BIO
// label sequence into byte-offset spans. It is safe to call concurrently (the
// session is serialized by m.mu).
func (m *onnxModel) Detect(ctx context.Context, text string) ([]Span, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if m.session == nil {
		return nil, errModelUnavailable{reason: "session closed"}
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	pieces := m.tok.tokenize(text)
	if len(pieces) == 0 {
		return nil, nil
	}
	if len(pieces) > maxSeqLen-2 {
		pieces = pieces[:maxSeqLen-2]
	}

	// Build the input id / attention sequences with [CLS] … [SEP].
	ids := make([]int64, 0, len(pieces)+2)
	ids = append(ids, m.tok.clsID)
	for _, p := range pieces {
		ids = append(ids, p.id)
	}
	ids = append(ids, m.tok.sepID)
	seqLen := len(ids)

	mask := make([]int64, seqLen)
	for i := range mask {
		mask[i] = 1
	}
	tokenType := make([]int64, seqLen) // all zeros (single segment)

	shape := ort.NewShape(1, int64(seqLen))
	inIDs, err := ort.NewTensor(shape, ids)
	if err != nil {
		return nil, fmt.Errorf("input_ids tensor: %w", err)
	}
	defer inIDs.Destroy()
	inMask, err := ort.NewTensor(shape, mask)
	if err != nil {
		return nil, fmt.Errorf("attention_mask tensor: %w", err)
	}
	defer inMask.Destroy()
	inType, err := ort.NewTensor(shape, tokenType)
	if err != nil {
		return nil, fmt.Errorf("token_type_ids tensor: %w", err)
	}
	defer inType.Destroy()

	inputVals := m.inputsFor(inIDs, inMask, inType)

	numLabels := int64(len(m.id2label))
	outShape := ort.NewShape(1, int64(seqLen), numLabels)
	out, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("logits tensor: %w", err)
	}
	defer out.Destroy()

	m.mu.Lock()
	err = m.session.Run(inputVals, []ort.Value{out})
	m.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("onnx run: %w", err)
	}

	logits := out.GetData()
	labelIdx := m.argmaxLabels(logits, seqLen, int(numLabels))
	scores := m.maxScores(logits, seqLen, int(numLabels))
	// index 0 is [CLS], index seqLen-1 is [SEP]; the piece labels are the
	// interior, aligned 1:1 with `pieces`.
	pieceLabels := make([]string, len(pieces))
	pieceScores := make([]float64, len(pieces))
	for i := range pieces {
		pieceLabels[i] = m.labelName(labelIdx[i+1])
		pieceScores[i] = scores[i+1]
	}

	return decodeBIO(text, pieces, pieceLabels, pieceScores), nil
}

// inputsFor returns the input Value slice matching the graph's declared inputs.
// The small tier has only input_ids+attention_mask; a two-input graph ignores
// the token_type tensor. We feed in the order the session was created, which is
// the graph's own input order from graphIONames.
func (m *onnxModel) inputsFor(inIDs, inMask, inType ort.Value) []ort.Value {
	// The session was created with the graph's input names in their declared
	// order (graphIONames). For a BERT token classifier that is input_ids,
	// attention_mask[, token_type_ids]. m.inputArity, captured at load, selects
	// the 2- or 3-input case; order is stable across BERT exports.
	switch m.inputArity {
	case 3:
		return []ort.Value{inIDs, inMask, inType}
	default:
		return []ort.Value{inIDs, inMask}
	}
}

// argmaxLabels reduces [seq, numLabels] logits to a per-token label index.
func (m *onnxModel) argmaxLabels(logits []float32, seqLen, numLabels int) []int {
	labels := make([]int, seqLen)
	for t := 0; t < seqLen; t++ {
		best := float32(math.Inf(-1))
		bestIdx := 0
		base := t * numLabels
		for l := 0; l < numLabels; l++ {
			if v := logits[base+l]; v > best {
				best, bestIdx = v, l
			}
		}
		labels[t] = bestIdx
	}
	return labels
}

// maxScores returns the softmax probability of the argmax label per token, used
// as the span confidence.
func (m *onnxModel) maxScores(logits []float32, seqLen, numLabels int) []float64 {
	scores := make([]float64, seqLen)
	for t := 0; t < seqLen; t++ {
		base := t * numLabels
		maxLogit := float32(math.Inf(-1))
		for l := 0; l < numLabels; l++ {
			if v := logits[base+l]; v > maxLogit {
				maxLogit = v
			}
		}
		var sum float64
		for l := 0; l < numLabels; l++ {
			sum += math.Exp(float64(logits[base+l] - maxLogit))
		}
		scores[t] = 1.0 / sum // exp(0)/sum; the argmax term is exp(maxLogit-maxLogit)=1
	}
	return scores
}

// decodeBIO turns a per-piece BIO label sequence into byte-offset Spans over the
// original text. A span opens on B-<CAT> (or an I-<CAT> with no open span, which
// some models emit) and extends across subsequent I-<CAT> pieces. The span's
// byte range is [first piece start, last piece end]; the confidence is the mean
// of the member piece scores. Pieces carry the byte offsets the tokenizer
// recorded, so offsets index the original UTF-8 text directly.
func decodeBIO(text string, pieces []piece, labels []string, scores []float64) []Span {
	var spans []Span
	var (
		open     bool
		cat      string
		start    int
		end      int
		scoreSum float64
		scoreN   int
	)
	flush := func() {
		if open && cat != "" && end > start {
			conf := 0.5
			if scoreN > 0 {
				conf = scoreSum / float64(scoreN)
			}
			spans = append(spans, Span{
				Category:   cat,
				Start:      start,
				End:        end,
				Text:       text[start:end],
				Confidence: conf,
				Source:     "model",
			})
		}
		open = false
		cat = ""
		scoreSum, scoreN = 0, 0
	}
	for i, p := range pieces {
		if i >= len(labels) {
			break
		}
		prefix, label := splitBIO(labels[i])
		score := scoreAt(scores, i)
		switch {
		case label == "" || label == "O":
			flush()
		case prefix == "B" || (prefix == "I" && (!open || label != cat)):
			flush()
			open = true
			cat = label
			start = p.start
			end = p.end
			scoreSum = score
			scoreN = 1
		case prefix == "I" && open && label == cat:
			end = p.end
			scoreSum += score
			scoreN++
		default:
			flush()
		}
	}
	flush()
	return spans
}

func (m *onnxModel) labelName(i int) string {
	if i >= 0 && i < len(m.id2label) {
		return m.id2label[i]
	}
	return "O"
}

// scoreAt safely reads a per-piece confidence, defaulting when out of range.
func scoreAt(scores []float64, i int) float64 {
	if i < 0 || i >= len(scores) {
		return 0.5
	}
	return scores[i]
}

// splitBIO splits a tag like "B-EMAIL_ADDRESS" into ("B","EMAIL_ADDRESS").
// "O" returns ("", "O"); a label with no prefix returns ("", label).
func splitBIO(tag string) (prefix, label string) {
	if tag == "O" || tag == "" {
		return "", "O"
	}
	if len(tag) > 2 && (tag[1] == '-') && (tag[0] == 'B' || tag[0] == 'I') {
		return string(tag[0]), tag[2:]
	}
	return "", tag
}

// --- tokenizer -------------------------------------------------------------

// piece is one WordPiece token with the byte span of the original text it
// covers. Continuation pieces ("##..") still carry the byte span of the chars
// they consumed, so a span over several pieces maps to a contiguous byte range.
type piece struct {
	id    int64
	start int // byte offset into original text (inclusive)
	end   int // byte offset into original text (exclusive)
}

// wordPieceTokenizer is a minimal, offset-preserving BERT tokenizer: Unicode
// normalization is limited to lowercasing (the small tier's do_lower_case=true)
// and whitespace/punctuation splitting, matching BERT's BasicTokenizer closely
// enough for English PII. It records byte offsets so model labels map back to
// the exact source substring.
type wordPieceTokenizer struct {
	vocab     map[string]int64
	lowercase bool
	unkID     int64
	clsID     int64
	sepID     int64
	maxChars  int
}

func newWordPieceTokenizer(vocab map[string]int64, lowercase bool) *wordPieceTokenizer {
	get := func(tok string) int64 {
		if id, ok := vocab[tok]; ok {
			return id
		}
		return -1
	}
	return &wordPieceTokenizer{
		vocab:     vocab,
		lowercase: lowercase,
		unkID:     get("[UNK]"),
		clsID:     get("[CLS]"),
		sepID:     get("[SEP]"),
		maxChars:  100,
	}
}

// tokenize splits text into byte-offset-tracked WordPiece tokens. It walks the
// original (unmodified) text so offsets are exact; lowercasing for the vocab
// lookup is applied per-rune without changing offsets (ASCII and most Latin
// lowercasing is same byte-length; for the rare width-changing case we fall
// back to [UNK] on the original slice, preserving offsets).
func (t *wordPieceTokenizer) tokenize(text string) []piece {
	var pieces []piece
	runes := []rune(text)
	// Precompute byte offset of each rune start.
	byteOffsets := make([]int, len(runes)+1)
	{
		b := 0
		for i, r := range runes {
			byteOffsets[i] = b
			b += utf8RuneLen(r)
		}
		byteOffsets[len(runes)] = b
	}

	i := 0
	for i < len(runes) {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if isPunct(r) {
			// Each punctuation char is its own token.
			bs, be := byteOffsets[i], byteOffsets[i+1]
			t.appendWord(&pieces, text, bs, be)
			i++
			continue
		}
		// Accumulate a word: letters/digits/symbols until space or punctuation.
		j := i
		for j < len(runes) && !unicode.IsSpace(runes[j]) && !isPunct(runes[j]) {
			j++
		}
		bs, be := byteOffsets[i], byteOffsets[j]
		t.appendWord(&pieces, text, bs, be)
		i = j
	}
	return pieces
}

// appendWord applies greedy WordPiece over the word text[bs:be], emitting one
// or more pieces with byte offsets carved out of [bs,be].
func (t *wordPieceTokenizer) appendWord(pieces *[]piece, text string, bs, be int) {
	word := text[bs:be]
	lookup := word
	if t.lowercase {
		lookup = strings.ToLower(word)
	}
	// If lowercasing changed byte length, offset math over sub-slices breaks; in
	// that uncommon case emit a single [UNK] covering the whole word.
	if len(lookup) != len(word) {
		*pieces = append(*pieces, piece{id: t.unkOrZero(), start: bs, end: be})
		return
	}
	if len([]rune(word)) > t.maxChars {
		*pieces = append(*pieces, piece{id: t.unkOrZero(), start: bs, end: be})
		return
	}

	// Greedy longest-match WordPiece over the (byte-aligned) lowercased word.
	chars := []rune(lookup)
	byteOffsets := make([]int, len(chars)+1)
	{
		b := 0
		for i, r := range chars {
			byteOffsets[i] = b
			b += utf8RuneLen(r)
		}
		byteOffsets[len(chars)] = b
	}

	start := 0
	first := true
	var sub []piece
	bad := false
	for start < len(chars) {
		end := len(chars)
		var curID int64 = -1
		var curEnd int
		for end > start {
			piece := string(chars[start:end])
			if !first {
				piece = "##" + piece
			}
			if id, ok := t.vocab[piece]; ok {
				curID = id
				curEnd = end
				break
			}
			end--
		}
		if curID == -1 {
			bad = true
			break
		}
		sub = append(sub, pieceStruct(curID, bs+byteOffsets[start], bs+byteOffsets[curEnd]))
		start = curEnd
		first = false
	}
	if bad {
		*pieces = append(*pieces, piece{id: t.unkOrZero(), start: bs, end: be})
		return
	}
	*pieces = append(*pieces, sub...)
}

func pieceStruct(id int64, start, end int) piece { return piece{id: id, start: start, end: end} }

func (t *wordPieceTokenizer) unkOrZero() int64 {
	if t.unkID >= 0 {
		return t.unkID
	}
	return 0
}

func isPunct(r rune) bool {
	if r >= 33 && r <= 47 || r >= 58 && r <= 64 || r >= 91 && r <= 96 || r >= 123 && r <= 126 {
		return true
	}
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func utf8RuneLen(r rune) int {
	switch {
	case r < 0x80:
		return 1
	case r < 0x800:
		return 2
	case r < 0x10000:
		return 3
	default:
		return 4
	}
}

// --- model artifact loaders ------------------------------------------------

func loadVocab(path string) (map[string]int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	vocab := make(map[string]int64, 32000)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024), 1024*1024)
	var i int64
	for sc.Scan() {
		vocab[strings.TrimRight(sc.Text(), "\r\n")] = i
		i++
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(vocab) == 0 {
		return nil, fmt.Errorf("empty vocab")
	}
	return vocab, nil
}

// modelConfig is the subset of a HF config.json the engine needs: the BIO label
// vocabulary and the tokenizer's casing mode. do_lower_case is read (not
// hardcoded) so the engine serves BOTH an uncased classifier (the small tier
// gravitee-io/bert-small-pii, do_lower_case=true) and a cased classifier (the
// large tier's cased DistilBERT, do_lower_case=false) correctly — a cased model
// lowercased at tokenize time would miss WordPiece pieces and drift the byte
// offsets, so the flag must match the model.
type modelConfig struct {
	id2label  []string
	lowercase bool
}

// loadModelConfig reads config.json's id2label map (into an index-ordered slice)
// and the do_lower_case flag. do_lower_case defaults to true when absent so a
// minimal config (the small tier's shape) keeps its existing behavior.
func loadModelConfig(path string) (modelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return modelConfig{}, err
	}
	var cfg struct {
		ID2Label    map[string]string `json:"id2label"`
		DoLowerCase *bool             `json:"do_lower_case"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return modelConfig{}, err
	}
	if len(cfg.ID2Label) == 0 {
		return modelConfig{}, fmt.Errorf("config.json has no id2label")
	}
	labels := make([]string, len(cfg.ID2Label))
	for k, v := range cfg.ID2Label {
		var idx int
		if _, err := fmt.Sscanf(k, "%d", &idx); err != nil {
			return modelConfig{}, fmt.Errorf("non-integer id2label key %q", k)
		}
		if idx < 0 || idx >= len(labels) {
			return modelConfig{}, fmt.Errorf("id2label index %d out of range for %d labels", idx, len(labels))
		}
		labels[idx] = v
	}
	// do_lower_case lives in tokenizer_config.json for DistilBERT/BERT; HF also
	// mirrors it into config.json for many exports. Prefer config.json; fall back
	// to tokenizer_config.json next to it; default true (the small tier's mode).
	lowercase := true
	if cfg.DoLowerCase != nil {
		lowercase = *cfg.DoLowerCase
	} else if v, ok := readDoLowerCase(filepath.Join(filepath.Dir(path), "tokenizer_config.json")); ok {
		lowercase = v
	}
	return modelConfig{id2label: labels, lowercase: lowercase}, nil
}

// readDoLowerCase reads do_lower_case from a tokenizer_config.json if present.
func readDoLowerCase(path string) (val, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, false
	}
	var tc struct {
		DoLowerCase *bool `json:"do_lower_case"`
	}
	if err := json.Unmarshal(data, &tc); err != nil || tc.DoLowerCase == nil {
		return false, false
	}
	return *tc.DoLowerCase, true
}

func fileReadable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
