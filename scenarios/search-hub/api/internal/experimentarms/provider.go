package experimentarms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
)

// ProviderCapabilities is an explicit refusal-capable declaration. A provider
// must report unsupported operations before the orchestrator schedules them.
type ProviderCapabilities struct {
	ProviderID      string
	OwnerID         string
	RecipeSchemes   []string
	CanStage        bool
	CanCatchUp      bool
	CanValidate     bool
	CanPromote      bool
	CanRollback     bool
	CanCancel       bool
	CanDelete       bool
	CapabilityBuild string
}

// GenerationRequest binds all provider side effects to one persisted arm,
// recipe digest, source snapshot, and idempotency key.
type GenerationRequest struct {
	ArmID           string
	ExperimentID    string
	RecipeID        string
	RecipeCanonical []byte
	SourceSnapshot  string
	Watermark       string
	IdempotencyKey  string
}

// ProviderReceipt is immutable evidence returned by an owner adapter. Search
// Hub records the receipt reference; provider-owned storage remains owned by
// the provider.
type ProviderReceipt struct {
	ProviderID   string
	Operation    string
	OperationID  string
	ArmID        string
	RecipeID     string
	GenerationID string
	State        string
	Evidence     EvidenceRef
}

// ProviderAdapter is the future physical-store boundary. Implementations own
// their data and must make every mutating operation idempotent. No adapter is
// invoked by this contract package, and no Qdrant/SQLite implementation is
// provided here.
type ProviderAdapter interface {
	Capabilities(context.Context) (ProviderCapabilities, error)
	Stage(context.Context, GenerationRequest) (ProviderReceipt, error)
	CatchUp(context.Context, GenerationRequest) (ProviderReceipt, error)
	Validate(context.Context, GenerationRequest) (ProviderReceipt, error)
	Promote(context.Context, GenerationRequest) (ProviderReceipt, error)
	Rollback(context.Context, GenerationRequest) (ProviderReceipt, error)
	Cancel(context.Context, GenerationRequest) (ProviderReceipt, error)
	Delete(context.Context, GenerationRequest) (ProviderReceipt, error)
}

// Validate rejects incomplete or changed generation identities before an
// adapter performs provider-owned work.
func (r GenerationRequest) Validate() error {
	for name, value := range map[string]string{
		"arm_id": r.ArmID, "experiment_id": r.ExperimentID, "source_snapshot": r.SourceSnapshot,
		"watermark": r.Watermark, "idempotency_key": r.IdempotencyKey,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("generation request %s is required", name)
		}
	}
	recipe, err := aisearch.EmbeddingRecipeFromCanonical(r.RecipeCanonical, strings.TrimPrefix(r.RecipeID, "sha256:"))
	if err != nil {
		return fmt.Errorf("generation request recipe identity is invalid: %w", err)
	}
	if r.RecipeID != recipe.ID() {
		return errors.New("generation request recipe ID does not match its canonical recipe")
	}
	return nil
}
