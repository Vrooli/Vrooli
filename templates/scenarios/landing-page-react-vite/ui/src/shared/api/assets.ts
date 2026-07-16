import { fromJson, type JsonValue } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';
import {
  AssetsService,
  AssetSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/assets_pb';
import type { Asset } from '@vrooli/proto-types/landing-page-react-vite/v1/assets_pb';

import { REST_API_BASE, decodeApiError, uploadFile, transport } from './client';

const assetsClient = createClient(AssetsService, transport);

export type AssetCategory = 'logo' | 'favicon' | 'og_image' | 'general';

export interface UploadAssetOptions {
  category?: AssetCategory;
  altText?: string;
  uploadedBy?: string;
}

/**
 * Uploads an asset file. Multipart upload is a deliberate REST exception (it
 * cannot be a Connect RPC), so this posts FormData to /api/v1/admin/assets/upload
 * and decodes the returned Asset JSON with the proto schema.
 */
export async function uploadAsset(file: File, options: UploadAssetOptions = {}): Promise<Asset> {
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

  const res = await uploadFile('/admin/assets/upload', formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  return fromJson(AssetSchema, (await res.json()) as JsonValue, { ignoreUnknownFields: true });
}

/**
 * Resolves an asset URL. Full URLs and data URLs pass through; storage paths
 * are prefixed with the static /uploads REST base.
 */
export function getAssetUrl(urlOrPath: string): string {
  if (!urlOrPath) {
    return '';
  }
  if (urlOrPath.startsWith('http://') || urlOrPath.startsWith('https://')) {
    return urlOrPath;
  }
  if (urlOrPath.startsWith('data:')) {
    return urlOrPath;
  }
  const host = REST_API_BASE.replace(/\/api\/v1$/, '');
  if (urlOrPath.startsWith('/api/v1/uploads/')) {
    return `${host}${urlOrPath}`;
  }
  const cleanPath = urlOrPath.startsWith('/') ? urlOrPath.slice(1) : urlOrPath;
  return `${REST_API_BASE}/uploads/${cleanPath}`;
}

/** Lists uploaded assets, optionally filtered by category (admin). */
export async function listAssets(category?: AssetCategory): Promise<Asset[]> {
  const resp = await assetsClient.listAssets({ category: category ?? '' });
  return resp.assets;
}

/** Deletes an asset by id (admin). */
export async function deleteAsset(id: bigint): Promise<boolean> {
  const resp = await assetsClient.deleteAsset({ id });
  return resp.deleted;
}

export type { Asset };
