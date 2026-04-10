import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { RemoteProfiles } from './RemoteProfiles';
import type { UseRemoteProfilesFormReturn } from '../hooks/useRemoteProfilesForm';

const addToastMock = vi.fn();
const handleLoadSessionLinksMock = vi.fn();
const handleRevokeRemoteSessionsMock = vi.fn();
const handleRevokeIncomingSessionMock = vi.fn();
const refreshIncomingSessionsMock = vi.fn();
const refreshMock = vi.fn();

const baseHookState: UseRemoteProfilesFormReturn = {
  profiles: [
    {
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
    },
  ],
  incomingSessions: [
    {
      session_id: 'remote-session-1',
      admin_email: 'admin@example.com',
      connector_id: 'connector-1',
      profile_tag: 'prod',
      origin: 'local',
      created_at: '2025-01-01T00:00:00Z',
      last_activity: '2025-01-01T01:00:00Z',
      expires_at: '2025-01-01T02:00:00Z',
    },
  ],
  sessionLinksByProfileId: {
    1: {
      profile_id: 1,
      profile_tag: 'prod',
      connector_id: 'connector-1',
      local_has_session: true,
      local_status: 'active',
      remote_sessions: [
        {
          session_id: 'remote-session-1',
          admin_email: 'admin@example.com',
          connector_id: 'connector-1',
          origin: 'local',
          created_at: '2025-01-01T00:00:00Z',
          last_activity: '2025-01-01T01:00:00Z',
          expires_at: '2025-01-01T02:00:00Z',
        },
      ],
    },
  },
  loading: false,
  error: null,
  actions: {
    creating: false,
    updatingId: null,
    deletingId: null,
    loginId: null,
    logoutId: null,
    testingId: null,
    refreshing: false,
    loadingLinksId: null,
    remoteRevokeId: null,
    incomingRefreshing: false,
    incomingRevokeSessionId: null,
  },
  refresh: refreshMock,
  refreshIncomingSessions: refreshIncomingSessionsMock,
  handleLoadSessionLinks: handleLoadSessionLinksMock,
  handleRevokeRemoteSessions: handleRevokeRemoteSessionsMock,
  handleRevokeIncomingSession: handleRevokeIncomingSessionMock,
  handleCreate: vi.fn(),
  handleUpdate: vi.fn(),
  handleDelete: vi.fn(),
  handleLogin: vi.fn(),
  handleLogout: vi.fn(),
  handleTest: vi.fn(),
};

const useRemoteProfilesFormMock = vi.fn<[], UseRemoteProfilesFormReturn>();

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));

vi.mock('../components/PageHeader', () => ({
  PageHeader: ({ title, actions }: { title: string; actions: ReactNode }) => (
    <div>
      <h1>{title}</h1>
      <div>{actions}</div>
    </div>
  ),
}));

vi.mock('../../../shared/ui/useToast', () => ({
  useToast: () => ({ addToast: addToastMock }),
}));

vi.mock('../hooks/useRemoteProfilesForm', () => ({
  useRemoteProfilesForm: () => useRemoteProfilesFormMock(),
}));

describe('RemoteProfiles route', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useRemoteProfilesFormMock.mockReturnValue(baseHookState);
    handleLoadSessionLinksMock.mockResolvedValue({ success: true, message: 'Remote state loaded' });
    handleRevokeRemoteSessionsMock.mockResolvedValue({ success: true, message: 'Remote sessions revoked' });
    handleRevokeIncomingSessionMock.mockResolvedValue({ success: true, message: 'Incoming remote session revoked' });
    vi.stubGlobal('confirm', vi.fn(() => true));
  });

  it('renders incoming and remote side visibility sections', async () => {
    render(<RemoteProfiles />);

    expect(screen.getByText('Remote Profiles')).toBeInTheDocument();
    expect(screen.getByText('Incoming Remote Sessions')).toBeInTheDocument();
    expect(screen.getByText('Remote Side Visibility')).toBeInTheDocument();
    expect(screen.getByText(/Linked sessions on remote instance/)).toBeInTheDocument();
  });

  it('calls inspect remote action and emits success toast', async () => {
    const user = userEvent.setup();
    render(<RemoteProfiles />);

    await user.click(screen.getByRole('button', { name: /Inspect Remote/i }));

    await waitFor(() => {
      expect(handleLoadSessionLinksMock).toHaveBeenCalledWith(1);
      expect(addToastMock).toHaveBeenCalledWith({
        type: 'success',
        message: 'Remote state loaded',
      });
    });
  });

  it('confirms and revokes remote sessions', async () => {
    const user = userEvent.setup();
    render(<RemoteProfiles />);

    await user.click(screen.getByRole('button', { name: /Revoke Remote/i }));

    await waitFor(() => {
      expect(handleRevokeRemoteSessionsMock).toHaveBeenCalledWith(1);
    });
  });

  it('does not revoke incoming session when confirm returns false', async () => {
    vi.stubGlobal('confirm', vi.fn(() => false));
    const user = userEvent.setup();
    render(<RemoteProfiles />);

    await user.click(screen.getByRole('button', { name: /^Revoke$/i }));

    expect(handleRevokeIncomingSessionMock).not.toHaveBeenCalled();
  });

  it('shows incoming sessions even when no remote profiles exist', async () => {
    useRemoteProfilesFormMock.mockReturnValue({
      ...baseHookState,
      profiles: [],
      sessionLinksByProfileId: {},
      incomingSessions: [
        {
          session_id: 'remote-session-only',
          admin_email: 'admin@example.com',
          connector_id: 'connector-only',
          profile_tag: 'prod',
          origin: 'local',
          created_at: '2025-01-01T00:00:00Z',
          last_activity: '2025-01-01T01:00:00Z',
          expires_at: '2025-01-01T02:00:00Z',
        },
      ],
    });

    render(<RemoteProfiles />);

    expect(screen.getByText('Incoming Remote Sessions')).toBeInTheDocument();
    expect(screen.getByText(/admin@example\.com/i)).toBeInTheDocument();
    expect(screen.getByText(/No remote profiles configured yet/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Revoke$/i })).toBeInTheDocument();
  });
});
