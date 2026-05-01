package mocks

import (
	"sync"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// FakeBroadcaster records event broadcasts behind a lock so executor tests can
// inspect async status/event fanout without racing run mutations.
type FakeBroadcaster struct {
	mu sync.Mutex

	statusBroadcasts   []*domain.Run
	eventBroadcasts    []*domain.RunEvent
	progressBroadcasts []ProgressBroadcast
}

type ProgressBroadcast struct {
	RunID   uuid.UUID
	Phase   domain.RunPhase
	Percent int
	Action  string
}

func NewFakeBroadcaster() *FakeBroadcaster {
	return &FakeBroadcaster{}
}

func (b *FakeBroadcaster) BroadcastEvent(event *domain.RunEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventBroadcasts = append(b.eventBroadcasts, event)
}

func (b *FakeBroadcaster) BroadcastRunStatus(run *domain.Run) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if run == nil {
		b.statusBroadcasts = append(b.statusBroadcasts, nil)
		return
	}
	snapshot := *run
	b.statusBroadcasts = append(b.statusBroadcasts, &snapshot)
}

func (b *FakeBroadcaster) BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.progressBroadcasts = append(b.progressBroadcasts, ProgressBroadcast{
		RunID:   runID,
		Phase:   phase,
		Percent: percent,
		Action:  action,
	})
}

func (b *FakeBroadcaster) StatusBroadcasts() []*domain.Run {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]*domain.Run(nil), b.statusBroadcasts...)
}

func (b *FakeBroadcaster) EventBroadcasts() []*domain.RunEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]*domain.RunEvent(nil), b.eventBroadcasts...)
}

func (b *FakeBroadcaster) ProgressBroadcasts() []ProgressBroadcast {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]ProgressBroadcast(nil), b.progressBroadcasts...)
}
