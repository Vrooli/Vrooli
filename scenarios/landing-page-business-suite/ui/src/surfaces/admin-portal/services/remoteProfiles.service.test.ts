import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from '../../../shared/api';
import type { RemoteProfile } from '../../../shared/api';
import {
  createRemoteProfile,
  DEFAULT_REMOTE_PROFILE_FORM,
  DEFAULT_REMOTE_PROFILE_LOGIN,
  deleteRemoteProfile,
  fetchIncomingRemoteProfileSessions,
  fetchRemoteProfiles,
  getRemoteProfileSessionLinks,
  getRemoteProfileStatusMeta,
  loginRemoteProfile,
  logoutRemoteProfile,
  normalizeRemoteProfileForm,
  revokeIncomingRemoteProfileSession,
  revokeRemoteProfileSessions,
  testRemoteProfile,
  updateRemoteProfile,
  validateRemoteProfileForm,
  validateRemoteProfileLoginForm,
} from './remoteProfiles.service';

vi.mock('../../../shared/api');

describe('remote profile service', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('normalizes and validates profile and login forms before sending credentials', () => {
    expect(DEFAULT_REMOTE_PROFILE_FORM).toEqual({ tag: '', label: '', apiBase: '' });
    expect(DEFAULT_REMOTE_PROFILE_LOGIN).toEqual({ email: '', password: '' });
    expect(normalizeRemoteProfileForm({ tag: ' partner ', label: ' Partner admin ', apiBase: ' https://partner.example/api/v1 ' })).toEqual({ tag: 'partner', label: 'Partner admin', api_base: 'https://partner.example/api/v1' });
    expect(validateRemoteProfileForm({ tag: '', label: '', apiBase: '' })).toBe('Tag is required');
    expect(validateRemoteProfileForm({ tag: 'partner', label: '', apiBase: '' })).toBe('API base is required');
    expect(validateRemoteProfileForm({ tag: 'partner', label: '', apiBase: 'https://partner.example' })).toBe('API base must include /api/v1');
    expect(validateRemoteProfileForm({ tag: 'partner', label: '', apiBase: 'https://partner.example/api/v1' })).toBeNull();
    expect(validateRemoteProfileLoginForm({ email: '', password: '' })).toBe('Email is required');
    expect(validateRemoteProfileLoginForm({ email: 'admin@example.com', password: '' })).toBe('Password is required');
    expect(validateRemoteProfileLoginForm({ email: 'admin@example.com', password: 'secret' })).toBeNull();
  });

  it('maps profile status to operator-facing state', () => {
    const profile = (status: RemoteProfile['status'], hasSession = true): RemoteProfile => ({
      id: 1, tag: 'partner', api_base: 'https://partner.example/api/v1', status, has_session: hasSession,
      created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
    });
    expect(getRemoteProfileStatusMeta(profile('active'))).toMatchObject({ label: 'Active', tone: 'success', description: 'Session active' });
    expect(getRemoteProfileStatusMeta(profile('expired'))).toMatchObject({ label: 'Expired', tone: 'warning' });
    expect(getRemoteProfileStatusMeta(profile('error'))).toMatchObject({ label: 'Error', tone: 'error' });
    expect(getRemoteProfileStatusMeta(profile('unknown', false))).toMatchObject({ label: 'Unknown', tone: 'info', description: 'No session' });
  });

  it('delegates lifecycle, session, and incoming-session actions with normalized results', async () => {
    vi.mocked(api.listRemoteProfilesAdmin).mockResolvedValue({ profiles: [] });
    vi.mocked(api.createRemoteProfileAdmin).mockResolvedValue({ id: 1 } as never);
    vi.mocked(api.updateRemoteProfileAdmin).mockResolvedValue({ id: 1 } as never);
    vi.mocked(api.loginRemoteProfileAdmin).mockResolvedValue({ id: 1 } as never);
    vi.mocked(api.logoutRemoteProfileAdmin).mockResolvedValue({ id: 1 } as never);
    vi.mocked(api.testRemoteProfileAdmin).mockResolvedValue({ id: 1 } as never);
    vi.mocked(api.getRemoteProfileSessionLinksAdmin).mockResolvedValue({ profile_id: 1 } as never);
    vi.mocked(api.revokeRemoteProfileSessionsAdmin).mockResolvedValue({ profile_id: 1 } as never);
    vi.mocked(api.listIncomingRemoteProfileSessionsAdmin).mockResolvedValue({} as never);
    vi.mocked(api.deleteRemoteProfileAdmin).mockResolvedValue({});
    vi.mocked(api.revokeIncomingRemoteProfileSessionAdmin).mockResolvedValue({});

    await expect(fetchRemoteProfiles()).resolves.toEqual([]);
    await createRemoteProfile({ tag: ' partner ', label: ' Partner ', apiBase: ' https://partner.example/api/v1 ' });
    await updateRemoteProfile(1, { tag: ' partner ', label: '', apiBase: ' https://partner.example/api/v1 ' });
    await deleteRemoteProfile(1);
    await loginRemoteProfile(1, { email: ' admin@example.com ', password: 'secret' });
    await logoutRemoteProfile(1);
    await testRemoteProfile(1);
    await getRemoteProfileSessionLinks(1);
    await revokeRemoteProfileSessions(1);
    await expect(fetchIncomingRemoteProfileSessions('connector-1')).resolves.toEqual([]);
    await revokeIncomingRemoteProfileSession('session-1');
    expect(api.createRemoteProfileAdmin).toHaveBeenCalledWith({ tag: 'partner', label: 'Partner', api_base: 'https://partner.example/api/v1' });
    expect(api.loginRemoteProfileAdmin).toHaveBeenCalledWith(1, { email: 'admin@example.com', password: 'secret' });
    expect(api.getRemoteProfileSessionLinksAdmin).toHaveBeenCalledWith(1);
    expect(api.revokeRemoteProfileSessionsAdmin).toHaveBeenCalledWith(1);
    expect(api.listIncomingRemoteProfileSessionsAdmin).toHaveBeenCalledWith('connector-1');
  });
});
