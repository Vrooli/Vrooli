import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { AdminAuthProvider, useAdminAuth } from './AdminAuthProvider';

// Test component that uses the auth context
function TestComponent() {
  const { isAuthenticated, user, login, logout } = useAdminAuth();

  return (
    <div>
      <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>
      <div data-testid="user-email">{user?.email || 'no-user'}</div>
      <button onClick={() => login('test@example.com', 'password')} data-testid="login-btn">
        Login
      </button>
      <button onClick={logout} data-testid="logout-btn">
        Logout
      </button>
    </div>
  );
}

describe('AdminAuthProvider [REQ:ADMIN-AUTH]', () => {
  const originalFetch = global.fetch;
  const originalLocation = window.location;

  beforeEach(() => {
    // Mock fetch
    global.fetch = vi.fn();

    // Mock window.location
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin/home' };
  });

  afterEach(() => {
    global.fetch = originalFetch;
    window.location = originalLocation;
  });

  it('[REQ:ADMIN-AUTH] should provide authentication context to children', () => {
    // Mock successful session check
    vi.mocked(global.fetch).mockResolvedValue({
      ok: false,
    } as Response);

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    expect(screen.getByTestId('user-email')).toHaveTextContent('no-user');
  });

  it('[REQ:ADMIN-AUTH] should check session on mount for admin routes', async () => {
    const mockSessionData = { authenticated: true, email: 'admin@example.com' };

    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => mockSessionData,
    } as Response);

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/admin/session'),
        expect.objectContaining({
          credentials: 'include',
          headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
        })
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('admin@example.com');
    });
  });

  it('[REQ:ADMIN-AUTH] should NOT check session on non-admin routes', async () => {
    // Change to public route
    window.location.pathname = '/';

    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      json: async () => ({ email: 'admin@example.com' }),
    } as Response);

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    // Wait a bit to ensure fetch is not called
    await new Promise((resolve) => setTimeout(resolve, 100));

    expect(global.fetch).not.toHaveBeenCalled();
    expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
  });

  it('[REQ:ADMIN-AUTH] should handle login successfully', async () => {
    const mockUserData = { email: 'test@example.com' };

    // Mock initial session check (fail)
    vi.mocked(global.fetch)
      .mockResolvedValueOnce({
        ok: false,
      } as Response)
      // Mock login success
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockUserData,
      } as Response);

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    const loginBtn = screen.getByTestId('login-btn');
    loginBtn.click();

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/admin/login'),
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: 'test@example.com', password: 'password' }),
          credentials: 'include',
        })
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('test@example.com');
    });
  });

  it('[REQ:ADMIN-AUTH] should handle login failure', async () => {
    // Mock initial session check (fail)
    vi.mocked(global.fetch)
      .mockResolvedValueOnce({
        ok: false,
      } as Response)
      // Mock login failure
      .mockResolvedValueOnce({
        ok: false,
      } as Response);

    // Create a component that catches the login error
    function TestComponentWithErrorHandling() {
      const { isAuthenticated, user, login, logout } = useAdminAuth();

      const handleLogin = async () => {
        try {
          await login('test@example.com', 'password');
        } catch (err) {
          // Catch and suppress the expected error
        }
      };

      return (
        <div>
          <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>
          <div data-testid="user-email">{user?.email || 'no-user'}</div>
          <button onClick={handleLogin} data-testid="login-btn">
            Login
          </button>
          <button onClick={logout} data-testid="logout-btn">
            Logout
          </button>
        </div>
      );
    }

    render(
      <AdminAuthProvider>
        <TestComponentWithErrorHandling />
      </AdminAuthProvider>
    );

    const loginBtn = screen.getByTestId('login-btn');

    // Click should trigger login attempt which will fail
    loginBtn.click();

    // Wait for fetch to be called with login endpoint
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/admin/login'),
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: 'test@example.com', password: 'password' }),
          credentials: 'include',
        })
      );
    });

    // Should remain unauthenticated after failed login
    expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
  });

  it('[REQ:ADMIN-AUTH] should handle logout successfully', async () => {
    const mockUserData = { email: 'test@example.com' };

    // Mock successful session check
    vi.mocked(global.fetch)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => mockUserData,
      } as Response)
      // Mock logout success
      .mockResolvedValueOnce({
        ok: true,
      } as Response);

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    // Wait for initial session check
    await waitFor(() => {
      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
    });

    const logoutBtn = screen.getByTestId('logout-btn');
    logoutBtn.click();

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/admin/logout'),
        expect.objectContaining({
          method: 'POST',
          credentials: 'include',
          headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
        })
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user');
    });
  });

  it('should throw error when useAdminAuth is used outside AdminAuthProvider', () => {
    // Suppress console.error for this test
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

    expect(() => {
      render(<TestComponent />);
    }).toThrow('useAdminAuth must be used within AdminAuthProvider');

    consoleError.mockRestore();
  });
});
