package dbdetect

import (
	"context"
	"strings"
)

// ManifestCollector emits one observation per declared resource type
// in the service manifest. SQLite is not declared as a manifest resource
// (it is library-only), so the sqlite profile does not consume this collector.
type ManifestCollector struct{}

func (ManifestCollector) Name() string { return "manifest" }

func (ManifestCollector) Collect(_ context.Context, in ScenarioInputs) ([]Observation, error) {
	if in.Manifest == nil {
		return nil, nil
	}
	var out []Observation
	for _, r := range in.Manifest.Resources() {
		if !r.Required && !r.Enabled {
			continue
		}
		t := strings.ToLower(strings.TrimSpace(r.Type))
		if t == "" {
			continue
		}
		out = append(out, Observation{
			Collector: "manifest",
			Value:     t,
			Count:     1,
			Locations: []string{".vrooli/service.json"},
		})
	}
	return out, nil
}
