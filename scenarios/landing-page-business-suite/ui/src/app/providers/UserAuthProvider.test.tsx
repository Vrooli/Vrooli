import { cleanup, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { UserAuthProvider } from './UserAuthProvider';
import { useUserAuth } from './useUserAuth';

const { mockGetUserMe, mockIsApiError, mockRefreshUserTokens, mockUserLogout } = vi.hoisted(() => ({
  mockGetUserMe: vi.fn(),
  mockIsApiError: vi.fn(),
  mockRefreshUserTokens: vi.fn(),
  mockUserLogout: vi.fn(),
}));

vi.mock('../../shared/api', () => ({
  getUserMe: mockGetUserMe,
  isApiError: mockIsApiError,
  refreshUserTokens: mockRefreshUserTokens,
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
    vi.clearAllMocks();
    mockGetUserMe.mockResolvedValue({ user });
    mockIsApiError.mockReturnValue(false);
    mockRefreshUserTokens.mockResolvedValue(undefined);
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

  it('does not make an unauthenticated request until an access-token cookie exists', () => {
    renderProvider();

    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(screen.getByTestId('loading')).toHaveTextContent('false');
    expect(mockGetUserMe).not.toHaveBeenCalled();
  });

  it('loads an existing user session when an access-token cookie exists', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('access_token=session-token');

    renderProvider();

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent('true');
    });

    expect(screen.getByTestId('email')).toHaveTextContent(user.email);
    expect(mockGetUserMe).toHaveBeenCalledTimes(1);
  });

  it('refreshes an expired authorized session before marking the user unauthenticated', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('access_token=expired-token');
    mockGetUserMe.mockRejectedValueOnce(new Error('expired')).mockResolvedValueOnce({ user });
    mockIsApiError.mockReturnValue(true);

    renderProvider();

    await waitFor(() => {
      expect(screen.getByTestId('authenticated')).toHaveTextContent('true');
    });

    expect(mockRefreshUserTokens).toHaveBeenCalledWith('');
    expect(mockGetUserMe).toHaveBeenCalledTimes(2);
  });

  it('marks the session unauthenticated when the existing cookie is not authorized', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('access_token=invalid-token');
    mockGetUserMe.mockRejectedValueOnce(new Error('network unavailable'));

    renderProvider();

    await waitFor(() => {
      expect(mockGetUserMe).toHaveBeenCalledTimes(1);
    });
    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false');
    });

    expect(screen.getByTestId('authenticated')).toHaveTextContent('false');
    expect(mockRefreshUserTokens).not.toHaveBeenCalled();
  });

  it('marks the session unauthenticated when token refresh fails', async () => {
    vi.spyOn(document, 'cookie', 'get').mockReturnValue('access_token=expired-token');
    mockGetUserMe.mockRejectedValueOnce(new Error('expired'));
    mockIsApiError.mockReturnValue(true);
    mockRefreshUserTokens.mockRejectedValueOnce(new Error('refresh rejected'));

    renderProvider();

    await waitFor(() => {
      expect(mockRefreshUserTokens).toHaveBeenCalledWith('');
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
});
