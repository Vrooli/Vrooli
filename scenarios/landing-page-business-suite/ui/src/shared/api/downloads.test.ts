import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as downloads from './downloads';
import { apiCall } from './common';

const downloadClient = vi.hoisted(() => ({
  authorizeDownload: vi.fn(),
  listDownloadApps: vi.fn(),
  saveDownloadApp: vi.fn(),
  createDownloadApp: vi.fn(),
  deleteDownloadApp: vi.fn(),
}));

vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => downloadClient) }));
vi.mock('./common', () => ({ apiCall: vi.fn(), CONNECT_API_BASE: 'http://api.test' }));
const mockApiCall = vi.mocked(apiCall);

function generatedAsset(asset: { bundle_key: string; app_key: string; platform: string; artifact_url: string; release_version: string; requires_entitlement: boolean; artifact_source?: string }) {
  return { id: 0n, bundleKey: asset.bundle_key, appKey: asset.app_key, platform: asset.platform, artifactUrl: asset.artifact_url, artifactSource: asset.artifact_source ?? 'direct', releaseVersion: asset.release_version, releaseNotes: '', checksum: '', requiresEntitlement: asset.requires_entitlement, variantKey: '', artifactFilename: '', artifactSizeBytes: 0n, artifactCount: 0 };
}

function generatedApp(app: { bundle_key: string; app_key: string; name: string; platforms: Array<ReturnType<typeof generatedAsset>> }) {
  return { id: 0n, bundleKey: app.bundle_key, appKey: app.app_key, name: app.name, tagline: '', description: '', iconUrl: '', screenshotUrl: '', installOverview: '', installSteps: [], storefronts: [], displayOrder: 0, platforms: app.platforms, updateApiKey: '' };
}

