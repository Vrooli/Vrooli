package staleness_test

import (
	"context"
	"testing"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	staleness "development-toolchain-validator/internal/staleness"
	smocks "development-toolchain-validator/internal/staleness/mocks"

	"github.com/stretchr/testify/require"
)

func newSvc(manifests []manifest.Manifest, goldenVersions, skillVersions map[string]string, overrides map[[2]string]time.Time) staleness.Service {
	return staleness.NewService(
		&smocks.FakeManifestSource{Manifests: manifests, Overrides: overrides},
		&smocks.FakeGoldenSource{Versions: goldenVersions},
		&smocks.FakeSkillSource{Versions: skillVersions},
	)
}

func sampleManifest(skillID, goldenSlug, templatePin, skillPin string, updatedAt time.Time) manifest.Manifest {
	return manifest.Manifest{
		SkillID: skillID, GoldenSlug: goldenSlug,
		TemplateVersionPinned: templatePin, SkillVersionPinned: skillPin,
		WildcardAllowed: true,
		UpdatedAt:       updatedAt,
	}
}

func TestListStale_TemplateDrift(t *testing.T) {
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "1.0.0", "v1", time.Now().UTC())},
		map[string]string{"ref-vite": "1.1.0"},
		map[string]string{"plan-skill": "v1"},
		nil,
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, staleness.StaleKindTemplateDrift, got[0].Kind)
}

func TestListStale_SkillDrift(t *testing.T) {
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "1.0.0", "v1", time.Now().UTC())},
		map[string]string{"ref-vite": "1.0.0"},
		map[string]string{"plan-skill": "v2"},
		nil,
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, staleness.StaleKindSkillDrift, got[0].Kind)
}

func TestListStale_BothDrift(t *testing.T) {
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "1.0.0", "v1", time.Now().UTC())},
		map[string]string{"ref-vite": "1.1.0"},
		map[string]string{"plan-skill": "v2"},
		nil,
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, staleness.StaleKindBoth, got[0].Kind)
}

func TestListStale_NoDriftReturnsEmpty(t *testing.T) {
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "1.0.0", "v1", time.Now().UTC())},
		map[string]string{"ref-vite": "1.0.0"},
		map[string]string{"plan-skill": "v1"},
		nil,
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestListStale_ManualOverrideSuppresses(t *testing.T) {
	manifestUpdated := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	cleared := time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC) // after manifest update
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "1.0.0", "v1", manifestUpdated)},
		map[string]string{"ref-vite": "1.1.0"},
		map[string]string{"plan-skill": "v1"},
		map[[2]string]time.Time{{"plan-skill", "ref-vite"}: cleared},
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Empty(t, got, "manual clear after manifest update suppresses staleness")
}

func TestListStale_OverrideBeforeManifestUpdateIgnored(t *testing.T) {
	manifestUpdated := time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)
	cleared := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC) // before manifest update
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "1.0.0", "v1", manifestUpdated)},
		map[string]string{"ref-vite": "1.1.0"},
		map[string]string{"plan-skill": "v1"},
		map[[2]string]time.Time{{"plan-skill", "ref-vite"}: cleared},
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1, "stale override predating the manifest must be ignored")
}

func TestListStale_EmptyPinValueSkipped(t *testing.T) {
	svc := newSvc(
		[]manifest.Manifest{sampleManifest("plan-skill", "ref-vite", "", "v1", time.Now().UTC())},
		map[string]string{"ref-vite": "anything"},
		map[string]string{"plan-skill": "v1"},
		nil,
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Empty(t, got, "blank pin means no expectation, so no drift")
}

func TestListStale_OrderedBySkillAndGolden(t *testing.T) {
	svc := newSvc(
		[]manifest.Manifest{
			sampleManifest("z", "z", "1", "v1", time.Now().UTC()),
			sampleManifest("a", "z", "1", "v1", time.Now().UTC()),
			sampleManifest("a", "a", "1", "v1", time.Now().UTC()),
		},
		map[string]string{"a": "2", "z": "2"},
		map[string]string{"a": "v1", "z": "v1"},
		nil,
	)
	got, err := svc.ListStale(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "a", got[0].SkillID)
	require.Equal(t, "a", got[0].GoldenSlug)
}
