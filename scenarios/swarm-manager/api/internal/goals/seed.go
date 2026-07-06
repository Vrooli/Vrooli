package goals

import (
	"fmt"
	"sort"
)

// SeedSpec maps a de-facto goal tag to a seeded goal's name/title.
type SeedSpec struct {
	Tag   string
	Name  string
	Title string
}

// DefaultSeedSpecs are the four de-facto goals that already exist as tags. Each
// tagged item becomes a target; the transitive closure pulls in prerequisites,
// and initiative membership surfaces the owning initiatives in scope.
var DefaultSeedSpecs = []SeedSpec{
	{Tag: "desktop-deploy-v1", Name: "desktop-deploy-v1", Title: "Desktop Deploy v1"},
	{Tag: "monetization-v1", Name: "monetization-v1", Title: "Monetization v1"},
	{Tag: "audio-reliability-v1", Name: "audio-reliability-v1", Title: "Audio Reliability v1"},
	{Tag: "self-host-v1", Name: "self-host-v1", Title: "Self-Host v1"},
}

// SeedFromTags creates a goal for each spec whose tag matches at least one
// backlog item, using the tagged items as targets. It is idempotent: existing
// goals and tags with no matching items are skipped. Returns the count created.
func (s *Service) SeedFromTags(specs []SeedSpec) (int, error) {
	items, err := s.backlog.LoadAll(nil)
	if err != nil {
		return 0, fmt.Errorf("seed goals: load backlog: %w", err)
	}
	created := 0
	for _, spec := range specs {
		if s.store.Exists(spec.Name) {
			continue
		}
		var targets []string
		for _, it := range items {
			if hasTag(it.Tags, spec.Tag) {
				targets = append(targets, string(it.Kind)+"/"+it.Name)
			}
		}
		if len(targets) == 0 {
			continue
		}
		sort.Strings(targets)
		if _, err := s.Create(CreateRequest{
			Name:    spec.Name,
			Title:   spec.Title,
			Targets: targets,
			Seeded:  true,
		}); err != nil {
			return created, fmt.Errorf("seed goal %q: %w", spec.Name, err)
		}
		created++
	}
	return created, nil
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
