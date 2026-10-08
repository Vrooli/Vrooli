package main

import (
	"context"
	"fmt"
	"sync"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/storagehealth"

	coredb "github.com/vrooli/api-core/database"
)

// storageHealthRuntime wires Agent Manager's storage self-management: the
// measurement and reclaim service, its HTTP contract, and the pause of this
// process's own background writers that a fenced compaction requires.
type storageHealthRuntime struct {
	service *storagehealth.Service
	handler *storagehealth.Handler

	mu         sync.Mutex
	workersCtx context.Context
}

// buildStorageHealth must run after the maintenance handler exists: the fence
// observer and owner authorizer both come from it.
func (s *Server) buildStorageHealth(routed *coredb.RoutedDB, repoRoot string) error {
	path, err := database.SQLiteFilePath()
	if err != nil {
		return fmt.Errorf("resolve database file for storage health: %w", err)
	}
	runtime := &storageHealthRuntime{}
	service, err := storagehealth.New(storagehealth.Options{
		DB:     routed.Primary,
		Path:   path,
		Budget: storagehealth.DeclaredBudget(repoRoot, "agent-manager", "data", nil),
		Fence: func(ctx context.Context) (storagehealth.FenceState, error) {
			standing, err := s.maintenance.Observe(ctx)
			return storagehealth.FenceState{Closed: standing.Closed, Drained: standing.Drained, Revision: standing.Revision}, err
		},
		Pause:      runtime.pauser(s),
		Watermarks: s.rowidWatermarks(),
		Logger:     obs.Component("storage-health"),
	})
	if err != nil {
		return err
	}
	handler, err := storagehealth.NewHandler(service, s.maintenance.AuthorizeOwner)
	if err != nil {
		return err
	}
	runtime.service, runtime.handler = service, handler
	s.storageHealth = runtime
	if s.reconciler != nil {
		s.reconciler.SetStorageMaintainer(service)
	}
	return nil
}

// rowidWatermarks are the in-memory run_events rowid positions a VACUUM must
// carry. Persisted positions are rewritten by the compaction itself.
func (s *Server) rowidWatermarks() []storagehealth.NamedWatermark {
	if s.statsEngine == nil {
		return nil
	}
	return []storagehealth.NamedWatermark{{Name: "stats_engine.watermark", Holder: s.statsEngine}}
}

func (s *Server) storageHealthHandler() *storagehealth.Handler {
	if s.storageHealth == nil {
		return nil
	}
	return s.storageHealth.handler
}

// setWorkersContext records the context background workers run under so a
// compaction restarts them with the same lifetime.
func (r *storageHealthRuntime) setWorkersContext(ctx context.Context) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.workersCtx = ctx
	r.mu.Unlock()
}

func (r *storageHealthRuntime) workers() context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.workersCtx == nil {
		return context.Background()
	}
	return r.workersCtx
}

// pauser stops the writers that would otherwise wait on VACUUM's lock and the
// consumers whose run_events rowid positions VACUUM may renumber: the
// conversation indexer (refusing while a generation builds), supervision watch
// processing, the reconciler (and with it imported-payload compaction), and
// the transcript and friction schedulers. Admitted agent work is already
// drained by the fence.
func (r *storageHealthRuntime) pauser(s *Server) storagehealth.Pauser {
	return func(context.Context) (func(), error) {
		if s.conversationIndexer != nil {
			for _, job := range s.conversationIndexer.SortedJobs() {
				if job.State == conversationsearch.ReindexQueued || job.State == conversationsearch.ReindexRunning {
					return nil, fmt.Errorf("conversation index generation %s is %s; compact after it finishes", job.ID, job.State)
				}
			}
			s.conversationIndexer.Stop()
		}
		resumeSupervision := func() {}
		if s.supervisionScheduler != nil {
			resumeSupervision = s.supervisionScheduler.Pause()
		}
		if s.reconciler != nil {
			_ = s.reconciler.Stop()
		}
		if s.transcriptImporter != nil {
			s.transcriptImporter.Stop()
		}
		if s.frictionPublisher != nil {
			s.frictionPublisher.Stop()
		}
		return func() {
			resumeSupervision()
			ctx := r.workers()
			if s.conversationIndexer != nil {
				s.conversationIndexer.Start(ctx)
			}
			if s.reconciler != nil {
				if err := s.reconciler.Start(ctx); err != nil {
					obs.Component("storage-health").Warn("reconciler restart after compaction failed", "error", err.Error())
				}
			}
			if s.transcriptImporter != nil {
				s.transcriptImporter.Start(ctx)
			}
			if s.frictionPublisher != nil {
				s.frictionPublisher.Start(ctx)
			}
		}, nil
	}
}
