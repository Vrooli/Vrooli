import { createClient } from '@connectrpc/connect';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import { AssetsService, type Asset as GeneratedAsset } from '@vrooli/proto-types/landing-page-business-suite/assets_pb';
import { API_BASE, CONNECT_API_BASE } from './common';
import { AssetSchema } from './schemas/common.schema';
import { safeParseJson } from '../lib/utils';
import type { Asset, AssetCategory } from './types';

const assetsClient = createClient(AssetsService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

export interface UploadAssetOptions {
  category?: AssetCategory;
  altText?: string;
  uploadedBy?: string;
}

function assetFromProto(value: GeneratedAsset): Asset {
  const createdAt = value.createdAt
    ? new Date(Number(value.createdAt.seconds) * 1000 + value.createdAt.nanos / 1_000_000)
      .toISOString()
      .replace('.000Z', 'Z')
    : undefined;
  const derivatives = Object.keys(value.derivatives).length > 0 ? value.derivatives : undefined;

  return AssetSchema.parse({
    id: Number(value.id),
    filename: value.filename,
    original_filename: value.originalFilename,
    mime_type: value.mimeType,
    size_bytes: Number(value.sizeBytes),
    storage_path: value.storagePath,
    ...(value.thumbnailPath ? { thumbnail_path: value.thumbnailPath } : {}),
    ...(value.altText ? { alt_text: value.altText } : {}),
    category: value.category,
    ...(value.uploadedBy ? { uploaded_by: value.uploadedBy } : {}),
    ...(createdAt ? { created_at: createdAt } : {}),
    url: value.url,
    ...(derivatives ? { derivatives } : {}),
  });
}

/**
 * Upload an asset file to the server.
 * @param file - The file to upload
 * @param options - Optional metadata (category, alt text, uploader)
 * @returns The created Asset with its URL
 */
export async function uploadAsset(
  file: File,
  options: UploadAssetOptions = {}
): Promise<Asset> {
  const formData = new FormData();
  formData.append('file', file);

  if (options.category) {
    formData.append('category', options.category);
  }
  if (options.altText) {
    formData.append('alt_text', options.altText);
  }
  if (options.uploadedBy) {
    formData.append('uploaded_by', options.uploadedBy);
  }

  const response = await fetch(`${API_BASE}/admin/assets/upload`, {
    method: 'POST',
    credentials: 'include',
    body: formData,
    // Note: Don't set Content-Type header - browser sets it with boundary for multipart
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Upload failed with status ${String(response.status)}`);
  }

  const raw = await response.text();
  return AssetSchema.parse(safeParseJson(raw));
}

/**
 * Resolve an asset URL. Handles:
 * - Full URLs (https://...) - returned as-is
 * - Relative paths (uploads/...) - prefixed with API base
 * - Already-resolved URLs - returned as-is
 *
 * @param urlOrPath - The URL or storage path to resolve
 * @returns The fully-qualified URL
 */
export function getAssetUrl(urlOrPath: string): string {
  if (!urlOrPath) {
    return '';
  }

  // Already a full URL
  if (urlOrPath.startsWith('http://') || urlOrPath.startsWith('https://')) {
    return urlOrPath;
  }

  // Data URL (base64 embedded)
  if (urlOrPath.startsWith('data:')) {
    return urlOrPath;
  }

  // Already includes /api/v1/uploads prefix
  if (urlOrPath.startsWith('/api/v1/uploads/')) {
    return `${API_BASE.replace('/api/v1', '')}${urlOrPath}`;
  }

  // Relative path - prefix with uploads endpoint
  const cleanPath = urlOrPath.startsWith('/') ? urlOrPath.slice(1) : urlOrPath;
  return `${API_BASE}/uploads/${cleanPath}`;
}

/**
 * List assets, optionally filtered by category.
 * @param category - Optional category filter
 * @returns Array of assets
 */
export async function listAssets(category?: AssetCategory): Promise<Asset[]> {
  const response = await assetsClient.listAssets({ category: category ?? '' });
  return response.assets.map(assetFromProto);
}

/**
 * Delete an asset by ID.
 * @param id - The asset ID to delete
 */
export async function deleteAsset(id: number): Promise<void> {
  const response = await assetsClient.deleteAsset({ id: BigInt(id) });
  if (!response.deleted) throw new Error('Asset deletion was not acknowledged');
}
