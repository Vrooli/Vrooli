package aisearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

const lifecycleManifestVersion = 3
const legacyLifecycleManifestVersion = 2

// persistedGeneration is the restart-safe portion of one engine. Build jobs
// are process-local and are deliberately not treated as complete after restart.
type persistedGeneration struct {
	Collection      string             `json:"collection"`
	Spec            pkg.CollectionSpec `json:"spec"`
	LastActiveEpoch uint64             `json:"last_active_epoch,omitempty"`
	Tuning          pkg.TuningConfig   `json:"tuning"`
	RecipeID        string             `json:"recipe_id,omitempty"`
	RecipeCanonical string             `json:"recipe_canonical,omitempty"`
	Request         *ShadowRequest     `json:"request,omitempty"`
}

type lifecycleManifest struct {
	Version      int                   `json:"version"`
	PointerEpoch uint64                `json:"pointer_epoch"`
	Active       persistedGeneration   `json:"active"`
	Retained     *persistedGeneration  `json:"retained,omitempty"`
	Shadows      []persistedGeneration `json:"shadows,omitempty"`
}

// RestoreLifecycle rebuilds the durable collection pointers after process
// restart. It does not reindex or mutate Qdrant: a recovered shadow has no
// trusted in-process job receipt and must be explicitly resumed by repeating
// its original stage request. Activation still calls EnsureCollection, which
// checks the physical collection sentinel before moving the pointer.
func (s *Service) RestoreLifecycle() error {
	if s == nil || strings.TrimSpace(s.lifecyclePath) == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	manifest, err := readLifecycleManifest(s.lifecyclePath)
	if errors.Is(err, os.ErrNotExist) {
		return s.persistLifecycleLocked()
	}
	if err != nil {
		return err
	}
	if manifest.Version != lifecycleManifestVersion && manifest.Version != legacyLifecycleManifestVersion {
		return fmt.Errorf("aisearch: unsupported lifecycle manifest version %d", manifest.Version)
	}
	if strings.TrimSpace(manifest.Active.Collection) == "" {
		return errors.New("aisearch: lifecycle manifest has no active collection")
	}
	active, err := s.recoverGenerationLocked(manifest.Active)
	if err != nil {
		return fmt.Errorf("aisearch: restore active collection: %w", err)
	}
	var retained *engine
	if manifest.Retained != nil {
		retained, err = s.recoverGenerationLocked(*manifest.Retained)
		if err != nil {
			return fmt.Errorf("aisearch: restore retained collection: %w", err)
		}
	}
	shadows := make(map[string]*engine, len(manifest.Shadows))
	for _, record := range manifest.Shadows {
		candidate, buildErr := s.recoverGenerationLocked(record)
		if buildErr != nil {
			return fmt.Errorf("aisearch: restore shadow %q: %w", record.Collection, buildErr)
		}
		if record.Request != nil {
			candidate.shadowRequest = *record.Request
		}
		// A job id belongs to its old process and cannot be trusted as a live
		// job receipt after reconstruction.
		candidate.shadowJobID = ""
		shadows[record.Collection] = candidate
	}
	if manifest.PointerEpoch == 0 {
		return errors.New("aisearch: lifecycle manifest has an invalid zero pointer epoch")
	}
	active.lastActiveEpoch = manifest.PointerEpoch
	s.eng, s.retained, s.shadows = active, retained, shadows
	s.shadowJobs = make(map[string]*pkg.Service)
	s.lifecycleEpoch = manifest.PointerEpoch
	return nil
}

