import { apiCall } from './common';
import type { DownloadAsset } from './types';

export function requestDownload(platform: string, user?: string) {
  const params = new URLSearchParams({ platform });
  if (user) {
    params.set('user', user);
  }

  return apiCall<DownloadAsset>(`/downloads?${params.toString()}`);
}
