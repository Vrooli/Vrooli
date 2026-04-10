// Package pipeline provides artifact readiness helpers.
//
// This file extracts artifact filtering logic from stage implementations
// into named, reusable functions. These helpers centralize the definition
// of what makes an artifact "ready" for use in subsequent stages.
package pipeline

import "scenario-to-desktop-api/build"

// IsArtifactReady returns true if a platform build result is ready for use.
// A result is considered ready if:
//   - Status is "ready" (indicating successful build)
//   - Artifact path is non-empty
func IsArtifactReady(result *build.PlatformResult) bool {
	if result == nil {
		return false
	}
	return result.Status == BuildStatusReady && result.Artifact != ""
}

// GetReadyArtifacts extracts artifact paths from platform results that are ready.
// Returns a map of platform name to artifact path for all successful builds.
func GetReadyArtifacts(platformResults map[string]*build.PlatformResult) map[string]string {
	artifacts := make(map[string]string)
	if platformResults == nil {
		return artifacts
	}
	for platform, result := range platformResults {
		if IsArtifactReady(result) {
			artifacts[platform] = result.Artifact
		}
	}
	return artifacts
}

// CountReadyArtifacts returns the number of platform results that are ready.
func CountReadyArtifacts(platformResults map[string]*build.PlatformResult) int {
	count := 0
	for _, result := range platformResults {
		if IsArtifactReady(result) {
			count++
		}
	}
	return count
}

// FindFirstReadyArtifact returns the first ready artifact found, along with its platform.
// Returns empty strings if no ready artifact is found.
// Useful for smoke testing when any single artifact is sufficient.
func FindFirstReadyArtifact(platformResults map[string]*build.PlatformResult) (platform, artifactPath string) {
	for p, result := range platformResults {
		if IsArtifactReady(result) {
			return p, result.Artifact
		}
	}
	return "", ""
}

// FindArtifactForPlatform returns the artifact path for a specific platform if ready.
// Returns empty string if the platform is not found or not ready.
func FindArtifactForPlatform(platformResults map[string]*build.PlatformResult, targetPlatform string) string {
	if result, ok := platformResults[targetPlatform]; ok && IsArtifactReady(result) {
		return result.Artifact
	}
	return ""
}
