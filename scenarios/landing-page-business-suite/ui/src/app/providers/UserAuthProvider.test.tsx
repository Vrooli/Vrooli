import { act, cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { MemoryRouter, Navigate, Route, Routes } from 'react-router-dom';
import { UserAuthProvider } from './UserAuthProvider';
import { useUserAuth } from './useUserAuth';

const { mockGetUserMe, mockIsApiError, mockRefreshSessionOnce, mockUserLogout } = vi.hoisted(() => ({
  mockGetUserMe: vi.fn(),
  mockIsApiError: vi.fn(),
  mockRefreshSessionOnce: vi.fn(),
  mockUserLogout: vi.fn(),
}));

vi.mock('../../shared/api', () => ({
  getUserMe: mockGetUserMe,
  isApiError: mockIsApiError,
  refreshSessionOnce: mockRefreshSessionOnce,
  userLogout: mockUserLogout,
}));

function UserAuthConsumer() {
  const { isAuthenticated, isSessionLoading, logout, refreshSession, user } = useUserAuth();

  return (
    <div>
      <p data-testid="authenticated">{String(isAuthenticated)}</p>
      <p data-testid="loading">{String(isSessionLoading)}</p>
      <p data-testid="email">{user?.email ?? 'none'}</p>
      <button type="button" onClick={() => void refreshSession()}>
        Refresh
      </button>
      <button type="button" onClick={() => void logout()}>
        Log out
      </button>
    </div>
  );
}

const user = {
  id: 'user-123',
  email: 'customer@example.com',
  email_verified: true,
};

describe('UserAuthProvider', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockGetUserMe.mockResolvedValue({ user });
    mockIsApiError.mockReturnValue(false);
    mockRefreshSessionOnce.mockResolvedValue(undefined);
    mockUserLogout.mockResolvedValue(undefined);
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('');
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  function renderProvider() {
    render(
      <UserAuthProvider>
        <UserAuthConsumer />
      </UserAuthProvider>,
    );
  }

  it('does not make an unauthenticated request until a session hint exists', () => {
    renderProvider();

    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(screen.getByTestId('loading')).toHaveTextContent('false');
    expect(mockGetUserMe).not.toHaveBeenCalled();
  });

  it('loads an existing user session when a session hint exists', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');

    renderProvider();

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent('true');
    });

    expect(screen.getByTestId('email')).toHaveTextContent(user.email);
    expect(mockGetUserMe).toHaveBeenCalledTimes(1);
  });

  it('holds a reopened protected route until its hinted session is checked', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('__Host-lpbs_session_hint=1');
    let resolveSession!: (value: { user: typeof user }) => void;
    mockGetUserMe.mockReturnValue(new Promise((resolve) => { resolveSession = resolve; }));
    function ProtectedPage() {
      const auth = useUserAuth();
      if (auth.isSessionLoading) return <p>Checking reopened session</p>;
      return auth.isAuthenticated ? <h1>Security for {auth.user?.email}</h1> : <Navigate to="/auth/login" replace />;
    }
    render(<MemoryRouter initialEntries={['/account/security']}><UserAuthProvider><Routes>
      <Route path="/account/security" element={<ProtectedPage />} />
      <Route path="/auth/login" element={<h1>Sign in again</h1>} />
    </Routes></UserAuthProvider></MemoryRouter>);
    expect(screen.getByText('Checking reopened session')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Sign in again' })).not.toBeInTheDocument();
    resolveSession({ user });
    expect(await screen.findByRole('heading', { name: `Security for ${user.email}` })).toBeInTheDocument();
    expect(mockGetUserMe).toHaveBeenCalledTimes(1);
  });

  it('refreshes an expired authorized session before marking the user unauthenticated', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');
    mockGetUserMe.mockRejectedValueOnce(new Error('expired')).mockResolvedValueOnce({ user });
    mockIsApiError.mockReturnValue(true);

    renderProvider();

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent('true');
    });

    expect(mockRefreshSessionOnce).toHaveBeenCalledTimes(1);
    expect(mockGetUserMe).toHaveBeenCalledTimes(2);
  });

  it('marks the session unauthenticated when the existing cookie is not authorized', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');
    mockGetUserMe.mockRejectedValueOnce(new Error('network unavailable'));

    renderProvider();

    await waitFor(() => {
      expect(mockGetUserMe).toHaveBeenCalledTimes(1);
    });
    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });

    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(mockRefreshSessionOnce).not.toHaveBeenCalled();
  });

  it('marks the session unauthenticated when token refresh fails', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');
    mockGetUserMe.mockRejectedValueOnce(new Error('expired'));
    mockIsApiError.mockReturnValue(true);
    mockRefreshSessionOnce.mockRejectedValueOnce(new Error('refresh rejected'));

    renderProvider();

    await waitFor(() => {
      expect(mockRefreshSessionOnce).toHaveBeenCalledTimes(1);
    });
    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });

    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(screen.getByTestId('email')).toHaveTextContent('none');
  });

  it('clears local authentication even when remote logout fails', async () => {
    const userEvents = userEvent.setup();
    mockUserLogout.mockRejectedValueOnce(new Error('network unavailable'));
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    renderProvider();
    await userEvents.click(screen.getByRole('button', { name: 'Log out' }));

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    });
    expect(screen.getByTestId('email')).toHaveTextContent('none');
    expect(mockUserLogout).toHaveBeenCalledTimes(1);
  });

  it('never restores a late session read after local logout', async () => {
    let resolveSession!: (value: { user: typeof user }) => void;
    mockGetUserMe.mockReturnValueOnce(new Promise((resolve) => { resolveSession = resolve; }));
    renderProvider();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    fireEvent.click(screen.getByRole('button', { name: 'Log out' }));
    await waitFor(() => { expect(mockUserLogout).toHaveBeenCalledTimes(1); });
    await act(async () => { resolveSession({ user }); await Promise.resolve(); });
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(screen.getByTestId('email')).toHaveTextContent('none');
    expect(screen.getByTestId('loading')).toHaveTextContent('false');
  });

  it('clears mounted private state on another tab sign-out without trusting the event as a session', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');
    renderProvider();
    await waitFor(() => { expect(screen.getByTestId('email')).toHaveTextContent(user.email); });
    await act(() => window.dispatchEvent(new StorageEvent('storage', {
      key: 'lpbs.auth.session-event', newValue: JSON.stringify({ kind: 'signed-out', nonce: 'synthetic-event' }),
    })));
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(screen.getByTestId('email')).toHaveTextContent('none');
    expect(mockGetUserMe).toHaveBeenCalledTimes(1);
  });

  it('checks session authority on focus and hides a denied expired session', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');
    renderProvider();
    await waitFor(() => { expect(screen.getByTestId('email')).toHaveTextContent(user.email); });
    mockGetUserMe.mockRejectedValueOnce(new Error('session expired'));
    mockIsApiError.mockReturnValue(true);
    mockRefreshSessionOnce.mockRejectedValueOnce(new Error('refresh session expired'));
    await act(() => window.dispatchEvent(new Event('focus')));
    await waitFor(() => { expect(screen.getByTestId('authenticated')).toHaveTextContent('false'); });
    expect(screen.getByTestId('email')).toHaveTextContent('none');
    expect(mockGetUserMe).toHaveBeenCalledTimes(2);
  });

  it('does not restore a locally signed-out account on passive focus after failed remote logout', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('lpbs_session_hint=1');
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    renderProvider();
    await waitFor(() => { expect(screen.getByTestId('email')).toHaveTextContent(user.email); });
    mockUserLogout.mockRejectedValueOnce(new Error('response unavailable'));
    fireEvent.click(screen.getByRole('button', { name: 'Log out' }));
    await waitFor(() => { expect(screen.getByTestId('email')).toHaveTextContent('none'); });
    await act(() => window.dispatchEvent(new Event('focus')));
    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(mockGetUserMe).toHaveBeenCalledTimes(1);
  });

  it('rejects a superseded required read instead of completing the older account handoff', async () => {
    let resolveOld!: (value: { user: typeof user }) => void;
    let resolveNew!: (value: { user: typeof user }) => void;
    const other = { ...user, id: 'new-user', email: 'new@example.com' };
    mockGetUserMe.mockReturnValueOnce(new Promise((resolve) => { resolveOld = resolve; }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveNew = resolve; }));
    const outcome = vi.fn();
    function RequiredReader() {
      const auth = useUserAuth();
      return <><output>{auth.user?.email ?? 'none'}</output>
        <button onClick={() => void auth.refreshSession({ requireAuthenticated: true, expectedEmail: user.email }).then(() => { outcome('old-success'); }, () => { outcome('old-rejected'); })}>Old read</button>
        <button onClick={() => void auth.refreshSession({ requireAuthenticated: true, expectedEmail: other.email }).then(() => { outcome('new-success'); }, () => { outcome('new-rejected'); })}>New read</button></>;
    }
    render(<UserAuthProvider><RequiredReader /></UserAuthProvider>);
    fireEvent.click(screen.getByText('Old read'));fireEvent.click(screen.getByText('New read'));
    await act(async () => { resolveNew({ user: other }); await Promise.resolve(); });
    await act(async () => { resolveOld({ user }); await Promise.resolve(); });
    expect(screen.getByRole('status')).toHaveTextContent(other.email);
    expect(outcome).toHaveBeenCalledWith('new-success');
    expect(outcome).toHaveBeenCalledWith('old-rejected');
    expect(outcome).not.toHaveBeenCalledWith('old-success');
  });
});
