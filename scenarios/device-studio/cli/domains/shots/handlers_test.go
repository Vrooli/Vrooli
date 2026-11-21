package shots

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestReadSnapshotUsesGeneratedProtoJSONSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"motionMode":"MOTION_MODE_FIXED","scene":{"displayWidthPixels":400}}`), 0o600))
	snapshot, err := readSnapshot(path)
	require.NoError(t, err)
	require.Equal(t, shotsv1.MotionMode_MOTION_MODE_FIXED, snapshot.MotionMode)
	require.EqualValues(t, 400, snapshot.Scene.DisplayWidthPixels)

	require.NoError(t, os.WriteFile(path, []byte(`{"unknownRecipePayload":true}`), 0o600))
	_, err = readSnapshot(path)
	require.Error(t, err, "unknown fields must not become an arbitrary recipe escape hatch")
}

func TestShotConflictRequiresTypedDetail(t *testing.T) {
	plain := connect.NewError(connect.CodeAborted, errors.New("server busy"))
	_, ok := shotConflict(plain)
	require.False(t, ok)

	conflict := connect.NewError(connect.CodeAborted, errors.New("conflict"))
	detail, err := connect.NewErrorDetail(&shotsv1.ShotConflict{ShotId: "shot-1", ExpectedRevision: 2, CurrentRevision: 3})
	require.NoError(t, err)
	conflict.AddDetail(detail)
	message, ok := shotConflict(conflict)
	require.True(t, ok)
	require.ErrorContains(t, message, "shot-1 revision conflict: expected 2, current 3")
}
