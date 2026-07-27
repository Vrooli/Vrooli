import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { UserAccounts } from './UserAccounts';
import type { UseUserManagementReturn } from '../hooks/useUserManagement';

const loadUsers = vi.fn();
const setSearch = vi.fn();
const setPage = vi.fn();
const selectUser = vi.fn();
const loadUserSessions = vi.fn();
const handleRevokeSession = vi.fn();
const handleRevokeAllSessions = vi.fn();
const clearError = vi.fn();
const useUserManagementMock = vi.fn<[], UseUserManagementReturn>();

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

vi.mock('../components/PageHeader', () => ({
  PageHeader: ({ title, actions }: { title: string; actions: ReactNode }) => <><h1>{title}</h1>{actions}</>,
}));

vi.mock('../hooks/useUserManagement', () => ({
  useUserManagement: () => useUserManagementMock(),
}));

const user = {
  id: 'customer_123',
  email: 'customer@example.com',
  email_verified: true,
  created_at: '2026-01-01T00:00:00Z',
  last_login_at: '2026-01-02T00:00:00Z',
  session_count: 2,
  subscription: { plan_tier: 'pro', status: 'active' },
  credits: { balance: 1_000, bonus: 25 },
};

const activeSession = {
  id: 'session_active',
  created_at: '2026-01-01T00:00:00Z',
  last_used_at: '2026-01-02T00:00:00Z',
  expires_at: '2030-01-01T00:00:00Z',
  ip_address: '192.0.2.10',
  user_agent: 'Mozilla/5.0 Chrome Windows',
  revoked: false,
};

const baseState: UseUserManagementReturn = {
  users: [user], total: 25, page: 2, perPage: 20, totalPages: 2, search: '', setSearch,
  selectedUser: user, selectedUserSessions: [activeSession], loading: false, error: null,
  detailsLoading: false, sessionsLoading: false, actionLoading: null, loadUsers, setPage, selectUser,
  loadUserDetails: vi.fn(), loadUserSessions, handleRevokeSession, handleRevokeAllSessions, clearError,
};

describe('UserAccounts route', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useUserManagementMock.mockReturnValue(baseState);
    handleRevokeSession.mockResolvedValue({ success: true });
    handleRevokeAllSessions.mockResolvedValue({ success: true });
  });

  it('renders customer subscription, credit, and active-session information', () => {
    render(<UserAccounts />);

    expect(screen.getByRole('heading', { name: 'User Accounts' })).toBeInTheDocument();
    expect(screen.getAllByText('customer@example.com')).not.toHaveLength(0);
    expect(screen.getAllByText('Pro')).not.toHaveLength(0);
    expect(screen.getAllByText('1,025')).not.toHaveLength(0);
    expect(screen.getByText('Sessions (1 active)')).toBeInTheDocument();
    expect(screen.getByText('Chrome on Windows')).toBeInTheDocument();
  });

  it('delegates refresh, filtering, and selection to the management hook', async () => {
    const actor = userEvent.setup();
    render(<UserAccounts />);

    await actor.click(screen.getByRole('button', { name: /refresh/i }));
    await actor.type(screen.getByPlaceholderText('Search by email...'), 'billing@example.com');
    const customerRow = screen.getAllByText('customer@example.com').find((element) => element.closest('tr'))?.closest('tr');
    if (!customerRow) throw new Error('customer row was not rendered');
    await actor.click(customerRow);

    expect(loadUsers).toHaveBeenCalledOnce();
    expect(setSearch).toHaveBeenCalled();
    expect(selectUser).toHaveBeenCalledWith(user);
    expect(loadUserSessions).toHaveBeenCalledWith(user.id);
  });

  it('executes both scoped session-revocation controls', async () => {
    const actor = userEvent.setup();
    render(<UserAccounts />);

    await actor.click(screen.getByTitle('Revoke session'));
    await actor.click(screen.getByRole('button', { name: /revoke all/i }));

    await waitFor(() => {
      expect(handleRevokeSession).toHaveBeenCalledWith(user.id, activeSession.id);
      expect(handleRevokeAllSessions).toHaveBeenCalledWith(user.id);
    });
  });

  it('shows a useful empty/search state and allows clearing displayed errors', async () => {
    useUserManagementMock.mockReturnValue({
      ...baseState, users: [], total: 0, totalPages: 0, selectedUser: null,
      selectedUserSessions: [], search: 'missing@example.com', error: 'Customer service unavailable',
    });
    const actor = userEvent.setup();
    render(<UserAccounts />);

    expect(screen.getByText('No users found')).toBeInTheDocument();
    expect(screen.getByText('Try a different search term')).toBeInTheDocument();
    expect(screen.getByText('Select a user to view details')).toBeInTheDocument();
    const dismissButton = screen.getByText('Customer service unavailable').parentElement?.querySelector('button');
    if (!dismissButton) throw new Error('error dismissal button was not rendered');
    await actor.click(dismissButton);
    expect(clearError).toHaveBeenCalledOnce();
  });

  it('shows explicit loading affordances while customer and session data is loading', () => {
    useUserManagementMock.mockReturnValue({
      ...baseState, users: [], selectedUser: user, selectedUserSessions: [], loading: true,
      detailsLoading: true, sessionsLoading: true,
    });
    render(<UserAccounts />);

    expect(screen.getByText('Loading users...')).toBeInTheDocument();
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });
});
