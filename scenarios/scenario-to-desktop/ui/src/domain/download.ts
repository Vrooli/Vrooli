/**
 * Download Domain Logic
 *
 * Pure functions for download-related operations.
 * This module handles platform artifact organization, file size formatting,
 * and download URL resolution. No side effects - all testable in isolation.
 */

import type {
  DesktopBuildArtifact,
  Platform,
  PlatformArtifactGroup,
  PlatformDisplayInfo
} from "./types";

// Re-export types for convenience
export type { DesktopBuildArtifact, Platform, PlatformArtifactGroup, PlatformDisplayInfo };

// ============================================================================
// Types
// ============================================================================

export interface DownloadResolverOptions {
  scenarioName: string;
  platform: Platform;
}

// ============================================================================
// Constants
// ============================================================================

export const PLATFORM_DISPLAY_INFO: Record<Platform, PlatformDisplayInfo> = {
  win: { icon: "🪟", name: "Windows", shortName: "Win" },
  mac: { icon: "🍎", name: "macOS", shortName: "Mac" },
  linux: { icon: "🐧", name: "Linux", shortName: "Linux" }
};

export const VALID_PLATFORMS: Platform[] = ["win", "mac", "linux"];

// ============================================================================
// Platform Validation
// ============================================================================

/**
 * Check if a string is a valid platform identifier.
 */
export function isValidPlatform(value: string): value is Platform {
  return VALID_PLATFORMS.includes(value as Platform);
}

/**
 * Safely convert a string to Platform, returning undefined if invalid.
 */
export function parsePlatform(value: string | undefined): Platform | undefined {
  if (value && isValidPlatform(value)) {
    return value;
  }
  return undefined;
}

// ============================================================================
// Platform Display
// ============================================================================

/**
 * Get display information for a platform.
 */
export function getPlatformDisplayInfo(platform: string): PlatformDisplayInfo {
  if (isValidPlatform(platform)) {
    return PLATFORM_DISPLAY_INFO[platform];
  }
  return { icon: "📦", name: "Unknown", shortName: "?" };
}

/**
 * Get the icon for a platform.
 */
export function getPlatformIcon(platform: string): string {
  return getPlatformDisplayInfo(platform).icon;
}

/**
 * Get the display name for a platform.
 */
export function getPlatformName(platform: string): string {
  return getPlatformDisplayInfo(platform).name;
}

// ============================================================================
// Artifact Grouping
// ============================================================================

/**
 * Group artifacts by platform.
 * Returns a map of platform to artifacts with computed totals.
 */
export function groupArtifactsByPlatform(
  artifacts: DesktopBuildArtifact[] | undefined
): Map<Platform | "unknown", PlatformArtifactGroup> {
  const groups = new Map<Platform | "unknown", PlatformArtifactGroup>();

  if (!artifacts?.length) {
    return groups;
  }

  for (const artifact of artifacts) {
    const platformKey = parsePlatform(artifact.platform) ?? "unknown";

    if (!groups.has(platformKey)) {
      groups.set(platformKey, {
        platform: platformKey as Platform,
        artifacts: [],
        totalSizeBytes: 0
      });
    }

    const group = groups.get(platformKey)!;
    group.artifacts.push(artifact);
    group.totalSizeBytes += artifact.size_bytes ?? 0;
  }

  return groups;
}

/**
 * Get a sorted list of platform artifact groups.
 * Sorts platforms in a consistent order: win, mac, linux, then unknown.
 */
export function getSortedPlatformGroups(
  artifacts: DesktopBuildArtifact[] | undefined
): PlatformArtifactGroup[] {
  const groups = groupArtifactsByPlatform(artifacts);
  const platformOrder: (Platform | "unknown")[] = [...VALID_PLATFORMS, "unknown"];

  return platformOrder
    .filter((platform) => groups.has(platform))
    .map((platform) => groups.get(platform)!);
}

// ============================================================================
// File Size Formatting
// ============================================================================

const SIZE_UNITS = ["Bytes", "KB", "MB", "GB", "TB"];
const SIZE_BASE = 1024;

/**
 * Format bytes into a human-readable string.
 */
export function formatBytes(bytes: number | undefined): string {
  if (bytes === undefined || bytes === 0) {
    return "0 Bytes";
  }

  const unitIndex = Math.floor(Math.log(bytes) / Math.log(SIZE_BASE));
  const clampedIndex = Math.min(unitIndex, SIZE_UNITS.length - 1);
  const value = bytes / Math.pow(SIZE_BASE, clampedIndex);
  const rounded = Math.round(value * 100) / 100;

  return `${rounded} ${SIZE_UNITS[clampedIndex]}`;
}

/**
 * Compute total size of artifacts in bytes.
 */
export function computeTotalArtifactSize(artifacts: DesktopBuildArtifact[] | undefined): number {
  if (!artifacts?.length) {
    return 0;
  }
  return artifacts.reduce((sum, artifact) => sum + (artifact.size_bytes ?? 0), 0);
}

// ============================================================================
// Download URL Building
// ============================================================================

/**
 * Build the download API path for a scenario/platform combination.
 * This returns the path segment only - the full URL is constructed by the API client.
 */
export function buildDownloadPath(options: DownloadResolverOptions): string {
  const { scenarioName, platform } = options;

  if (!scenarioName) {
    throw new Error("scenarioName is required");
  }
  if (!isValidPlatform(platform)) {
    throw new Error(`Invalid platform: ${platform}. Must be one of: ${VALID_PLATFORMS.join(", ")}`);
  }

  return `/desktop/download/${encodeURIComponent(scenarioName)}/${platform}`;
}

/**
 * Check if artifacts are available for download for a given platform.
 */
export function hasDownloadableArtifacts(
  artifacts: DesktopBuildArtifact[] | undefined,
  platform: Platform
): boolean {
  if (!artifacts?.length) {
    return false;
  }
  return artifacts.some((artifact) => artifact.platform === platform);
}

/**
 * Get available platforms from artifacts.
 */
export function getAvailablePlatforms(artifacts: DesktopBuildArtifact[] | undefined): Platform[] {
  if (!artifacts?.length) {
    return [];
  }

  const platforms = new Set<Platform>();
  for (const artifact of artifacts) {
    const platform = parsePlatform(artifact.platform);
    if (platform) {
      platforms.add(platform);
    }
  }

  // Return in consistent order
  return VALID_PLATFORMS.filter((p) => platforms.has(p));
}
