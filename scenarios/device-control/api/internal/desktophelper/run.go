package desktophelper

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"device-control/internal/native/atspi"
	"device-control/internal/native/x11"
	"device-control/internal/sessions"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Run is invoked only by the lifecycle-managed helper executable. The protected
// bootstrap config is provisioned by the admission owner; it contains no private
// signing key and cannot mint grants. Missing configuration never enables input.
func Run(ctx context.Context, path string) error { return RunBound(ctx, path, "") }

// RunBound waits for owner provisioning to match the protected bootstrap before
// holding the helper lock. This prevents an old generated config from winning
// a managed restart race during an explicit accessibility rebind.
func RunBound(ctx context.Context, path, ownerPath string) error {
	config, err := LoadConfig(path)
	if err != nil {
		return err
	}
	if ownerPath != "" {
		if err := checkOwnerConfiguration(ownerPath, path, config); err != nil {
			return err
		}
	}
	if err = privateDirectory(config.StateDirectory); err != nil {
		return err
	}
	unlock, err := lockDirectory(config.StateDirectory)
	if err != nil {
		return err
	}
	defer unlock()
	authorityBytes, err := readPrivate(config.XAuthorityFile, 1024*1024)
	if err != nil {
		return err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return ErrBootstrap
	}
	cookie, err := xCookie(authorityBytes, config.Display, hostname)
	if err != nil {
		return err
	}
	backend, err := x11.NewForSession(ctx, config.Display, cookie, config.SessionID)
	if err != nil {
		return ErrBootstrap
	}
	defer backend.Close()
	var native sessions.DesktopNative = backend
	if config.AccessibilitySocket != "" {
		conn, err := atspi.DialBound(ctx, config.AccessibilitySocket, config.AccessibilityBusID, func(check context.Context, _, _ uint32) error { return backend.CheckSession(check) })
		if err != nil {
			return ErrBootstrap
		}
		defer conn.Close()
		client, err := atspi.New(conn, func(check context.Context, ref atspi.Ref) error {
			if ref.BusID != config.AccessibilityBusID {
				return atspi.ErrRefused
			}
			return backend.CheckSession(check)
		})
		if err != nil {
			return ErrBootstrap
		}
		native = &semanticBackend{pixels: backend, access: client, busID: config.AccessibilityBusID}
	}
	key, err := base64.StdEncoding.DecodeString(config.PublicKey)
	if err != nil {
		return ErrBootstrap
	}
	authority, err := sessions.NewSignedDesktopAuthority(key, func(ctx context.Context, id string) (bool, error) {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return grantActive(config.GrantStatusFile, id, nowUTC())
	})
	if err != nil {
		return err
	}
	// The directory is private. Refuse links to unrelated databases rather than
	// allowing SQLite to follow a caller-controlled target.
	databasePath := filepath.Join(config.StateDirectory, "sessions.sqlite")
	if info, err := os.Lstat(databasePath); err == nil {
		if !info.Mode().IsRegular() {
			return ErrBootstrap
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	repo, err := sessions.NewSQLiteDesktopRepository(ctx, db, config.SessionID)
	if err != nil {
		return err
	}
	if err = os.Chmod(databasePath, 0600); err != nil {
		return err
	}
	helperID := uuid.NewString()
	controller, err := sessions.NewDesktopController(repo, authority, native, config.Surface, config.SessionID, helperID)
	if err != nil {
		return err
	}
	var previous string
	var epoch uint64
	if err = repo.Update(ctx, func(s *sessions.DesktopState) error { previous = s.HelperID; epoch = s.Epoch; return nil }); err != nil {
		return err
	}
	// Bind before activation so an existing listener cannot be displaced. A
	// stale socket is removed only under the exclusive helper lock after a
	// refused connection; an active socket is never displaced.
	socketPath := filepath.Join(config.StateDirectory, "helper.sock")
	if info, statErr := os.Lstat(socketPath); statErr == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return ErrBootstrap
		}
		probe, probeErr := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
		if probeErr == nil {
			probe.Close()
			return ErrBootstrap
		}
		if !errors.Is(probeErr, syscall.ECONNREFUSED) {
			return ErrBootstrap
		}
		if err = os.Remove(socketPath); err != nil {
			return err
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return err
	}
	defer listener.Close()
	if err = os.Chmod(socketPath, 0600); err != nil {
		return err
	}
	if err = controller.ActivateHelper(ctx, previous, epoch); err != nil {
		return err
	}
	helper, err := sessions.NewDesktopUnixHelper(controller, authority)
	if err != nil {
		return err
	}
	// This safe metadata lets admission name the exact helper generation/epoch.
	registration := Registration{HelperID: helperID, Epoch: epoch + 1, SessionID: config.SessionID, Surface: config.Surface, SocketPath: socketPath}
	if err = writeRegistration(config.StateDirectory, registration); err != nil {
		return err
	}
	return helper.Serve(ctx, listener)
}

func checkOwnerConfiguration(ownerPath, helperPath string, generated Config) error {
	owner, err := LoadOwnerBootstrap(ownerPath)
	if err != nil || owner.HelperConfigPath != helperPath {
		return ErrBootstrap
	}
	generated.PublicKey = ""
	if owner.Helper != generated {
		return ErrBootstrap
	}
	return nil
}
