import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as downloads from './downloads';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('download API transport', () => {
  beforeEach(() => { vi.clearAllMocks(); mockApiCall.mockResolvedValue({} as never); });

  it('uses public/app/storage endpoints and rejects malformed required payloads', async () => {
    const app = { name: 'Desktop', platforms: [] };
    await expect(downloads.requestDownload('desktop', 'windows', 'customer')).rejects.toThrow('Invalid download asset response');
    await expect(downloads.saveDownloadAppAdmin('desktop', app)).rejects.toThrow('Invalid download app response');
    await expect(downloads.createDownloadAppAdmin(app)).rejects.toThrow('Invalid download app response');
    await expect(downloads.getDownloadStorageAdmin()).rejects.toThrow('Invalid download storage settings response');
    await expect(downloads.updateDownloadStorageAdmin({ bucket: 'installers' })).rejects.toThrow('Invalid download storage settings response');
    expect(mockApiCall).toHaveBeenCalledWith('/downloads?app=desktop&platform=windows&user=customer');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-apps/desktop', expect.objectContaining({ method: 'PUT' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-apps', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-storage');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-storage', expect.objectContaining({ method: 'PUT' }));
  });

  it('uses artifact list, upload, commit, retrieval, apply, and current-version endpoints with safe fallbacks', async () => {
    await expect(downloads.listDownloadAppsAdmin()).resolves.toEqual({ apps: [] });
    await expect(downloads.listDownloadArtifactsAdmin()).resolves.toEqual({ artifacts: [], page: 1, page_size: 10, total: 0 });
    await expect(downloads.listDownloadArtifactsAdmin({ query: 'desktop', platform: 'windows', app_key: 'desktop', page: 2, page_size: 25 })).resolves.toEqual({ artifacts: [], page: 1, page_size: 10, total: 0 });
    await expect(downloads.listDownloadArtifactsByAppAdmin({ app_key: 'desktop', platform: 'windows', page: 2, page_size: 25 })).resolves.toEqual({ artifacts: [], page: 1, page_size: 10, total: 0 });
    await expect(downloads.presignDownloadArtifactUploadAdmin({ filename: 'desktop.exe' })).rejects.toThrow('Invalid presign upload response');
    await expect(downloads.commitDownloadArtifactAdmin({ bucket: 'installers', object_key: 'desktop.exe' })).rejects.toThrow('Invalid download artifact response');
    await expect(downloads.presignDownloadArtifactGetAdmin(9)).rejects.toThrow('Invalid presign get response');
    await expect(downloads.applyDownloadArtifactAdmin({ app_key: 'desktop', platform: 'windows', artifact_id: 9 })).rejects.toThrow('Invalid download asset response');
    await expect(downloads.setArtifactAsCurrentAdmin({ app_key: 'desktop', platform: 'windows', artifact_id: 9 })).rejects.toThrow('Invalid download asset response');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-artifacts');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-artifacts?query=desktop&platform=windows&app_key=desktop&page=2&page_size=25');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-artifacts/by-app?app_key=desktop&platform=windows&page=2&page_size=25');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-artifacts/presign-upload', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-artifacts/commit', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-artifacts/9/presign-get');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-assets/apply', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-assets/set-current', expect.objectContaining({ method: 'POST' }));
  });

  it('accepts generic success envelopes for destructive/storage probes', async () => {
    await expect(downloads.deleteDownloadAppAdmin('desktop')).resolves.toEqual({});
    await expect(downloads.testDownloadStorageAdmin()).resolves.toEqual({});
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-apps/desktop', { method: 'DELETE' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-storage/test', { method: 'POST' });
  });

  it('returns validated download, storage, and artifact records from successful responses', async () => {
    const asset = { bundle_key: 'bundle', app_key: 'desktop', platform: 'windows', artifact_url: 'https://cdn.example.test/app.exe', release_version: '1.0.0', requires_entitlement: true };
    const app = { bundle_key: 'bundle', app_key: 'desktop', name: 'Desktop', platforms: [asset] };
    const settings = { provider: 's3', force_path_style: false, signed_url_ttl_seconds: 900, access_key_id_set: true, secret_access_key_set: true, session_token_set: false, credentials_from_env: true, settings_row_available: true };
    const artifact = { id: 1, bundle_key: 'bundle', provider: 's3', bucket: 'releases', object_key: 'desktop/app.exe', metadata: {}, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' };
    mockApiCall
      .mockResolvedValueOnce(asset as never)
      .mockResolvedValueOnce({ apps: [app] } as never)
      .mockResolvedValueOnce(app as never)
      .mockResolvedValueOnce({ settings } as never)
      .mockResolvedValueOnce({ artifacts: [artifact], page: 2, page_size: 25, total: 1 } as never)
      .mockResolvedValueOnce({ upload_url: 'https://storage.example.test/upload', required_headers: {}, bucket: 'releases', object_key: 'desktop/app.exe', expires_at: '2026-01-02T00:00:00Z', stable_object_uri: 's3://releases/desktop/app.exe' } as never)
      .mockResolvedValueOnce(artifact as never)
      .mockResolvedValueOnce({ url: 'https://storage.example.test/get' } as never);

    await expect(downloads.requestDownload('desktop', 'windows')).resolves.toEqual(asset);
    await expect(downloads.listDownloadAppsAdmin()).resolves.toEqual({ apps: [app] });
    await expect(downloads.createDownloadAppAdmin({ name: 'Desktop', platforms: [] })).resolves.toEqual(app);
    await expect(downloads.getDownloadStorageAdmin()).resolves.toEqual({ settings });
    await expect(downloads.listDownloadArtifactsAdmin({ page: 2, page_size: 25 })).resolves.toEqual({ artifacts: [artifact], page: 2, page_size: 25, total: 1 });
    await expect(downloads.presignDownloadArtifactUploadAdmin({ filename: 'app.exe' })).resolves.toMatchObject({ bucket: 'releases' });
    await expect(downloads.commitDownloadArtifactAdmin({ bucket: 'releases', object_key: 'desktop/app.exe' })).resolves.toEqual(artifact);
    await expect(downloads.presignDownloadArtifactGetAdmin(1)).resolves.toEqual({ url: 'https://storage.example.test/get' });
  });
});
