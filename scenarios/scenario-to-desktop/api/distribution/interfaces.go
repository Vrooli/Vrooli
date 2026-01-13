package distribution

import (
	"context"
	"io"
	"os"
	"time"
)

// Uploader abstracts cloud storage upload operations.
// Implementations exist for S3, R2, and other S3-compatible services.
type Uploader interface {
	// Upload uploads a file to the configured destination.
	Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error)

	// ValidateCredentials verifies the credentials work.
	ValidateCredentials(ctx context.Context) error

	// ListObjects lists objects at a given prefix (for verification).
	ListObjects(ctx context.Context, prefix string, maxKeys int) ([]ObjectInfo, error)

	// DeleteObject removes an object (for cleanup/rollback).
	DeleteObject(ctx context.Context, key string) error

	// GetPresignedURL generates a temporary download URL.
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// GetPublicURL returns the public URL for an object (if CDN configured).
	GetPublicURL(key string) string
}

// Service orchestrates distribution operations.
type Service interface {
	// Distribute uploads build artifacts to specified targets.
	// If targetNames is empty, uploads to all enabled targets.
	Distribute(ctx context.Context, req *DistributeRequest) (*DistributeResponse, error)

	// GetDistributionStatus retrieves the status of a distribution operation.
	GetDistributionStatus(distributionID string) (*DistributionStatus, bool)

	// ListDistributions returns all tracked distribution operations.
	ListDistributions() []*DistributionStatus

	// CancelDistribution cancels an in-progress distribution.
	CancelDistribution(distributionID string) bool

	// ValidateTargets validates all (or specified) targets.
	ValidateTargets(ctx context.Context, targetNames []string) *ValidationResult
}

// Repository persists distribution configurations.
// Unlike signing (per-scenario), distribution config is GLOBAL.
type Repository interface {
	// Get retrieves the global distribution config.
	Get(ctx context.Context) (*DistributionConfig, error)

	// Save stores the global distribution config.
	Save(ctx context.Context, config *DistributionConfig) error

	// GetTarget retrieves a specific target config.
	GetTarget(ctx context.Context, name string) (*DistributionTarget, error)

	// SaveTarget creates or updates a specific target.
	SaveTarget(ctx context.Context, name string, target *DistributionTarget) error

	// DeleteTarget removes a target.
	DeleteTarget(ctx context.Context, name string) error

	// ListTargets returns all target names.
	ListTargets(ctx context.Context) ([]string, error)

	// GetPath returns the path to the distribution.json file.
	GetPath() string
}

// Store persists distribution operation status.
type Store interface {
	// Save creates or updates a distribution status.
	Save(status *DistributionStatus)

	// Get retrieves a distribution status by ID.
	Get(distributionID string) (*DistributionStatus, bool)

	// Update updates a distribution status.
	Update(distributionID string, fn func(status *DistributionStatus)) bool

	// List returns all distribution statuses.
	List() []*DistributionStatus

	// Delete removes a distribution status.
	Delete(distributionID string) bool
}

// UploaderFactory creates Uploader instances for targets.
type UploaderFactory interface {
	// Create creates an Uploader for the given target config.
	Create(target *DistributionTarget, envReader EnvironmentReader) (Uploader, error)
}

// EnvironmentReader abstracts environment variable access for testing.
type EnvironmentReader interface {
	// GetEnv retrieves an environment variable value.
	GetEnv(key string) string

	// LookupEnv retrieves an environment variable and reports if it exists.
	LookupEnv(key string) (string, bool)
}

// FileSystem abstracts file operations for testing.
type FileSystem interface {
	// Exists checks if a file exists.
	Exists(path string) bool

	// ReadFile reads file content.
	ReadFile(path string) ([]byte, error)

	// WriteFile writes file content.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// Stat returns file info.
	Stat(path string) (os.FileInfo, error)

	// Open opens a file for reading.
	Open(path string) (io.ReadCloser, error)

	// MkdirAll creates directories.
	MkdirAll(path string, perm os.FileMode) error

	// Remove removes a file.
	Remove(path string) error
}

// TimeProvider abstracts time for testing.
type TimeProvider interface {
	Now() time.Time
	NowUnix() int64
}

// IDGenerator generates unique identifiers.
type IDGenerator interface {
	Generate() string
}

// Logger provides structured logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// CancelManager manages cancellation functions.
type CancelManager interface {
	Set(distributionID string, cancel context.CancelFunc)
	Get(distributionID string) (context.CancelFunc, bool)
	Delete(distributionID string)
}

// RealEnvironmentReader implements EnvironmentReader using os package.
type RealEnvironmentReader struct{}

// GetEnv retrieves an environment variable value.
func (r *RealEnvironmentReader) GetEnv(key string) string {
	return os.Getenv(key)
}

// LookupEnv retrieves an environment variable and reports if it exists.
func (r *RealEnvironmentReader) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// RealFileSystem implements FileSystem using os package.
type RealFileSystem struct{}

// Exists checks if a file exists.
func (r *RealFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads file content.
func (r *RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes file content.
func (r *RealFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat returns file info.
func (r *RealFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// Open opens a file for reading.
func (r *RealFileSystem) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// MkdirAll creates directories.
func (r *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Remove removes a file.
func (r *RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// RealTimeProvider implements TimeProvider using time package.
type RealTimeProvider struct{}

// Now returns the current time.
func (r *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// NowUnix returns the current Unix timestamp.
func (r *RealTimeProvider) NowUnix() int64 {
	return time.Now().Unix()
}
