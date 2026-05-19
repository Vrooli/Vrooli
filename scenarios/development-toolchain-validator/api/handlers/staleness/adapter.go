package staleness

import (
	stalenessdom "development-toolchain-validator/internal/staleness"

	stalenessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness"
)

func domainToProto(e stalenessdom.Entry) *stalenessv1.StaleEntry {
	return &stalenessv1.StaleEntry{
		SkillId:                       e.SkillID,
		GoldenSlug:                    e.GoldenSlug,
		Kind:                          kindDomainToProto(e.Kind),
		ManifestTemplateVersionPinned: e.ManifestTemplateVersionPinned,
		ManifestSkillVersionPinned:    e.ManifestSkillVersionPinned,
		GoldenTemplateVersionCurrent:  e.GoldenTemplateVersionCurrent,
		SkillVersionCurrent:           e.SkillVersionCurrent,
	}
}

func kindDomainToProto(k stalenessdom.StaleKind) stalenessv1.StaleKind {
	switch k {
	case stalenessdom.StaleKindTemplateDrift:
		return stalenessv1.StaleKind_STALE_KIND_TEMPLATE_DRIFT
	case stalenessdom.StaleKindSkillDrift:
		return stalenessv1.StaleKind_STALE_KIND_SKILL_DRIFT
	case stalenessdom.StaleKindBoth:
		return stalenessv1.StaleKind_STALE_KIND_BOTH
	default:
		return stalenessv1.StaleKind_STALE_KIND_UNSPECIFIED
	}
}
