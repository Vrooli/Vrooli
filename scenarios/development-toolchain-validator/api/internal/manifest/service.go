package manifest

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// Service is the application-layer surface the manifest handlers depend
// on. Owns input validation and ClearStale orchestration. Read paths
// pass through to the repository.
type Service interface {
	List(ctx context.Context) ([]Manifest, error)
	Get(ctx context.Context, skillID, goldenSlug string) (Manifest, error)
	Upsert(ctx context.Context, in UpsertInput) (Manifest, error)
	ClearStale(ctx context.Context, skillID, goldenSlug string) (time.Time, error)
}

type service struct {
	repo  Repository
	clock schedule.Clock
}

// NewService constructs the production Service.
func NewService(repo Repository, clk schedule.Clock) Service {
	return &service{repo: repo, clock: clk}
}

var _ Service = (*service)(nil)

var (
	skillIDPattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)
	goldenSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
)

func (s *service) List(ctx context.Context) ([]Manifest, error) {
	return s.repo.List(ctx)
}

func (s *service) Get(ctx context.Context, skillID, goldenSlug string) (Manifest, error) {
	skillID = strings.TrimSpace(skillID)
	goldenSlug = strings.TrimSpace(goldenSlug)
	if skillID == "" {
		return Manifest{}, ErrInvalidManifest{Field: "skill_id", Reason: "required"}
	}
	if goldenSlug == "" {
		return Manifest{}, ErrInvalidManifest{Field: "golden_slug", Reason: "required"}
	}
	return s.repo.Get(ctx, skillID, goldenSlug)
}

func (s *service) Upsert(ctx context.Context, in UpsertInput) (Manifest, error) {
	in.SkillID = strings.TrimSpace(in.SkillID)
	in.GoldenSlug = strings.TrimSpace(in.GoldenSlug)
	if !skillIDPattern.MatchString(in.SkillID) {
		return Manifest{}, ErrInvalidManifest{Field: "skill_id", Reason: "must be slug-shape"}
	}
	if !goldenSlugPattern.MatchString(in.GoldenSlug) {
		return Manifest{}, ErrInvalidManifest{Field: "golden_slug", Reason: "must be lowercase slug-shape"}
	}
	if !in.WildcardAllowed && len(in.AllowedPaths) == 0 && len(in.ContentRules) == 0 {
		return Manifest{}, ErrInvalidManifest{
			Field:  "allowed_paths",
			Reason: "must declare at least one allowed path or content rule when wildcard_allowed is false",
		}
	}
	for i, p := range in.AllowedPaths {
		if strings.TrimSpace(p) == "" {
			return Manifest{}, ErrInvalidManifest{
				Field:  "allowed_paths",
				Reason: "entry at position " + itoa(i) + " is blank",
			}
		}
	}
	for i, rule := range in.ContentRules {
		if strings.TrimSpace(rule.PathGlob) == "" {
			return Manifest{}, ErrInvalidManifest{
				Field:  "content_rules",
				Reason: "rule at position " + itoa(i) + " has blank path_glob",
			}
		}
	}
	if in.ConvergenceTarget == ConvergenceTargetUnspecified {
		in.ConvergenceTarget = ConvergenceTargetNone
	}
	m := Manifest{
		SkillID:               in.SkillID,
		GoldenSlug:            in.GoldenSlug,
		AllowedPaths:          in.AllowedPaths,
		ContentRules:          in.ContentRules,
		WildcardAllowed:       in.WildcardAllowed,
		ConvergenceTarget:     in.ConvergenceTarget,
		TemplateVersionPinned: in.TemplateVersionPinned,
		SkillVersionPinned:    in.SkillVersionPinned,
		UpdatedAt:             s.clock.Now().UTC(),
	}
	return s.repo.Upsert(ctx, m)
}

func (s *service) ClearStale(ctx context.Context, skillID, goldenSlug string) (time.Time, error) {
	skillID = strings.TrimSpace(skillID)
	goldenSlug = strings.TrimSpace(goldenSlug)
	if skillID == "" {
		return time.Time{}, ErrInvalidManifest{Field: "skill_id", Reason: "required"}
	}
	if goldenSlug == "" {
		return time.Time{}, ErrInvalidManifest{Field: "golden_slug", Reason: "required"}
	}
	now := s.clock.Now().UTC()
	if err := s.repo.ClearStaleOverride(ctx, skillID, goldenSlug, now); err != nil {
		return time.Time{}, err
	}
	return now, nil
}

// itoa is an allocation-free int-to-string for small positive ints
// used only in error messages.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
