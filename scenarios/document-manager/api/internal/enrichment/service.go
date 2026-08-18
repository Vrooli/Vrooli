package enrichment

import (
	"context"
	"fmt"
	"time"

	"document-manager/internal/gatewayreq"
	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

func (s Service) Enrich(ctx context.Context, hash, text string, class documentpb.PrivacyClass) (Enrichment, error) {
	if s.Gateway == nil || s.Builder == nil {
		return s.recordUnavailable(ctx, hash)
	}
	request, err := s.Builder.For(ctx, gatewayreq.DocumentClass{PrivacyClass: class}, gatewayreq.Options{Profile: gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE, Kind: gatewaypb.RequestKind_REQUEST_KIND_TEXT_GENERATION, Role: "summarize", Operation: "document-enrichment", Timeout: 30 * time.Second})
	if err != nil {
		return s.recordUnavailable(ctx, hash)
	}
	request.Metadata = map[string]string{"text": text}
	result, err := s.Gateway.Invoke(ctx, request)
	if err != nil {
		return s.recordUnavailable(ctx, hash)
	}
	record := Enrichment{DocumentHash: hash, Status: "enriched", Summary: result.Summary, SuggestedPrivacyClass: result.SuggestedPrivacy, CreatedAt: time.Now().UTC()}
	if err := s.Repo.SaveEnrichment(ctx, record); err != nil {
		return Enrichment{}, err
	}
	return record, nil
}

func (s Service) Embed(ctx context.Context, hash, unitID, text string, class documentpb.PrivacyClass, role string) (Embedding, error) {
	if role == "" {
		role = "semantic-query"
	}
	if s.Gateway == nil || s.Builder == nil {
		return Embedding{}, fmt.Errorf("embedding unavailable")
	}
	request, err := s.Builder.For(ctx, gatewayreq.DocumentClass{PrivacyClass: class}, gatewayreq.Options{Profile: gatewaypb.Profile_PROFILE_PRIVACY_SENSITIVE, Kind: gatewaypb.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, Role: "embed", Operation: role, Timeout: 30 * time.Second})
	if err != nil {
		return Embedding{}, err
	}
	request.Metadata = map[string]string{"text": text, "reader": readerForRole(role)}
	result, err := s.Gateway.Invoke(ctx, request)
	if err != nil {
		return Embedding{}, err
	}
	embedding := Embedding{ID: hash + ":" + unitID + ":" + role, DocumentHash: hash, UnitID: unitID, Role: role, Model: result.Model, Dimension: result.Dimension, ContentVersion: 1, RetargetStrategy: "re-embed-on-model-change", Vector: result.Vector, CreatedAt: time.Now().UTC()}
	if err := embedding.Validate(); err != nil {
		return Embedding{}, err
	}
	if err := s.Repo.SaveEmbedding(ctx, embedding); err != nil {
		return Embedding{}, err
	}
	return embedding, nil
}

func (s Service) recordUnavailable(ctx context.Context, hash string) (Enrichment, error) {
	record := Enrichment{DocumentHash: hash, Status: "unenriched", CreatedAt: time.Now().UTC()}
	if err := s.Repo.SaveEnrichment(ctx, record); err != nil {
		return Enrichment{}, err
	}
	return record, nil
}

func readerForRole(role string) string {
	if role == "near-duplicate" {
		return ReaderIntake
	}
	return ReaderRetrieval
}
