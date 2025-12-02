package bundleruntime

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure conditions.
var (
	// ErrManifestRequired indicates the manifest was not provided.
	ErrManifestRequired = errors.New("manifest is required")

	// ErrCyclicDependency indicates a cycle was detected in service dependencies.
	ErrCyclicDependency = errors.New("cycle detected in dependencies")

	// ErrNoFreePorts indicates no available ports could be found in the specified range.
	ErrNoFreePorts = errors.New("no free ports in range")

	// ErrGPURequired indicates a required GPU was not detected.
	ErrGPURequired = errors.New("gpu required but not available")
)

// ServiceError represents an error related to a specific service.
type ServiceError struct {
	ServiceID string
	Op        string // Operation that failed (e.g., "start", "health_check", "migration")
	Err       error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("service %s: %s: %v", e.ServiceID, e.Op, e.Err)
	}
	return fmt.Sprintf("service %s: %s failed", e.ServiceID, e.Op)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// NewServiceError creates a new ServiceError.
func NewServiceError(serviceID, op string, err error) *ServiceError {
	return &ServiceError{ServiceID: serviceID, Op: op, Err: err}
}

// SecretError represents an error related to secret management.
type SecretError struct {
	SecretID  string
	ServiceID string
	Reason    string
}

func (e *SecretError) Error() string {
	if e.ServiceID != "" {
		return fmt.Sprintf("secret %s for service %s: %s", e.SecretID, e.ServiceID, e.Reason)
	}
	return fmt.Sprintf("secret %s: %s", e.SecretID, e.Reason)
}

// NewSecretError creates a new SecretError.
func NewSecretError(secretID, serviceID, reason string) *SecretError {
	return &SecretError{SecretID: secretID, ServiceID: serviceID, Reason: reason}
}

// PortError represents an error related to port allocation.
type PortError struct {
	ServiceID string
	PortName  string
	Range     PortRange
	Reason    string
}

func (e *PortError) Error() string {
	if e.Range.Min > 0 && e.Range.Max > 0 {
		return fmt.Sprintf("port %s for service %s in range %d-%d: %s",
			e.PortName, e.ServiceID, e.Range.Min, e.Range.Max, e.Reason)
	}
	return fmt.Sprintf("port %s for service %s: %s", e.PortName, e.ServiceID, e.Reason)
}

// NewPortError creates a new PortError.
func NewPortError(serviceID, portName string, rng PortRange, reason string) *PortError {
	return &PortError{ServiceID: serviceID, PortName: portName, Range: rng, Reason: reason}
}

// AssetError represents an error related to asset verification.
type AssetError struct {
	ServiceID string
	Path      string
	Reason    string
	Expected  int64 // For size-related errors
	Actual    int64
}

func (e *AssetError) Error() string {
	if e.Expected > 0 && e.Actual > 0 {
		return fmt.Sprintf("asset %s for service %s: %s (expected %d bytes, got %d)",
			e.Path, e.ServiceID, e.Reason, e.Expected, e.Actual)
	}
	return fmt.Sprintf("asset %s for service %s: %s", e.Path, e.ServiceID, e.Reason)
}

// NewAssetError creates a new AssetError.
func NewAssetError(serviceID, path, reason string) *AssetError {
	return &AssetError{ServiceID: serviceID, Path: path, Reason: reason}
}

// MigrationError represents an error during migration execution.
type MigrationError struct {
	ServiceID string
	Version   string
	Err       error
}

func (e *MigrationError) Error() string {
	return fmt.Sprintf("migration %s for service %s: %v", e.Version, e.ServiceID, e.Err)
}

func (e *MigrationError) Unwrap() error {
	return e.Err
}

// NewMigrationError creates a new MigrationError.
func NewMigrationError(serviceID, version string, err error) *MigrationError {
	return &MigrationError{ServiceID: serviceID, Version: version, Err: err}
}

// HealthCheckError represents a failed health check.
type HealthCheckError struct {
	ServiceID string
	Type      string // "http", "tcp", "command", "log_match"
	Attempts  int
	LastErr   error
}

func (e *HealthCheckError) Error() string {
	if e.LastErr != nil {
		return fmt.Sprintf("health check %s for service %s failed after %d attempts: %v",
			e.Type, e.ServiceID, e.Attempts, e.LastErr)
	}
	return fmt.Sprintf("health check %s for service %s failed after %d attempts",
		e.Type, e.ServiceID, e.Attempts)
}

func (e *HealthCheckError) Unwrap() error {
	return e.LastErr
}

// NewHealthCheckError creates a new HealthCheckError.
func NewHealthCheckError(serviceID, checkType string, attempts int, lastErr error) *HealthCheckError {
	return &HealthCheckError{ServiceID: serviceID, Type: checkType, Attempts: attempts, LastErr: lastErr}
}

// IsServiceError checks if an error is a ServiceError for a specific service.
func IsServiceError(err error, serviceID string) bool {
	var se *ServiceError
	if errors.As(err, &se) {
		return se.ServiceID == serviceID
	}
	return false
}

// IsSecretError checks if an error is related to secrets.
func IsSecretError(err error) bool {
	var se *SecretError
	return errors.As(err, &se)
}

// IsAssetError checks if an error is related to assets.
func IsAssetError(err error) bool {
	var ae *AssetError
	return errors.As(err, &ae)
}

// IsMigrationError checks if an error is related to migrations.
func IsMigrationError(err error) bool {
	var me *MigrationError
	return errors.As(err, &me)
}
