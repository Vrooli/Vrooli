package storage

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/resilience"
)

// ResilientStorage wraps a primary storage backend with a fallback,
// using a circuit breaker to detect and recover from failures.
type ResilientStorage struct {
	primary   StorageInterface
	fallback  StorageInterface
	breaker   *resilience.Breaker
	log       *logrus.Logger
	useFallback atomic.Bool
}

// ResilientStorageConfig configures the resilient storage wrapper.
type ResilientStorageConfig struct {
	Primary        StorageInterface
	Fallback       StorageInterface
	BreakerConfig  resilience.BreakerConfig
	Logger         *logrus.Logger
}

// NewResilientStorage creates a storage wrapper with circuit breaker and fallback.
func NewResilientStorage(cfg ResilientStorageConfig) *ResilientStorage {
	if cfg.Logger == nil {
		cfg.Logger = logrus.StandardLogger()
	}
	if cfg.BreakerConfig.Name == "" {
		cfg.BreakerConfig.Name = "storage"
	}
	cfg.BreakerConfig.Logger = cfg.Logger

	// Set up state change callback to log fallback activation
	originalOnStateChange := cfg.BreakerConfig.OnStateChange
	cfg.BreakerConfig.OnStateChange = func(name string, from, to resilience.BreakerState) {
		if to == resilience.StateOpen {
			cfg.Logger.WithField("breaker", name).Warn("Storage circuit breaker opened, activating fallback storage")
		} else if from == resilience.StateOpen && to == resilience.StateClosed {
			cfg.Logger.WithField("breaker", name).Info("Storage circuit breaker closed, primary storage restored")
		}
		if originalOnStateChange != nil {
			originalOnStateChange(name, from, to)
		}
	}

	return &ResilientStorage{
		primary:  cfg.Primary,
		fallback: cfg.Fallback,
		breaker:  resilience.NewBreaker(cfg.BreakerConfig),
		log:      cfg.Logger,
	}
}

// StoreScreenshot stores a screenshot, falling back to secondary storage on failure.
func (r *ResilientStorage) StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*ScreenshotInfo, error) {
	// Try primary with circuit breaker
	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		return r.primary.StoreScreenshot(ctx, executionID, stepName, data, contentType)
	})

	if err == nil {
		r.useFallback.Store(false)
		return result.(*ScreenshotInfo), nil
	}

	// If circuit breaker is open or primary failed, try fallback
	if r.fallback != nil {
		if errors.Is(err, resilience.ErrCircuitOpen) {
			r.log.WithFields(logrus.Fields{
				"execution_id": executionID,
				"step":         stepName,
			}).Debug("Using fallback storage (circuit breaker open)")
		} else {
			r.log.WithError(err).WithFields(logrus.Fields{
				"execution_id": executionID,
				"step":         stepName,
			}).Warn("Primary storage failed, using fallback")
		}

		r.useFallback.Store(true)
		return r.fallback.StoreScreenshot(ctx, executionID, stepName, data, contentType)
	}

	return nil, err
}

// GetScreenshot retrieves a screenshot, trying fallback if primary fails.
func (r *ResilientStorage) GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	// If we're in fallback mode, try fallback first for recently stored items
	if r.useFallback.Load() && r.fallback != nil {
		reader, info, err := r.fallback.GetScreenshot(ctx, objectName)
		if err == nil {
			return reader, info, nil
		}
		// Fall through to try primary
	}

	// Try primary with circuit breaker
	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		reader, info, err := r.primary.GetScreenshot(ctx, objectName)
		if err != nil {
			return nil, err
		}
		return &getResult{reader: reader, info: info}, nil
	})

	if err == nil {
		res := result.(*getResult)
		return res.reader, res.info, nil
	}

	// Try fallback
	if r.fallback != nil {
		return r.fallback.GetScreenshot(ctx, objectName)
	}

	return nil, nil, err
}

type getResult struct {
	reader io.ReadCloser
	info   *minio.ObjectInfo
}

