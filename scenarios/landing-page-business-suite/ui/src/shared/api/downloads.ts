import { apiCall } from './common';
import { parseOrNull } from './safeParse';
import {
  DownloadAssetSchema,
  DownloadAppSchema,
  DownloadAppsListResponseSchema,
  DownloadStorageSettingsResponseSchema,
  ListDownloadArtifactsResponseSchema,
  PresignUploadResponseSchema,
  DownloadArtifactSchema,
  PresignGetResponseSchema,
} from './schemas/downloads.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
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

  return apiCall<DownloadAsset>(`/downloads?${params.toString()}`).then((resp) => {
    const validated = parseOrNull(DownloadAssetSchema, resp, 'DownloadAsset');
    if (!validated) {
      throw new Error('Invalid download asset response from API');
    }
    return validated;
  });
}

export function listDownloadAppsAdmin() {
  return apiCall<{ apps: DownloadApp[] }>('/admin/download-apps').then((resp) => {
    const validated = parseOrNull(DownloadAppsListResponseSchema, resp, 'DownloadAppsListResponse');
    if (!validated) {
      return { apps: [] };
    }
    return validated;
  });
}

export function saveDownloadAppAdmin(appKey: string, payload: DownloadAppInput) {
  return apiCall<DownloadApp>(`/admin/download-apps/${appKey}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(DownloadAppSchema, resp, 'DownloadApp');
    if (!validated) {
      throw new Error('Invalid download app response from API');
    }
    return validated;
  });
}

export function createDownloadAppAdmin(payload: DownloadAppInput) {
  return apiCall<DownloadApp>('/admin/download-apps', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(DownloadAppSchema, resp, 'DownloadApp');
    if (!validated) {
      throw new Error('Invalid download app response from API');
    }
    return validated;
  });
}

export function deleteDownloadAppAdmin(appKey: string) {
  return apiCall<{ success: boolean }>(`/admin/download-apps/${appKey}`, {
    method: 'DELETE',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'DeleteDownloadAppResponse');
    if (!validated) {
      throw new Error('Invalid delete download app response from API');
    }
    return validated;
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
  return apiCall<{ settings: DownloadStorageSettingsSnapshot }>('/admin/download-storage').then((resp) => {
    const validated = parseOrNull(DownloadStorageSettingsResponseSchema, resp, 'DownloadStorageSettingsResponse');
    if (!validated) {
      throw new Error('Invalid download storage settings response from API');
    }
    return validated;
  });
}

export function updateDownloadStorageAdmin(payload: DownloadStorageSettingsUpdate) {
  return apiCall<{ settings: DownloadStorageSettingsSnapshot }>('/admin/download-storage', {
    method: 'PUT',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(DownloadStorageSettingsResponseSchema, resp, 'DownloadStorageSettingsResponse');
    if (!validated) {
      throw new Error('Invalid download storage settings response from API');
    }
    return validated;
  });
}

export function testDownloadStorageAdmin() {
  return apiCall<{ success: boolean }>('/admin/download-storage/test', {
    method: 'POST',
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'TestDownloadStorageResponse');
    if (!validated) {
      throw new Error('Invalid test download storage response from API');
    }
    return validated;
  });
}

export interface ListDownloadArtifactsResponse {
  artifacts: DownloadArtifact[];
  page: number;
  page_size: number;
  total: number;
}

export function listDownloadArtifactsAdmin(params?: { query?: string; platform?: string; app_key?: string; page?: number; page_size?: number }) {
  const search = new URLSearchParams();
  if (params?.query) search.set('query', params.query);
  if (params?.platform) search.set('platform', params.platform);
  if (params?.app_key) search.set('app_key', params.app_key);
  if (params?.page) search.set('page', String(params.page));
  if (params?.page_size) search.set('page_size', String(params.page_size));
  const suffix = search.toString() ? `?${search.toString()}` : '';
  return apiCall<ListDownloadArtifactsResponse>(`/admin/download-artifacts${suffix}`).then((resp) => {
    const validated = parseOrNull(ListDownloadArtifactsResponseSchema, resp, 'ListDownloadArtifactsResponse');
    if (!validated) {
      return { artifacts: [], page: 1, page_size: 10, total: 0 };
    }
    return validated;
  });
}

export function listDownloadArtifactsByAppAdmin(params: { app_key: string; platform?: string; page?: number; page_size?: number }) {
  const search = new URLSearchParams();
  search.set('app_key', params.app_key);
  if (params.platform) search.set('platform', params.platform);
  if (params.page) search.set('page', String(params.page));
  if (params.page_size) search.set('page_size', String(params.page_size));
  return apiCall<ListDownloadArtifactsResponse>(`/admin/download-artifacts/by-app?${search.toString()}`).then((resp) => {
    const validated = parseOrNull(ListDownloadArtifactsResponseSchema, resp, 'ListDownloadArtifactsResponse');
    if (!validated) {
      return { artifacts: [], page: 1, page_size: 10, total: 0 };
    }
    return validated;
  });
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
  }).then((resp) => {
    const validated = parseOrNull(PresignUploadResponseSchema, resp, 'PresignUploadResponse');
    if (!validated) {
      throw new Error('Invalid presign upload response from API');
    }
    return validated;
  });
}

export function commitDownloadArtifactAdmin(payload: {
  bucket: string;
  object_key: string;
  original_filename?: string;
  content_type?: string;
  app_key?: string;
  platform?: string;
  release_version?: string;
  sha256?: string;
  metadata?: Record<string, unknown>;
  set_as_current?: boolean;
}) {
  return apiCall<DownloadArtifact>('/admin/download-artifacts/commit', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(DownloadArtifactSchema, resp, 'DownloadArtifact');
    if (!validated) {
      throw new Error('Invalid download artifact response from API');
    }
    return validated;
  });
}

export function presignDownloadArtifactGetAdmin(artifactId: number) {
  return apiCall<{ url: string }>(`/admin/download-artifacts/${String(artifactId)}/presign-get`).then((resp) => {
    const validated = parseOrNull(PresignGetResponseSchema, resp, 'PresignGetResponse');
    if (!validated) {
      throw new Error('Invalid presign get response from API');
    }
    return validated;
  });
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
  }).then((resp) => {
    const validated = parseOrNull(DownloadAssetSchema, resp, 'DownloadAsset');
    if (!validated) {
      throw new Error('Invalid download asset response from API');
    }
    return validated;
  });
}

export function setArtifactAsCurrentAdmin(payload: {
  artifact_id: number;
  app_key: string;
  platform: string;
}) {
  return apiCall<DownloadAsset>('/admin/download-assets/set-current', {
    method: 'POST',
    body: JSON.stringify(payload),
    headers: {
      'Content-Type': 'application/json',
    },
  }).then((resp) => {
    const validated = parseOrNull(DownloadAssetSchema, resp, 'DownloadAsset');
    if (!validated) {
      throw new Error('Invalid download asset response from API');
    }
    return validated;
  });
}
