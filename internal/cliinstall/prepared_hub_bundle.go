package cliinstall

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/config"
)

// PrepareHubBundle publishes a durable snapshot and its resume locator in that
// order. A locator write error can follow publication; retain the snapshot.
func PrepareHubBundle(source, snapshot, pointer string) error {
	return prepareHubBundle(source, snapshot, pointer, PrepareBundle, config.WriteOwnedFileDurable)
}

func prepareHubBundle(source, snapshot, pointer string, prepare func(string, string) error, write func(string, []byte, os.FileMode) error) error {
	if err := prepare(source, snapshot); err != nil {
		return err
	}
	if err := write(filepath.Join(snapshot, ".lease"), nil, 0600); err != nil {
		return fmt.Errorf("publish snapshot lease: %w", err)
	}
	if err := write(filepath.Join(snapshot, ".ready"), nil, 0600); err != nil {
		return fmt.Errorf("publish snapshot readiness: %w", err)
	}
	if err := write(pointer, []byte(snapshot+"\n"), 0600); err != nil {
		return fmt.Errorf("publish snapshot locator: %w", err)
	}
	return nil
}
