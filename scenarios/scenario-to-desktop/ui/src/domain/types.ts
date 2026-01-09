/**
 * Domain Types
 *
 * Core domain types used across the scenario-to-desktop application.
 * These types represent the fundamental concepts of the domain model.
 */

// ============================================================================
// Build Artifacts
// ============================================================================

/**
 * Represents a single desktop build artifact (executable, installer, etc.).
 * This is the core domain type for download operations.
 */
export interface DesktopBuildArtifact {
  platform?: string;
  file_name: string;
  size_bytes?: number;
  modified_at?: string;
  absolute_path?: string;
  relative_path?: string;
}

// ============================================================================
// Telemetry Events
// ============================================================================

/**
 * A single telemetry event from the runtime.
 * Events are recorded in JSONL format.
 */
export interface TelemetryEvent {
  event?: string;
  timestamp?: string;
  detail?: string;
  [key: string]: unknown;
}

/**
 * Operating system identifiers for telemetry paths.
 */
export type OperatingSystem = "Windows" | "macOS" | "Linux";

/**
 * A platform-specific telemetry file path.
 */
export interface TelemetryFilePath {
  os: OperatingSystem;
  path: string;
}

// ============================================================================
// Download Types
// ============================================================================

/**
 * Valid platform identifiers for desktop builds.
 */
export type Platform = "win" | "mac" | "linux";

/**
 * Display information for a platform.
 */
export interface PlatformDisplayInfo {
  icon: string;
  name: string;
  shortName: string;
}

/**
 * A group of artifacts for a single platform.
 */
export interface PlatformArtifactGroup {
  platform: Platform;
  artifacts: DesktopBuildArtifact[];
  totalSizeBytes: number;
}
