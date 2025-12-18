const ABSOLUTE_URL_PATTERN = /^[a-zA-Z][a-zA-Z\d+.-]*:/;

export const stripApiSuffix = (value?: string | null): string | null => {
  if (!value) {
    return null;
  }

  try {
    const parsed = new URL(value);
    const cleanedPath = parsed.pathname
      .replace(/\/api\/v1\/?$/i, '')
      .replace(/\/+$/, '');
    return cleanedPath ? `${parsed.origin}${cleanedPath}` : parsed.origin;
  } catch {
    const cleaned = value.replace(/\/api\/v1\/?$/i, '').replace(/\/+$/, '');
    return cleaned.length > 0 ? cleaned : null;
  }
};

export const withAssetBasePath = (value: string) => {
  if (!value || ABSOLUTE_URL_PATTERN.test(value)) {
    return value;
  }
  const base = import.meta.env.BASE_URL || '/';
  if (base === '/' || value.startsWith(base)) {
    return value;
  }
  const normalizedBase = base.endsWith('/') ? base : `${base}/`;
  const normalizedValue = value.startsWith('/') ? value.slice(1) : value;
  return `${normalizedBase}${normalizedValue}`;
};

export const formatCapturedLabel = (count: number, noun: string): string => {
  const rounded = Math.round(count);
  const suffix = rounded === 1 ? '' : 's';
  return `${rounded} ${noun}${suffix}`;
};

export const normalizePreviewStatus = (value?: string | null): string => {
  if (!value) {
    return '';
  }
  return value.trim().toLowerCase();
};

export const describePreviewStatusMessage = (
  status: string,
  fallback?: string | null,
  metrics?: { capturedFrames?: number; assetCount?: number },
): string => {
  if (fallback && fallback.trim().length > 0) {
    return fallback.trim();
  }
  const capturedFrames = metrics?.capturedFrames ?? 0;
  switch (status) {
    case 'pending':
      if (capturedFrames > 0) {
        return `Replay export pending – ${formatCapturedLabel(capturedFrames, 'frame')} captured so far`;
      }
      return 'Replay export pending – timeline frames not captured yet';
    case 'unavailable':
      if (capturedFrames > 0) {
        return `Replay export unavailable – ${formatCapturedLabel(capturedFrames, 'frame')} captured but not yet playable`;
      }
      return 'Replay export unavailable – execution did not capture any timeline frames';
    default:
      return 'Replay export unavailable';
  }
};

export const resolveBackgroundAsset = (relativePath: string) => {
  const url = new URL(relativePath, import.meta.url);
  return withAssetBasePath(url.pathname || url.href);
};

export const geometricPrismUrl = resolveBackgroundAsset(
  '../assets/replay-backgrounds/geometric-prism.jpg',
);
export const geometricOrbitUrl = resolveBackgroundAsset(
  '../assets/replay-backgrounds/geometric-orbit.jpg',
);
export const geometricMosaicUrl = resolveBackgroundAsset(
  '../assets/replay-backgrounds/geometric-mosaic.jpg',
);

export type ExportStatusLabel = 'ready' | 'pending' | 'error' | 'unavailable' | 'unknown';

export const mapExportStatus = (status?: number | null): ExportStatusLabel => {
  switch (status) {
    case 1: // READY
      return 'ready';
    case 2: // PENDING
      return 'pending';
    case 3: // ERROR
      return 'error';
    case 4: // UNAVAILABLE
      return 'unavailable';
    default:
      return 'unknown';
  }
};
