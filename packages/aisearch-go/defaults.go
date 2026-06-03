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

	// DefaultQdrantURL is the local Qdrant address used when none is configured.
	DefaultQdrantURL = "http://127.0.0.1:6333"
)
