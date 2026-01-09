package storage

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/resilience"
)

// NewScreenshotStorage chooses a storage backend based on env configuration, defaulting to local filesystem.
// BAS_SCREENSHOT_STORAGE:
//   - "local" (default): filesystem store rooted at recordingsRoot
//   - "minio": use MinIO/S3-compatible object storage with circuit breaker and local fallback
//   - "minio-only": use MinIO without fallback (for testing)
//
// When using "minio" mode, the storage is wrapped in a resilient layer that:
//   - Uses a circuit breaker to detect MinIO failures
//   - Automatically falls back to local storage when MinIO is unavailable
//   - Recovers to MinIO when the circuit breaker closes
//
// Circuit breaker configuration via environment variables:
//   - MINIO_CB_TIMEOUT: cooldown period before retrying (default: 30s)
//   - MINIO_CB_FAILURE_THRESHOLD: failures before opening circuit (default: 5)
//   - MINIO_CB_FAILURE_RATIO: failure ratio to trip breaker (default: 0.6)
func NewScreenshotStorage(log *logrus.Logger, recordingsRoot string) StorageInterface {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_SCREENSHOT_STORAGE")))
	if mode == "" {
		mode = "local"
	}

	// Create local storage as potential fallback
	local, localErr := NewFileStorage(recordingsRoot, log)
	if localErr != nil && log != nil {
		log.WithError(localErr).Debug("Local filesystem storage unavailable for fallback")
	}

	if mode == "minio" || mode == "minio-only" {
		client, err := NewMinIOClient(log)
		if err != nil {
			if log != nil {
				log.WithError(err).Warn("MinIO initialization failed, using local storage")
			}
			if local == nil {
				if log != nil {
					log.Error("Screenshot storage unavailable: both MinIO and local storage failed")
				}
				return nil
			}
			return local
		}

		// For minio-only mode, return raw client without resilience wrapper
		if mode == "minio-only" {
			return client
		}

		// Wrap MinIO with circuit breaker and local fallback
		if local != nil {
			cfg := resilience.ConfigFromEnv("MINIO", "minio-storage")
			cfg.Logger = log
			if log != nil {
				log.Info("Using resilient MinIO storage with local fallback")
			}
			return NewResilientStorage(ResilientStorageConfig{
				Primary:       client,
				Fallback:      local,
				BreakerConfig: cfg,
				Logger:        log,
			})
		}

		// No fallback available, return raw client
		if log != nil {
			log.Warn("Using MinIO storage without fallback (local storage unavailable)")
		}
		return client
	}

	// Local mode
	if local == nil {
		if log != nil {
			log.Error("Screenshot storage unavailable; filesystem backend failed to initialize")
		}
		return nil
	}
	return local
}
