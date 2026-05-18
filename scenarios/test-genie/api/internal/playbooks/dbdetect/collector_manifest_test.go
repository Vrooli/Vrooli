package dbdetect_test

import (
	"context"
	"testing"

	"test-genie/internal/playbooks/dbdetect"
	"test-genie/internal/playbooks/dbdetect/mocks"
)

func TestManifestCollector(t *testing.T) {
	cases := []struct {
		name      string
		resources []dbdetect.ManifestResource
		want      []string
	}{
		{
			name: "postgres only",
			resources: []dbdetect.ManifestResource{
				{Key: "pg", Type: "postgres", Required: true, Enabled: true},
			},
			want: []string{"postgres"},
		},
		{
			name: "postgres and redis",
			resources: []dbdetect.ManifestResource{
				{Key: "pg", Type: "postgres", Required: true},
				{Key: "rd", Type: "redis", Enabled: true},
			},
			want: []string{"postgres", "redis"},
		},
		{
			name:      "no resources",
			resources: nil,
			want:      nil,
		},
		{
			name: "disabled and not required is skipped",
			resources: []dbdetect.ManifestResource{
				{Key: "pg", Type: "postgres", Required: false, Enabled: false},
			},
			want: nil,
		},
		{
			name: "empty type ignored",
			resources: []dbdetect.ManifestResource{
				{Key: "x", Type: "  ", Required: true},
			},
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &mocks.FakeManifest{ResourcesList: tc.resources}
			obs, err := dbdetect.ManifestCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{Manifest: m})
			if err != nil {
				t.Fatalf("collect: %v", err)
			}
			values := make([]string, 0, len(obs))
			for _, o := range obs {
				values = append(values, o.Value)
			}
			if !equalUnordered(values, tc.want) {
				t.Fatalf("got %v want %v", values, tc.want)
			}
		})
	}
}

func TestManifestCollectorNilManifest(t *testing.T) {
	obs, err := dbdetect.ManifestCollector{}.Collect(context.Background(), dbdetect.ScenarioInputs{})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(obs) != 0 {
		t.Fatalf("expected no observations, got %v", obs)
	}
}

func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[string]int{}
	for _, x := range a {
		counts[x]++
	}
	for _, x := range b {
		counts[x]--
	}
	for _, v := range counts {
		if v != 0 {
			return false
		}
	}
	return true
}
