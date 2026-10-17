import { useState, useEffect, useCallback } from 'react';
import {
  type UserAccount,
  type UserSession,
  type UsersListResponse,
  listUsers,
  getUserDetails,
  getUserSessions,
  revokeSession,
  revokeAllSessions,
} from '../services/users.service';

export interface UseUserManagementReturn {
  // List state
  users: UserAccount[];
  total: number;
  page: number;
  perPage: number;
  totalPages: number;

  // Search state
  search: string;
  setSearch: (search: string) => void;

  // Selected user state
  selectedUser: UserAccount | null;
  selectedUserSessions: UserSession[];

  // UI state
  loading: boolean;
  error: string | null;
  detailsLoading: boolean;
  sessionsLoading: boolean;
  actionLoading: string | null;

  // Actions
  loadUsers: () => Promise<void>;
  setPage: (page: number) => void;
  selectUser: (user: UserAccount | null) => void;
  loadUserDetails: (id: string) => Promise<void>;
  loadUserSessions: (id: string) => Promise<void>;
  handleRevokeSession: (
    userId: string,
    sessionId: string
  ) => Promise<{ success: boolean; message?: string }>;
  handleRevokeAllSessions: (
    userId: string
  ) => Promise<{ success: boolean; message?: string }>;
  clearError: () => void;
}

/**
 * Hook for managing user accounts in the admin portal.
 *
 * Provides state and handlers for:
 * - Paginated user listing with search
 * - User details with subscription/credit info
 * - Session management (view, revoke)
 */
export function useUserManagement(): UseUserManagementReturn {
  // List state
  const [users, setUsers] = useState<UserAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [perPage] = useState(20);
  const [totalPages, setTotalPages] = useState(0);

  // Search state
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  // Selected user state
  const [selectedUser, setSelectedUser] = useState<UserAccount | null>(null);
  const [selectedUserSessions, setSelectedUserSessions] = useState<UserSession[]>([]);

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [detailsLoading, setDetailsLoading] = useState(false);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1); // Reset to first page on search change
    }, 300);
    return () => { clearTimeout(timer); };
  }, [search]);

  /**
   * Load users from API
   */
  const loadUsers = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response: UsersListResponse = await listUsers({
        search: debouncedSearch || undefined,
        page,
        per_page: perPage,
      });
      setUsers(response.users);
      setTotal(response.total);
      setTotalPages(response.total_pages);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, page, perPage]);

  // Load users on mount and when search/page changes
  useEffect(() => {
    void loadUsers();
  }, [loadUsers]);

  /**
   * Select a user and load their details
   */
  const selectUser = useCallback((user: UserAccount | null) => {
    setSelectedUser(user);
    setSelectedUserSessions([]);
  }, []);

  /**
   * Load full details for a user
   */
  const loadUserDetails = useCallback(async (id: string) => {
    setDetailsLoading(true);
    try {
      const user = await getUserDetails(id);
      setSelectedUser(user);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load user details');
    } finally {
      setDetailsLoading(false);
    }
  }, []);

  /**
   * Load sessions for a user
   */
  const loadUserSessions = useCallback(async (id: string) => {
    setSessionsLoading(true);
    try {
      const sessions = await getUserSessions(id);
      setSelectedUserSessions(sessions);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sessions');
    } finally {
      setSessionsLoading(false);
    }
  }, []);

  /**
   * Revoke a specific session
   */
  const handleRevokeSession = useCallback(
    async (
      userId: string,
      sessionId: string
    ): Promise<{ success: boolean; message?: string }> => {
      setActionLoading(sessionId);
      try {
        await revokeSession(userId, sessionId);
        // Update local state
        setSelectedUserSessions((prev) =>
          prev.map((s) => (s.id === sessionId ? { ...s, revoked: true } : s))
        );
        // Refresh user data to update session count
        if (selectedUser?.id === userId) {
          void loadUserDetails(userId);
        }
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to revoke session';
        return { success: false, message };
      } finally {
        setActionLoading(null);
      }
    },
    [selectedUser, loadUserDetails]
  );

  /**
   * Revoke all sessions for a user
   */
  const handleRevokeAllSessions = useCallback(
    async (userId: string): Promise<{ success: boolean; message?: string }> => {
      setActionLoading('all');
      try {
        const result = await revokeAllSessions(userId);
        // Update local state
        setSelectedUserSessions((prev) =>
          prev.map((s) => ({ ...s, revoked: true }))
        );
        // Refresh user data to update session count
        if (selectedUser?.id === userId) {
          void loadUserDetails(userId);
        }
        return { success: true, message: `${String(result.sessions_revoked)} sessions revoked` };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to revoke sessions';
        return { success: false, message };
      } finally {
        setActionLoading(null);
      }
    },
    [selectedUser, loadUserDetails]
  );

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    // List state
    users,
    total,
    page,
    perPage,
    totalPages,

    // Search state
    search,
    setSearch,

    // Selected user state
    selectedUser,
    selectedUserSessions,

    // UI state
    loading,
    error,
    detailsLoading,
    sessionsLoading,
    actionLoading,

    // Actions
    loadUsers,
    setPage,
    selectUser,
    loadUserDetails,
    loadUserSessions,
    handleRevokeSession,
    handleRevokeAllSessions,
    clearError,
  };
}
