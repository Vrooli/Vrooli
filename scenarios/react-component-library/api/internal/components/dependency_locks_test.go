package components

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestCatalogVersionDependencyLocksAreCompleteAndLive(t *testing.T) {
	require.NoError(t, ValidateVersionDependencyLocks(os.DirFS("../../../library")))
}

func TestValidateVersionDependencyLocksRejectsMissingLock(t *testing.T) {
	fsys := fstest.MapFS{
		"components/Button/versions/1.0.0/Button.tsx": {Data: []byte("export const Button = 1")},
	}
	require.ErrorContains(t, ValidateVersionDependencyLocks(fsys), "missing dependencies.json")
}

func TestValidateVersionDependencyLocksRejectsMissingTargetVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"components/Button/versions/1.0.0/Button.tsx":        {Data: []byte("export const Button = 1")},
		"components/Button/versions/1.0.0/dependencies.json": {Data: []byte(`{"schemaVersion":1,"libraryId":"react-component-library:Button","version":"1.0.0","resolvedAt":"2026-08-27T00:00:00Z","dependencies":[{"libraryId":"react-component-library:Missing","version":"9.9.9","rank":3}]}`)},
	}
	require.ErrorContains(t, ValidateVersionDependencyLocks(fsys), "has no materialized version directory")
}
