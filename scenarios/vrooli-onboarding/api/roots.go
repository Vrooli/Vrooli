package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/envx"
)

// Roots is the single environment-to-storage mapping used by the onboarding
// API. RepoRoot is the source tree, CatalogRoot is the immutable manifest
// catalog, and StateRoot is this scenario's contract-routed writable state
// directory. Repository metadata and runtime state are deliberately separate.
type Roots struct {
	RepoRoot    string
	StateRoot   string
	CatalogRoot string
	Source      string
}

var rootEnvironment envx.Reader = envx.OS{}

func resolveRoots() (Roots, error) {
	repo := strings.TrimSpace(rootEnvironment.Getenv("VROOLI_ROOT"))
	storageRoot := strings.TrimSpace(rootEnvironment.Getenv("VROOLI_STORAGE_ROOT"))
	bundle := strings.TrimSpace(rootEnvironment.Getenv("BUNDLE_ROOT"))
	roots := Roots{RepoRoot: repo}
	switch {
	case repo != "":
		roots.Source = "VROOLI_ROOT"
		roots.CatalogRoot = repo
	case storageRoot != "":
		roots.Source = "VROOLI_STORAGE_ROOT"
		if bundle != "" {
			roots.Source = "VROOLI_STORAGE_ROOT+BUNDLE_ROOT"
			roots.CatalogRoot = filepath.Join(bundle, "catalog")
		}
	case bundle != "":
		roots.Source = "BUNDLE_ROOT"
		roots.CatalogRoot = filepath.Join(bundle, "catalog")
	default:
		return Roots{}, fmt.Errorf("VROOLI_ROOT, VROOLI_STORAGE_ROOT, or BUNDLE_ROOT is required")
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
		EnvGet:  rootEnvironment.Getenv,
	})
	if err != nil {
		return Roots{}, fmt.Errorf("create storage resolver: %w", err)
	}
	options := storage.Options{ScenarioID: onboardingScenarioName}
	if storageRoot == "" && bundle != "" {
		options.RootOverride = filepath.Join(bundle, "app-data")
	}
	paths, err := resolver.Resolve(options)
	if err != nil {
		return Roots{}, fmt.Errorf("resolve onboarding state root: %w", err)
	}
	roots.StateRoot = paths.StateDir
	return roots, nil
}
