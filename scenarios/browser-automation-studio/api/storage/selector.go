package storage

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// NewScreenshotStorage chooses a storage backend based on env configuration, defaulting to local filesystem.
// BAS_SCREENSHOT_STORAGE:
//   - "local" (default): filesystem store rooted at screenshotRoot
//   - "minio": use MinIO/S3-compatible object storage, falling back to local on init failure
func NewScreenshotStorage(log *logrus.Logger, screenshotRoot string) StorageInterface {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_SCREENSHOT_STORAGE")))
	if mode == "" {
		mode = "local"
	}

	if mode == "minio" {
		client, err := NewMinIOClient(log)
		if err != nil {
			if log != nil {
				log.WithError(err).Warn("Falling back to local screenshot storage after MinIO init failure")
			}
		} else {
			return client
		}
	}

	local, err := NewFileStorage(screenshotRoot, log)
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Screenshot storage unavailable; filesystem backend failed to initialize")
		}
		return nil
	}
	return local
}
