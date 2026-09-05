// Package retrieval owns query planning, retrieval legs, fusion, and ranking.
package retrieval

import "context"

type Regime string

const (
	RegimeExact        Regime = "exact"
	RegimeNatural      Regime = "natural_language"
	RegimeRelationship Regime = "relationship"
	RegimeContract     Regime = "contract"
)

type Query struct {
	Text       string
	Target     string
	Scope      string
	Roles      []string
	Families   []string
	Languages  []string
	Generation string
	Limit      int
}

type RankEvidence struct {
	Leg   string
	Rank  int
	Score float64
}

type Candidate struct {
	ID           string
	Path         string
	Title        string
	Text         string
	SourceHash   string
	Generation   string
	Role         string
	Language     string
	Scope        string
	Authority    string
	Kind         string
	StartLine    int
	EndLine      int
	Regime       Regime
	Explanation  string
	Evidence     string
	Proof        string
	ScoreFactors map[string]float64
	Score        float64
	RankEvidence []RankEvidence
}

type Plan struct {
	Regime      Regime
	UseLexical  bool
	UseSemantic bool
	UseReranker bool
}

type (
	Planner         interface{ Plan(Query) Plan }
	LexicalSearcher interface {
		SearchLexical(context.Context, Query) ([]Candidate, error)
	}
	SemanticSearcher interface {
		SearchSemantic(context.Context, Query) ([]Candidate, error)
	}
	Reranker interface {
		Rerank(context.Context, Query, []Candidate) ([]Candidate, error)
	}
	Fusion interface {
		Fuse(Query, map[string][]Candidate) []Candidate
	}
)

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type VectorRecord struct {
	ID         string
	Vector     []float32
	SourceHash string
	Generation string
	Payload    map[string]string
}

type VectorStore interface {
	Upsert(context.Context, []VectorRecord) error
	Delete(context.Context, []string) error
	Query(context.Context, []float32, Query) ([]Candidate, error)
}

type Admission interface {
	Acquire(context.Context, string, int) (release func(), err error)
}
