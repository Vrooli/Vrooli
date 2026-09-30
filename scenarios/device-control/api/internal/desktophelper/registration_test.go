package desktophelper

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"device-control/internal/sessions"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
)

func TestReadRegistrationUsesCurrentEpochAndRefusesStoppedGeneration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux bootstrap")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0700))
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}
	config := Config{Version: 1, Surface: surface, SessionID: "os-session", StateDirectory: dir, XAuthorityFile: filepath.Join(dir, "authority"), GrantStatusFile: filepath.Join(dir, "grants"), PublicKey: base64.StdEncoding.EncodeToString(make([]byte, 32))}
	data, err := json.Marshal(config)
	require.NoError(t, err)
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	require.NoError(t, writeRegistration(dir, Registration{Surface: surface, SessionID: config.SessionID, HelperID: "helper", Epoch: 1, SocketPath: filepath.Join(dir, "helper.sock")}))
	databasePath := filepath.Join(dir, "sessions.sqlite")
	db, err := sql.Open("sqlite", databasePath)
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()
	repo, err := sessions.NewSQLiteDesktopRepository(ctx, db, config.SessionID)
	require.NoError(t, err)
	require.NoError(t, os.Chmod(databasePath, 0600))
	require.NoError(t, repo.Update(ctx, func(s *sessions.DesktopState) error { s.HelperID = "helper"; s.Epoch = 9; return nil }))
	registration, err := ReadRegistration(ctx, path)
	require.NoError(t, err)
	require.Equal(t, uint64(9), registration.Epoch, "startup epoch must not be reused")
	for _, change := range []func(*sessions.DesktopState){
		func(s *sessions.DesktopState) { s.Halted = true },
		func(s *sessions.DesktopState) { s.Halted = false; s.CleanupPending = true },
		func(s *sessions.DesktopState) { s.CleanupPending = false; s.HelperID = "replacement" },
	} {
		require.NoError(t, repo.Update(ctx, func(s *sessions.DesktopState) error { change(s); return nil }))
		_, err = ReadRegistration(ctx, path)
		require.ErrorIs(t, err, ErrBootstrap)
	}
	require.NoError(t, os.Chmod(databasePath, 0644))
	_, err = ReadRegistration(ctx, path)
	require.ErrorIs(t, err, ErrBootstrap)
}
