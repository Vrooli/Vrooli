package autosteer

import "github.com/ecosystem-manager/api/pkg/skillmap"

// PromptLoaderCatalog adapts the prompt-manager-backed PromptLoader to the
// skillmap.CatalogSource seam: it surfaces each cached skill's declared target
// dimensions so the controller can build its capability map.
type PromptLoaderCatalog struct {
	loader *PromptLoader
}

var _ skillmap.CatalogSource = (*PromptLoaderCatalog)(nil)

// NewPromptLoaderCatalog wraps a PromptLoader as a skill catalog source.
func NewPromptLoaderCatalog(loader *PromptLoader) *PromptLoaderCatalog {
	return &PromptLoaderCatalog{loader: loader}
}

// Skills returns one declaration per cached skill that declares any target
// dimensions. Validation against the vocabulary happens in skillmap.
func (c *PromptLoaderCatalog) Skills() []skillmap.SkillDeclaration {
	if c == nil || c.loader == nil {
		return nil
	}
	raw := c.loader.GetCachedSkills()
	out := make([]skillmap.SkillDeclaration, 0, len(raw))
	for _, p := range raw {
		if len(p.TargetDimensions) == 0 {
			continue
		}
		out = append(out, skillmap.SkillDeclaration{
			ID:         p.ID,
			Dimensions: p.TargetDimensions,
		})
	}
	return out
}
