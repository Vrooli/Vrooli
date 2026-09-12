import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useStorageWizard } from './useStorageWizard';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api', async () => ({
  ...(await vi.importActual('../../../shared/api')),
  getDownloadStorageAdmin: vi.fn(), updateDownloadStorageAdmin: vi.fn(), testDownloadStorageAdmin: vi.fn(),
}));

const settings = {
  provider: 's3' as const, bucket: 'releases', region: 'us-east-1', endpoint: '', force_path_style: false, default_prefix: 'apps/',
  signed_url_ttl_seconds: 900, public_base_url: '', access_key_id_set: true, secret_access_key_set: true,
  session_token_set: false, credentials_from_env: false, settings_row_available: true,
};

describe('useStorageWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getDownloadStorageAdmin).mockResolvedValue({ settings });
    vi.mocked(api.updateDownloadStorageAdmin).mockResolvedValue({ settings });
    vi.mocked(api.testDownloadStorageAdmin).mockResolvedValue({});
  });

  it('enforces provider and bucket progression before exposing credential and verification steps', () => {
    const { result } = renderHook(() => useStorageWizard({ onComplete: vi.fn() }));
    expect(result.current).toMatchObject({ currentStepId: 'provider', canGoNext: false, canGoBack: false });

    act(() => { result.current.setProvider('aws-s3'); });
    expect(result.current.canGoNext).toBe(true);
    act(() => { result.current.goNext(); });
    expect(result.current.currentStepId).toBe('configure');
    expect(result.current.canGoNext).toBe(false);
    act(() => { result.current.setForm({ bucket: 'releases' }); });
    act(() => { result.current.goNext(); });
    act(() => { result.current.goNext(); });
    expect(result.current).toMatchObject({ currentStepId: 'verify', isLastStep: true, canGoNext: false });
  });

  it('loads R2 settings, identifies the provider, and derives the account endpoint without persisting secrets', async () => {
    vi.mocked(api.getDownloadStorageAdmin).mockResolvedValue({ settings: { ...settings, endpoint: 'https://acct-123.r2.cloudflarestorage.com', region: 'auto' } });
    const { result } = renderHook(() => useStorageWizard({ onComplete: vi.fn() }));
    await act(async () => { await result.current.loadExistingSettings(); });

    expect(result.current.state).toMatchObject({ provider: 'cloudflare-r2', cloudflareAccountId: 'acct-123', loading: false });
    expect(result.current.state.form).toMatchObject({ bucket: 'releases', endpoint: 'https://acct-123.r2.cloudflarestorage.com' });
  });

  it('surfaces a connection-test failure and permits a successful retry', async () => {
    vi.mocked(api.testDownloadStorageAdmin).mockRejectedValueOnce(new Error('Access key rejected'));
    const { result } = renderHook(() => useStorageWizard({ onComplete: vi.fn() }));
    await act(async () => { await result.current.testConnection(); });
    expect(result.current.state).toMatchObject({ testStatus: 'error', testError: 'Access key rejected' });

    await act(async () => { await result.current.testConnection(); });
    expect(result.current.state).toMatchObject({ testStatus: 'success', testError: null });
  });

  it('saves an explicit provider configuration and completes only after durable persistence', async () => {
    const onComplete = vi.fn();
    const { result } = renderHook(() => useStorageWizard({ onComplete }));
    act(() => {
      result.current.setProvider('minio');
      result.current.setForm({ bucket: 'releases', endpoint: 'http://minio.local:9000', region: 'us-east-1' });
      result.current.setCredentials({ accessKeyId: 'operator', secretAccessKey: 'secret' });
    });
    await act(async () => { await result.current.saveSettings(); });

    await waitFor(() => { expect(api.updateDownloadStorageAdmin).toHaveBeenCalledWith(expect.objectContaining({ bucket: 'releases', endpoint: 'http://minio.local:9000', access_key_id: 'operator' })); });
    expect(result.current.state.saveStatus).toBe('success');
    expect(onComplete).toHaveBeenCalledOnce();
  });

  it.each([
    ['minio', { ...settings, endpoint: 'http://minio.test:9000', force_path_style: true }, 'minio'],
    ['aws', { ...settings, endpoint: '', region: 'eu-west-1', force_path_style: false }, 'aws-s3'],
    ['custom', { ...settings, endpoint: 'https://object.example.test', region: '', force_path_style: false }, 'custom'],
  ] as const)('detects %s provider settings without assuming credentials are present', async (_name, storedSettings, provider) => {
    vi.mocked(api.getDownloadStorageAdmin).mockResolvedValue({ settings: storedSettings });
    const { result } = renderHook(() => useStorageWizard({ onComplete: vi.fn() }));

    await act(async () => { await result.current.loadExistingSettings(); });

    expect(result.current.state).toMatchObject({ provider, loading: false, loadError: null });
    expect(result.current.state.form.bucket).toBe('releases');
  });

  it('retains a safe wizard state when settings cannot be loaded or the failure is non-Error', async () => {
    vi.mocked(api.getDownloadStorageAdmin).mockRejectedValueOnce(new Error('Storage control plane unavailable'));
    const { result } = renderHook(() => useStorageWizard({ onComplete: vi.fn() }));

    await act(async () => { await result.current.loadExistingSettings(); });
    expect(result.current.state).toMatchObject({ loading: false, loadError: 'Storage control plane unavailable', provider: null });

    vi.mocked(api.getDownloadStorageAdmin).mockRejectedValueOnce('unknown failure');
    await act(async () => { await result.current.loadExistingSettings(); });
    expect(result.current.state.loadError).toBe('Failed to load settings');
  });

  it('does not navigate past bounds and resets all transient wizard state', () => {
    const { result } = renderHook(() => useStorageWizard({ onComplete: vi.fn() }));
    act(() => {
      result.current.goToStep(-1);
      result.current.goToStep(99);
      result.current.goBack();
      result.current.setProvider('cloudflare-r2');
      result.current.setCloudflareAccountId('account-42');
      result.current.goToStep(3);
      result.current.goNext();
      result.current.reset();
    });
    expect(result.current.state).toMatchObject({ step: 0, provider: null, cloudflareAccountId: '', testStatus: 'idle', saveStatus: 'idle' });
    expect(result.current.currentStepId).toBe('provider');
  });

  it('reports safe fallback messages for non-Error test and save failures without completing', async () => {
    const onComplete = vi.fn();
    vi.mocked(api.testDownloadStorageAdmin).mockRejectedValueOnce('network failure');
    vi.mocked(api.updateDownloadStorageAdmin).mockRejectedValueOnce('persistence failure');
    const { result } = renderHook(() => useStorageWizard({ onComplete }));

    await act(async () => { await result.current.testConnection(); });
    expect(result.current.state).toMatchObject({ testStatus: 'error', testError: 'Connection test failed' });
    await act(async () => { await result.current.saveSettings(); });
    expect(result.current.state).toMatchObject({ saveStatus: 'error', saveError: 'Failed to save settings' });
    expect(onComplete).not.toHaveBeenCalled();
  });
});
