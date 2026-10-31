package desktophelper

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
)

func record(family uint16, host, display string, cookie []byte) []byte {
	var data bytes.Buffer
	_ = binary.Write(&data, binary.BigEndian, family)
	for _, field := range [][]byte{[]byte(host), []byte(display), []byte("MIT-MAGIC-COOKIE-1"), cookie} {
		_ = binary.Write(&data, binary.BigEndian, uint16(len(field)))
		data.Write(field)
	}
	return data.Bytes()
}
func TestXAuthoritySelectsExactDisplayAndRejectsAmbiguity(t *testing.T) {
	cookie := bytes.Repeat([]byte{1}, 16)
	data := append(record(256, "host", "1", cookie), record(256, "host", "2", bytes.Repeat([]byte{2}, 16))...)
	got, err := xCookie(data, 1, "host")
	require.NoError(t, err)
	require.Equal(t, "01010101010101010101010101010101", got)
	_, err = xCookie(data, 3, "host")
	require.Error(t, err)
	_, err = xCookie(data, 1, "other-host")
	require.Error(t, err)
	_, err = xCookie(append(data, record(65535, "", "1", bytes.Repeat([]byte{3}, 16))...), 1, "host")
	require.Error(t, err)
	_, err = xCookie(data[:len(data)-1], 1, "host")
	require.Error(t, err)
}
func TestXAuthorityGDMEmptyDisplayRemainsHostBoundAndUnambiguous(t *testing.T) {
	cookie := bytes.Repeat([]byte{1}, 16)
	data := append(record(256, "host", "", cookie), record(65535, "host", "", cookie)...)
	got, err := xCookie(data, 0, "host")
	require.NoError(t, err)
	require.Equal(t, "01010101010101010101010101010101", got)
	_, err = xCookie(record(256, "other-host", "", cookie), 0, "host")
	require.Error(t, err)
	_, err = xCookie(append(data, record(256, "host", "0", bytes.Repeat([]byte{2}, 16))...), 0, "host")
	require.Error(t, err, "conflicting wildcard and explicit cookies must not be guessed")
}

func TestPrivateBootstrapFilesAndExclusiveLock(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux bootstrap")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0700))
	require.NoError(t, privateDirectory(dir))
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte("private"), 0600))
	_, err := readPrivate(path, 1024)
	require.NoError(t, err)
	link := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(path, link))
	_, err = readPrivate(link, 1024)
	require.Error(t, err)
	require.NoError(t, os.Chmod(path, 0644))
	_, err = readPrivate(path, 1024)
	require.Error(t, err)
	release, err := lockDirectory(dir)
	require.NoError(t, err)
	_, err = lockDirectory(dir)
	require.Error(t, err)
	release()
	release, err = lockDirectory(dir)
	require.NoError(t, err)
	release()
}
func TestGrantStatusExpiresAndNeverDefaultsToAllow(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux bootstrap")
	}
	path := filepath.Join(t.TempDir(), "grants.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"active":["g1"],"observed_at":"2026-09-05T00:00:00Z","expires_at":"2026-09-05T00:00:01Z"}`), 0600))
	now := time.Date(2026, 9, 5, 0, 0, 0, 500000000, time.UTC)
	active, err := grantActive(path, "g1", now)
	require.NoError(t, err)
	require.True(t, active)
	active, err = grantActive(path, "g2", now)
	require.NoError(t, err)
	require.False(t, active)
	active, err = grantActive(path, "g1", now.Add(time.Minute))
	require.Error(t, err)
	require.False(t, active)
}

func TestActiveWaylandSessionNeverUsesX11CompatibilityDisplay(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "")
	require.True(t, activeWaylandSession())
	t.Setenv("XDG_SESSION_TYPE", "x11")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	require.True(t, activeWaylandSession())
	t.Setenv("WAYLAND_DISPLAY", "")
	require.False(t, activeWaylandSession())
}

func TestWaylandConfigUsesNamedPortalBackendInsteadOfXAuthority(t *testing.T) {
	dir := t.TempDir()
	config := Config{
		Version:         1,
		Surface:         targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"},
		SessionID:       "session",
		GrantStatusFile: filepath.Join(dir, "grants"),
		StateDirectory:  filepath.Join(dir, "state"),
		PublicKey:       base64.StdEncoding.EncodeToString(make([]byte, 32)),
		WaylandBackend:  "gnome",
	}
	require.NoError(t, validateConfig(config))
	config.XAuthorityFile = filepath.Join(dir, "authority")
	require.Error(t, validateConfig(config))
	config.XAuthorityFile = ""
	config.WaylandBackend = "sway"
	require.Error(t, validateConfig(config))
}
