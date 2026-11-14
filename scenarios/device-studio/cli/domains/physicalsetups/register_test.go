package physicalsetups

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadSetupRequiresStrictBoundedProtoJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "setup.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"displayId":"display"}`), 0o600))
	setup, err := readSetup(path)
	require.NoError(t, err)
	require.Equal(t, "display", setup.GetDisplayId())
	require.NoError(t, os.WriteFile(path, []byte(`{"unknown":true}`), 0o600))
	_, err = readSetup(path)
	require.ErrorContains(t, err, "decode strict")
	require.NoError(t, os.WriteFile(path, []byte(strings.Repeat("x", maxSetupBytes+1)), 0o600))
	_, err = readSetup(path)
	require.ErrorContains(t, err, "at most")
}

func TestPhysicalSetupCLIParsersRejectInvalidCursorAndRange(t *testing.T) {
	_, err := parseOptionalUint32("-1")
	require.Error(t, err)
	_, err = parsePage("101")
	require.Error(t, err)
	_, err = parsePage("0")
	require.Error(t, err)
	got, err := parseOptionalUint32("")
	require.NoError(t, err)
	require.Zero(t, got)
}
