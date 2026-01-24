import { z } from 'zod';
import { MetadataSchema, StorageProviderSchema } from './common.schema';
// Re-export download schemas from landing.schema.ts to avoid duplication
export {
  DownloadStorefrontSchema,
  DownloadAssetSchema,
  DownloadAppSchema,
  type DownloadStorefront,
  type DownloadAsset,
  type DownloadApp,
} from './landing.schema';

/**
 * Downloads-related Zod schemas for API response validation.
 * Note: DownloadStorefront, DownloadAsset, and DownloadApp schemas are re-exported from landing.schema.ts
 */

import { DownloadAppSchema as BaseDownloadAppSchema } from './landing.schema';

// Download apps list response schema
export const DownloadAppsListResponseSchema = z.object({
  apps: z.array(BaseDownloadAppSchema),
});

// Download storage settings snapshot schema
export const DownloadStorageSettingsSnapshotSchema = z.object({
  provider: StorageProviderSchema,
  bucket: z.string().optional(),
  region: z.string().optional(),
  endpoint: z.string().optional(),
  force_path_style: z.boolean(),
  default_prefix: z.string().optional(),
  signed_url_ttl_seconds: z.number(),
  public_base_url: z.string().optional(),
  access_key_id_set: z.boolean(),
  secret_access_key_set: z.boolean(),
  session_token_set: z.boolean(),
  credentials_from_env: z.boolean(),
  settings_row_available: z.boolean(),
});

// Download storage settings response schema
export const DownloadStorageSettingsResponseSchema = z.object({
  settings: DownloadStorageSettingsSnapshotSchema,
});

// Download artifact schema
export const DownloadArtifactSchema = z.object({
  id: z.number(),
  bundle_key: z.string(),
  provider: StorageProviderSchema,
  bucket: z.string(),
  object_key: z.string(),
  etag: z.string().optional(),
  size_bytes: z.number().optional(),
  sha256: z.string().optional(),
  content_type: z.string().optional(),
  original_filename: z.string().optional(),
  platform: z.string().optional(),
  release_version: z.string().optional(),
  metadata: MetadataSchema,
  created_at: z.string(),
  updated_at: z.string(),
  stable_object_uri: z.string().optional(),
});

// List download artifacts response schema
export const ListDownloadArtifactsResponseSchema = z.object({
  artifacts: z.array(DownloadArtifactSchema),
  page: z.number(),
  page_size: z.number(),
  total: z.number(),
});

// Presign upload response schema
export const PresignUploadResponseSchema = z.object({
  upload_url: z.string(),
  required_headers: z.record(z.string(), z.string()),
  bucket: z.string(),
  object_key: z.string(),
  expires_at: z.string(),
  stable_object_uri: z.string(),
});

// Presign get response schema
export const PresignGetResponseSchema = z.object({
  url: z.string(),
});

// Export inferred types (DownloadStorefront, DownloadAsset, DownloadApp are re-exported above)
export type DownloadAppsListResponse = z.infer<typeof DownloadAppsListResponseSchema>;
export type DownloadStorageSettingsSnapshot = z.infer<typeof DownloadStorageSettingsSnapshotSchema>;
export type DownloadStorageSettingsResponse = z.infer<typeof DownloadStorageSettingsResponseSchema>;
export type DownloadArtifact = z.infer<typeof DownloadArtifactSchema>;
export type ListDownloadArtifactsResponse = z.infer<typeof ListDownloadArtifactsResponseSchema>;
export type PresignUploadResponse = z.infer<typeof PresignUploadResponseSchema>;
export type PresignGetResponse = z.infer<typeof PresignGetResponseSchema>;
