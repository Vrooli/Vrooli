package deps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/deps"
	depsmocks "react-component-library/internal/deps/mocks"
)

type fakePkgReader struct {
	bytesByScenario map[string][]byte
	err             error
}

func (f *fakePkgReader) Read(_ context.Context, scenario string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	b, ok := f.bytesByScenario[scenario]
	if !ok {
		return nil, errors.New("not found")
	}
	return b, nil
}

func newSvcWithDecls(t *testing.T, componentID string, libraryID string, ds []deps.DeclarationFields, pkgs deps.PackageJSONReader) deps.Service {
	t.Helper()
	repo := depsmocks.NewFakeRepository()
	require.NoError(t, repo.SyncForComponent(context.Background(), deps.SyncInput{
		ComponentID:  componentID,
		LibraryID:    libraryID,
		Declarations: ds,
	}))
	return deps.NewService(repo, pkgs)
}

func TestValidateAdoption_OK(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{"react":"^18.2.0","lodash":"^4.17.0"}}`),
	}}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{DepName: "react", VersionRange: "^18.0.0"},
		{DepName: "lodash", VersionRange: "^4.17.0"},
	}, pkg)

	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictOK, v.Kind)
	require.Empty(t, v.Issues)
}

func TestValidateAdoption_Warn_MissingDep(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{"react":"^18.2.0"}}`),
	}}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{DepName: "react", VersionRange: "^18.0.0"},
		{DepName: "lodash", VersionRange: "^4.17.0"},
	}, pkg)

	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictWarn, v.Kind)
	require.Len(t, v.Issues, 1)
	require.Equal(t, deps.IssueMissingDep, v.Issues[0].Kind)
	require.Equal(t, "lodash", v.Issues[0].DepName)
}

func TestValidateAdoption_Block_IncompatibleMajor(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{"react":"^19.0.0"}}`),
	}}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{DepName: "react", VersionRange: "^18.0.0"},
	}, pkg)

	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictBlock, v.Kind)
	require.Len(t, v.Issues, 1)
	require.Equal(t, deps.IssueIncompatibleMajor, v.Issues[0].Kind)
}

func TestValidateAdoption_Block_BeatsWarn(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{"react":"^19.0.0"}}`),
	}}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{DepName: "react", VersionRange: "^18.0.0"},
		{DepName: "lodash", VersionRange: "^4.0.0"},
	}, pkg)

	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictBlock, v.Kind)
	require.Len(t, v.Issues, 2)
}

func TestValidateAdoption_UsesRequestedComponentVersion(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{"react":"^17.0.0"}}`),
	}}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{Version: "1.0.0", DepName: "react", VersionRange: "^17.0.0"},
		{Version: "1.1.0", DepName: "react", VersionRange: "^18.0.0"},
	}, pkg)

	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "1.0.0", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictOK, v.Kind)
	require.Empty(t, v.Issues)

	v, err = svc.ValidateAdoption(context.Background(), "cmp-1", "1.1.0", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictBlock, v.Kind)
	require.Len(t, v.Issues, 1)
	require.Equal(t, "1.1.0", v.Issues[0].Version)
}

func TestValidateAdoption_MissingPeerDependencyBlocks(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{"react":"^18.2.0"}}`),
	}}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{Version: "1.0.0", DepName: "react", VersionRange: "^18.0.0"},
		{Version: "1.0.0", DepName: "lucide-react", VersionRange: "^0.424.0", Kind: deps.DepKindPeer},
	}, pkg)

	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "1.0.0", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictBlock, v.Kind)
	require.Len(t, v.Issues, 1)
	require.Equal(t, deps.IssueMissingDep, v.Issues[0].Kind)
	require.Equal(t, deps.DepKindPeer, v.Issues[0].DepKind)
}

func TestValidateAdoption_NoDeclarations_OK(t *testing.T) {
	pkg := &fakePkgReader{bytesByScenario: map[string][]byte{
		"target-app": []byte(`{"dependencies":{}}`),
	}}
	repo := depsmocks.NewFakeRepository()
	svc := deps.NewService(repo, pkg)
	v, err := svc.ValidateAdoption(context.Background(), "cmp-1", "", "target-app")
	require.NoError(t, err)
	require.Equal(t, deps.VerdictOK, v.Kind)
}

func TestValidateAdoption_ScenarioPkgMissing(t *testing.T) {
	pkg := &fakePkgReader{err: errors.New("ENOENT")}
	svc := newSvcWithDecls(t, "cmp-1", "rcl:Button", []deps.DeclarationFields{
		{DepName: "react", VersionRange: "^18.0.0"},
	}, pkg)

	_, err := svc.ValidateAdoption(context.Background(), "cmp-1", "", "missing-app")
	require.Error(t, err)
	var sentinel deps.ErrScenarioPackageJSONMissing
	require.ErrorAs(t, err, &sentinel)
}

func TestSyncForComponent_StripsBlankNames(t *testing.T) {
	repo := depsmocks.NewFakeRepository()
	svc := deps.NewService(repo, nil)
	require.NoError(t, svc.SyncForComponent(context.Background(), deps.SyncInput{
		ComponentID: "cmp-1",
		Declarations: []deps.DeclarationFields{
			{DepName: "react", VersionRange: "^18"},
			{DepName: "  ", VersionRange: ""},
		},
	}))
	got, err := svc.ListForComponent(context.Background(), "cmp-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "react", got[0].DepName)
}
