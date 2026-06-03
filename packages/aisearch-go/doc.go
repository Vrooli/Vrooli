// Package aisearch is the shared retrieval library extracted from the proven
// cli-health / ui-health / security-health engines. It is the single place
// that owns embedding, chunking, two-level drift reconciliation, hybrid
// (dense + sparse) vector search, and reranking, so that every scenario that
// needs semantic search composes the same primitives instead of re-rolling a
// ~80%-copied engine.
//
// Ownership and provenance:
//   - This package is created and owned by the Knowledge Observatory search
//     cutover plan: docs/plans/knowledge-observatory-search-cutover-plan.md.
//   - The federated `search-hub` plan
//     (docs/plans/unified-search-hub-plan.md) *consumes* its result by
//     federating providers that implement search with this library; the router
//     never imports it. The two plans meet only at the provider contract.
//   - The interface NAMES in this file are a published contract referenced by
//     the search-hub plan's Required Reading. Keep them stable:
//     Source, SourceDoc, Chunk, Chunker, EmbeddingTextComposer, Reranker,
//     plus the hybrid VectorStore and the search Service surface.
//
// Status: Phase 0 (contracts only). This file defines the interfaces and the
// data-transfer types they exchange. There are deliberately NO concrete
// implementations here yet — the engine is lifted from cli-health in Phase 1,
// generalized for the 1-source -> N-chunk fan-out, and proven by migrating
// cli-health onto it in Phase 2 before any Knowledge Observatory behavior
// changes. Consumers inject concrete impls; tests inject fakes.
//
// The generalization over the current command/surface engines is the
// 1-source -> N-chunk fan-out: a single CLI command embeds as one vector, but
// a single documentation file fans out into many chunks. Everything else
// (drift skip via payload hash, ghost deletion, bounded-concurrency embed,
// auto -> ai -> text fallback) carries over unchanged.
package aisearch
