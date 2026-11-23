package wiring

import (
	"context"
	"os"
	"path/filepath"

	"agent-manager/internal/adapters/runner/opencodearchive"
	"agent-manager/internal/config"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/runstate"
	"agent-manager/internal/transcriptindex"

	corestorage "github.com/vrooli/api-core/storage"
)

// newTranscriptIndex builds the Transcript Indexer over the orchestrator. Its
// roots are the operator's home harness directories, web-console's session
// Codex homes, the run-state root's run homes and the configured extra roots.
func newTranscriptIndex(orch *orchestration.Orchestrator, store transcriptindex.Store, runStates runstate.RootResolver, levers *config.Levers) *orchestration.TranscriptIndex {
	home, err := os.UserHomeDir()
	if err != nil {
		obs.Component("transcript-index").Warn("home directory unavailable; only configured roots are indexed", obs.KeyError, err.Error())
	}
	webConsoleCodexRoot, err := transcriptindex.ResolveWebConsoleCodexRoot(corestorage.ResolverConfig{})
	if err != nil {
		obs.Component("transcript-index").Warn("web-console session homes unavailable; they are not indexed", obs.KeyError, err.Error())
	}
	runStateRoot := ""
	if runStates != nil {
		if root, err := runStates.Resolve(context.Background()); err == nil {
			runStateRoot = root
		}
	}
	openCodeArchiveDB := opencodearchive.DatabasePathForEnv(nil)
	cacheRoot, cacheErr := os.UserCacheDir()
	if cacheErr != nil {
		obs.Component("transcript-index").Warn("OpenCode archive cache unavailable", obs.KeyError, cacheErr.Error())
		cacheRoot = ""
	} else {
		cacheRoot = opencodearchive.CacheDirectory(openCodeArchiveDB, filepath.Join(cacheRoot, "agent-manager", "opencode-archives"))
	}
	return orchestration.NewTranscriptIndex(orch, store, transcriptindex.Config{
		Roots: transcriptindex.RootConfig{
			Home:                home,
			CursorStateRoot:     cursorStateRoot(home),
			GrokSessionsRoot:    grokSessionsRoot(os.Getenv("GROK_HOME")),
			WebConsoleCodexRoot: webConsoleCodexRoot,
			RunStateRoot:        runStateRoot,
			OpenCodeArchiveDB:   openCodeArchiveDB,
			OpenCodeArchiveRoot: cacheRoot,
			ExtraRoots:          levers.TranscriptImport.ScanRoots,
		},
		Interval:       levers.TranscriptImport.Interval,
		RecallMaxBytes: levers.Storage.ImportedCorpusMaxBytes,
	})
}

func cursorStateRoot(home string) string {
	for _, candidate := range []string{
		filepath.Join(home, ".config", "Cursor", "User", "workspaceStorage"),
		filepath.Join(home, ".cursor", "User", "workspaceStorage"),
	} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join(home, ".config", "Cursor", "User", "workspaceStorage")
}

func grokSessionsRoot(grokHome string) string {
	if grokHome == "" {
		return ""
	}
	return filepath.Join(grokHome, "sessions")
}
