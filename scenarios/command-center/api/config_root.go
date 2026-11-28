package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

var errConfigRootUnavailable = errors.New("command-center config root unavailable")

// ensureConfigRoot keeps per-deployment configuration selection explicit. A
// missing selected root is unavailable; handlers must not create a new root
// that could look like the saved instance has been replaced by defaults.
func ensureConfigRoot() error {
	info, err := os.Stat(catalogDir())
	if err != nil || !info.IsDir() {
		return errConfigRootUnavailable
	}
	return nil
}

// ensureCatalogDirectory may create a catalog child inside an existing root,
// but never creates the selected root itself.
func ensureCatalogDirectory(name string) (string, error) {
	if err := ensureConfigRoot(); err != nil {
		return "", err
	}
	dir := filepath.Join(catalogDir(), name)
	if err := os.Mkdir(dir, 0o755); err != nil && !os.IsExist(err) {
		if rootErr := ensureConfigRoot(); rootErr != nil {
			return "", rootErr
		}
		return "", err
	}
	return dir, nil
}

func isConfigRootUnavailable(err error) bool {
	return errors.Is(err, errConfigRootUnavailable)
}

func writeConfigReadError(w http.ResponseWriter, err error) {
	if isConfigRootUnavailable(err) {
		writeError(w, http.StatusServiceUnavailable, "config_unavailable", "selected configuration is unavailable; restore the configured root and retry", nil)
		return
	}
	writeError(w, http.StatusInternalServerError, "config_read_failed", "Command Center configuration could not be read", nil)
}

func requireConfigRoot(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := ensureConfigRoot(); err != nil {
			writeConfigReadError(w, err)
			return
		}
		next(w, r)
	}
}
