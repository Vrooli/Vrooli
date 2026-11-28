import type { DownloadAsset, SigningNotice } from '../../../shared/api';

/**
 * Detected platform type
 */
export type DetectedPlatform = 'windows' | 'mac' | 'linux' | 'unknown';

/**
 * Detects the current platform based on the user agent.
 * @returns The detected platform identifier
 */
export function detectPlatform(): DetectedPlatform {
  if (typeof navigator === 'undefined') return 'unknown';

  const ua = navigator.userAgent.toLowerCase();

  if (ua.includes('windows')) return 'windows';
  if (ua.includes('macintosh') || ua.includes('mac os')) return 'mac';
  if (ua.includes('linux')) return 'linux';

  return 'unknown';
}

/**
 * Generates a unique key for a download asset
 * @param download - The download asset
 * @returns A unique string key for the asset
 */
export function getDownloadAssetKey(download: DownloadAsset): string {
  if (typeof download.id === 'number') {
    return `asset-${String(download.id)}`;
  }
  const version = download.release_version || 'unknown';
  const artifact = download.artifact_url || 'na';
  return `app-${download.app_key}-${download.platform}-${version}-${artifact}`;
}

/**
 * Sanitizes an artifact URL to prevent XSS and other injection attacks
 * @param artifactUrl - The URL to sanitize
 * @returns Sanitized URL or empty string if invalid
 */
export function sanitizeArtifactUrl(artifactUrl?: string): string {
  if (typeof artifactUrl !== 'string') return '';
  const trimmed = artifactUrl.trim();
  if (!trimmed) return '';
  const lower = trimmed.toLowerCase();
  if (lower.startsWith('javascript:') || lower.startsWith('data:') || lower.startsWith('vbscript:')) return '';
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  if (/^[./]/.test(trimmed) || !trimmed.includes(':')) return trimmed;
  return '';
}

/**
 * Opens a download URL in a new window
 * @param url - The URL to open
 * @returns True if the window was opened successfully
 */
export function openDownloadWindow(url: string): boolean {
  if (typeof window === 'undefined' || typeof window.open !== 'function') return false;
  const target = window.open(url, '_blank', 'noopener,noreferrer');
  return target !== null;
}

/**
 * Extracts a human-readable variant label from an installer's artifact URL or release notes
 * @param installer - The download asset
 * @returns Human-readable variant label
 */
export function getVariantLabel(installer: DownloadAsset): string {
  const url = installer.artifact_url.toLowerCase();
  const notes = installer.release_notes?.toLowerCase() ?? '';

  // Check for specific formats
  if (url.includes('.appimage') || notes.includes('appimage')) return 'AppImage';
  if (url.includes('.deb') || notes.includes('.deb')) return '.deb';
  if (url.includes('.rpm') || notes.includes('.rpm')) return '.rpm';
  if (url.includes('.tar.gz') || url.includes('.tgz')) return '.tar.gz';
  if (url.includes('.dmg') || notes.includes('.dmg')) return '.dmg';
  if (url.includes('.pkg') || notes.includes('.pkg')) return '.pkg';
  if (url.includes('.exe') || notes.includes('.exe')) return 'Installer';
  if (url.includes('.msi') || notes.includes('.msi')) return '.msi';
  if (url.includes('.zip')) return '.zip';

  // Check for architecture
  if (url.includes('arm64') || url.includes('aarch64') || notes.includes('arm')) return 'ARM64';
  if (url.includes('x64') || url.includes('x86_64') || url.includes('amd64')) return '64-bit';
  if (url.includes('x86') || url.includes('i386') || url.includes('i686')) return '32-bit';

  // Fallback to version
  return installer.release_version ? `v${installer.release_version}` : 'Download';
}

/**
 * Platform display labels
 */
export const PLATFORM_LABELS: Record<string, string> = {
  windows: 'Windows',
  mac: 'macOS',
  linux: 'Linux',
};

/**
 * Gets the display label for a platform
 * @param platform - Platform identifier
 * @returns Human-readable label
 */
export function getPlatformLabel(platform: string): string {
  return PLATFORM_LABELS[platform.toLowerCase()] ?? platform;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : undefined;
}

/** Unwraps the protobuf JSON envelopes that may wrap catalog metadata values. */
function unwrapMetadataValue(value: unknown): unknown {
  const record = asRecord(value);
  if (!record) return value;
  if (record.fields) return asRecord(record.fields);
  if ('stringValue' in record) return record.stringValue;
  if ('string_value' in record) return record.string_value;
  if ('boolValue' in record) return record.boolValue;
  if ('bool_value' in record) return record.bool_value;
  if ('structValue' in record) return unwrapMetadataValue(record.structValue);
  if ('struct_value' in record) return unwrapMetadataValue(record.struct_value);
  return value;
}

function noticeText(value: unknown): string {
  const unwrapped = unwrapMetadataValue(value);
  return typeof unwrapped === 'string' ? unwrapped.trim() : '';
}

/**
 * Reads the validated signing notice from a catalog entry. Returns undefined
 * when the notice is absent, explicitly suppressed, or missing required copy.
 */
export function getSigningNotice(metadata?: Record<string, unknown>): SigningNotice | undefined {
  const raw = asRecord(unwrapMetadataValue(metadata?.signing_notice));
  if (!raw) return undefined;
  if (raw.enabled === false) return undefined;
  const title = noticeText(raw.title);
  const body = noticeText(raw.body);
  if (!title || !body) return undefined;
  const severity = noticeText(raw.severity).toLowerCase() === 'warning' ? 'warning' : 'info';
  const linkLabel = noticeText(raw.link_label ?? raw.linkLabel);
  const linkUrl = noticeText(raw.link_url ?? raw.linkUrl);
  return {
    title,
    body,
    severity,
    ...(linkLabel ? { link_label: linkLabel } : {}),
    ...(linkUrl ? { link_url: linkUrl } : {}),
  };
}

/**
 * Resolves the notice to show for the selected release. A platform-specific
 * notice wins over the app default, and an explicit `enabled: false` on the
 * platform suppresses the app default entirely.
 */
export function resolveSigningNotice(
  appMetadata?: Record<string, unknown>,
  assetMetadata?: Record<string, unknown>,
): SigningNotice | undefined {
  const assetRaw = asRecord(unwrapMetadataValue(assetMetadata?.signing_notice));
  if (assetRaw) {
    if (assetRaw.enabled === false) return undefined;
    return getSigningNotice(assetMetadata);
  }
  return getSigningNotice(appMetadata);
}
