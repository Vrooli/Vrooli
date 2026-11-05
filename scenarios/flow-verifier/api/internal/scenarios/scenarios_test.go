package scenarios_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/scenarios"

	"github.com/stretchr/testify/require"
)

// fakeFlowLister returns a canned answer per scenario root so tests
// can drive flow-count and discovery-error branches without writing
// real flow.json files.
type fakeFlowLister struct {
	byPath map[string]flowResult
}

type flowResult struct {
	rows []flows.Summary
	err  error
}

func (f *fakeFlowLister) List(root string) ([]flows.Summary, error) {
	r, ok := f.byPath[root]
	if !ok {
		return nil, nil
	}
	return r.rows, r.err
}

// writeVrooliTree builds a minimal Vrooli root under tmp with the
// requested scenario directories so ResolveVrooliRoot + Service.List
// have something real to walk.
func writeVrooliTree(t *testing.T, scenariosByID map[string]string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios"), 0o755))
	for id, displayName := range scenariosByID {
		dir := filepath.Join(root, "scenarios", id)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755))
		body := `{"service":{"displayName":"` + displayName + `","description":"desc for ` + id + `"}}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(body), 0o644))
	}
	return root
}

func TestResolveVrooliRoot_EnvOverride(t *testing.T) {
	root := writeVrooliTree(t, nil)
	got, err := scenarios.ResolveVrooliRoot("/tmp", func(k string) string {
		if k == scenarios.EnvVrooliRoot {
			return root
		}
		return ""
	})
	require.NoError(t, err)
	require.Equal(t, root, got)
}

func TestResolveVrooliRoot_RejectsNonVrooliEnv(t *testing.T) {
	bad := t.TempDir()
	_, err := scenarios.ResolveVrooliRoot("/tmp", func(k string) string {
		if k == scenarios.EnvVrooliRoot {
			return bad
		}
		return ""
	})
	require.Error(t, err)
}

func TestResolveVrooliRoot_WalkUp(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{"foo": "Foo"})
	deep := filepath.Join(root, "scenarios", "foo", "api", "internal")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	got, err := scenarios.ResolveVrooliRoot(deep, func(string) string { return "" })
	require.NoError(t, err)
	require.Equal(t, root, got)
}

func TestResolveVrooliRoot_NotFound(t *testing.T) {
	// /tmp/<random> has no scenarios/ or .vrooli/ ancestor on any sane
	// box. Use the temp dir directly as the start, env empty.
	tmp := t.TempDir()
	_, err := scenarios.ResolveVrooliRoot(tmp, func(string) string { return "" })
	require.Error(t, err)
}

func TestService_List_HappyPath(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{
		"alpha": "Alpha",
		"beta":  "Beta",
	})
	fake := &fakeFlowLister{byPath: map[string]flowResult{
		filepath.Join(root, "scenarios", "alpha"): {rows: []flows.Summary{{FlowID: "x"}, {FlowID: "y"}}},
		filepath.Join(root, "scenarios", "beta"):  {rows: nil},
	}}
	svc := scenarios.NewService(root, fake)
	got, err := svc.List()
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "alpha", got[0].ID)
	require.Equal(t, "Alpha", got[0].DisplayName)
	require.Equal(t, 2, got[0].FlowCount)
	require.Equal(t, "beta", got[1].ID)
	require.Equal(t, 0, got[1].FlowCount)
}

func TestService_List_SkipsDirsWithoutServiceJson(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{"alpha": "Alpha"})
	// A bare scenarios/scratch/ dir — no service.json — must be skipped.
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "scratch"), 0o755))
	svc := scenarios.NewService(root, &fakeFlowLister{})
	got, err := svc.List()
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "alpha", got[0].ID)
}

func TestService_List_PartialFailureSurfacedPerRow(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{
		"alpha": "Alpha",
		"beta":  "Beta",
	})
	fake := &fakeFlowLister{byPath: map[string]flowResult{
		filepath.Join(root, "scenarios", "alpha"): {rows: []flows.Summary{{FlowID: "x"}}},
		filepath.Join(root, "scenarios", "beta"):  {err: errors.New("permission denied")},
	}}
	svc := scenarios.NewService(root, fake)
	got, err := svc.List()
	require.NoError(t, err, "one bad scenario must not fail the whole list")
	require.Len(t, got, 2)
	require.Equal(t, 1, got[0].FlowCount)
	require.Empty(t, got[0].DiscoveryErr)
	require.Equal(t, 0, got[1].FlowCount)
	require.Contains(t, got[1].DiscoveryErr, "permission denied")
}

func TestService_Detail_HappyPath(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{"alpha": "Alpha"})
	rows := []flows.Summary{{FlowID: "a"}, {FlowID: "b"}}
	fake := &fakeFlowLister{byPath: map[string]flowResult{
		filepath.Join(root, "scenarios", "alpha"): {rows: rows},
	}}
	svc := scenarios.NewService(root, fake)
	got, err := svc.Detail("alpha")
	require.NoError(t, err)
	require.Equal(t, "alpha", got.ID)
	require.Equal(t, "Alpha", got.DisplayName)
	require.Equal(t, 2, got.FlowCount)
	require.Equal(t, rows, got.Flows)
}

func TestService_Detail_NotFound(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{"alpha": "Alpha"})
	svc := scenarios.NewService(root, &fakeFlowLister{})
	_, err := svc.Detail("missing")
	require.ErrorIs(t, err, scenarios.ErrScenarioNotFound)
}

func TestService_Detail_PropagatesFlowError(t *testing.T) {
	root := writeVrooliTree(t, map[string]string{"alpha": "Alpha"})
	fake := &fakeFlowLister{byPath: map[string]flowResult{
		filepath.Join(root, "scenarios", "alpha"): {err: errors.New("boom")},
	}}
	svc := scenarios.NewService(root, fake)
	_, err := svc.Detail("alpha")
	require.Error(t, err)
}
