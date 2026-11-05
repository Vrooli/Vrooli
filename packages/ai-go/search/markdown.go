package aisearch

import "strings"

// markdown.go holds the corpus-agnostic markdown pieces any documentation-style
// hybrid adopter needs: a header-aware Chunker and an Anthropic-style contextual
// EmbeddingTextComposer. Only the markdown logic lives here — the manifest /
// scope / authority discovery that decides WHICH files to index and how to
// classify them stays adopter-local (it is not corpus-agnostic). These were
// promoted out of knowledge-observatory so adopter #2+ imports them instead of
// re-cloning ~300 LOC.

// Contextual-composition meta keys. The Chunker writes MetaHeadingPath; the
// ContextualComposer reads MetaTitle/MetaDescription/MetaHeadingPath to build the
// self-contained embedding prefix. An adopter's Source must populate
// title/description in SourceDoc.Meta under these keys (heading_path is
// chunk-derived). They are also the Qdrant payload key names, so keep the string
// values stable to avoid a re-index.
const (
	MetaTitle       = "title"        // doc title (manifest title, or first H1 / filename)
	MetaDescription = "description"  // doc description (may be empty)
	MetaHeadingPath = "heading_path" // chunk-derived "H1 > H2 > H3"
)

// Chunking budget. Tokens are estimated (~4 chars/token); these are the tuned
// defaults from 2025/26 RAG guidance for documentation retrieval. A unit larger
// than HardMaxTokens (e.g. a big code fence) becomes its own oversized chunk
// rather than being split mid-fence.
const (
	TargetTokens    = 900 // pack units until adding the next would exceed this
	OverlapTokens   = 135 // ~15% carried into the next chunk for continuity
	MinFlushTokens  = 400 // below this, a heading boundary does not force a flush
	HardMaxTokens   = 1200
	charsPerToken   = 4
	headingPathJoin = " > "
)

// MarkdownChunker splits a markdown documentation body into header-aware chunks.
// It keeps fenced code blocks and tables atomic (never split mid-fence/-row),
// tracks the heading path so each chunk records the section it belongs to,
// flushes at heading boundaries once a chunk is substantial, and seeds each new
// chunk with a small token overlap of the previous one.
type MarkdownChunker struct{}

var _ Chunker = MarkdownChunker{}

// NewMarkdownChunker returns the documentation chunker.
func NewMarkdownChunker() MarkdownChunker { return MarkdownChunker{} }

// unitKind classifies a parsed block for packing decisions.
type unitKind int

const (
	unitText    unitKind = iota // paragraph / list run (overlap-eligible)
	unitHeading                 // an ATX heading line (forces section boundary)
	unitAtomic                  // fenced code block or table (never split, not overlapped)
)

// unit is one parsed block plus the heading path active where it begins.
type unit struct {
	kind        unitKind
	text        string
	headingPath string
}

// Chunk parses the body into units and greedily packs them into chunks.
func (MarkdownChunker) Chunk(doc SourceDoc) ([]Chunk, error) {
	units := parseUnits(doc.Body)
	if len(units) == 0 {
		return nil, nil
	}

	var chunks []Chunk
	var cur []unit
	curTokens := 0

	emit := func() {
		if len(cur) == 0 {
			return
		}
		chunks = append(chunks, newChunk(doc, cur, len(chunks)))
	}

	// seedOverlap returns the trailing overlap-eligible units of the just-
	// flushed chunk to carry into the next one (continuity across the cut).
	seedOverlap := func(prev []unit) ([]unit, int) {
		var seed []unit
		toks := 0
		for i := len(prev) - 1; i >= 0; i-- {
			u := prev[i]
			if u.kind != unitText {
				break // don't duplicate headings or atomic code/table blocks
			}
			t := estimateTokens(u.text)
			if toks+t > OverlapTokens && len(seed) > 0 {
				break
			}
			seed = append([]unit{u}, seed...)
			toks += t
		}
		return seed, toks
	}

	flush := func() {
		emit()
		seed, toks := seedOverlap(cur)
		cur = seed
		curTokens = toks
	}

	for _, u := range units {
		uTokens := estimateTokens(u.text)

		// A heading starts a new section: flush first if the current chunk is
		// already substantial, so sections don't bleed together.
		if u.kind == unitHeading && curTokens >= MinFlushTokens {
			flush()
		}

		// Adding this unit would overflow the target: flush (unless the chunk is
		// empty, in which case an oversized atomic unit becomes its own chunk).
		if curTokens > 0 && curTokens+uTokens > TargetTokens {
			flush()
		}

		cur = append(cur, u)
		curTokens += uTokens

		// An oversized single unit (huge code fence) can't be split — close it
		// out immediately so it doesn't drag siblings over the hard max.
		if curTokens >= HardMaxTokens {
			emit()
			cur = nil
			curTokens = 0
		}
	}
	emit()

	// Re-index sequentially (overlap seeding can otherwise leave gaps).
	for i := range chunks {
		chunks[i].Index = i
	}
	return chunks, nil
}

// newChunk assembles a Chunk from a run of units. Body is the joined unit text;
// Meta is the doc metadata plus the chunk's heading path (taken from its first
// unit). SourceID/Index are (re)assigned by the reconciler.
func newChunk(doc SourceDoc, units []unit, index int) Chunk {
	parts := make([]string, 0, len(units))
	for _, u := range units {
		parts = append(parts, u.text)
	}
	meta := make(map[string]any, len(doc.Meta)+1)
	for k, v := range doc.Meta {
		meta[k] = v
	}
	// The heading path is the most specific section the chunk reaches: the last
	// heading unit it contains, falling back to the path carried into its first
	// unit when the chunk is pure body/overlap with no heading of its own.
	headingPath := units[0].headingPath
	for _, u := range units {
		if u.kind == unitHeading {
			headingPath = u.headingPath
		}
	}
	meta[MetaHeadingPath] = headingPath

	return Chunk{
		SourceID: doc.ID,
		Index:    index,
		Body:     strings.Join(parts, "\n\n"),
		Meta:     meta,
	}
}

