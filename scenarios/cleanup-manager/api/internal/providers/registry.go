package providers

import (
	"fmt"
	"sort"

	"cleanup-manager/internal/cleanup"
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
		NewTrashProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "trash",
			Name:        "Trash",
			Roots:       deps.TrashRoots,
			Description: "Remove aged XDG trash entries",
		}),
		NewTmpProvider(deps.FileSystem, deps.Clock, FileProviderConfig{
			ID:          "tmp",
			Name:        "Temporary files",
			Roots:       deps.TmpRoots,
			Description: "Remove aged temporary files below configured roots",
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

	for _, provider := range providers {
		if err := cleanup.ValidateProvider(provider); err != nil {
			return nil, err
		}
	}
	return providers, nil
}

type BuiltInDeps struct {
	FileSystem           cleanup.FileSystem
	ProcessRunner        cleanup.ProcessRunner
	Docker               cleanup.DockerClient
	Journal              cleanup.JournalClient
	Clock                cleanup.Clock
	TrashRoots           []string
	TmpRoots             []string
	GoBuildCacheRoots    []string
	PlaywrightCacheRoots []string
}