func (s *Service) recoverGenerationLocked(record persistedGeneration) (*engine, error) {
	if record.Collection == "" || record.Spec.Name != record.Collection {
		return nil, errors.New("collection identity is missing or inconsistent")
	}
	if record.Request != nil {
		request := record.Request
		if request.Collection != record.Collection || request.Embedding.Model != record.Spec.Model || request.Embedding.Role != record.Spec.Role || request.Embedding.Dimensions != record.Spec.DenseSize || request.Embedding.PolicySchemaVersion != record.Spec.PolicySchemaVersion || request.EmbeddingRecipeID != record.RecipeID || request.EmbeddingRecipeCanonical != record.RecipeCanonical {
			return nil, errors.New("persisted shadow request disagrees with its generation identity")
		}
	}
	if s.buildShadow == nil {
		return nil, errors.New("service has no generation builder")
	}
	policy := pkg.EmbeddingPolicy{Model: record.Spec.Model, Role: record.Spec.Role, Dimensions: record.Spec.DenseSize, PolicySchemaVersion: record.Spec.PolicySchemaVersion}
	var engine *engine
	if s.buildPersisted != nil && record.Tuning.Engine != "" {
		engine = s.buildPersisted(policy, record.Collection, record.Tuning)
	} else {
		engine = s.buildShadow(policy, record.Collection)
	}
	if engine == nil || engine.svc == nil || engine.vectorStore == nil || engine.spec != record.Spec {
		return nil, fmt.Errorf("generation builder did not reproduce the persisted collection spec for %q", record.Collection)
	}
	if record.RecipeID != "" || record.RecipeCanonical != "" {
		if record.RecipeID == "" || record.RecipeCanonical == "" {
			return nil, errors.New("persisted generation has an incomplete recipe identity")
		}
		recipe, err := pkg.EmbeddingRecipeFromCanonical([]byte(record.RecipeCanonical), record.RecipeID)
		if err != nil {
			return nil, fmt.Errorf("persisted generation recipe is invalid: %w", err)
		}
		input, err := recipe.Input()
		if err != nil {
			return nil, err
		}
		if input.ModelID != record.Spec.Model || input.Dimensions != record.Spec.DenseSize {
			return nil, errors.New("persisted generation recipe does not match collection model and dimensions")
		}
		engine.recipeID, engine.recipeCanonical = record.RecipeID, record.RecipeCanonical
	}
	engine.lastActiveEpoch = record.LastActiveEpoch
	return engine, nil
}

func readLifecycleManifest(path string) (lifecycleManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return lifecycleManifest{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	var manifest lifecycleManifest
	if err := decoder.Decode(&manifest); err != nil {
		return lifecycleManifest{}, fmt.Errorf("aisearch: decode lifecycle manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return lifecycleManifest{}, errors.New("aisearch: lifecycle manifest has trailing JSON")
		}
		return lifecycleManifest{}, fmt.Errorf("aisearch: read lifecycle manifest trailer: %w", err)
	}
	return manifest, nil
}

func (s *Service) persistLifecycleLocked() error {
	if strings.TrimSpace(s.lifecyclePath) == "" {
		return nil
	}
	if s.eng == nil {
		return errors.New("aisearch: cannot persist lifecycle without active engine")
	}
	manifest := lifecycleManifest{Version: lifecycleManifestVersion, PointerEpoch: s.lifecycleEpoch, Active: persistEngine(s.eng)}
	if s.retained != nil {
		retained := persistEngine(s.retained)
		manifest.Retained = &retained
	}
	for _, collection := range sortedEngineKeys(s.shadows) {
		if candidate := s.shadows[collection]; candidate != nil {
			manifest.Shadows = append(manifest.Shadows, persistEngine(candidate))
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("aisearch: encode lifecycle manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.lifecyclePath), 0o700); err != nil {
		return fmt.Errorf("aisearch: create lifecycle manifest directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.lifecyclePath), ".embedding-lifecycle-*.tmp")
	if err != nil {
		return fmt.Errorf("aisearch: create lifecycle manifest temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("aisearch: set lifecycle manifest permissions: %w", err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("aisearch: write lifecycle manifest: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("aisearch: sync lifecycle manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("aisearch: close lifecycle manifest: %w", err)
	}
	if err := os.Rename(tmpName, s.lifecyclePath); err != nil {
		return fmt.Errorf("aisearch: replace lifecycle manifest: %w", err)
	}
	if directory, err := os.Open(filepath.Dir(s.lifecyclePath)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}

func persistEngine(engine *engine) persistedGeneration {
	if engine == nil {
		return persistedGeneration{}
	}
	record := persistedGeneration{Collection: engine.spec.Name, Spec: engine.spec, LastActiveEpoch: engine.lastActiveEpoch, Tuning: engine.tuning, RecipeID: engine.recipeID, RecipeCanonical: engine.recipeCanonical}
	if engine.shadowRequest.Collection != "" {
		request := engine.shadowRequest
		record.Request = &request
	}
	return record
}
