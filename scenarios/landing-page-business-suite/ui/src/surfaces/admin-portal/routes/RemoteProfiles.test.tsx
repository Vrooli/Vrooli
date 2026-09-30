import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { RemoteProfiles } from './RemoteProfiles';
import * as profilesHook from '../hooks/useRemoteProfilesForm';

vi.mock('../hooks/useRemoteProfilesForm');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));
const addToast = vi.fn();
vi.mock('../../../shared/ui/useToast', () => ({ useToast: () => ({ addToast }) }));

function formState(overrides: Record<string, unknown> = {}) {
  return { profiles: [], incomingSessions: [], sessionLinksByProfileId: {}, loading: false, error: null, actions: { refreshing: false, incomingRefreshing: false, incomingRevokeSessionId: null, loginId: null, logoutId: null, testingId: null, deletingId: null, updatingId: null, creating: false, loadingLinksId: null, remoteRevokeId: null }, refresh: vi.fn(), refreshIncomingSessions: vi.fn(), handleLoadSessionLinks: vi.fn().mockResolvedValue({ success: true, message: 'Remote state loaded' }), handleRevokeRemoteSessions: vi.fn().mockResolvedValue({ success: true, message: 'Sessions revoked' }), handleRevokeIncomingSession: vi.fn().mockResolvedValue({ success: true, message: 'Incoming revoked' }), handleCreate: vi.fn().mockResolvedValue({ success: true, message: 'Profile created' }), handleUpdate: vi.fn().mockResolvedValue({ success: true, message: 'Profile updated' }), handleDelete: vi.fn().mockResolvedValue({ success: true, message: 'Profile deleted' }), handleLogin: vi.fn().mockResolvedValue({ success: true, message: 'Authenticated' }), handleLogout: vi.fn().mockResolvedValue({ success: true, message: 'Logged out' }), handleTest: vi.fn().mockResolvedValue({ success: true, message: 'Connected' }), ...overrides } as unknown as ReturnType<typeof profilesHook.useRemoteProfilesForm>;
}

