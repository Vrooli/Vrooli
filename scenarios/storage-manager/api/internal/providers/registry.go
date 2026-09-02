package providers

import (
	"context"
	"fmt"
	"sort"

	"github.com/vrooli/vrooli/packages/artifactledger"

	"storage-manager/internal/cleanup"
)

type Registry struct {
	byID map[string]cleanup.Provider
}

func NewRegistry(providers ...cleanup.Provider) (*Registry, error) {
	registry := &Registry{byID: make(map[string]cleanup.Provider, len(providers))}
	for _, provider := range providers {
		if err := registry.Register(provider); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(provider cleanup.Provider) error {
	if r == nil {
		return fmt.Errorf("provider registry is nil")
	}
	if err := cleanup.ValidateProvider(provider); err != nil {
		return err
	}
	meta := provider.Metadata()
	if _, exists := r.byID[meta.ID]; exists {
		return fmt.Errorf("provider %q already registered", meta.ID)
	}
	r.byID[meta.ID] = provider
	return nil
}

func (r *Registry) Get(id string) (cleanup.Provider, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.byID[id]
	return provider, ok
}

func (r *Registry) List() []cleanup.ProviderMetadata {
	if r == nil {
		return nil
	}
	ids := make([]string, 0, len(r.byID))
	for id := range r.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]cleanup.ProviderMetadata, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.byID[id].Metadata())
	}
	return out
}

func ConservativeBuiltIns(deps BuiltInDeps) ([]cleanup.Provider, error) {
	providers := []cleanup.Provider{
		NewUndeclaredWorkloadProvider(deps.ProcessRunner, UndeclaredWorkloadProviderConfig{HistoricalNames: map[string]string{"airbyte-abctl-control-plane": "agent-experiments/airbyte-abctl-control-plane absent manifest"}, Posture: "vrooli_only"}),
		NewScenarioBinariesProvider(deps.FileSystem, deps.ProcessLiveness, deps.Clock, ScenarioBinariesProviderConfig{Root: deps.ScenarioBinariesRoot, Ledger: deps.RemovalLedger}),
		NewTrashProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "trash",
			Name:        "Trash",
			Roots:       deps.TrashRoots,
			Description: "Remove aged trash entries",
			// A trashed item is a unit: its payload and its metadata record
			// are deleted together or not at all.
			TopLevelEntries: true,
		}),
		NewTmpProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "tmp",
			Name:        "Temporary files",
			Roots:       deps.TmpRoots,
			Description: "Remove aged temporary entries below configured roots",
			// A temp staging directory is a unit. Deleting the files inside it
			// individually would reclaim the bytes but leave the directory
			// behind, and would age each file separately — which can strand a
			// half-deleted fragment of an otherwise coherent directory.
			TopLevelEntries: true,
		}),
		NewScratchProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "agent-scratch",
			Name:        "Agent scratch",
			Roots:       deps.ScratchRoots,
			Description: "Remove aged entries from the repository scratch directory",
			// A plan's scratch directory is a unit, for the same reason a temp
			// staging directory is: aging its files independently can leave a
			// half-deleted fragment that is worse than either keeping or
			// removing the whole capture.
			TopLevelEntries: true,
		}),
		NewCacheProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "go-build-cache",
			Name:        "Go build cache",
			Roots:       deps.GoBuildCacheRoots,
			Description: "Remove aged Go build cache entries",
		}),
		NewCacheProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "playwright-cache",
			Name:        "Playwright browser cache",
			Roots:       deps.PlaywrightCacheRoots,
			Description: "Remove aged Playwright browser cache entries",
		}),
		NewDockerProvider(deps.Docker),
		NewJournalProvider(deps.Journal),
		NewCommandMetadataProvider(CommandProviderConfig{
			ID:                 "apt-metadata",
			Name:               "APT metadata",
			SafetyTier:         cleanup.SafetyTierConditional,
			DefaultMode:        cleanup.ProviderModeDisabled,
			DefaultApproval:    cleanup.ApprovalModeOperator,
			RequiredPrivileges: []string{"sudo"},
			TestSubstitute:     "fake-process-runner",
			EstimateCommand:    cleanup.ProcessCommand{Name: "apt-cache", Args: []string{"stats"}},
			PreviewAction:      "apt-metadata-clean",
		}, deps.ProcessRunner),
	}
	if deps.DockerImageLedger != nil {
		providers = append(providers, NewDockerUnusedImagesProvider(deps.Docker, deps.DockerImageLedger))
	}
	if deps.OllamaModelProvider != nil {
		providers = append(providers, deps.OllamaModelProvider)
	}
	providers = append(providers, OwnerScenarioProviders(deps.OwnerProviderConfigs, deps.OwnerScenarioClient)...)
	for _, cfg := range deps.RuntimeHomeProviders {
		providers = append(providers, NewRuntimeHomeProvider(deps.FileSystem, deps.Clock, cfg))
	}

	for _, provider := range providers {
		if err := cleanup.ValidateProvider(provider); err != nil {
			return nil, err
		}
	}
	return providers, nil
}

func OwnerScenarioProviders(configs []OwnerProviderConfig, client cleanup.ScenarioProviderClient) []cleanup.Provider {
	out := make([]cleanup.Provider, 0, len(configs))
	for _, cfg := range configs {
		out = append(out, NewOwnerScenarioProvider(cfg, client))
	}
	return out
}

type BuiltInDeps struct {
	FileSystem           cleanup.FileSystem
	ProcessLiveness      cleanup.ProcessLiveness
	ProcessRunner        cleanup.ProcessRunner
	Docker               cleanup.DockerClient
	DockerImageLedger    imageUsageLedger
	OllamaModelProvider  cleanup.Provider
	Journal              cleanup.JournalClient
	OwnerScenarioClient  cleanup.ScenarioProviderClient
	OwnerProviderConfigs []OwnerProviderConfig
	Clock                cleanup.Clock
	TrashRoots           []string
	TmpRoots             []string
	ScratchRoots         []string
	GoBuildCacheRoots    []string
	PlaywrightCacheRoots []string
	ScenarioBinariesRoot string
	RuntimeHomeProviders []FileProviderConfig
	Saturated            func(context.Context) (bool, error)
	// RemovalLedger makes reclamations attributable. A built-in that deletes
	// install-root artifacts refuses to run without it.
	RemovalLedger *artifactledger.Ledger
}
