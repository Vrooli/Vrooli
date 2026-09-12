import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useRemoteProfilesForm } from './useRemoteProfilesForm';
import type {
  createRemoteProfile,
  deleteRemoteProfile,
  fetchIncomingRemoteProfileSessions,
  fetchRemoteProfiles,
  getRemoteProfileSessionLinks,
  loginRemoteProfile,
  logoutRemoteProfile,
  revokeIncomingRemoteProfileSession,
  revokeRemoteProfileSessions,
  testRemoteProfile,
  updateRemoteProfile,
} from '../services/remoteProfiles.service';
import type { IncomingRemoteProfileSession, RemoteProfile } from '../../../shared/api';

type FetchRemoteProfilesFn = typeof fetchRemoteProfiles;
type FetchIncomingFn = typeof fetchIncomingRemoteProfileSessions;
type GetLinksFn = typeof getRemoteProfileSessionLinks;
type RevokeRemoteFn = typeof revokeRemoteProfileSessions;
type RevokeIncomingFn = typeof revokeIncomingRemoteProfileSession;
type CreateFn = typeof createRemoteProfile;
type UpdateFn = typeof updateRemoteProfile;
type DeleteFn = typeof deleteRemoteProfile;
type LoginFn = typeof loginRemoteProfile;
type LogoutFn = typeof logoutRemoteProfile;
type TestFn = typeof testRemoteProfile;

const fetchRemoteProfilesMock = vi.fn<FetchRemoteProfilesFn>();
const fetchIncomingRemoteProfileSessionsMock = vi.fn<FetchIncomingFn>();
const getRemoteProfileSessionLinksMock = vi.fn<GetLinksFn>();
const revokeRemoteProfileSessionsMock = vi.fn<RevokeRemoteFn>();
const revokeIncomingRemoteProfileSessionMock = vi.fn<RevokeIncomingFn>();
const createRemoteProfileMock = vi.fn<CreateFn>();
const updateRemoteProfileMock = vi.fn<UpdateFn>();
const deleteRemoteProfileMock = vi.fn<DeleteFn>();
const loginRemoteProfileMock = vi.fn<LoginFn>();
const logoutRemoteProfileMock = vi.fn<LogoutFn>();
const testRemoteProfileMock = vi.fn<TestFn>();

vi.mock('../services/remoteProfiles.service', async () => {
  const actual = await vi.importActual<typeof import('../services/remoteProfiles.service')>(
    '../services/remoteProfiles.service'
  );
  return {
    ...actual,
    fetchRemoteProfiles: (...args: Parameters<FetchRemoteProfilesFn>) => fetchRemoteProfilesMock(...args),
    fetchIncomingRemoteProfileSessions: (...args: Parameters<FetchIncomingFn>) =>
      fetchIncomingRemoteProfileSessionsMock(...args),
    getRemoteProfileSessionLinks: (...args: Parameters<GetLinksFn>) => getRemoteProfileSessionLinksMock(...args),
    revokeRemoteProfileSessions: (...args: Parameters<RevokeRemoteFn>) => revokeRemoteProfileSessionsMock(...args),
    revokeIncomingRemoteProfileSession: (...args: Parameters<RevokeIncomingFn>) =>
      revokeIncomingRemoteProfileSessionMock(...args),
    createRemoteProfile: (...args: Parameters<CreateFn>) => createRemoteProfileMock(...args),
    updateRemoteProfile: (...args: Parameters<UpdateFn>) => updateRemoteProfileMock(...args),
    deleteRemoteProfile: (...args: Parameters<DeleteFn>) => deleteRemoteProfileMock(...args),
    loginRemoteProfile: (...args: Parameters<LoginFn>) => loginRemoteProfileMock(...args),
    logoutRemoteProfile: (...args: Parameters<LogoutFn>) => logoutRemoteProfileMock(...args),
    testRemoteProfile: (...args: Parameters<TestFn>) => testRemoteProfileMock(...args),
  };
});

