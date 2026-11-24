import type { DownloadAsset } from './types';

/** One readiness rule for admin health and editable download forms. */
export function isDownloadAssetConfigured(asset: Partial<DownloadAsset>): boolean {
  if (!asset.release_version?.trim()) return false;
  const source = asset.artifact_source ?? (asset.artifact_id ? 'managed' : 'direct');
  if (source === 'managed') {
    return Number.isSafeInteger(asset.artifact_id) && (asset.artifact_id ?? 0) > 0 && Boolean(asset.checksum?.trim());
  }
  return Boolean(asset.artifact_url?.trim());
}
