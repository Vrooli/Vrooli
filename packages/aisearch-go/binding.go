package aisearch

import (
	"context"
	"fmt"
)

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

// NewHybridBinding builds a SourceBinding for a hybrid (dense + sparse) corpus
// that fans out one source into many chunks — the markdown-doc / record shape.
// It collapses the seven-field SourceBinding literal a hybrid adopter would
// otherwise hand-write (the call site that made the silent-sparse cliff easy to
// hit) into one call, threading the markdown chunker, the contextual composer,
// and the sparse encoder explicitly:
//
//	b := NewHybridBinding("doc", "ko:", eng.VectorStore, source,
//	    NewMarkdownChunker(), NewContextualComposer(), eng.SparseEncoder)
//
// Pair it with NewHybridEngine (whose Spec already sets Sparse=true) and ensure
// the collection via EnsureCollectionForBinding so the cliff is closed.
func NewHybridBinding(kind, idPrefix string, store VectorStore, source Source, chunker Chunker, composer EmbeddingTextComposer, sparse SparseEncoder) SourceBinding {
	return SourceBinding{
		Kind:     kind,
		Store:    store,
		Source:   source,
		Chunker:  chunker,
		Composer: composer,
		Sparse:   sparse,
		IDPrefix: idPrefix,
	}
}

// AssertCollectionMatchesBinding fails loudly when a binding's retrieval shape
// disagrees with the collection spec it will be reconciled against. The one that
// matters: a hybrid binding (Sparse encoder set) MUST target a Sparse=true
// collection — otherwise Qdrant has no sparse named-vector to receive the sparse
// half of each point, the sparse leg is silently dropped, and search degrades to
// dense-only with NO error. This is the "silent cliff" a hand-wired hybrid that
// forgot Spec.Sparse=true falls off; the assertion turns it into a boot-time
// failure with a fix in the message.
func AssertCollectionMatchesBinding(b SourceBinding, spec CollectionSpec) error {
	if b.Sparse != nil && !spec.Sparse {
		return fmt.Errorf("aisearch: hybrid binding %q has a sparse encoder but its collection spec (%q) is dense-only (Sparse=false) — the sparse vectors would be silently dropped and search would never fuse; assemble with NewHybridEngine or set Spec.Sparse=true", b.Kind, spec.Name)
	}
	return nil
}

// EnsureCollectionForBinding asserts the binding/spec shape agree (closing the
// silent-sparse cliff) and then ensures the collection. It is the assembly path
// the README recipe and both in-tree adopters use instead of calling
// store.EnsureCollection directly, so a hybrid/dense mismatch can never reach
// Qdrant.
func EnsureCollectionForBinding(ctx context.Context, store VectorStore, b SourceBinding, spec CollectionSpec) error {
	if err := AssertCollectionMatchesBinding(b, spec); err != nil {
		return err
	}
	return store.EnsureCollection(ctx, spec)
}
