package staleness

import (
	"context"
	"sort"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
)

// ManifestSource exposes the manifest-domain methods the staleness
// derivation needs. Declared here (consumer-owned) per
// seam-discovery-and-enforcement.
//
// seam: ManifestSource
type ManifestSource interface {
	List(ctx context.Context) ([]manifest.Manifest, error)
	GetStaleOverride(ctx context.Context, skillID, goldenSlug string) (clearedAt time.Time, err error)
}

// GoldenSource exposes the per-golden current template-version pin.
//
// seam: GoldenSource
type GoldenSource interface {
	CurrentTemplateVersion(ctx context.Context, goldenSlug string) (string, error)
}

// SkillSource exposes the per-skill current version pin (from the
// local skill_catalog mirror).
//
// seam: SkillSource
type SkillSource interface {
	CurrentSkillVersion(ctx context.Context, skillID string) (string, error)
}

// Service is the application-layer surface the staleness handler
// depends on.
type Service interface {
	ListStale(ctx context.Context) ([]Entry, error)
}

type service struct {
	manifests ManifestSource
	goldens   GoldenSource
	skills    SkillSource
}

// NewService constructs the production Service.
func NewService(m ManifestSource, g GoldenSource, s SkillSource) Service {
	return &service{manifests: m, goldens: g, skills: s}
}

var _ Service = (*service)(nil)

func (s *service) ListStale(ctx context.Context) ([]Entry, error) {
	rows, err := s.manifests.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, m := range rows {
		// If a manual ClearStale override exists and is at-or-after
		// the manifest's UpdatedAt, suppress this tuple's staleness.
		cleared, err := s.manifests.GetStaleOverride(ctx, m.SkillID, m.GoldenSlug)
		if err != nil {
			return nil, err
		}
		if !cleared.IsZero() && !cleared.Before(m.UpdatedAt) {
			continue
		}

		currentTemplate, err := s.goldens.CurrentTemplateVersion(ctx, m.GoldenSlug)
		if err != nil {
			// Missing golden / lookup error: skip silently — the
			// manifest may reference a golden that's not registered
			// yet. Don't fail the whole list.
			continue
		}
		currentSkill, err := s.skills.CurrentSkillVersion(ctx, m.SkillID)
		if err != nil {
			continue
		}

		templateDrift := m.TemplateVersionPinned != "" && currentTemplate != "" && m.TemplateVersionPinned != currentTemplate
		skillDrift := m.SkillVersionPinned != "" && currentSkill != "" && m.SkillVersionPinned != currentSkill

		if !templateDrift && !skillDrift {
			continue
		}
		kind := StaleKindUnspecified
		switch {
		case templateDrift && skillDrift:
			kind = StaleKindBoth
		case templateDrift:
			kind = StaleKindTemplateDrift
		case skillDrift:
			kind = StaleKindSkillDrift
		}
		out = append(out, Entry{
			SkillID:                       m.SkillID,
			GoldenSlug:                    m.GoldenSlug,
			Kind:                          kind,
			ManifestTemplateVersionPinned: m.TemplateVersionPinned,
			ManifestSkillVersionPinned:    m.SkillVersionPinned,
			GoldenTemplateVersionCurrent:  currentTemplate,
			SkillVersionCurrent:           currentSkill,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SkillID != out[j].SkillID {
			return out[i].SkillID < out[j].SkillID
		}
		return out[i].GoldenSlug < out[j].GoldenSlug
	})
	return out, nil
}
