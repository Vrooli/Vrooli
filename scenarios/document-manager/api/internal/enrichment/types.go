package enrichment

import (
	"context"
	"fmt"
	"time"

	"document-manager/internal/gatewayreq"

	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

const (
	ReaderRetrieval = "retrieval"
	ReaderIntake    = "intake"
)

type Enrichment struct {
	DocumentHash          string
	Status                string
	Summary               string
	SuggestedPrivacyClass documentpb.PrivacyClass
	CreatedAt             time.Time
}

type Embedding struct {
	ID               string
	DocumentHash     string
	UnitID           string
	Role             string
	Model            string
	Dimension        int
	ContentVersion   int
	RetargetStrategy string
	Vector           []float32
	CreatedAt        time.Time
}

func (e Embedding) Validate() error {
	if e.Role == "" || e.Model == "" || e.Dimension <= 0 || e.ContentVersion <= 0 || e.RetargetStrategy == "" {
		return fmt.Errorf("embedding metadata requires role, model, positive dimension, content version, and retarget strategy")
	}
	if len(e.Vector) != e.Dimension {
		return fmt.Errorf("embedding vector length %d does not match dimension %d", len(e.Vector), e.Dimension)
	}
	return nil
}

type Repository interface {
	SaveEnrichment(context.Context, Enrichment) error
	GetEnrichment(context.Context, string) (Enrichment, error)
	SaveEmbedding(context.Context, Embedding) error
	ListEmbeddings(context.Context, string) ([]Embedding, error)
	ListAllEmbeddings(context.Context) ([]Embedding, error)
}

type Gateway interface {
	Invoke(context.Context, *gatewaypb.GatewayRequest) (GatewayResult, error)
}

type GatewayResult struct {
	Summary          string
	SuggestedPrivacy documentpb.PrivacyClass
	Vector           []float32
	Model            string
	Dimension        int
}

type Service struct {
	Repo    Repository
	Builder gatewayreq.Builder
	Gateway Gateway
}

func NewService(repo Repository, builder gatewayreq.Builder, gateway Gateway) Service {
	return Service{Repo: repo, Builder: builder, Gateway: gateway}
}
