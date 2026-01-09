/**
 * Pure utility functions for preflight validation.
 * These functions have no side effects and can be tested in isolation.
 */

// ============================================================================
// Duration & Time Formatting
// ============================================================================

/**
 * Format a duration in milliseconds to a human-readable string.
 * Returns "n/a" for invalid inputs.
 *
 * @example
 * formatDuration(5000) // "5s"
 * formatDuration(65000) // "1m 5s"
 * formatDuration(3665000) // "1h 1m"
 */
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) {
    return "n/a";
  }
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

/**
 * Parse an ISO timestamp string to a Unix timestamp in milliseconds.
 * Returns null for invalid or very old dates.
 */
export function parseTimestamp(value?: string): number | null {
  if (!value) {
    return null;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime()) || date.getFullYear() < 2000) {
    return null;
  }
  return date.getTime();
}

/**
 * Format an ISO timestamp string to a localized time string.
 * Returns empty string for invalid timestamps.
 */
export function formatTimestamp(value?: string): string {
  const ts = parseTimestamp(value);
  if (!ts) {
    return "";
  }
  return new Date(ts).toLocaleTimeString();
}

// ============================================================================
// Size & Count Formatting
// ============================================================================

/**
 * Format a byte count to a human-readable string (B, KB, MB).
 * Returns empty string for invalid or zero values.
 */
export function formatBytes(value?: number): string {
  if (!value || value <= 0) {
    return "";
  }
  if (value < 1024) {
    return `${value} B`;
  }
  const kb = value / 1024;
  if (kb < 1024) {
    return `${kb.toFixed(1)} KB`;
  }
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

/**
 * Count the number of non-empty lines in a text string.
 */
export function countLines(text?: string): number {
  if (!text) {
    return 0;
  }
  const lines = text.split(/\r?\n/);
  if (lines.length === 0) {
    return 0;
  }
  const last = lines[lines.length - 1];
  return last === "" ? lines.length - 1 : lines.length;
}

// ============================================================================
// URL & Port Extraction
// ============================================================================

/**
 * Extract a localhost URL from a "listening on <port>" detail message.
 * Returns null if no valid port is found.
 */
export function getListenURL(detail?: string): string | null {
  if (!detail) {
    return null;
  }
  const match = detail.match(/listening on (\d+)/i);
  if (!match) {
    return null;
  }
  const port = Number.parseInt(match[1], 10);
  if (!Number.isFinite(port) || port <= 0) {
    return null;
  }
  return `http://localhost:${port}`;
}

export type ServiceURLResult = {
  url: string;
  port: number;
  portName: string;
} | null;

/**
 * Get the URL for a service from the ports map.
 * Optionally specify a preferred port name; otherwise uses a priority list.
 *
 * @param serviceId - The service identifier
 * @param ports - Map of service ID -> port name -> port number
 * @param preferredPortName - Optional specific port to use
 */
export function getServiceURL(
  serviceId: string,
  ports?: Record<string, Record<string, number>>,
  preferredPortName?: string
): ServiceURLResult {
  if (!ports) {
    return null;
  }
  const portMap = ports[serviceId];
  if (!portMap) {
    return null;
  }
  if (preferredPortName) {
    const port = portMap[preferredPortName];
    if (Number.isFinite(port) && port > 0) {
      return { url: `http://localhost:${port}`, port, portName: preferredPortName };
    }
    return null;
  }
  const preferred = ["ui", "api", "http"];
  let portName = preferred.find((name) => Number.isFinite(portMap[name]));
  if (!portName) {
    const names = Object.keys(portMap);
    portName = names.length > 0 ? names[0] : undefined;
  }
  if (!portName) {
    return null;
  }
  const port = portMap[portName];
  if (!Number.isFinite(port) || port <= 0) {
    return null;
  }
  return { url: `http://localhost:${port}`, port, portName };
}

// ============================================================================
// Manifest Parsing
// ============================================================================

export type ManifestHealthConfig = {
  type?: string;
  path?: string;
  portName?: string;
};

/**
 * Extract health check configuration from a bundle manifest for a specific service.
 */
export function getManifestHealthConfig(manifest: unknown, serviceId: string): ManifestHealthConfig | null {
  if (!manifest || typeof manifest !== "object") {
    return null;
  }
  const services = (manifest as { services?: unknown }).services;
  if (!Array.isArray(services)) {
    return null;
  }
  const service = services.find((entry) => {
    if (!entry || typeof entry !== "object") {
      return false;
    }
    const id = (entry as { id?: unknown }).id;
    return typeof id === "string" && id === serviceId;
  }) as { health?: { type?: unknown; path?: unknown; port_name?: unknown } } | undefined;
  if (!service || !service.health || typeof service.health !== "object") {
    return null;
  }
  const type = typeof service.health.type === "string" ? service.health.type : undefined;
  const path = typeof service.health.path === "string" ? service.health.path : undefined;
  const portName = typeof service.health.port_name === "string" ? service.health.port_name : undefined;
  return { type, path, portName };
}

/**
 * Normalize a health check path to ensure it starts with "/".
 */
export function normalizeHealthPath(path?: string): string | null {
  const trimmed = path?.trim();
  if (!trimmed) {
    return null;
  }
  return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}

// ============================================================================
// Port Summary
// ============================================================================

/**
 * Generate a summary string of all service ports.
 * Format: "service1(name1:port1, name2:port2) · service2(name3:port3)"
 */
export function formatPortSummary(ports?: Record<string, Record<string, number>>): string {
  if (!ports) {
    return "";
  }
  return Object.entries(ports)
    .map(([svc, portMap]) => {
      const pairs = Object.entries(portMap)
        .map(([name, port]) => `${name}:${port}`)
        .join(", ");
      return `${svc}(${pairs})`;
    })
    .join(" · ");
}

// ============================================================================
// Bundle Path Utilities
// ============================================================================

/**
 * Extract the bundle root directory from a manifest path.
 * Removes the filename component to get the parent directory.
 */
export function getBundleRootFromManifestPath(manifestPath: string): string {
  const trimmed = manifestPath.trim();
  if (!trimmed) {
    return "";
  }
  return trimmed.replace(/[/\\][^/\\]+$/, "");
}

/**
 * Check if missing artifacts might be due to a bundle root mismatch.
 * Returns true if artifacts are missing and the path doesn't look like staging.
 */
export function detectLikelyRootMismatch(
  validationValid: boolean | undefined,
  missingAssetsCount: number,
  missingBinariesCount: number,
  bundleManifestPath: string
): boolean {
  const hasMissingArtifacts = Boolean(
    validationValid === false && (missingAssetsCount > 0 || missingBinariesCount > 0)
  );
  return Boolean(
    hasMissingArtifacts &&
    bundleManifestPath.trim() &&
    !bundleManifestPath.includes("scenario-to-desktop/data/staging")
  );
}
