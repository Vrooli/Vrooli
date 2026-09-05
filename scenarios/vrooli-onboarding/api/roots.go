package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Roots is the single environment-to-storage mapping used by the onboarding
// API. RepoRoot is the source tree, CatalogRoot is the immutable manifest
// catalog, and StorageRoot is the writable application-data root.
type Roots struct {
	RepoRoot    string
	StorageRoot string
	CatalogRoot string
	Source      string
}

func resolveRoots() (Roots, error) {
	repo := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	storage := strings.TrimSpace(os.Getenv("VROOLI_STORAGE_ROOT"))
	bundle := strings.TrimSpace(os.Getenv("BUNDLE_ROOT"))
	roots := Roots{RepoRoot: repo}
	switch {
	case repo != "":
		roots.Source = "VROOLI_ROOT"
		roots.StorageRoot = storage
		if roots.StorageRoot == "" {
			roots.StorageRoot = filepath.Join(repo, ".vrooli")
		}
		roots.CatalogRoot = repo
	case storage != "":
		roots.Source = "VROOLI_STORAGE_ROOT"
		roots.StorageRoot = storage
		if bundle != "" {
			roots.Source = "VROOLI_STORAGE_ROOT+BUNDLE_ROOT"
			roots.CatalogRoot = filepath.Join(bundle, "catalog")
		}
	case bundle != "":
		roots.Source = "BUNDLE_ROOT"
		roots.StorageRoot = filepath.Join(bundle, "app-data")
		roots.CatalogRoot = filepath.Join(bundle, "catalog")
	default:
		return Roots{}, fmt.Errorf("VROOLI_ROOT, VROOLI_STORAGE_ROOT, or BUNDLE_ROOT is required")
	}
	return roots, nil
}
