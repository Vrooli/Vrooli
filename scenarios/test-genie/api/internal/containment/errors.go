package containment

import "errors"

var (
	// ErrNoCommand is returned when ExecutionConfig has no command specified.
	ErrNoCommand = errors.New("no command specified")

	// ErrContainmentUnavailable is returned when the requested containment is not available.
	ErrContainmentUnavailable = errors.New("containment method unavailable")

	// ErrInvalidConfig is returned when the execution config is invalid.
	ErrInvalidConfig = errors.New("invalid execution configuration")

	// ErrDockerNotRunning is returned when Docker is installed but not running.
	ErrDockerNotRunning = errors.New("Docker is installed but not running")

	// ErrDockerImageNotFound is returned when the required Docker image is missing.
	ErrDockerImageNotFound = errors.New("Docker image not found")
)
