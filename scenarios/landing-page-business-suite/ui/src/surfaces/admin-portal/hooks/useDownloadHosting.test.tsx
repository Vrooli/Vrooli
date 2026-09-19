import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useDownloadHosting } from './useDownloadHosting';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => ({
  ...(await vi.importActual('../../../shared/api')),
  applyDownloadArtifactAdmin: vi.fn(),
  commitDownloadArtifactAdmin: vi.fn(),
  getDownloadStorageAdmin: vi.fn(),
  listDownloadArtifactsAdmin: vi.fn(),
  listDownloadArtifactsByAppAdmin: vi.fn(),
  presignDownloadArtifactUploadAdmin: vi.fn(),
  setArtifactAsCurrentAdmin: vi.fn(),
  testDownloadStorageAdmin: vi.fn(),
  updateDownloadStorageAdmin: vi.fn(),
}));

const loadApps = vi.fn().mockResolvedValue(undefined);
const getFirstAppKey = vi.fn(() => 'desktop-app');
const artifact = {
  id: 42,
  bucket: 'releases',
  object_key: 'desktop-app/1.2.3/app.exe',
  original_filename: 'app.exe',
  platform: 'windows' as const,
  created_at: '2026-07-27T00:00:00Z',
};
const storageSettings = {
  bucket: 'releases',
  region: 'us-east-1',
  endpoint: 'https://storage.example.test',
  force_path_style: false,
  default_prefix: 'desktop-app/',
  signed_url_ttl_seconds: 900,
  public_base_url: '',
};

function renderHosting(activeTab: 'apps' | 'hosting' = 'apps') {
  return renderHook(({ tab }) => useDownloadHosting({ activeTab: tab, loadApps, getFirstAppKey }), {
    initialProps: { tab: activeTab },
  });
}

