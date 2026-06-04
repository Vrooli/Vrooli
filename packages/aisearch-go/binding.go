package aisearch

// NewDenseBinding builds a SourceBinding for the overwhelmingly common case: a
// dense-only collection whose sources are single-chunk (1 SourceDoc -> 1 Chunk)
// and whose embedding text is the SourceDoc.Body verbatim. It collapses the
// seven-field SourceBinding literal a 1:1 consumer would otherwise write into
// one call:
//
//	b := NewDenseBinding("command", "cli-health:", store, source)
//
// It wires the identity chunker, the identity composer (Composer == nil), and
// no sparse encoder (Sparse == nil). A consumer that fans out into many chunks
// (e.g. the KO markdown docs adopter) or that needs hybrid sparse retrieval
// keeps the full SourceBinding literal instead.
func NewDenseBinding(kind, idPrefix string, store VectorStore, source Source) SourceBinding {
	return SourceBinding{
		Kind:     kind,
		Store:    store,
		Source:   source,
		Chunker:  NewIdentityChunker(),
		IDPrefix: idPrefix,
	}
}
