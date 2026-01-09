import { API_BASE } from './common';
import type { Asset, AssetCategory } from './types';

export interface UploadAssetOptions {
  category?: AssetCategory;
  altText?: string;
  uploadedBy?: string;
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
    throw new Error(text || `Upload failed with status ${response.status}`);
  }

  return response.json();
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
  const url = category
    ? `${API_BASE}/admin/assets?category=${encodeURIComponent(category)}`
    : `${API_BASE}/admin/assets`;

  const response = await fetch(url, {
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`Failed to list assets: ${response.status}`);
  }

  const data = await response.json();
  return data.assets || [];
}

/**
 * Delete an asset by ID.
 * @param id - The asset ID to delete
 */
export async function deleteAsset(id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/admin/assets/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    throw new Error(`Failed to delete asset: ${response.status}`);
  }
}