// GetArtifact retrieves an artifact, trying fallback if primary fails.
func (r *ResilientStorage) GetArtifact(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error) {
	// Similar pattern to GetScreenshot
	if r.useFallback.Load() && r.fallback != nil {
		reader, info, err := r.fallback.GetArtifact(ctx, objectName)
		if err == nil {
			return reader, info, nil
		}
	}

	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		reader, info, err := r.primary.GetArtifact(ctx, objectName)
		if err != nil {
			return nil, err
		}
		return &getResult{reader: reader, info: info}, nil
	})

	if err == nil {
		res := result.(*getResult)
		return res.reader, res.info, nil
	}

	if r.fallback != nil {
		return r.fallback.GetArtifact(ctx, objectName)
	}

	return nil, nil, err
}

// DeleteScreenshot deletes a screenshot from storage.
func (r *ResilientStorage) DeleteScreenshot(ctx context.Context, objectName string) error {
	// Try to delete from both storages if we've been using fallback
	var primaryErr, fallbackErr error

	_, primaryErr = r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		return nil, r.primary.DeleteScreenshot(ctx, objectName)
	})

	if r.fallback != nil {
		fallbackErr = r.fallback.DeleteScreenshot(ctx, objectName)
	}

	// Return primary error if fallback succeeded, otherwise return fallback error
	if primaryErr != nil && !errors.Is(primaryErr, resilience.ErrCircuitOpen) {
		return primaryErr
	}
	if fallbackErr != nil {
		return fallbackErr
	}
	return primaryErr
}

// ListExecutionScreenshots lists screenshots for an execution.
func (r *ResilientStorage) ListExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]string, error) {
	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		return r.primary.ListExecutionScreenshots(ctx, executionID)
	})

	if err == nil {
		return result.([]string), nil
	}

	if r.fallback != nil {
		return r.fallback.ListExecutionScreenshots(ctx, executionID)
	}

	return nil, err
}

// StoreScreenshotFromFile stores a screenshot from a file path.
func (r *ResilientStorage) StoreScreenshotFromFile(ctx context.Context, executionID uuid.UUID, stepName string, filePath string) (*ScreenshotInfo, error) {
	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		return r.primary.StoreScreenshotFromFile(ctx, executionID, stepName, filePath)
	})

	if err == nil {
		r.useFallback.Store(false)
		return result.(*ScreenshotInfo), nil
	}

	if r.fallback != nil {
		r.useFallback.Store(true)
		return r.fallback.StoreScreenshotFromFile(ctx, executionID, stepName, filePath)
	}

	return nil, err
}

// StoreArtifactFromFile stores an artifact from a file path.
func (r *ResilientStorage) StoreArtifactFromFile(ctx context.Context, executionID uuid.UUID, label string, filePath string, contentType string) (*ArtifactInfo, error) {
	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		return r.primary.StoreArtifactFromFile(ctx, executionID, label, filePath, contentType)
	})

	if err == nil {
		r.useFallback.Store(false)
		return result.(*ArtifactInfo), nil
	}

	if r.fallback != nil {
		r.useFallback.Store(true)
		return r.fallback.StoreArtifactFromFile(ctx, executionID, label, filePath, contentType)
	}

	return nil, err
}

// StoreArtifact stores raw artifact data.
func (r *ResilientStorage) StoreArtifact(ctx context.Context, objectName string, data []byte, contentType string) (*ArtifactInfo, error) {
	result, err := r.breaker.ExecuteContext(ctx, func(ctx context.Context) (any, error) {
		return r.primary.StoreArtifact(ctx, objectName, data, contentType)
	})

	if err == nil {
		r.useFallback.Store(false)
		return result.(*ArtifactInfo), nil
	}

	if r.fallback != nil {
		r.useFallback.Store(true)
		return r.fallback.StoreArtifact(ctx, objectName, data, contentType)
	}

	return nil, err
}

// GetBucketName returns the bucket name from the primary storage.
func (r *ResilientStorage) GetBucketName() string {
	return r.primary.GetBucketName()
}

// HealthCheck checks if the primary storage is healthy.
func (r *ResilientStorage) HealthCheck(ctx context.Context) error {
	// Health check bypasses the circuit breaker to allow recovery probing
	return r.primary.HealthCheck(ctx)
}

// CircuitBreakerState returns the current state of the circuit breaker.
func (r *ResilientStorage) CircuitBreakerState() string {
	return string(r.breaker.State())
}

// IsUsingFallback returns true if currently using fallback storage.
func (r *ResilientStorage) IsUsingFallback() bool {
	return r.useFallback.Load()
}

// Compile-time interface check
var _ StorageInterface = (*ResilientStorage)(nil)
