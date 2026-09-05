import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useUserManagement } from './useUserManagement';
import * as usersService from '../services/users.service';
import type { UserAccount, UserSession } from '../services/users.service';

vi.mock('../services/users.service', async () => ({
  ...(await vi.importActual('../services/users.service')),
  listUsers: vi.fn(),
  getUserDetails: vi.fn(),
  getUserSessions: vi.fn(),
  revokeSession: vi.fn(),
  revokeAllSessions: vi.fn(),
}));

const listUsers = vi.mocked(usersService.listUsers);
const getUserDetails = vi.mocked(usersService.getUserDetails);
const getUserSessions = vi.mocked(usersService.getUserSessions);
const revokeSession = vi.mocked(usersService.revokeSession);
const revokeAllSessions = vi.mocked(usersService.revokeAllSessions);

const user: UserAccount = {
  id: 'user_123',
  email: 'customer@example.com',
  email_verified: true,
  created_at: '2026-01-01T00:00:00Z',
  session_count: 2,
  subscription: { plan_tier: 'pro', status: 'active' },
  credits: { balance: 100, bonus: 25 },
};

const activeSession: UserSession = {
  id: 'session_123',
  created_at: '2026-01-01T00:00:00Z',
  last_used_at: '2026-01-02T00:00:00Z',
  expires_at: '2030-01-01T00:00:00Z',
  revoked: false,
};

describe('useUserManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listUsers.mockResolvedValue({ users: [user], total: 1, page: 1, per_page: 20, total_pages: 1 });
    getUserDetails.mockResolvedValue(user);
    getUserSessions.mockResolvedValue([activeSession]);
    revokeSession.mockResolvedValue({ success: true, message: 'Session revoked' });
    revokeAllSessions.mockResolvedValue({ success: true, message: 'Sessions revoked', sessions_revoked: 1 });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('loads the first customer page with the declared pagination contract', async () => {
    const { result } = renderHook(() => useUserManagement());

    await waitFor(() => { expect(result.current.loading).toBe(false); });

    expect(listUsers).toHaveBeenCalledWith({ search: undefined, page: 1, per_page: 20 });
    expect(result.current.users).toEqual([user]);
    expect(result.current.total).toBe(1);
  });

  it('uses safe messages and clears loading state for non-Error list, detail, and session failures', async () => {
    listUsers.mockRejectedValue('offline');
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.error).toBe('Failed to load users');

    getUserDetails.mockRejectedValue('offline');
    await act(async () => { await result.current.loadUserDetails(user.id); });
    expect(result.current.error).toBe('Failed to load user details');
    expect(result.current.detailsLoading).toBe(false);

    getUserSessions.mockRejectedValue('offline');
    await act(async () => { await result.current.loadUserSessions(user.id); });
    expect(result.current.error).toBe('Failed to load sessions');
    expect(result.current.sessionsLoading).toBe(false);
  });

  it('debounces search, resets pagination, and loads the filtered customer list', async () => {
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    act(() => { result.current.setPage(2); });
    await waitFor(() => { expect(listUsers).toHaveBeenLastCalledWith({ search: undefined, page: 2, per_page: 20 }); });

    act(() => { result.current.setSearch('customer@example.com'); });
    await waitFor(() => { expect(listUsers).toHaveBeenLastCalledWith({ search: 'customer@example.com', page: 1, per_page: 20 }); });
  });

  it('loads sessions for the selected customer and clears them on deselection', async () => {
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    act(() => { result.current.selectUser(user); });
    await act(async () => { await result.current.loadUserSessions(user.id); });

    expect(getUserSessions).toHaveBeenCalledWith(user.id);
    expect(result.current.selectedUserSessions).toEqual([activeSession]);

    act(() => { result.current.selectUser(null); });
    expect(result.current.selectedUser).toBeNull();
    expect(result.current.selectedUserSessions).toEqual([]);
  });

  it('marks a revoked session locally and refreshes the selected customer details', async () => {
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    act(() => { result.current.selectUser(user); });
    await act(async () => { await result.current.loadUserSessions(user.id); });

    let revokeResult: { success: boolean; message?: string } | undefined;
    await act(async () => { revokeResult = await result.current.handleRevokeSession(user.id, activeSession.id); });

    expect(revokeResult).toEqual({ success: true });
    expect(revokeSession).toHaveBeenCalledWith(user.id, activeSession.id);
    expect(result.current.selectedUserSessions[0]).toMatchObject({ id: activeSession.id, revoked: true });
    expect(getUserDetails).toHaveBeenCalledWith(user.id);
  });

  it('marks all current sessions revoked after a successful revoke-all operation', async () => {
    const secondSession = { ...activeSession, id: 'session_456' };
    getUserSessions.mockResolvedValue([activeSession, secondSession]);
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    act(() => { result.current.selectUser(user); });
    await act(async () => { await result.current.loadUserSessions(user.id); });

    let revokeResult: { success: boolean; message?: string } | undefined;
    await act(async () => { revokeResult = await result.current.handleRevokeAllSessions(user.id); });

    expect(revokeResult).toEqual({ success: true, message: '1 sessions revoked' });
    expect(revokeAllSessions).toHaveBeenCalledWith(user.id);
    expect(result.current.selectedUserSessions.every((session) => session.revoked)).toBe(true);
  });

  it('keeps customer session state intact and returns a safe failure on revocation errors', async () => {
    revokeSession.mockRejectedValue(new Error('Authorization expired'));
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    act(() => { result.current.selectUser(user); });
    await act(async () => { await result.current.loadUserSessions(user.id); });

    let revokeResult: { success: boolean; message?: string } | undefined;
    await act(async () => { revokeResult = await result.current.handleRevokeSession(user.id, activeSession.id); });

    expect(revokeResult).toEqual({ success: false, message: 'Authorization expired' });
    expect(result.current.selectedUserSessions[0]?.revoked).toBe(false);
    expect(result.current.actionLoading).toBeNull();
  });

  it('does not reload unselected users and returns safe fallbacks for non-Error revocation failures', async () => {
    revokeSession.mockRejectedValue('offline');
    revokeAllSessions.mockRejectedValue('offline');
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    getUserDetails.mockClear();

    let sessionResult: { success: boolean; message?: string } | undefined;
    let allResult: { success: boolean; message?: string } | undefined;
    await act(async () => { sessionResult = await result.current.handleRevokeSession('another-user', activeSession.id); });
    await act(async () => { allResult = await result.current.handleRevokeAllSessions('another-user'); });

    expect(sessionResult).toEqual({ success: false, message: 'Failed to revoke session' });
    expect(allResult).toEqual({ success: false, message: 'Failed to revoke sessions' });
    expect(getUserDetails).not.toHaveBeenCalled();
    expect(result.current.actionLoading).toBeNull();
  });

  it('clears a list error on request and keeps a selected customer when their details refresh', async () => {
    listUsers.mockRejectedValue(new Error('Temporary error'));
    const { result } = renderHook(() => useUserManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.error).toBe('Temporary error');

    act(() => { result.current.clearError(); });
    expect(result.current.error).toBeNull();
    await act(async () => { await result.current.loadUserDetails(user.id); });
    expect(result.current.selectedUser).toEqual(user);
  });
});
