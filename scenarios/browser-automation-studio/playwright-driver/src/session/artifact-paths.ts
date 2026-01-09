/**
 * Artifact Path Resolution
 *
 * Centralizes decisions about where artifacts (HAR, video, trace) are stored.
 *
 * DECISION LOGIC (priority order):
 * 1. Explicit path from spec (artifact_paths.har_path, etc.)
 * 2. Derived from artifact root (artifact_paths.root + standard subdir)
 * 3. Fallback to temp directory
 *
 * This module makes artifact path decisions:
 * - Explicit and named (not inline ternaries)
 * - Testable (pure functions)
 * - Consistent (same fallback logic for all artifact types)
 *
 * CHANGE AXIS: New Artifact Type
 * 1. Add resolver function following existing patterns
 * 2. Add to ARTIFACT_SUBDIRS if using standard subdirectory structure
 * 3. Export from module
 */

import path from 'path';

// =============================================================================
// Types
// =============================================================================

/**
 * Artifact paths configuration from session spec.
 */
export interface ArtifactPathsSpec {
  /** Root directory for all artifacts */
  root?: string;
  /** Explicit path for HAR file */
  har_path?: string;
  /** Explicit directory for video files */
  video_dir?: string;
  /** Explicit path for trace file */
  trace_path?: string;
}

/**
 * Resolved artifact paths.
 * All paths are either resolved or undefined (artifact disabled).
 */
export interface ResolvedArtifactPaths {
  harPath?: string;
  videoDir?: string;
  tracePath?: string;
}

// =============================================================================
// Constants
// =============================================================================

/**
 * Standard subdirectories for each artifact type.
 * Used when deriving paths from artifact root.
 */
const ARTIFACT_SUBDIRS = {
  har: 'har',
  videos: 'videos',
  traces: 'traces',
} as const;

/**
 * File extensions for artifact types.
 */
const ARTIFACT_EXTENSIONS = {
  har: '.har',
  trace: '.zip',
} as const;

// =============================================================================
// Path Resolution Decisions
// =============================================================================

/**
 * Resolve HAR file path.
 *
 * DECISION: Path selection priority
 * 1. Explicit har_path from spec
 * 2. Derived from root: {root}/har/execution-{id}.har
 * 3. Temp fallback: /tmp/har-{id}-{timestamp}.har
 *
 * @param spec - Artifact paths configuration
 * @param executionId - Execution ID for filename
 * @returns Resolved HAR path
 */
export function resolveHarPath(
  spec: ArtifactPathsSpec | undefined,
  executionId: string
): string {
  const artifactRoot = spec?.root?.trim() || '';

  // Priority 1: Explicit path
  if (spec?.har_path) {
    return spec.har_path;
  }

  // Priority 2: Derived from root
  if (artifactRoot) {
    return path.join(
      artifactRoot,
      ARTIFACT_SUBDIRS.har,
      `execution-${executionId}${ARTIFACT_EXTENSIONS.har}`
    );
  }

  // Priority 3: Temp fallback
  return `/tmp/har-${executionId}-${Date.now()}${ARTIFACT_EXTENSIONS.har}`;
}

/**
 * Resolve video directory path.
 *
 * DECISION: Path selection priority
 * 1. Explicit video_dir from spec
 * 2. Derived from root: {root}/videos
 * 3. Temp fallback: /tmp/videos-{id}-{timestamp}
 *
 * Note: Videos use directory (multiple files per session)
 * unlike HAR/trace which are single files.
 *
 * @param spec - Artifact paths configuration
 * @param executionId - Execution ID for fallback directory name
 * @returns Resolved video directory
 */
export function resolveVideoDir(
  spec: ArtifactPathsSpec | undefined,
  executionId: string
): string {
  const artifactRoot = spec?.root?.trim() || '';

  // Priority 1: Explicit directory
  if (spec?.video_dir) {
    return spec.video_dir;
  }

  // Priority 2: Derived from root
  if (artifactRoot) {
    return path.join(artifactRoot, ARTIFACT_SUBDIRS.videos);
  }

  // Priority 3: Temp fallback
  return `/tmp/videos-${executionId}-${Date.now()}`;
}

/**
 * Resolve trace file path.
 *
 * DECISION: Path selection priority
 * 1. Explicit trace_path from spec
 * 2. Derived from root: {root}/traces/execution-{id}.zip
 * 3. Temp fallback: /tmp/trace-{id}-{timestamp}.zip
 *
 * @param spec - Artifact paths configuration
 * @param executionId - Execution ID for filename
 * @returns Resolved trace path
 */
export function resolveTracePath(
  spec: ArtifactPathsSpec | undefined,
  executionId: string
): string {
  const artifactRoot = spec?.root?.trim() || '';

  // Priority 1: Explicit path
  if (spec?.trace_path) {
    return spec.trace_path;
  }

  // Priority 2: Derived from root
  if (artifactRoot) {
    return path.join(
      artifactRoot,
      ARTIFACT_SUBDIRS.traces,
      `execution-${executionId}${ARTIFACT_EXTENSIONS.trace}`
    );
  }

  // Priority 3: Temp fallback
  return `/tmp/trace-${executionId}-${Date.now()}${ARTIFACT_EXTENSIONS.trace}`;
}

// =============================================================================
// Capability Decisions
// =============================================================================

/**
 * Check if HAR recording should be enabled.
 *
 * DECISION: HAR recording is enabled when:
 * - required_capabilities.har is true
 *
 * @param capabilities - Required capabilities from spec
 * @returns true if HAR should be recorded
 */
export function shouldRecordHar(capabilities?: { har?: boolean }): boolean {
  return capabilities?.har === true;
}

/**
 * Check if video recording should be enabled.
 *
 * DECISION: Video recording is enabled when:
 * - required_capabilities.video is true
 *
 * @param capabilities - Required capabilities from spec
 * @returns true if video should be recorded
 */
export function shouldRecordVideo(capabilities?: { video?: boolean }): boolean {
  return capabilities?.video === true;
}

/**
 * Check if tracing should be enabled.
 *
 * DECISION: Tracing is enabled when:
 * - required_capabilities.tracing is true
 *
 * @param capabilities - Required capabilities from spec
 * @returns true if tracing should be enabled
 */
export function shouldEnableTracing(capabilities?: { tracing?: boolean }): boolean {
  return capabilities?.tracing === true;
}

// =============================================================================
// Combined Resolution
// =============================================================================

/**
 * Resolve all artifact paths based on spec and capabilities.
 *
 * Only resolves paths for artifacts that are enabled via capabilities.
 * This prevents unnecessary directory creation for disabled artifacts.
 *
 * @param spec - Artifact paths configuration
 * @param capabilities - Required capabilities
 * @param executionId - Execution ID for filenames
 * @returns Object with resolved paths (only for enabled artifacts)
 */
export function resolveArtifactPaths(
  spec: ArtifactPathsSpec | undefined,
  capabilities: { har?: boolean; video?: boolean; tracing?: boolean } | undefined,
  executionId: string
): ResolvedArtifactPaths {
  const result: ResolvedArtifactPaths = {};

  if (shouldRecordHar(capabilities)) {
    result.harPath = resolveHarPath(spec, executionId);
  }

  if (shouldRecordVideo(capabilities)) {
    result.videoDir = resolveVideoDir(spec, executionId);
  }

  if (shouldEnableTracing(capabilities)) {
    result.tracePath = resolveTracePath(spec, executionId);
  }

  return result;
}

/**
 * Get the parent directory of a file path.
 * Used when ensuring artifact directories exist.
 */
export function getArtifactDir(artifactPath: string): string {
  return path.dirname(artifactPath);
}
