package supervision

// [REQ:REQ-P2-010]

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runsignal"
	"context"
	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"testing"
	"time"
)

type supervisionRunRepo struct {
	repository.RunRepository
	run *domain.Run
}

func (r supervisionRunRepo) Get(context.Context, uuid.UUID) (*domain.Run, error) { return r.run, nil }

type frictionFixture struct {
	filter    invocationreadmodel.Filter
	limit     int
	available bool
	through   time.Time
}

func (f *frictionFixture) Watermark(context.Context, string) (*invocationreadmodel.Watermark, error) {
	if !f.available {
		return nil, nil
	}
	return &invocationreadmodel.Watermark{EpisodeClassifierVersion: runsignal.EpisodeClassifierVersion, LastEventAt: f.through}, nil
}
func (f *frictionFixture) Episodes(_ context.Context, filter invocationreadmodel.Filter, limit int) ([]invocationreadmodel.Episode, error) {
	f.filter = filter
	f.limit = limit
	return []invocationreadmodel.Episode{{FrictionEpisode: runsignal.FrictionEpisode{EpisodeID: "episode", Severity: "high", Pattern: "repeated-validation", Fingerprint: "same", SuspectedOwnerScenario: "test-genie"}}}, nil
}
func TestProductionSubjectResolutionCarriesBoundedAttributedFriction(t *testing.T) {
	id := uuid.New()
	now := time.Now().UTC()
	source := &frictionFixture{available: true, through: now}
	resolver := NewRunSubjectResolver(supervisionRunRepo{run: &domain.Run{ID: id, Status: domain.RunStatusRunning}}, source)
	resolver.now = func() time.Time { return now }
	result, err := resolver.Resolve(context.Background(), []*domainpb.WatchSubject{{RunId: id.String()}})
	if err != nil || len(result) != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if source.filter.RunID != id.String() || source.limit != 4 || !source.filter.From.Equal(now.Add(-15*time.Minute)) {
		t.Fatalf("unbounded/wrong-subject query: %+v limit=%d", source.filter, source.limit)
	}
	if result[0].FrictionUnavailable || len(result[0].Friction) != 1 || result[0].Friction[0].Owner != "test-genie" || result[0].FrictionScore != .9 {
		t.Fatalf("lost evidence: %+v", result)
	}
	source.available = false
	result, err = resolver.Resolve(context.Background(), []*domainpb.WatchSubject{{RunId: id.String()}})
	if err != nil || !result[0].FrictionUnavailable {
		t.Fatalf("missing projection treated as zero: %+v %v", result, err)
	}
}