const baseProfile: RemoteProfile = {
  id: 1,
  tag: 'prod',
  label: 'Production',
  api_base: 'https://example.com/api/v1',
  connector_id: 'connector-1',
  remote_session_id: 'remote-session-1',
  status: 'active',
  has_session: true,
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

const incomingSession: IncomingRemoteProfileSession = {
  session_id: 'remote-session-1',
  admin_email: 'admin@example.com',
  connector_id: 'connector-1',
  profile_tag: 'prod',
  created_at: '2025-01-01T00:00:00Z',
  last_activity: '2025-01-01T01:00:00Z',
  expires_at: '2025-01-01T02:00:00Z',
};

describe('useRemoteProfilesForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchRemoteProfilesMock.mockResolvedValue([baseProfile]);
    fetchIncomingRemoteProfileSessionsMock.mockResolvedValue([incomingSession]);
  });

  it('loads profiles and incoming sessions on mount', async () => {
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.profiles).toHaveLength(1);
    expect(result.current.incomingSessions).toHaveLength(1);
    expect(fetchRemoteProfilesMock).toHaveBeenCalledTimes(1);
    expect(fetchIncomingRemoteProfileSessionsMock).toHaveBeenCalledTimes(1);
  });

  it('loads and stores session links per profile id', async () => {
    getRemoteProfileSessionLinksMock.mockResolvedValue({
      profile_id: 1,
      profile_tag: 'prod',
      connector_id: 'connector-1',
      local_has_session: true,
      local_status: 'active',
      remote_sessions: [incomingSession],
    });
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    let action: Awaited<ReturnType<typeof result.current.handleLoadSessionLinks>> | undefined;
    await act(async () => {
      action = await result.current.handleLoadSessionLinks(1);
    });

    expect(action?.success).toBe(true);
    expect(result.current.sessionLinksByProfileId[1]?.remote_sessions).toHaveLength(1);
  });

  it('revokes remote sessions and refreshes profiles/incoming sessions', async () => {
    revokeRemoteProfileSessionsMock.mockResolvedValue({
      profile_id: 1,
      profile_tag: 'prod',
      connector_id: 'connector-1',
      local_has_session: false,
      local_status: 'expired',
      remote_sessions: [],
    });
    fetchRemoteProfilesMock.mockResolvedValueOnce([baseProfile]).mockResolvedValueOnce([{ ...baseProfile, has_session: false }]);
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    let action: Awaited<ReturnType<typeof result.current.handleRevokeRemoteSessions>> | undefined;
    await act(async () => {
      action = await result.current.handleRevokeRemoteSessions(1);
    });

    expect(action?.success).toBe(true);
    expect(fetchRemoteProfilesMock).toHaveBeenCalledTimes(2);
    expect(fetchIncomingRemoteProfileSessionsMock).toHaveBeenCalledTimes(2);
    expect(result.current.profiles[0]?.has_session).toBe(false);
  });

  it('removes revoked incoming session from local state', async () => {
    revokeIncomingRemoteProfileSessionMock.mockResolvedValue();
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    let action: Awaited<ReturnType<typeof result.current.handleRevokeIncomingSession>> | undefined;
    await act(async () => {
      action = await result.current.handleRevokeIncomingSession('remote-session-1');
    });

    expect(action?.success).toBe(true);
    expect(revokeIncomingRemoteProfileSessionMock).toHaveBeenCalledWith('remote-session-1');
    expect(result.current.incomingSessions).toHaveLength(0);
  });

  it('returns service error message when create fails', async () => {
    createRemoteProfileMock.mockRejectedValue(new Error('create failed'));
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    let action: Awaited<ReturnType<typeof result.current.handleCreate>> | undefined;
    await act(async () => {
      action = await result.current.handleCreate({
        tag: 'prod',
        label: 'Production',
        apiBase: 'https://example.com/api/v1',
      });
    });

    expect(action?.success).toBe(false);
    expect(action?.message).toBe('create failed');
  });

  it('creates, updates, logs into, tests, and logs out profiles while preserving the newest service state', async () => {
    const created: RemoteProfile = { ...baseProfile, id: 2, tag: 'staging', has_session: false };
    const updated: RemoteProfile = { ...created, label: 'Staging deployment' };
    const loggedIn: RemoteProfile = { ...updated, has_session: true, remote_session_id: 'remote-session-2' };
    const tested: RemoteProfile = { ...loggedIn, status: 'active' };
    const loggedOut: RemoteProfile = { ...tested, has_session: false, remote_session_id: null };
    createRemoteProfileMock.mockResolvedValue(created);
    updateRemoteProfileMock.mockResolvedValue(updated);
    loginRemoteProfileMock.mockResolvedValue(loggedIn);
    testRemoteProfileMock.mockResolvedValue(tested);
    logoutRemoteProfileMock.mockResolvedValue(loggedOut);
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    const form = { tag: 'staging', label: 'Staging', apiBase: 'https://staging.example/api/v1' };
    await act(async () => {
      await expect(result.current.handleCreate(form)).resolves.toMatchObject({ success: true, profile: created });
      await expect(result.current.handleUpdate(2, { ...form, label: 'Staging deployment' })).resolves.toMatchObject({ success: true, profile: updated });
      await expect(result.current.handleLogin(2, { email: 'admin@example.com', password: 'safe-password' })).resolves.toMatchObject({ success: true, profile: loggedIn });
      await expect(result.current.handleTest(2)).resolves.toMatchObject({ success: true, profile: tested });
      await expect(result.current.handleLogout(2)).resolves.toMatchObject({ success: true, profile: loggedOut });
    });

    expect(result.current.profiles.find((profile) => profile.id === 2)).toEqual(loggedOut);
    expect(result.current.actions).toMatchObject({ creating: false, updatingId: null, loginId: null, testingId: null, logoutId: null });
  });

  it('returns safe fallback messages and clears busy state when profile actions fail unexpectedly', async () => {
    updateRemoteProfileMock.mockRejectedValue('not-an-error');
    deleteRemoteProfileMock.mockRejectedValue('not-an-error');
    loginRemoteProfileMock.mockRejectedValue('not-an-error');
    logoutRemoteProfileMock.mockRejectedValue('not-an-error');
    testRemoteProfileMock.mockRejectedValue('not-an-error');
    getRemoteProfileSessionLinksMock.mockRejectedValue('not-an-error');
    revokeRemoteProfileSessionsMock.mockRejectedValue('not-an-error');
    revokeIncomingRemoteProfileSessionMock.mockRejectedValue('not-an-error');
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    await act(async () => {
      await expect(result.current.handleUpdate(1, { tag: 'prod', label: '', apiBase: baseProfile.api_base })).resolves.toMatchObject({ success: false, message: 'Failed to update profile' });
      await expect(result.current.handleDelete(1)).resolves.toMatchObject({ success: false, message: 'Failed to delete profile' });
      await expect(result.current.handleLogin(1, { email: 'admin@example.com', password: 'password' })).resolves.toMatchObject({ success: false, message: 'Remote login failed' });
      await expect(result.current.handleLogout(1)).resolves.toMatchObject({ success: false, message: 'Remote logout failed' });
      await expect(result.current.handleTest(1)).resolves.toMatchObject({ success: false, message: 'Remote session test failed' });
      await expect(result.current.handleLoadSessionLinks(1)).resolves.toMatchObject({ success: false, message: 'Failed to load remote session state' });
      await expect(result.current.handleRevokeRemoteSessions(1)).resolves.toMatchObject({ success: false, message: 'Failed to revoke remote sessions' });
      await expect(result.current.handleRevokeIncomingSession('remote-session-1')).resolves.toMatchObject({ success: false, message: 'Failed to revoke incoming session' });
    });

    expect(result.current.actions).toEqual(expect.objectContaining({
      updatingId: null, deletingId: null, loginId: null, logoutId: null, testingId: null,
      loadingLinksId: null, remoteRevokeId: null, incomingRevokeSessionId: null,
    }));
  });

  it('keeps initial profile-load errors visible while allowing a later silent refresh to recover', async () => {
    fetchRemoteProfilesMock
      .mockRejectedValueOnce(new Error('initial directory failure'))
      .mockResolvedValueOnce([baseProfile]);
    const { result } = renderHook(() => useRemoteProfilesForm());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    expect(result.current.error).toBe('initial directory failure');
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBeNull();
    expect(result.current.actions.refreshing).toBe(false);
  });
});
