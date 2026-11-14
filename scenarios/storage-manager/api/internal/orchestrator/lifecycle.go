package orchestrator

import (
	"context"
	"errors"
	"sort"

	"github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var ErrLifecycleQuiesced = errors.New("storage orchestrator is quiesced")

// beginLifecycleWork couples admission and ownership registration. A caller
// admitted before a fence is visible even while preparing its detached worker.
func (s *Service) beginLifecycleWork(id, kind string) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.lifecycleClosed {
		return ErrLifecycleQuiesced
	}
	s.lifecycleWork[id] = kind
	return nil
}

func (s *Service) endLifecycleWork(id string) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	delete(s.lifecycleWork, id)
	close(s.lifecycleChanged)
	s.lifecycleChanged = make(chan struct{})
}

// Snapshot reports only physical work owned by this process, never historical
// recovery records imported from the durable ledger.
func (s *Service) Snapshot(ctx context.Context) (lifecycle.Inventory, error) {
	if err := ctx.Err(); err != nil {
		return lifecycle.Inventory{Remaining: -1}, err
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	result := lifecycle.Inventory{Complete: true, Remaining: int32(len(s.lifecycleWork))}
	ids := make([]string, 0, len(s.lifecycleWork))
	for id := range s.lifecycleWork {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		result.Work = append(result.Work, &commonv1.LifecycleWorkItem{Id: id, Kind: s.lifecycleWork[id], Status: "running", PhysicalExecutorPresent: true})
	}
	return result, nil
}

// Quiesce closes admission before waiting; a cancelled wait keeps the fence
// closed until Resume, so a failed preparation cannot start untracked work.
func (s *Service) Quiesce(ctx context.Context) error {
	s.lifecycleMu.Lock()
	s.lifecycleClosed = true
	for len(s.lifecycleWork) != 0 {
		changed := s.lifecycleChanged
		s.lifecycleMu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
		s.lifecycleMu.Lock()
		if !s.lifecycleClosed {
			s.lifecycleMu.Unlock()
			return errors.New("storage orchestrator resumed while quiescing")
		}
	}
	s.lifecycleMu.Unlock()
	return nil
}

func (s *Service) Resume(context.Context) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.lifecycleClosed = false
	close(s.lifecycleChanged)
	s.lifecycleChanged = make(chan struct{})
	return nil
}
