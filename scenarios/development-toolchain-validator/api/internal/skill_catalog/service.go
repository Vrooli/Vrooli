package skill_catalog

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/api-core/schedule"
)

// SkillCatalogSource is the outbound seam that fetches the canonical
// skill catalog from prompt-manager. Production wires the REST adapter
// from api/integrations/prompt_manager/; tests substitute fakes from
// mocks/.
//
// seam: SkillCatalogSource
type SkillCatalogSource interface {
	// Fetch returns the current skill set from the upstream catalog
	// (prompt-manager). Implementations are responsible for transport
	// retries / re-resolution; this method either returns a populated
	// list or an error.
	Fetch(ctx context.Context) ([]Skill, error)
}

// Service is the application-layer surface the skill_catalog handlers
// depend on. Owns input validation, Sync orchestration (Fetch →
// reconcile), and any cross-handler policy.
type Service interface {
	Sync(ctx context.Context) (SyncResult, error)
	List(ctx context.Context) ([]Skill, error)
	Get(ctx context.Context, id string) (Skill, error)
}

// SyncResult is the explicit output DTO Service.Sync returns. Used by
// the handler to populate SyncResponse counts.
type SyncResult struct {
	Skills  []Skill
	Added   int
	Updated int
	Removed int
}

type service struct {
	repo   Repository
	source SkillCatalogSource
	clock  schedule.Clock
}

// NewService constructs the production Service.
func NewService(repo Repository, source SkillCatalogSource, clk schedule.Clock) Service {
	return &service{repo: repo, source: source, clock: clk}
}

var _ Service = (*service)(nil)

// idPattern enforces a conservative kebab-case shape for skill ids.
// Mirrors prompt-manager's slug convention.
var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func (s *service) Sync(ctx context.Context) (SyncResult, error) {
	if s.source == nil {
		return SyncResult{}, ErrSyncFailed{Reason: "no upstream source configured"}
	}
	upstream, err := s.source.Fetch(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	now := s.clock.Now().UTC()
	keep := make([]string, 0, len(upstream))
	added := 0
	updated := 0
	for i := range upstream {
		sk := upstream[i]
		sk.ID = strings.TrimSpace(sk.ID)
		if !idPattern.MatchString(sk.ID) {
			// Skip malformed upstream entries rather than fail the whole
			// sync; record but don't count.
			continue
		}
		sk.SyncedAt = now
		ins, changed, err := s.repo.Upsert(ctx, sk)
		if err != nil {
			return SyncResult{}, err
		}
		if ins {
			added++
		} else if changed {
			updated++
		}
		keep = append(keep, sk.ID)
	}
	removed, err := s.repo.DeleteMissing(ctx, keep)
	if err != nil {
		return SyncResult{}, err
	}
	after, err := s.repo.List(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	// repository already orders by id; defensive sort for tests that may
	// substitute non-ordering fakes.
	sort.Slice(after, func(i, j int) bool { return after[i].ID < after[j].ID })
	return SyncResult{Skills: after, Added: added, Updated: updated, Removed: removed}, nil
}

func (s *service) List(ctx context.Context) ([]Skill, error) {
	return s.repo.List(ctx)
}

func (s *service) Get(ctx context.Context, id string) (Skill, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Skill{}, ErrInvalidSkill{Field: "id", Reason: "required"}
	}
	return s.repo.Get(ctx, id)
}
