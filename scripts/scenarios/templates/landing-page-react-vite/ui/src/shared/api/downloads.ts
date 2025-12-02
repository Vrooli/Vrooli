import { apiCall } from './common';
import type { DownloadAsset, DownloadApp, DownloadStorefront } from './types';

export interface DownloadAssetInput {
  platform: string;
  artifact_url: string;
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
