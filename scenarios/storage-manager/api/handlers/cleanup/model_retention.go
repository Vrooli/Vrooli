package cleanup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/packages/capacity/modelretention"

	cleanupcore "storage-manager/internal/cleanup"
	"storage-manager/internal/providers"
)

// ModelRetentionPath serves the current model-store eviction decision (the
// ranking, bands, budgets and selection) without applying anything.
const ModelRetentionPath = "/api/v1/cleanup/model-retention"

// ModelRetentionPolicyPath is where the bands, idle window and stores live.
func ModelRetentionPolicyPath(repoRoot string) string {
	return filepath.Join(repoRoot, "scenarios", "storage-manager", "config", "model-retention-policy.json")
}

func newModelStoreRetentionProvider(repoRoot string) (*providers.ModelStoreRetentionProvider, error) {
	policy, err := providers.LoadModelRetentionPolicy(ModelRetentionPolicyPath(repoRoot))
	if err != nil {
		return nil, err
	}
	stores := make([]providers.ModelStore, 0, len(policy.Stores))
	for _, s := range policy.Stores {
		stores = append(stores, providers.CommandModelStore{Name: s.Resource, Command: s.Command})
	}
	env := providers.ModelStoreEnvironment{
		UsagePath: func(resource string) (string, error) {
			paths, err := modelretention.StorePaths(resource)
			return paths.UsageFile, err
		},
		Space:  providers.MeasureFilesystemSpace,
		Budget: func(resource string) (int64, error) { return resourceModelBudget(repoRoot, resource) },
	}
	return providers.NewModelStoreRetentionProvider(policy, stores, env, schedule.System()), nil
}

// resourceModelBudget reads the store's declared budget from its resource
// manifest (storage.entries.models.budget.max_bytes), the single source of
// truth for how large the store may grow.
func resourceModelBudget(repoRoot, resource string) (int64, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "resources", resource, "resource.json"))
	if err != nil {
		return 0, err
	}
	var manifest struct {
		Storage struct {
			Entries map[string]struct {
				Budget *struct {
					MaxBytes string `json:"max_bytes"`
				} `json:"budget"`
			} `json:"entries"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return 0, fmt.Errorf("decode %s resource manifest: %w", resource, err)
	}
	entry, ok := manifest.Storage.Entries["models"]
	if !ok || entry.Budget == nil || strings.TrimSpace(entry.Budget.MaxBytes) == "" {
		return 0, nil
	}
	return coreRetention.ParseBytes(entry.Budget.MaxBytes)
}

type providerLookup interface {
	Provider(string) (cleanupcore.Provider, bool)
}

// ModelStoreRetention returns the registered model-store provider.
func ModelStoreRetention(service any) (*providers.ModelStoreRetentionProvider, bool) {
	lookup, ok := service.(providerLookup)
	if !ok {
		return nil, false
	}
	p, ok := lookup.Provider("model-store-retention")
	if !ok {
		return nil, false
	}
	msp, ok := p.(*providers.ModelStoreRetentionProvider)
	return msp, ok
}

func mountModelRetention(r *mux.Router, service any) {
	provider, ok := ModelStoreRetention(service)
	if !ok {
		return
	}
	r.HandleFunc(ModelRetentionPath, func(w http.ResponseWriter, req *http.Request) {
		decision := provider.Decide(req.Context())
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			providers.ModelEvictionDecision
			Lines []string `json:"lines"`
		}{decision, providers.DecisionLines(decision)})
	}).Methods(http.MethodGet)
}