describe('download API transport', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockApiCall.mockResolvedValue({});
    downloadClient.authorizeDownload.mockResolvedValue({});
    downloadClient.listDownloadApps.mockResolvedValue({ apps: [] });
    downloadClient.saveDownloadApp.mockResolvedValue({});
    downloadClient.createDownloadApp.mockResolvedValue({});
    downloadClient.deleteDownloadApp.mockResolvedValue({});
  });

  it('uses public/app/storage endpoints and rejects malformed required payloads', async () => {
    const app = { name: 'Desktop', platforms: [] };
    await expect(downloads.requestDownload('desktop', 'windows', 'customer')).rejects.toThrow('Invalid download asset response');
    await expect(downloads.saveDownloadAppAdmin('desktop', app)).rejects.toThrow('Invalid download app response');
    await expect(downloads.createDownloadAppAdmin(app)).rejects.toThrow('Invalid download app response');
    await expect(downloads.getDownloadStorageAdmin()).rejects.toThrow('Invalid download storage settings response');
    await expect(downloads.updateDownloadStorageAdmin({ bucket: 'installers' })).rejects.toThrow('Invalid download storage settings response');
    expect(downloadClient.authorizeDownload).toHaveBeenCalledWith({ app: 'desktop', platform: 'windows' });
    expect(downloadClient.saveDownloadApp).toHaveBeenCalledWith(expect.objectContaining({ appKey: 'desktop' }));
    expect(downloadClient.createDownloadApp).toHaveBeenCalledTimes(1);
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
    expect(downloadClient.deleteDownloadApp).toHaveBeenCalledWith({ appKey: 'desktop' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/download-storage/test', { method: 'POST' });
  });

  it('returns validated download, storage, and artifact records from successful responses', async () => {
    const asset = { bundle_key: 'bundle', app_key: 'desktop', platform: 'windows', artifact_url: 'https://cdn.example.test/app.exe', release_version: '1.0.0', requires_entitlement: true };
    const app = { bundle_key: 'bundle', app_key: 'desktop', name: 'Desktop', platforms: [asset] };
    const settings = { provider: 's3', force_path_style: false, signed_url_ttl_seconds: 900, access_key_id_set: true, secret_access_key_set: true, session_token_set: false, credentials_from_env: true, settings_row_available: true };
    const artifact = { id: 1, bundle_key: 'bundle', provider: 's3', bucket: 'releases', object_key: 'desktop/app.exe', metadata: {}, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' };
    downloadClient.authorizeDownload.mockResolvedValueOnce({ asset: generatedAsset(asset) });
    downloadClient.listDownloadApps.mockResolvedValueOnce({ apps: [generatedApp({ ...app, platforms: [generatedAsset(asset)] })] });
    downloadClient.createDownloadApp.mockResolvedValueOnce({ app: generatedApp({ ...app, platforms: [generatedAsset(asset)] }) });
    mockApiCall
      .mockResolvedValueOnce({ settings })
      .mockResolvedValueOnce({ artifacts: [artifact], page: 2, page_size: 25, total: 1 })
      .mockResolvedValueOnce({ upload_url: 'https://storage.example.test/upload', required_headers: {}, bucket: 'releases', object_key: 'desktop/app.exe', expires_at: '2026-01-02T00:00:00Z', stable_object_uri: 's3://releases/desktop/app.exe' })
      .mockResolvedValueOnce(artifact)
      .mockResolvedValueOnce({ url: 'https://storage.example.test/get' });

    await expect(downloads.requestDownload('desktop', 'windows')).resolves.toMatchObject(asset);
    await expect(downloads.listDownloadAppsAdmin()).resolves.toMatchObject({ apps: [app] });
    await expect(downloads.createDownloadAppAdmin({ name: 'Desktop', platforms: [] })).resolves.toMatchObject(app);
    await expect(downloads.getDownloadStorageAdmin()).resolves.toEqual({ settings });
    await expect(downloads.listDownloadArtifactsAdmin({ page: 2, page_size: 25 })).resolves.toEqual({ artifacts: [artifact], page: 2, page_size: 25, total: 1 });
    await expect(downloads.presignDownloadArtifactUploadAdmin({ filename: 'app.exe' })).resolves.toMatchObject({ bucket: 'releases' });
    await expect(downloads.commitDownloadArtifactAdmin({ bucket: 'releases', object_key: 'desktop/app.exe' })).resolves.toEqual(artifact);
    await expect(downloads.presignDownloadArtifactGetAdmin(1)).resolves.toEqual({ url: 'https://storage.example.test/get' });
  });

  it('preserves valid app, storage, and applied-artifact responses for operator workflows', async () => {
    const asset = { bundle_key: 'bundle', app_key: 'desktop', platform: 'windows', artifact_url: 'https://cdn.example.test/app.exe', release_version: '1.0.0', requires_entitlement: false };
    const app = { bundle_key: 'bundle', app_key: 'desktop', name: 'Desktop', platforms: [asset] };
    const settings = { provider: 's3', force_path_style: true, signed_url_ttl_seconds: 600, access_key_id_set: false, secret_access_key_set: false, session_token_set: false, credentials_from_env: false, settings_row_available: true };
    downloadClient.saveDownloadApp.mockResolvedValueOnce({ app: generatedApp({ ...app, platforms: [generatedAsset(asset)] }) });
    mockApiCall
      .mockResolvedValueOnce({ settings })
      .mockResolvedValueOnce({ success: true })
      .mockResolvedValueOnce(asset)
      .mockResolvedValueOnce(asset);

    await expect(downloads.saveDownloadAppAdmin('desktop', { name: 'Desktop', platforms: [] })).resolves.toMatchObject(app);
    await expect(downloads.updateDownloadStorageAdmin({ bucket: 'releases', force_path_style: true })).resolves.toEqual({ settings });
    await expect(downloads.testDownloadStorageAdmin()).resolves.toEqual({ success: true });
    await expect(downloads.applyDownloadArtifactAdmin({ app_key: 'desktop', platform: 'windows', artifact_id: 1 })).resolves.toEqual(asset);
    await expect(downloads.setArtifactAsCurrentAdmin({ app_key: 'desktop', platform: 'windows', artifact_id: 1 })).resolves.toEqual(asset);
  });
});
