package wiring

import (
	"context"
	"os"

	"agent-manager/internal/config"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/runstate"
	"agent-manager/internal/transcriptindex"
)

// newTranscriptIndex builds the Transcript Indexer over the orchestrator. Its
// roots are the operator's home harness directories, the run-state root's run
// homes and the configured extra roots.
func newTranscriptIndex(orch *orchestration.Orchestrator, store transcriptindex.Store, runStates runstate.RootResolver, levers *config.Levers) *orchestration.TranscriptIndex {
	home, err := os.UserHomeDir()
	if err != nil {
		obs.Component("transcript-index").Warn("home directory unavailable; only configured roots are indexed", obs.KeyError, err.Error())
	}
	runStateRoot := ""
	if runStates != nil {
		if root, err := runStates.Resolve(context.Background()); err == nil {
			runStateRoot = root
		}
	}
	return orchestration.NewTranscriptIndex(orch, store, transcriptindex.Config{
		Roots:          transcriptindex.RootConfig{Home: home, RunStateRoot: runStateRoot, ExtraRoots: levers.TranscriptImport.ScanRoots},
		Interval:       levers.TranscriptImport.Interval,
		RecallMaxBytes: levers.Storage.ImportedCorpusMaxBytes,
	})
}
