package uxcapture

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	bundle "github.com/vrooli/browser-automation-studio/services/uxcapture"
)

var (
	errCaptureStorageUnavailable = errors.New("capture storage unavailable")
	errCaptureBundleIDInvalid    = errors.New("capture bundle ID invalid")
)

// persistCaptureBundle is the single handler owner for validating the private
// destination and writing a decoded driver bundle. Route-specific source
// checks remain with each caller before this function is reached.
func persistCaptureBundle(capturesRoot string, manifest bundle.Bundle, payloads []bundle.Payload) error {
	if strings.TrimSpace(capturesRoot) == "" {
		return errCaptureStorageUnavailable
	}
	bundleDir, err := bundleDirectory(capturesRoot, manifest.ID)
	if err != nil {
		return errCaptureBundleIDInvalid
	}
	if err := bundle.WriteBundleWithArtifacts(bundleDir, manifest, payloads); err != nil {
		return fmt.Errorf("persist private capture bundle: %w", err)
	}
	return nil
}

func respondCapturePersistenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCaptureStorageUnavailable):
		writeError(w, http.StatusInternalServerError, "BAS private capture storage is unavailable")
	case errors.Is(err, errCaptureBundleIDInvalid):
		writeError(w, http.StatusBadGateway, "Driver returned an invalid UX capture bundle ID")
	default:
		cause := strings.TrimPrefix(err.Error(), "persist private capture bundle: ")
		writeError(w, http.StatusInternalServerError, "Could not persist the private capture bundle: "+cause)
	}
}
