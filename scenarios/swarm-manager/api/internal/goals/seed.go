package goals

import (
	"fmt"
	"sort"
	"strings"
)

// SeedSpec maps a de-facto goal tag to a fully specified seeded goal. A seed
// must carry the same explicit description and priority contract as an
// operator-authored goal; silent defaults would turn an omitted value into a
// ranked value and make the seed's intent unrecoverable.
type SeedSpec struct {
	Tag         string
	Name        string
	Title       string
	Description string
	Priority    *int
}

// DefaultSeedSpecs are the four de-facto goals that already exist as tags. Each
// tagged item becomes a target; the transitive closure pulls in prerequisites,
// and initiative membership surfaces the owning initiatives in scope.
var DefaultSeedSpecs = []SeedSpec{
	{Tag: "desktop-deploy-v1", Name: "desktop-deploy-v1", Title: "Desktop Deploy v1", Description: "Deliver signed, installable desktop builds across supported operating systems.", Priority: seedPriority(5)},
	{Tag: "monetization-v1", Name: "monetization-v1", Title: "Monetization v1", Description: "Establish the initial monetization capability and its operating guardrails.", Priority: seedPriority(5)},
	{Tag: "audio-reliability-v1", Name: "audio-reliability-v1", Title: "Audio Reliability v1", Description: "Improve reliability of the audio interaction and runtime surface.", Priority: seedPriority(5)},
	{Tag: "self-host-v1", Name: "self-host-v1", Title: "Self-Host v1", Description: "Make self-hosted Vrooli operation reliable and documented.", Priority: seedPriority(5)},
}

func seedPriority(value int) *int { return &value }

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
		if strings.TrimSpace(spec.Description) == "" {
			return created, fmt.Errorf("seed goal %q: description is required", spec.Name)
		}
		if spec.Priority == nil {
			return created, fmt.Errorf("seed goal %q: explicit priority is required", spec.Name)
		}
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
			Name:        spec.Name,
			Title:       spec.Title,
			Description: strings.TrimSpace(spec.Description),
			Priority:    *spec.Priority,
			Targets:     targets,
			Seeded:      true,
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
