// Package bundleruntime provides the desktop bundle runtime supervisor.
//
// This file contains type aliases re-exported from domain packages
// to provide a convenient public API.
package bundleruntime

import (
	"scenario-to-desktop-runtime/env"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/migrations"
	"scenario-to-desktop-runtime/ports"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/telemetry"
)

// =============================================================================
// Infrastructure Interfaces
// =============================================================================

// Clock abstracts time operations for testing.
type Clock = infra.Clock

// Ticker abstracts time.Ticker for testing.
type Ticker = infra.Ticker

// FileSystem abstracts file operations for testing.
type FileSystem = infra.FileSystem

// File abstracts an open file for testing.
type File = infra.File

// NetworkDialer abstracts network dialing for testing.
type NetworkDialer = infra.NetworkDialer

// HTTPClient abstracts HTTP requests for testing.
type HTTPClient = infra.HTTPClient

// ProcessRunner abstracts process execution for testing.
type ProcessRunner = infra.ProcessRunner

// Process abstracts a running process for testing.
type Process = infra.Process

// CommandRunner abstracts command execution for testing.
type CommandRunner = infra.CommandRunner

// EnvReader abstracts environment variable access for testing.
type EnvReader = infra.EnvReader

// =============================================================================
// Infrastructure Real Implementations
// =============================================================================

// RealClock is the production implementation of Clock.
type RealClock = infra.RealClock

// RealFileSystem is the production implementation of FileSystem.
type RealFileSystem = infra.RealFileSystem

// RealNetworkDialer is the production implementation of NetworkDialer.
type RealNetworkDialer = infra.RealNetworkDialer

// RealHTTPClient is the production implementation of HTTPClient.
type RealHTTPClient = infra.RealHTTPClient

// RealProcessRunner is the production implementation of ProcessRunner.
type RealProcessRunner = infra.RealProcessRunner

// RealCommandRunner is the production implementation of CommandRunner.
type RealCommandRunner = infra.RealCommandRunner

// RealEnvReader is the production implementation of EnvReader.
type RealEnvReader = infra.RealEnvReader

// =============================================================================
// Domain Types
// =============================================================================

// PortAllocator manages dynamic port allocation for services.
type PortAllocator = ports.Allocator

// PortRange defines a range of ports for allocation.
type PortRange = ports.Range

// SecretStore manages secret storage and retrieval.
type SecretStore = secrets.Store

// HealthChecker monitors service health and readiness.
type HealthChecker = health.Checker

// ServiceStatus represents the current status of a service.
type ServiceStatus = health.Status

// GPUDetector detects available GPU hardware.
type GPUDetector = gpu.Detector

// GPUStatus represents the result of GPU detection.
type GPUStatus = gpu.Status

// TelemetryRecorder records events for debugging and analytics.
type TelemetryRecorder = telemetry.Recorder

// EnvRenderer renders environment variable templates.
type EnvRenderer = env.Renderer

// MigrationsState tracks applied migrations per service.
type MigrationsState = migrations.State

// MigrationRunner defines the interface for migration execution.
type MigrationRunner = migrations.Runner

// =============================================================================
// Re-exported Constants and Defaults
// =============================================================================

// Signal constants for cross-platform compatibility.
var (
	Interrupt = infra.Interrupt
	Kill      = infra.Kill
)

// DefaultPortRange is the fallback range when not specified in manifest.
var DefaultPortRange = ports.DefaultRange

