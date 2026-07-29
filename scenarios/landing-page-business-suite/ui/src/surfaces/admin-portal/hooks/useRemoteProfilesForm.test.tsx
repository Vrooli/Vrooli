import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useRemoteProfilesForm } from './useRemoteProfilesForm';
import type {
  createRemoteProfile,
  fetchIncomingRemoteProfileSessions,
  fetchRemoteProfiles,
  getRemoteProfileSessionLinks,
  revokeIncomingRemoteProfileSession,
  revokeRemoteProfileSessions,
} from '../services/remoteProfiles.service';
import type { IncomingRemoteProfileSession, RemoteProfile } from '../../../shared/api';

type FetchRemoteProfilesFn = typeof fetchRemoteProfiles;
type FetchIncomingFn = typeof fetchIncomingRemoteProfileSessions;
type GetLinksFn = typeof getRemoteProfileSessionLinks;
type RevokeRemoteFn = typeof revokeRemoteProfileSessions;
type RevokeIncomingFn = typeof revokeIncomingRemoteProfileSession;
type CreateFn = typeof createRemoteProfile;

const fetchRemoteProfilesMock = vi.fn<FetchRemoteProfilesFn>();
const fetchIncomingRemoteProfileSessionsMock = vi.fn<FetchIncomingFn>();
const getRemoteProfileSessionLinksMock = vi.fn<GetLinksFn>();
const revokeRemoteProfileSessionsMock = vi.fn<RevokeRemoteFn>();
const revokeIncomingRemoteProfileSessionMock = vi.fn<RevokeIncomingFn>();
const createRemoteProfileMock = vi.fn<CreateFn>();

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
});
