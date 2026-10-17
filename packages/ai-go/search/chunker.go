package aisearch

// identityChunker maps one SourceDoc to exactly one Chunk. The 1:1 consumers
// (cli-health commands, ui-health surfaces) use it; documentation uses the
// shared markdown chunker (NewMarkdownChunker, markdown.go). The point ID is assigned by the
// reconciler from the source's natural key, so a single-chunk source keeps the
// un-suffixed ID an existing 1:1 collection already uses.
type identityChunker struct{}

// NewIdentityChunker returns the 1:1 chunker.
func NewIdentityChunker() Chunker { return identityChunker{} }

func (identityChunker) Chunk(doc SourceDoc) ([]Chunk, error) {
	return []Chunk{{
		SourceID: doc.ID,
		Index:    0,
		Body:     doc.Body,
		Meta:     doc.Meta,
	}}, nil
}

// identityComposer embeds a chunk's raw body verbatim — the composition the 1:1
// consumers use. The markdown consumer (Phase 3) supplies its own contextual
// composer.
type identityComposer struct{}

// NewIdentityComposer returns a composer that embeds the chunk body as-is.
func NewIdentityComposer() EmbeddingTextComposer { return identityComposer{} }

func (identityComposer) Compose(chunk Chunk) string { return chunk.Body }
