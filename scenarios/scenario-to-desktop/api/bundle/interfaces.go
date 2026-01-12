// Package bundle provides bundle packaging services for desktop applications.
// This domain handles packaging service binaries, assets, and runtime components
// into a distributable bundle for Electron wrapper integration.
package bundle

import (
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// Packager orchestrates bundle packaging operations.
type Packager interface {
	// Package packages a bundle from the given app path and manifest.
	Package(appPath, manifestPath string, requestedPlatforms []string) (*PackageResult, error)
}

// RuntimeResolver locates the runtime source directory.
type RuntimeResolver interface {
	// Resolve locates and returns the absolute path to the runtime source directory.
	Resolve() (string, error)
}

// RuntimeBuilder builds runtime binaries for target platforms.
type RuntimeBuilder interface {
	// Build compiles a runtime binary for the specified platform.
	Build(srcDir, outPath, goos, goarch, target string) error
}

// ServiceCompiler compiles service binaries for target platforms.
type ServiceCompiler interface {
	// Compile compiles a service binary for the specified platform.
	Compile(svc bundlemanifest.Service, platform, manifestRoot string) (string, error)
}

// CLIStager stages CLI helpers into the bundle.
type CLIStager interface {
	// Stage copies CLI binaries and creates shims for the specified platform.
	Stage(bundleRoot, platform string) error
}

// SizeCalculator calculates bundle sizes and generates warnings.
type SizeCalculator interface {
	// Calculate walks the bundle directory and returns size information.
	Calculate(bundleDir string) (totalSize int64, largeFiles []LargeFileInfo)

	// CheckWarning returns a warning if the bundle exceeds size thresholds.
	CheckWarning(totalSize int64, largeFiles []LargeFileInfo) *SizeWarning
}

// PlatformResolver resolves platform-specific configurations.
type PlatformResolver interface {
	// ParseKey parses a platform key into GOOS and GOARCH.
	ParseKey(key string) (goos, goarch string, err error)

	// NormalizeRuntime normalizes a platform key for runtime staging.
	NormalizeRuntime(platform string) string

	// RuntimeBinaryName returns the runtime binary name for a GOOS.
	RuntimeBinaryName(goos string) string

	// RuntimeCtlBinaryName returns the runtimectl binary name for a GOOS.
	RuntimeCtlBinaryName(goos string) string

	// ResolveBinaryForPlatform resolves the binary entry for a service on a platform.
	ResolveBinaryForPlatform(svc bundlemanifest.Service, platform string) (bundlemanifest.Binary, bool)
}

// FileOperations provides file system operations for bundling.
type FileOperations interface {
	// CopyFile copies a single file.
	CopyFile(src, dst string) error

	// CopyPath copies a file or directory.
	CopyPath(src, dst string) error

	// WithinBase checks if target path is within base directory.
	WithinBase(base, target string) bool

	// NormalizeBundlePath strips leading parent directory traversals.
	NormalizeBundlePath(rel string) string
}
