package desktophelper

import (
	"context"
	"database/sql"
	"device-control/internal/sessions"
	"encoding/json"
	"github.com/vrooli/api-core/targetmodel"
	"os"
	"path/filepath"
	"time"
)

func nowUTC() time.Time { return time.Now().UTC() }

type Registration struct {
	HelperID   string                 `json:"helper_id"`
	Epoch      uint64                 `json:"epoch"`
	SessionID  string                 `json:"session_id"`
	Surface    targetmodel.SurfaceRef `json:"surface"`
	SocketPath string                 `json:"socket_path"`
}

// ReadRegistration resolves the startup identity against current durable helper
// state. The registration file's initial epoch is not a reusable admission epoch.
// A stopped or replaced helper cannot authorize another owner grant.
func ReadRegistration(ctx context.Context, configPath string) (Registration, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return Registration{}, err
	}
	if err = privateDirectory(config.StateDirectory); err != nil {
		return Registration{}, err
	}
	data, err := readPrivate(filepath.Join(config.StateDirectory, "registration.json"), 64*1024)
	if err != nil {
		return Registration{}, err
	}
	var registration Registration
	if decodeStrict(data, &registration) != nil || registration.Surface != config.Surface || registration.SessionID != config.SessionID || registration.HelperID == "" || registration.SocketPath != filepath.Join(config.StateDirectory, "helper.sock") {
		return Registration{}, ErrBootstrap
	}
	databasePath := filepath.Join(config.StateDirectory, "sessions.sqlite")
	// Validate the file without loading a potentially large receipt database.
	if err = checkPrivateDatabase(databasePath); err != nil {
		return Registration{}, err
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return Registration{}, err
	}
	defer db.Close()
	repo, err := sessions.NewSQLiteDesktopRepository(ctx, db, config.SessionID)
	if err != nil {
		return Registration{}, err
	}
	err = repo.Update(ctx, func(state *sessions.DesktopState) error {
		if state.Halted || state.CleanupPending || state.HelperID != registration.HelperID || state.Epoch == 0 {
			return ErrBootstrap
		}
		registration.Epoch = state.Epoch
		return nil
	})
	if err != nil {
		return Registration{}, err
	}
	return registration, nil
}

func writeRegistration(directory string, registration Registration) error {
	return writePrivateJSON(filepath.Join(directory, "registration.json"), registration)
}

func writePrivateJSON(path string, value any) error {
	directory := filepath.Dir(path)
	if err := privateDirectory(directory); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".registration-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