describe('RemoteProfiles', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('shows loading/empty state and creates a validated remote profile', async () => {
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(formState({ loading: true }));
    const { rerender } = render(<RemoteProfiles />);
    expect(screen.getByText('Loading remote profiles...')).toBeInTheDocument();
    const state = formState();
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    rerender(<RemoteProfiles />);
    fireEvent.click(screen.getByRole('button', { name: 'Add Your First Profile' }));
    fireEvent.change(screen.getByLabelText('Tag'), { target: { value: 'production' } });
    fireEvent.change(screen.getByLabelText('API Base'), { target: { value: 'https://lpbs.example/api/v1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Profile' }));
    await waitFor(() => { expect(state.handleCreate).toHaveBeenCalledWith(expect.objectContaining({ tag: 'production', apiBase: 'https://lpbs.example/api/v1' })); });
    expect(addToast).toHaveBeenCalledWith(expect.objectContaining({ type: 'success', message: 'Profile created' }));
  });

  it('operates a stored remote session: test, inspect, logout, revoke, edit, and delete', async () => {
    const profile = { id: 5, tag: 'production', label: 'Production', api_base: 'https://lpbs.example/api/v1', has_session: true, connector_id: 'connector-1', remote_session_id: 'session-remote', session_expires_at: '2026-02-01T00:00:00Z', last_login_at: null, last_used_at: null };
    const state = formState({ profiles: [profile], incomingSessions: [{ session_id: 'incoming-1', connector_id: 'connector-2', profile_tag: 'staging', admin_email: 'admin@example.com', origin: 'test', last_activity: '2026-01-01T00:00:00Z', expires_at: '2026-02-01T00:00:00Z' }] });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);
    fireEvent.click(screen.getByRole('button', { name: 'Test' }));
    fireEvent.click(screen.getByRole('button', { name: 'Inspect Remote' }));
    fireEvent.click(screen.getByRole('button', { name: 'Logout' }));
    fireEvent.click(screen.getByRole('button', { name: 'Revoke Remote' }));
    fireEvent.click(screen.getByRole('button', { name: 'Revoke' }));
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    expect(screen.getByText('Edit Remote Profile')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    await waitFor(() => { expect(state.handleTest).toHaveBeenCalledWith(5); });
    expect(state.handleLoadSessionLinks).toHaveBeenCalledWith(5);
    expect(state.handleLogout).toHaveBeenCalledWith(5);
    expect(state.handleRevokeRemoteSessions).toHaveBeenCalledWith(5);
    expect(state.handleRevokeIncomingSession).toHaveBeenCalledWith('incoming-1');
    expect(state.handleDelete).toHaveBeenCalledWith(5);
  });

  it('opens remote login and delegates valid credentials to the profile-specific action', async () => {
    const profile = { id: 5, tag: 'production', api_base: 'https://lpbs.example/api/v1', has_session: false };
    const state = formState({ profiles: [profile] });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);
    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    fireEvent.change(screen.getByLabelText('Admin Email'), { target: { value: 'admin@example.com' } });
    fireEvent.change(screen.getByLabelText('Admin Password'), { target: { value: 'safe-password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Authenticate' }));
    await waitFor(() => { expect(state.handleLogin).toHaveBeenCalledWith(5, { email: 'admin@example.com', password: 'safe-password' }); });
    expect(addToast).toHaveBeenCalledWith(expect.objectContaining({ type: 'success', message: 'Authenticated' }));
  });

  it('rejects empty remote-login credentials before a session request is made', async () => {
    const profile = { id: 5, tag: 'production', api_base: 'https://lpbs.example/api/v1', has_session: false };
    const state = formState({ profiles: [profile] });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);
    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    fireEvent.click(screen.getByRole('button', { name: 'Authenticate' }));
    expect(await screen.findByText('Email is required')).toBeInTheDocument();
    expect(state.handleLogin).not.toHaveBeenCalled();
  });

  it('does not perform destructive remote-session operations when confirmation is declined', () => {
    const profile = { id: 5, tag: 'production', api_base: 'https://lpbs.example/api/v1', has_session: true };
    const state = formState({
      profiles: [profile],
      incomingSessions: [{ session_id: 'incoming-1', connector_id: 'connector-2', admin_email: 'admin@example.com', origin: 'test', last_activity: null, expires_at: null }],
    });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    vi.stubGlobal('confirm', vi.fn(() => false));
    render(<RemoteProfiles />);

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Revoke Remote' }));
    fireEvent.click(screen.getByRole('button', { name: 'Revoke' }));

    expect(state.handleDelete).not.toHaveBeenCalled();
    expect(state.handleRevokeRemoteSessions).not.toHaveBeenCalled();
    expect(state.handleRevokeIncomingSession).not.toHaveBeenCalled();
  });

  it('keeps failed profile changes open and reports the action error', async () => {
    const state = formState({ handleCreate: vi.fn().mockResolvedValue({ success: false, message: 'Profile tag is already in use' }) });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);
    fireEvent.click(screen.getByRole('button', { name: 'Add Profile' }));
    fireEvent.change(screen.getByLabelText('Tag'), { target: { value: 'production' } });
    fireEvent.change(screen.getByLabelText('API Base'), { target: { value: 'https://lpbs.example/api/v1' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Profile' }));

    expect(await screen.findByText('Profile tag is already in use')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create Profile' })).toBeInTheDocument();
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Profile tag is already in use' });
  });

  it('reports failed remote operations without claiming a session state change', async () => {
    const profile = { id: 5, tag: 'production', api_base: 'https://lpbs.example/api/v1', has_session: true };
    const state = formState({
      profiles: [profile],
      incomingSessions: [{ session_id: 'incoming-1', connector_id: 'connector-2', admin_email: 'admin@example.com', origin: '', last_activity: null, expires_at: null }],
      handleTest: vi.fn().mockResolvedValue({ success: false }),
      handleLogout: vi.fn().mockResolvedValue({ success: false }),
      handleLoadSessionLinks: vi.fn().mockResolvedValue({ success: false }),
      handleRevokeRemoteSessions: vi.fn().mockResolvedValue({ success: false }),
      handleRevokeIncomingSession: vi.fn().mockResolvedValue({ success: false }),
      handleDelete: vi.fn().mockResolvedValue({ success: false }),
    });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);

    fireEvent.click(screen.getByRole('button', { name: 'Test' }));
    fireEvent.click(screen.getByRole('button', { name: 'Logout' }));
    fireEvent.click(screen.getByRole('button', { name: 'Inspect Remote' }));
    fireEvent.click(screen.getByRole('button', { name: 'Revoke Remote' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Revoke' }));

    await waitFor(() => {
      expect(state.handleTest).toHaveBeenCalledWith(5);
      expect(state.handleLogout).toHaveBeenCalledWith(5);
      expect(state.handleLoadSessionLinks).toHaveBeenCalledWith(5);
      expect(state.handleRevokeRemoteSessions).toHaveBeenCalledWith(5);
      expect(state.handleDelete).toHaveBeenCalledWith(5);
      expect(state.handleRevokeIncomingSession).toHaveBeenCalledWith('incoming-1');
    });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Remote test failed' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Remote logout failed' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Failed to inspect remote state' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Failed to revoke remote sessions' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Failed to delete profile' });
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Failed to revoke incoming remote session' });
  });

  it('edits a sessionless profile and makes unavailable session actions visibly inert', async () => {
    const profile = {
      id: 7,
      tag: 'staging',
      label: '  ',
      api_base: 'https://staging.example/api/v1',
      has_session: false,
      connector_id: '',
      remote_session_id: '',
      session_expires_at: null,
      last_login_at: null,
      last_used_at: null,
    };
    const state = formState({ profiles: [profile], error: 'Remote directory is temporarily unavailable' });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);

    expect(screen.getByText('Remote directory is temporarily unavailable')).toBeInTheDocument();
    expect(screen.getByText('No incoming remote-profile sessions detected.')).toBeInTheDocument();
    expect(screen.getByText('staging')).toBeInTheDocument();
    expect(screen.getByText('Not logged in')).toBeInTheDocument();
    expect(screen.getByText('Connector: pending')).toBeInTheDocument();
    expect(screen.getByText('Remote session ID: unknown')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Test' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Logout' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Inspect Remote' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Revoke Remote' })).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
    expect(screen.getByDisplayValue('staging')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Label'), { target: { value: 'Staging deployment' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));
    await waitFor(() => {
      expect(state.handleUpdate).toHaveBeenCalledWith(7, {
        tag: 'staging', label: 'Staging deployment', apiBase: 'https://staging.example/api/v1',
      });
    });
    expect(addToast).toHaveBeenCalledWith({ type: 'success', message: 'Profile updated' });
  });

  it('keeps login open and shows the server failure when remote authentication is rejected', async () => {
    const profile = { id: 5, tag: 'production', api_base: 'https://lpbs.example/api/v1', has_session: false };
    const state = formState({
      profiles: [profile],
      handleLogin: vi.fn().mockResolvedValue({ success: false, message: 'Remote password was rejected' }),
    });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);

    fireEvent.click(screen.getByRole('button', { name: 'Login' }));
    fireEvent.change(screen.getByLabelText('Admin Email'), { target: { value: 'admin@example.com' } });
    fireEvent.change(screen.getByLabelText('Admin Password'), { target: { value: 'wrong-password' } });
    fireEvent.click(screen.getByRole('button', { name: 'Authenticate' }));

    expect(await screen.findByText('Remote password was rejected')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Authenticate' })).toBeInTheDocument();
    expect(addToast).toHaveBeenCalledWith({ type: 'error', message: 'Remote password was rejected' });
  });

  it('renders remote-side session visibility and busy controls without exposing unavailable actions', () => {
    const profile = {
      id: 9,
      tag: 'production',
      label: undefined,
      api_base: 'https://lpbs.example/api/v1',
      has_session: true,
      connector_id: 'connector-9',
      remote_session_id: 'remote-9',
      session_expires_at: '2026-02-01T00:00:00Z',
      last_login_at: '2026-01-01T00:00:00Z',
      last_used_at: '2026-01-02T00:00:00Z',
    };
    const state = formState({
      profiles: [profile],
      actions: {
        refreshing: true, incomingRefreshing: true, incomingRevokeSessionId: null,
        loginId: 9, logoutId: 9, testingId: 9, deletingId: 9, updatingId: 9,
        creating: false, loadingLinksId: 9, remoteRevokeId: 9,
      },
      sessionLinksByProfileId: {
        9: {
          remote_sessions: [{ session_id: 'remote-session-1', origin: '', last_activity: '2026-01-03T00:00:00Z' }],
        },
      },
    });
    vi.mocked(profilesHook.useRemoteProfilesForm).mockReturnValue(state);
    render(<RemoteProfiles />);

    expect(screen.getByText('Linked sessions on remote instance:')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
    expect(screen.getByText(/remote-session-1 .* unknown/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Refresh Incoming' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Re-login' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Test' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Logout' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Inspect Remote' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Revoke Remote' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Edit' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Delete' })).toBeDisabled();
  });
});
