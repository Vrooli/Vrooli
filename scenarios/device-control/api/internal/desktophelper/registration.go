package desktophelper

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"device-control/internal/sessions"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/targetmodel"
	_ "modernc.org/sqlite"
)

func nowUTC() time.Time { return time.Now().UTC() }

type Registration struct {
	ComponentGeneration string                 `json:"component_generation,omitempty"`
	HelperID            string                 `json:"helper_id"`
	Epoch               uint64                 `json:"epoch"`
	SessionID           string                 `json:"session_id"`
	Surface             targetmodel.SurfaceRef `json:"surface"`
	SocketPath          string                 `json:"socket_path"`
}

// LifecycleReceipt is a durable helper-local shutdown observation. A terminal
// cleanup result does not prove that the process exited; the process owner must
// still establish that boundary independently.
type LifecycleReceipt struct {
	ComponentGeneration string                 `json:"component_generation,omitempty"`
	HelperID            string                 `json:"helper_id"`
	Epoch               uint64                 `json:"epoch"`
	SessionID           string                 `json:"session_id"`
	Surface             targetmodel.SurfaceRef `json:"surface"`
	Status              string                 `json:"status"`
	ActiveRequests      int                    `json:"active_requests"`
	ErrorCode           string                 `json:"error_code,omitempty"`
	ObservedAt          time.Time              `json:"observed_at"`
}

const lifecycleReceiptName = "lifecycle.json"

func validLifecycleReceipt(receipt LifecycleReceipt) bool {
	switch receipt.Status {
	case "running", "stopping", "controller-cleanup-completed", "uncertain":
	default:
		return false
	}
	return receipt.HelperID != "" && receipt.Epoch > 0 && receipt.SessionID != "" && receipt.Surface.Validate() == nil && receipt.ActiveRequests >= 0 && !receipt.ObservedAt.IsZero() && len(receipt.ErrorCode) <= 128
}

// WriteLifecycleReceipt atomically persists a helper-generation-specific
// observation in the already protected helper state directory.
func WriteLifecycleReceipt(directory string, receipt LifecycleReceipt) error {
	if receipt.ObservedAt.IsZero() {
		receipt.ObservedAt = nowUTC()
	}
	if !validLifecycleReceipt(receipt) {
		return ErrBootstrap
	}
	return writePrivateJSON(filepath.Join(directory, lifecycleReceiptName), receipt)
}

// ReadLifecycleReceipt reads the shutdown observation without opening or
// mutating the helper's SQLite state. Missing or malformed receipts stay
// unknown to lifecycle inventory.
func ReadLifecycleReceipt(ctx context.Context, configPath string) (LifecycleReceipt, error) {
	if err := ctx.Err(); err != nil {
		return LifecycleReceipt{}, err
	}
	config, err := LoadConfig(configPath)
	if err != nil {
		return LifecycleReceipt{}, err
	}
	if err = privateDirectory(config.StateDirectory); err != nil {
		return LifecycleReceipt{}, err
	}
	data, err := readPrivate(filepath.Join(config.StateDirectory, lifecycleReceiptName), 64*1024)
	if err != nil {
		return LifecycleReceipt{}, err
	}
	var receipt LifecycleReceipt
	if decodeStrict(data, &receipt) != nil || !validLifecycleReceipt(receipt) || receipt.Surface != config.Surface || receipt.SessionID != config.SessionID {
		return LifecycleReceipt{}, ErrBootstrap
	}
	registrationData, err := readPrivate(filepath.Join(config.StateDirectory, "registration.json"), 64*1024)
	if err != nil {
		return LifecycleReceipt{}, err
	}
	var registration Registration
	if decodeStrict(registrationData, &registration) != nil || registration.Surface != config.Surface || registration.SessionID != config.SessionID || registration.SocketPath != filepath.Join(config.StateDirectory, "helper.sock") || registration.HelperID != receipt.HelperID || registration.Epoch != receipt.Epoch || registration.ComponentGeneration != receipt.ComponentGeneration {
		return LifecycleReceipt{}, ErrBootstrap
	}
	return receipt, nil
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
	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          databasePath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
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
