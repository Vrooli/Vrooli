import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as remoteProfiles from './remoteProfiles';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('remote profile API transport', () => {
  beforeEach(() => { vi.resetAllMocks(); mockApiCall.mockResolvedValue({} as never); });

  it('keeps invalid list payloads safely empty and sends all profile lifecycle requests to their scoped endpoints', async () => {
    await expect(remoteProfiles.listRemoteProfilesAdmin()).resolves.toEqual({ profiles: [] });
    await expect(remoteProfiles.createRemoteProfileAdmin({ tag: 'partner', api_base: 'https://partner.example/api/v1' })).rejects.toThrow('Invalid remote profile response from API');
    await expect(remoteProfiles.updateRemoteProfileAdmin(7, { label: 'Partner' })).rejects.toThrow('Invalid remote profile response from API');
    await expect(remoteProfiles.deleteRemoteProfileAdmin(7)).resolves.toEqual({});
    await expect(remoteProfiles.loginRemoteProfileAdmin(7, { email: 'admin@example.com', password: 'secret' })).rejects.toThrow('Invalid remote profile response from API');
    await expect(remoteProfiles.logoutRemoteProfileAdmin(7)).rejects.toThrow('Invalid remote profile response from API');
    await expect(remoteProfiles.testRemoteProfileAdmin(7)).rejects.toThrow('Invalid remote profile response from API');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7', expect.objectContaining({ method: 'PUT' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7', { method: 'DELETE' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7/login', expect.objectContaining({ method: 'POST' }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7/logout', { method: 'POST' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7/test', { method: 'POST' });
  });

  it('validates session links and keeps malformed incoming-session lists safely empty', async () => {
    await expect(remoteProfiles.getRemoteProfileSessionLinksAdmin(7)).rejects.toThrow('Invalid remote profile session links response from API');
    await expect(remoteProfiles.revokeRemoteProfileSessionsAdmin(7)).rejects.toThrow('Invalid remote profile session links response from API');
    await expect(remoteProfiles.listIncomingRemoteProfileSessionsAdmin()).resolves.toEqual({ sessions: [] });
    await expect(remoteProfiles.listIncomingRemoteProfileSessionsAdmin(' connector/1 ')).resolves.toEqual({ sessions: [] });
    await expect(remoteProfiles.revokeIncomingRemoteProfileSessionAdmin('session/1')).resolves.toEqual({});
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7/session-links');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profiles/7/remote-revoke', { method: 'POST' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profile-sessions');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profile-sessions?connector_id=connector%2F1');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/remote-profile-sessions/session%2F1', { method: 'DELETE' });
  });

  it('returns validated profile and session records for lifecycle actions', async () => {
    const profile = {
      id: 7, tag: 'partner', label: 'Partner', api_base: 'https://partner.example/api/v1', status: 'active', has_session: true,
      session_expires_at: '2026-02-01T00:00:00Z', last_login_at: '2026-01-01T00:00:00Z', last_used_at: '2026-01-02T00:00:00Z',
      created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-02T00:00:00Z',
    };
    const incoming = { session_id: 'session-1', admin_email: 'admin@example.com', connector_id: 'connector-1', created_at: '2026-01-01T00:00:00Z', last_activity: '2026-01-01T01:00:00Z', expires_at: '2026-02-01T00:00:00Z' };
    const links = { profile_id: 7, profile_tag: 'partner', connector_id: 'connector-1', local_has_session: true, local_status: 'active', local_session_expires_at: '2026-02-01T00:00:00Z', remote_sessions: [incoming] };
    mockApiCall
      .mockResolvedValueOnce({ profiles: [profile] } as never)
      .mockResolvedValueOnce(profile as never)
      .mockResolvedValueOnce(profile as never)
      .mockResolvedValueOnce(profile as never)
      .mockResolvedValueOnce(profile as never)
      .mockResolvedValueOnce(profile as never)
      .mockResolvedValueOnce(links as never)
      .mockResolvedValueOnce(links as never)
      .mockResolvedValueOnce({ sessions: [incoming] } as never);

    await expect(remoteProfiles.listRemoteProfilesAdmin()).resolves.toEqual({ profiles: [profile] });
    await expect(remoteProfiles.createRemoteProfileAdmin({ tag: 'partner', api_base: profile.api_base })).resolves.toMatchObject({ id: 7 });
    await expect(remoteProfiles.updateRemoteProfileAdmin(7, { label: 'Partner' })).resolves.toMatchObject({ status: 'active' });
    await expect(remoteProfiles.loginRemoteProfileAdmin(7, { email: 'admin@example.com', password: 'safe-password' })).resolves.toMatchObject({ has_session: true });
    await expect(remoteProfiles.logoutRemoteProfileAdmin(7)).resolves.toMatchObject({ tag: 'partner' });
    await expect(remoteProfiles.testRemoteProfileAdmin(7)).resolves.toMatchObject({ api_base: profile.api_base });
    await expect(remoteProfiles.getRemoteProfileSessionLinksAdmin(7)).resolves.toEqual(links);
    await expect(remoteProfiles.revokeRemoteProfileSessionsAdmin(7)).resolves.toEqual(links);
    await expect(remoteProfiles.listIncomingRemoteProfileSessionsAdmin('connector-1')).resolves.toEqual({ sessions: [incoming] });
  });
});
