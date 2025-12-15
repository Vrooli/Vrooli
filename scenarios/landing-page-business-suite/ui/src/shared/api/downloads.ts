import { apiCall } from './common';
import type { DownloadApp, DownloadArtifact, DownloadAsset, DownloadStorageSettingsSnapshot, DownloadStorefront } from './types';

export interface DownloadAssetInput {
  platform: string;
  artifact_url: string;
  artifact_source?: 'direct' | 'managed';
  artifact_id?: number;
  release_version: string;
  release_notes?: string;
  checksum?: string;
  requires_entitlement?: boolean;
  metadata?: Record<string, unknown>;
}

export interface DownloadAppInput {
  app_key?: string;
  name: string;
  tagline?: string;
  description?: string;
  icon_url?: string;
  screenshot_url?: string;
  install_overview?: string;
  install_steps?: string[];
  storefronts?: DownloadStorefront[];
  metadata?: Record<string, unknown>;
  display_order?: number;
  platforms: DownloadAssetInput[];
}

export function requestDownload(appKey: string, platform: string, user?: string) {
  const params = new URLSearchParams({ app: appKey, platform });
  if (user) {
    params.set('user', user);
  }

  return apiCall<DownloadAsset>(`/downloads?${params.toString()}`);
}

export function listDownloadAppsAdmin() {
  return apiCall<{ apps: DownloadApp[] }>('/admin/download-apps');
}

export function saveDownloadAppAdmin(appKey: string, payload: DownloadAppInput) {
  return apiCall<DownloadApp>(`/admin/download-apps/${appKey}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

export function createDownloadAppAdmin(payload: DownloadAppInput) {
  return apiCall<DownloadApp>('/admin/download-apps', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

export function deleteDownloadAppAdmin(appKey: string) {
  return apiCall<{ success: boolean }>(`/admin/download-apps/${appKey}`, {
    method: 'DELETE',
  });
}

export interface DownloadStorageSettingsUpdate {
  provider?: 's3';
  bucket?: string;
  region?: string;
  endpoint?: string;
  force_path_style?: boolean;
  default_prefix?: string;
  signed_url_ttl_seconds?: number;
  public_base_url?: string;
  access_key_id?: string;
  secret_access_key?: string;
  session_token?: string;
}

export function getDownloadStorageAdmin() {
  return apiCall<{ settings: DownloadStorageSettingsSnapshot }>('/admin/download-storage');
}

export function updateDownloadStorageAdmin(payload: DownloadStorageSettingsUpdate) {
  return apiCall<{ settings: DownloadStorageSettingsSnapshot }>('/admin/download-storage', {
    method: 'PUT',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

export function testDownloadStorageAdmin() {
  return apiCall<{ success: boolean }>('/admin/download-storage/test', {
    method: 'POST',
  });
}

export interface ListDownloadArtifactsResponse {
  artifacts: DownloadArtifact[];
  page: number;
  page_size: number;
  total: number;
}

export function listDownloadArtifactsAdmin(params?: { query?: string; platform?: string; page?: number; page_size?: number }) {
  const search = new URLSearchParams();
  if (params?.query) search.set('query', params.query);
  if (params?.platform) search.set('platform', params.platform);
  if (params?.page) search.set('page', String(params.page));
  if (params?.page_size) search.set('page_size', String(params.page_size));
  const suffix = search.toString() ? `?${search.toString()}` : '';
  return apiCall<ListDownloadArtifactsResponse>(`/admin/download-artifacts${suffix}`);
}

export interface PresignUploadResponse {
  upload_url: string;
  required_headers: Record<string, string>;
  bucket: string;
  object_key: string;
  expires_at: string;
  stable_object_uri: string;
}

export function presignDownloadArtifactUploadAdmin(payload: {
  filename: string;
  content_type?: string;
  app_key?: string;
  platform?: string;
  release_version?: string;
  metadata?: Record<string, unknown>;
}) {
  return apiCall<PresignUploadResponse>('/admin/download-artifacts/presign-upload', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

export function commitDownloadArtifactAdmin(payload: {
  bucket: string;
  object_key: string;
  original_filename?: string;
  content_type?: string;
  platform?: string;
  release_version?: string;
  sha256?: string;
  metadata?: Record<string, unknown>;
}) {
  return apiCall<DownloadArtifact>('/admin/download-artifacts/commit', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

export function presignDownloadArtifactGetAdmin(artifactId: number) {
  return apiCall<{ url: string }>(`/admin/download-artifacts/${artifactId}/presign-get`);
}

export function applyDownloadArtifactAdmin(payload: {
  app_key: string;
  platform: string;
  artifact_id: number;
  release_version?: string;
  release_notes?: string;
  checksum?: string;
  requires_entitlement?: boolean;
  metadata?: Record<string, unknown>;
}) {
  return apiCall<DownloadAsset>('/admin/download-assets/apply', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  });
}
