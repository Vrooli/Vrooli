package aisearch

// DenseEngine is the assembled bundle a dense-only, single-chunk adopter wires
// at boot: the embedder, the vector store, the reranker chain, and the
// CollectionSpec that matches them. It exists to remove the per-adopter
// foot-guns the first adoption (cli-health) exposed — the collection name
// threaded through two calls, and a CollectionSpec whose size/distance/model
// can drift from the store. With NewDenseEngine the collection name is named
// once and the spec is derived from cfg/defaults so it cannot disagree with the
// store.
type DenseEngine struct {
	Embedder    Embedder
	VectorStore VectorStore
	Reranker    *RerankerChain
	Spec        CollectionSpec
}

// NewDenseEngine assembles the dense-only common case from existing
// constructors only — it introduces no new seam and no new behavior, so it
// cannot diverge from the hand-wired path. A scenario that needs a non-default
// shape (hybrid sparse, a foreign embedding dimension, a custom reranker order)
// keeps hand-wiring the constructors directly; this is the foolproof default,
// not a mandate.
//
// The CollectionSpec uses the engine defaults for the dense vector (size 768,
// Cosine) and records cfg.EmbedModel as the collection model so the schema
// guard's model attribution always matches the embedder actually wired here.
// Adopters embedding with a model whose dimension differs from
// DefaultVectorSize must set Spec.DenseSize themselves (or hand-wire).
func NewDenseEngine(cfg Config, collection string) DenseEngine {
	return DenseEngine{
		Embedder:    NewEmbedder(cfg.EmbedModel),
		VectorStore: NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, collection),
		Reranker: NewRerankerChain(
			NewCrossEncoderReranker(),
			NewLLMReranker(cfg.RerankModel),
		),
		Spec: CollectionSpec{
			Name:          collection,
			DenseSize:     DefaultVectorSize,
			DenseDistance: DefaultDenseDistance,
			Model:         cfg.EmbedModel,
		},
	}
}
