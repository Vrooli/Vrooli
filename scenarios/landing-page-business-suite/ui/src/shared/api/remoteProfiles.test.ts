import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as remoteProfiles from './remoteProfiles';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('remote profile API transport', () => {
  beforeEach(() => { vi.clearAllMocks(); mockApiCall.mockResolvedValue({} as never); });

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
});
