package retrieval

import (
	"context"

	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type Unit struct {
	ID           string
	CollectionID string
	DocumentHash string
	PrivacyClass documentpb.PrivacyClass
	Text         string
	AnchorURI    string
}

type Query struct {
	Text             string
	Vector           []float32
	CollectionID     string
	CallerMaxPrivacy documentpb.PrivacyClass
	Federated        bool
	Limit            int
}

type Result struct {
	UnitID       string
	DocumentHash string
	AnchorURI    string
	Score        float64
}

type Response struct {
	Results []Result
	Partial bool
}

// VectorStore is the swappable semantic leg. The initial implementation is a
// SQLite-resident cosine scan; moving it to a vector service changes only this
// implementation, not query policy or RRF fusion.
type VectorStore interface {
	Similar(context.Context, []float32, []Unit, int) map[string]float64
}

type Service struct {
	Repo   Repository
	Vector VectorStore
}

type Repository interface {
	AddUnit(context.Context, Unit) error
	AddVector(context.Context, string, []float32) error
	Candidates(context.Context, Query) ([]Unit, error)
}
