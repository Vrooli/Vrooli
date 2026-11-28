package main

import (
	"sync"

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
