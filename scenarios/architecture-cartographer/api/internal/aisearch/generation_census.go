package aisearch

import (
	"sort"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

// GenerationCensusEntry describes the runtime collection pointers currently
// held by Architecture Cartographer. Recipe identity is verified from the
// persisted canonical recipe bytes for generations staged with the v2
// migration contract; legacy generations remain explicitly unknown.
type GenerationCensusEntry struct {
	Owner                 string `json:"owner"`
	StoreKind             string `json:"store_kind"`
	Collection            string `json:"collection"`
	GenerationID          string `json:"generation_id"`
	LifecycleState        string `json:"lifecycle_state"`
	PointerEpoch          uint64 `json:"pointer_epoch"`
	ActivationEpoch       uint64 `json:"activation_epoch,omitempty"`
	BuildJobID            string `json:"build_job_id,omitempty"`
	BuildJobState         string `json:"build_job_state,omitempty"`
	Model                 string `json:"model,omitempty"`
	Role                  string `json:"role,omitempty"`
	Dimensions            int    `json:"dimensions,omitempty"`
	PolicySchemaVersion   string `json:"policy_schema_version,omitempty"`
	RecipeID              string `json:"recipe_id,omitempty"`
	RecipeIdentityState   string `json:"recipe_identity_state"`
	SentinelID            string `json:"sentinel_id,omitempty"`
	SentinelIdentityState string `json:"sentinel_identity_state"`
	SentinelSchemaVersion int    `json:"sentinel_schema_version,omitempty"`
	Durability            string `json:"durability"`
	RestartRecovery       string `json:"restart_recovery"`
	PhysicalState         string `json:"physical_state"`
}

// GenerationCensus returns an in-process inventory of Architecture
// Cartographer's active, retained, and shadow Qdrant collection pointers.
// It reads owner-held memory only: it performs no Qdrant request and does not
// claim restart durability, physical point count, or a verified recipe for
// the existing model/dimension metadata.
func (s *Service) GenerationCensus() []GenerationCensusEntry {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]GenerationCensusEntry, 0, 2+len(s.shadows))
	seen := make(map[string]struct{}, 2+len(s.shadows))
	pointerEpoch := s.lifecycleEpoch
	if pointerEpoch == 0 {
		pointerEpoch = 1
	}
	add := func(candidate *engine, state string) {
		if candidate == nil {
			return
		}
		collection := strings.TrimSpace(candidate.spec.Name)
		if collection == "" {
			return
		}
		if _, ok := seen[collection]; ok {
			return
		}
		seen[collection] = struct{}{}
		durability, recovery := "in_process_only", "unsupported"
		if strings.TrimSpace(s.lifecyclePath) != "" {
			durability, recovery = "durable_manifest", "recoverable"
		}
		recipeID, recipeState := verifiedRecipeIdentity(candidate)
		entry := GenerationCensusEntry{
			Owner: "architecture-cartographer", StoreKind: "qdrant",
			Collection: collection, GenerationID: collection,
			PointerEpoch: pointerEpoch, ActivationEpoch: candidate.lastActiveEpoch,
			LifecycleState: state, Model: candidate.spec.Model,
			Role: candidate.spec.Role, Dimensions: candidate.spec.DenseSize,
			PolicySchemaVersion: candidate.spec.PolicySchemaVersion,
			RecipeID:            recipeID, RecipeIdentityState: recipeState, SentinelID: pkg.PointIDFor("__aisearch_meta__:", collection, 0, 1),
			SentinelIdentityState: "expected_from_owner_spec_unverified", SentinelSchemaVersion: 1,
			Durability: durability, RestartRecovery: recovery, PhysicalState: "unobserved",
		}
		if candidate.shadowJobID != "" {
			entry.BuildJobID = candidate.shadowJobID
			if job, ok := candidate.svc.ReindexStatus(candidate.shadowJobID); ok && job != nil {
				entry.BuildJobState = job.State
			} else {
				entry.BuildJobState = "unavailable"
			}
		} else if candidate.shadowRequest.Collection != "" && durability == "durable_manifest" {
			entry.BuildJobState = "recovered_pending"
		}
		entries = append(entries, entry)
	}
	add(s.eng, "active")
	add(s.retained, "retained")
	for _, collection := range sortedEngineKeys(s.shadows) {
		candidate := s.shadows[collection]
		if candidate == nil {
			continue
		}
		state := "shadow"
		if candidate.shadowRequest.DryRun {
			state = "shadow_dry_run"
		}
		add(candidate, state)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].LifecycleState != entries[j].LifecycleState {
			return entries[i].LifecycleState < entries[j].LifecycleState
		}
		return entries[i].Collection < entries[j].Collection
	})
	return entries
}

func verifiedRecipeIdentity(candidate *engine) (string, string) {
	if candidate == nil || candidate.recipeID == "" || candidate.recipeCanonical == "" {
		return "", "unknown"
	}
	recipe, err := pkg.EmbeddingRecipeFromCanonical([]byte(candidate.recipeCanonical), candidate.recipeID)
	if err != nil {
		return "", "unknown"
	}
	input, err := recipe.Input()
	if err != nil || input.ModelID != candidate.spec.Model || input.Dimensions != candidate.spec.DenseSize {
		return "", "unknown"
	}
	return candidate.recipeID, "verified_canonical"
}

func sortedEngineKeys(engines map[string]*engine) []string {
	keys := make([]string, 0, len(engines))
	for key := range engines {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
