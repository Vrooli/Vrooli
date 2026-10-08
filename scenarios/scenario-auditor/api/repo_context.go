package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/api-core/storage"
	"scenario-auditor/internal/repocontext"
)

var (
	appRepoContextMu sync.RWMutex
	appRepoContext   *repocontext.Context
)

func initRepoContext() (*repocontext.Context, error) {
	ctx, err := repocontext.FromEnvOrCWD()
	if err != nil {
		return nil, err
	}
	setRepoContext(ctx)
	return ctx, nil
}

func repoContext() (*repocontext.Context, error) {
	appRepoContextMu.RLock()
	ctx := appRepoContext
	appRepoContextMu.RUnlock()
	if ctx != nil {
		return ctx, nil
	}
	return initRepoContext()
}

func setRepoContext(ctx *repocontext.Context) {
	appRepoContextMu.Lock()
	defer appRepoContextMu.Unlock()
	appRepoContext = ctx
}

func clearRepoContext() {
	appRepoContextMu.Lock()
	defer appRepoContextMu.Unlock()
	appRepoContext = nil
}

func resolveScenarioAuditorDataDir() (string, error) {
	ctx, err := repoContext()
	if err != nil {
		return "", err
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	dataDir, err := storage.EnsureClassDir(
		resolver,
		storage.Options{ScenarioID: "scenario-auditor"},
		storage.ClassData,
		0o755,
	)
	if err != nil {
		return "", fmt.Errorf("ensure scenario-auditor data dir: %w", err)
	}

	legacy := filepath.Join(ctx.RepoRoot(), ".vrooli", "data", "scenario-auditor")
	if _, statErr := os.Stat(legacy); statErr == nil {
		if _, dstErr := os.Stat(dataDir); os.IsNotExist(dstErr) {
			if err := os.Rename(legacy, dataDir); err != nil {
				return "", fmt.Errorf("migrate legacy scenario-auditor data dir: %w", err)
			}
		}
	}

	return dataDir, nil
}