// parseUnits splits a markdown body into blocks while tracking the heading
// path. Fenced code blocks and tables are emitted as single atomic units;
// headings are their own units and update the path; everything else is split
// into paragraph/list runs on blank lines.
func parseUnits(body string) []unit {
	lines := strings.Split(body, "\n")
	var units []unit
	var headings []string // heading text by (level-1)

	var para []string
	flushPara := func() {
		if text := strings.TrimRight(strings.Join(para, "\n"), "\n"); strings.TrimSpace(text) != "" {
			units = append(units, unit{kind: unitText, text: text, headingPath: joinHeadings(headings)})
		}
		para = nil
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Fenced code block: consume verbatim through the closing fence.
		if fence := codeFence(trimmed); fence != "" {
			flushPara()
			block := []string{line}
			for i++; i < len(lines); i++ {
				block = append(block, lines[i])
				if strings.HasPrefix(strings.TrimSpace(lines[i]), fence) {
					break
				}
			}
			units = append(units, unit{kind: unitAtomic, text: strings.Join(block, "\n"), headingPath: joinHeadings(headings)})
			continue
		}

		// ATX heading: update the path and emit a heading unit.
		if level, htext := atxHeading(trimmed); level > 0 {
			flushPara()
			setHeading(&headings, level, htext)
			units = append(units, unit{kind: unitHeading, text: line, headingPath: joinHeadings(headings)})
			continue
		}

		// Table: consume the contiguous run of pipe rows as one atomic unit.
		if isTableRow(trimmed) {
			flushPara()
			block := []string{line}
			for i+1 < len(lines) && isTableRow(strings.TrimSpace(lines[i+1])) {
				i++
				block = append(block, lines[i])
			}
			units = append(units, unit{kind: unitAtomic, text: strings.Join(block, "\n"), headingPath: joinHeadings(headings)})
			continue
		}

		// Blank line: paragraph boundary.
		if trimmed == "" {
			flushPara()
			continue
		}
		para = append(para, line)
	}
	flushPara()
	return units
}

// codeFence returns the fence marker ("```" or "~~~") if the line opens a
// fenced code block, else "".
func codeFence(trimmed string) string {
	switch {
	case strings.HasPrefix(trimmed, "```"):
		return "```"
	case strings.HasPrefix(trimmed, "~~~"):
		return "~~~"
	default:
		return ""
	}
}

// atxHeading returns the level (1-6) and text of an ATX heading line, or 0.
func atxHeading(trimmed string) (int, string) {
	if !strings.HasPrefix(trimmed, "#") {
		return 0, ""
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, ""
	}
	if level < len(trimmed) && trimmed[level] != ' ' {
		return 0, "" // "#word" is not a heading
	}
	return level, strings.TrimSpace(strings.TrimLeft(trimmed, "# "))
}

// isTableRow reports whether a trimmed line looks like a markdown table row.
func isTableRow(trimmed string) bool {
	return strings.HasPrefix(trimmed, "|") && strings.Count(trimmed, "|") >= 2
}

// setHeading records a heading at its level and clears any deeper levels.
func setHeading(headings *[]string, level int, text string) {
	h := *headings
	if level <= len(h) {
		h = h[:level-1]
	}
	for len(h) < level-1 {
		h = append(h, "")
	}
	h = append(h, text)
	*headings = h
}

// joinHeadings renders the active heading path, skipping empty levels.
func joinHeadings(headings []string) string {
	parts := make([]string, 0, len(headings))
	for _, h := range headings {
		if h != "" {
			parts = append(parts, h)
		}
	}
	return strings.Join(parts, headingPathJoin)
}

// estimateTokens approximates a token count from character length.
func estimateTokens(text string) int {
	n := len([]rune(text)) / charsPerToken
	if n < 1 {
		return 1
	}
	return n
}

// ContextualComposer prepends a contextual header (title — description, then
// the heading path) to each chunk's body before embedding, so every chunk is
// self-contained (Anthropic-style contextual retrieval).
type ContextualComposer struct{}

var _ EmbeddingTextComposer = ContextualComposer{}

// NewContextualComposer returns the documentation embedding composer.
func NewContextualComposer() ContextualComposer { return ContextualComposer{} }

// Compose builds "{title} — {description}\n{headingPath}\n\n{body}", omitting
// any empty leading lines.
func (ContextualComposer) Compose(chunk Chunk) string {
	title, _ := chunk.Meta[MetaTitle].(string)
	desc, _ := chunk.Meta[MetaDescription].(string)
	headingPath, _ := chunk.Meta[MetaHeadingPath].(string)

	header := strings.TrimSpace(title)
	if d := strings.TrimSpace(desc); d != "" {
		if header != "" {
			header += " — " + d
		} else {
			header = d
		}
	}

	var prefix []string
	if header != "" {
		prefix = append(prefix, header)
	}
	if hp := strings.TrimSpace(headingPath); hp != "" {
		prefix = append(prefix, hp)
	}
	if len(prefix) == 0 {
		return chunk.Body
	}
	return strings.Join(prefix, "\n") + "\n\n" + chunk.Body
}
