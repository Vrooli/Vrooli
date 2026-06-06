package aisearch

// Engine-wide defaults shared by the concrete implementations. Consumers
// override per call (CollectionSpec carries its own size/distance/model); these
// only fill in the common case so a caller can stay terse.
const (
	// DefaultEmbedModel is the substrate-standard dense embedding model. All
	// current consumers (cli/ui/security) and the KO cutover standardize on it
	// at 768 dimensions (the stranded legacy 1024-dim KO collections used
	// mxbai-embed-large; see the plan §1).
	DefaultEmbedModel = "nomic-embed-text"
	// DefaultVectorSize is the nomic-embed-text dimension.
	DefaultVectorSize = 768
	// DefaultDenseDistance is the similarity metric for the dense vector.
	DefaultDenseDistance = "Cosine"
	// DefaultSparseModifier makes Qdrant apply IDF server-side over the
	// locally-computed term-frequency sparse weights (Phase 0 decision D2).
	DefaultSparseModifier = "idf"

	// DefaultReconcileParallelism / MaxReconcileParallelism bound the embed
	// worker pool (one in-flight embed per worker).
	DefaultReconcileParallelism = 4
	MaxReconcileParallelism     = 16

	// DefaultRerankShortlist is the over-fetch depth handed to the reranker: the
	// query pulls this many candidates (or the page size, whichever is larger) so
	// the reranker reorders a meaningful pool before the page is sliced. Higher =
	// better recall into the rerank, more candidates to score (latency cost on
	// the LLM leg; negligible on the cross-encoder). Bounded to
	// [MinRerankShortlist, MaxRerankShortlist].
	DefaultRerankShortlist = 50
	MinRerankShortlist     = 1
	MaxRerankShortlist     = 500

	// DefaultQdrantURL is the local Qdrant address used when none is configured.
	DefaultQdrantURL = "http://127.0.0.1:6333"

	// DefaultRelevanceMaxGap / DefaultRelevanceHardFloor tune ApplyRelevanceFloor
	// (WS2) for the dense-cosine regime (the rerank-off path). MaxGap is the
	// primary, query-adaptive gate; HardFloor is a garbage-only safety net.
	// FloorForLeg returns these for an unrecognized / "none" leg.
	DefaultRelevanceMaxGap    = 0.15
	DefaultRelevanceHardFloor = 0.35

	// Cross-encoder regime floor: junk collapses to ~0 and direct answers sit
	// high, so HardFloor is the primary gate (kills the ~0 gibberish) and MaxGap
	// is permissive so a legitimately weak-but-real answer far below the top hit
	// is not relatively cut. Chosen from the Track-C eval matrix.
	CrossEncoderRelevanceMaxGap    = 0.60
	CrossEncoderRelevanceHardFloor = 0.15

	// LLM regime floor: the 0..1 listwise judge spaces relevance more evenly than
	// the cross-encoder, so a moderate gap plus a low garbage floor fits.
	LLMRelevanceMaxGap    = 0.50
	LLMRelevanceHardFloor = 0.20

	// Fusion (RRF) regime floor: a server-side reciprocal-rank-fusion score is an
	// uncalibrated *ranking* signal, not a 0..1 relevance probability — its
	// magnitude depends on the fusion constant and leg count, and real answers to
	// a sparse query spread well below the top hit. So the query-adaptive relative
	// MaxGap is the ONLY gate, and the absolute HardFloor is disabled (0): the
	// cosine 0.35 HardFloor would wrongly annihilate genuine fused hits — exactly
	// the cliff the old ServiceOptions.ApplyFloor opt-out worked around. MaxGap is
	// generous for the same reason (cut only the far tail). FloorForMethodLeg
	// returns these for an RRF-fused leg (method "hybrid", no reranker).
	FusionRelevanceMaxGap    = 0.50
	FusionRelevanceHardFloor = 0.0
)