describe('useDownloadHosting', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    vi.clearAllMocks();
    loadApps.mockResolvedValue(undefined);
    vi.mocked(api.getDownloadStorageAdmin).mockResolvedValue({ settings: storageSettings } as Awaited<ReturnType<typeof api.getDownloadStorageAdmin>>);
    vi.mocked(api.listDownloadArtifactsAdmin).mockResolvedValue({ artifacts: [artifact] } as Awaited<ReturnType<typeof api.listDownloadArtifactsAdmin>>);
    vi.mocked(api.listDownloadArtifactsByAppAdmin).mockResolvedValue({ artifacts: [artifact] } as Awaited<ReturnType<typeof api.listDownloadArtifactsByAppAdmin>>);
    vi.mocked(api.updateDownloadStorageAdmin).mockResolvedValue({ settings: storageSettings } as Awaited<ReturnType<typeof api.updateDownloadStorageAdmin>>);
    vi.mocked(api.testDownloadStorageAdmin).mockResolvedValue({});
    vi.mocked(api.applyDownloadArtifactAdmin).mockResolvedValue({} as Awaited<ReturnType<typeof api.applyDownloadArtifactAdmin>>);
    vi.mocked(api.setArtifactAsCurrentAdmin).mockResolvedValue({} as Awaited<ReturnType<typeof api.setArtifactAsCurrentAdmin>>);
  });

  it('loads storage and artifacts only when the hosting tab becomes active', async () => {
    const { result, rerender } = renderHosting();
    expect(api.getDownloadStorageAdmin).not.toHaveBeenCalled();

    rerender({ tab: 'hosting' });
    await waitFor(() => {
      expect(result.current.storageSettings).toEqual(storageSettings);
    });

    expect(api.listDownloadArtifactsAdmin).toHaveBeenCalledWith({
      query: undefined,
      platform: undefined,
      app_key: undefined,
      page_size: 50,
    });
    expect(result.current.artifacts).toEqual([artifact]);
  });

  it('saves and tests storage configuration with user feedback', async () => {
    const { result } = renderHosting();
    await act(async () => { await result.current.loadStorage(); });

    await act(async () => { await result.current.handleSaveStorage(); });
    expect(api.updateDownloadStorageAdmin).toHaveBeenCalled();
    expect(result.current.storageSuccess).toBe('Saved storage settings.');

    await act(async () => { await result.current.handleTestStorage(); });
    expect(api.testDownloadStorageAdmin).toHaveBeenCalledOnce();
    expect(result.current.storageSuccess).toBe('Connection test succeeded.');
  });

  it('uses the by-app artifact endpoint when an app filter is selected', async () => {
    const { result } = renderHosting();
    act(() => {
      result.current.setArtifactsAppKey('desktop-app');
      result.current.setArtifactsPlatform('windows');
    });

    await act(async () => { await result.current.loadArtifacts(); });
    expect(api.listDownloadArtifactsByAppAdmin).toHaveBeenCalledWith({
      app_key: 'desktop-app', platform: 'windows', page_size: 50,
    });
  });

  it('requires an app before applying an artifact and preserves a clear error', async () => {
    const { result } = renderHosting();
    act(() => { result.current.setSelectedArtifact(artifact as typeof result.current.selectedArtifact); });

    await act(async () => { await result.current.handleApplyArtifact(); });
    expect(api.applyDownloadArtifactAdmin).not.toHaveBeenCalled();
    expect(result.current.artifactsError).toBe('Select an app to apply to.');
  });

  it('applies an artifact, refreshes apps, and reports completion', async () => {
    const { result } = renderHosting();
    act(() => {
      result.current.setSelectedArtifact(artifact as typeof result.current.selectedArtifact);
      result.current.setApplyTarget((target) => ({ ...target, appKey: 'desktop-app', releaseVersion: '1.2.3' }));
    });

    await act(async () => { await result.current.handleApplyArtifact(); });
    expect(api.applyDownloadArtifactAdmin).toHaveBeenCalledWith(expect.objectContaining({
      app_key: 'desktop-app', artifact_id: 42, platform: 'windows', release_version: '1.2.3',
    }));
    expect(loadApps).toHaveBeenCalledOnce();
    expect(result.current.storageSuccess).toBe('Applied artifact to download asset.');
  });

  it('does not start an upload until a file has been selected', async () => {
    const { result } = renderHosting();
    await act(async () => { await result.current.handleUploadArtifact(); });

    expect(api.presignDownloadArtifactUploadAdmin).not.toHaveBeenCalled();
    expect(result.current.uploadState.error).toBe('Choose a file first.');
  });

  it('surfaces storage, artifact, and current-version operation failures without leaving loading state behind', async () => {
    vi.mocked(api.getDownloadStorageAdmin).mockRejectedValueOnce(new Error('Storage unavailable'));
    vi.mocked(api.listDownloadArtifactsAdmin).mockRejectedValueOnce(new Error('Artifact listing unavailable'));
    vi.mocked(api.setArtifactAsCurrentAdmin).mockRejectedValueOnce(new Error('Current version rejected'));
    const { result } = renderHosting();

    await act(async () => { await result.current.loadStorage(); });
    await act(async () => { await result.current.loadArtifacts(); });
    await act(async () => { await result.current.handleSetArtifactAsCurrent(artifact as never, 'desktop-app', 'windows'); });

    expect(result.current.storageError).toBe('Storage unavailable');
    expect(result.current.artifactsError).toBe('Current version rejected');
    expect(result.current.storageLoading).toBe(false);
    expect(result.current.artifactsLoading).toBe(false);
  });

  it('keeps save, connection-test, and apply failures visible to the operator', async () => {
    vi.mocked(api.updateDownloadStorageAdmin).mockRejectedValueOnce('save unavailable');
    vi.mocked(api.testDownloadStorageAdmin).mockRejectedValueOnce('test unavailable');
    vi.mocked(api.applyDownloadArtifactAdmin).mockRejectedValueOnce(new Error('Apply rejected'));
    const { result } = renderHosting();
    act(() => {
      result.current.setSelectedArtifact(artifact as typeof result.current.selectedArtifact);
      result.current.setApplyTarget((target) => ({ ...target, appKey: 'desktop-app' }));
    });

    await act(async () => { await result.current.handleSaveStorage(); });
    expect(result.current.storageError).toBe('Failed to save settings');
    expect(result.current.storageSaving).toBe(false);

    await act(async () => { await result.current.handleTestStorage(); });
    expect(result.current.storageError).toBe('Connection test failed');

    await act(async () => { await result.current.handleApplyArtifact(); });
    expect(result.current.artifactsError).toBe('Apply rejected');
  });

  it('sets a current artifact and refreshes both artifact history and app state', async () => {
    const { result } = renderHosting();

    await act(async () => {
      await result.current.handleSetArtifactAsCurrent(artifact as never, 'desktop-app', 'windows');
    });

    expect(api.setArtifactAsCurrentAdmin).toHaveBeenCalledWith({ artifact_id: 42, app_key: 'desktop-app', platform: 'windows' });
    expect(loadApps).toHaveBeenCalledOnce();
    expect(result.current.storageSuccess).toBe('Set app.exe as the latest version.');
  });

  it('uploads a selected installer with safe headers, commits it, and refreshes artifact history', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200 });
    vi.stubGlobal('fetch', fetchMock);
    vi.mocked(api.presignDownloadArtifactUploadAdmin).mockResolvedValue({
      upload_url: 'https://storage.example.test/upload', bucket: 'releases', object_key: 'desktop-app/1.2.3/app.exe', required_headers: { Host: 'storage.example.test', 'x-amz-meta-version': '1.2.3' },
    } as never);
    vi.mocked(api.commitDownloadArtifactAdmin).mockResolvedValue({} as never);
    const { result } = renderHosting();
    const file = new File(['binary'], 'app.exe', { type: 'application/vnd.microsoft.portable-executable' });
    act(() => { result.current.setUploadState({ file, appKey: 'desktop-app', platform: 'windows', releaseVersion: '1.2.3', busy: false, message: '', error: '' }); });

    await act(async () => { await result.current.handleUploadArtifact(); });

    expect(api.presignDownloadArtifactUploadAdmin).toHaveBeenCalledWith(expect.objectContaining({ filename: 'app.exe', app_key: 'desktop-app', platform: 'windows' }));
    expect(fetchMock).toHaveBeenCalledWith('https://storage.example.test/upload', expect.objectContaining({ method: 'PUT', body: file }));
    expect(api.commitDownloadArtifactAdmin).toHaveBeenCalledWith(expect.objectContaining({ bucket: 'releases', object_key: 'desktop-app/1.2.3/app.exe' }));
    expect(result.current.uploadState).toMatchObject({ file: null, message: 'Upload committed.', error: '' });
  });

  it('preserves the selected file and reports failed upload responses', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: false, status: 403 });
    vi.stubGlobal('fetch', fetchMock);
    vi.mocked(api.presignDownloadArtifactUploadAdmin).mockResolvedValue({
      upload_url: 'https://storage.example.test/upload', bucket: 'releases', object_key: 'app.exe', required_headers: {},
    } as never);
    const { result } = renderHosting();
    const file = new File(['binary'], 'app.exe');
    act(() => { result.current.setUploadState({ file, appKey: '', platform: '', releaseVersion: '', busy: false, message: '', error: '' }); });

    await act(async () => { await result.current.handleUploadArtifact(); });

    expect(api.commitDownloadArtifactAdmin).not.toHaveBeenCalled();
    expect(result.current.uploadState).toMatchObject({ file, busy: false, error: 'Upload failed (403)' });
  });
});
