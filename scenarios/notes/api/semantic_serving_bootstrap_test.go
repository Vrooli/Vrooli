package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	schema "notes-api/internal/notes"
)

type bootstrapGenerationResolver struct{ config schema.FleetGenerationConfig }

func (r bootstrapGenerationResolver) ResolveActiveSemanticGeneration(context.Context) (schema.FleetGenerationConfig, error) {
	return r.config, nil
}

type bootstrapAttribution struct{}

func (bootstrapAttribution) ResolveSemanticAttribution(context.Context) (schema.SemanticAttribution, error) {
	return schema.SemanticAttribution{CallerID: "notes-api", RightsEvidenceURI: "evidence://rights/notes", RightsEvidenceSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", QualityEvidenceURI: "evidence://quality/notes", QualityEvidenceSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, nil
}

type bootstrapTenant struct{}

func (bootstrapTenant) ResolveSemanticTenant(_ context.Context, requested string) (string, error) {
	if requested == "" {
		return "", errors.New("authenticated tenant is required")
	}
	return requested, nil
}

type bootstrapStore struct{}

func (bootstrapStore) QueryActive(context.Context, schema.SemanticFleetQuery) (schema.SemanticFleetQueryResult, error) {
	return schema.SemanticFleetQueryResult{}, nil
}

func (bootstrapStore) UpsertActive(context.Context, schema.SemanticFleetUpsert) (schema.SemanticFleetOperationResult, error) {
	return schema.SemanticFleetOperationResult{}, nil
}

func (bootstrapStore) DeleteActive(context.Context, schema.SemanticFleetDelete) (schema.SemanticFleetOperationResult, error) {
	return schema.SemanticFleetOperationResult{}, nil
}

type bootstrapSemanticFactory struct {
	service *schema.SemanticServing
	err     error
}

func (f bootstrapSemanticFactory) Build(context.Context, *sql.DB) (*schema.SemanticServing, error) {
	return f.service, f.err
}

func TestInitializeSemanticServingCallsInjectedFactoryAndBindsQualifiedOwner(t *testing.T) {
	config := schema.FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-generation-8", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	service := &schema.SemanticServing{
		Generations: bootstrapGenerationResolver{config: config},
		Tenants:     bootstrapTenant{}, Attribution: bootstrapAttribution{}, Store: bootstrapStore{}, Mutations: bootstrapStore{},
	}
	previous := semanticServing
	defer setSemanticServing(previous)
	if err := initializeSemanticServing(context.Background(), &sql.DB{}, bootstrapSemanticFactory{service: service}); err != nil {
		t.Fatalf("initializeSemanticServing() error = %v", err)
	}
	if semanticServing != service {
		t.Fatal("production bootstrap did not install the qualified Notes owner")
	}
}

func TestInitializeSemanticServingRefusesMissingFactoryOrGenerationFacts(t *testing.T) {
	previous := semanticServing
	defer setSemanticServing(previous)
	setSemanticServing(previous)
	if err := initializeSemanticServing(context.Background(), &sql.DB{}, nil); err == nil {
		t.Fatal("startup accepted missing admitted owner factory")
	}
	if semanticServing != nil {
		t.Fatal("missing owner factory left a stale serving owner installed")
	}
	invalid := schema.FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 384}
	service := &schema.SemanticServing{
		Generations: bootstrapGenerationResolver{config: invalid},
		Tenants:     bootstrapTenant{}, Attribution: bootstrapAttribution{}, Store: bootstrapStore{}, Mutations: bootstrapStore{},
	}
	if err := initializeSemanticServing(context.Background(), &sql.DB{}, bootstrapSemanticFactory{service: service}); err == nil {
		t.Fatalf("startup accepted generation without collection identity: %v", err)
	}
	if semanticServing != nil {
		t.Fatal("invalid owner factory left semantic serving enabled")
	}
}
