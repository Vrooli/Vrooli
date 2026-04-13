import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { AdminAuthProvider, useAdminAuth } from './AdminAuthProvider';
import { adminLogin, adminLogout, checkAdminSession } from '../../shared/api';

const { mockAdminLogin, mockAdminLogout, mockCheckAdminSession } = vi.hoisted(() => ({
  mockAdminLogin: vi.fn(),
  mockAdminLogout: vi.fn(),
  mockCheckAdminSession: vi.fn(),
}));

vi.mock('../../shared/api', () => ({
  adminLogin: mockAdminLogin,
  adminLogout: mockAdminLogout,
  checkAdminSession: mockCheckAdminSession,
}));

// Test component that uses the auth context
function TestComponent() {
  const { isAuthenticated, user, login, logout, canResetDemoData } = useAdminAuth();

  return (
    <div>
      <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>
      <div data-testid="user-email">{user?.email || 'no-user'}</div>
      <div data-testid="reset-flag">{canResetDemoData ? 'reset-enabled' : 'reset-disabled'}</div>
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
  const originalLocation = window.location;

  beforeEach(() => {
    // Mock window.location
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin/home' };

    mockCheckAdminSession.mockResolvedValue({
      authenticated: false,
      email: undefined,
      reset_enabled: false,
    });
    mockAdminLogin.mockResolvedValue({ authenticated: true, email: 'test@example.com', reset_enabled: true });
    mockAdminLogout.mockResolvedValue({});
  });

  afterEach(() => {
    window.location = originalLocation;
    vi.clearAllMocks();
  });

  it('[REQ:ADMIN-AUTH] should provide authentication context to children', () => {
    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    expect(screen.getByTestId('user-email')).toHaveTextContent('no-user');
    expect(screen.getByTestId('reset-flag')).toHaveTextContent('reset-disabled');
  });

  it('[REQ:ADMIN-AUTH] should check session on mount for admin routes', async () => {
    const mockSessionData = { authenticated: true, email: 'admin@example.com', reset_enabled: true };

    mockCheckAdminSession.mockResolvedValue(mockSessionData);

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    await waitFor(() => {
      expect(mockCheckAdminSession).toHaveBeenCalled();
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('admin@example.com');
      expect(screen.getByTestId('reset-flag')).toHaveTextContent('reset-enabled');
    });
  });

  it('[REQ:ADMIN-AUTH] should NOT check session on non-admin routes', async () => {
    // Change to public route
    window.location.pathname = '/';

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    // Wait a bit to ensure the effect would have run if eligible
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(mockCheckAdminSession).not.toHaveBeenCalled();
    expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    expect(screen.getByTestId('reset-flag')).toHaveTextContent('reset-disabled');
  });

  it('[REQ:ADMIN-AUTH] should handle login successfully', async () => {
    const mockUserData = { email: 'test@example.com', reset_enabled: true };

    mockCheckAdminSession.mockResolvedValue({ authenticated: false, reset_enabled: false });
    mockAdminLogin.mockResolvedValue({ authenticated: true, ...mockUserData });

    render(
      <AdminAuthProvider>
        <TestComponent />
      </AdminAuthProvider>
    );

    const loginBtn = screen.getByTestId('login-btn');
    loginBtn.click();

    await waitFor(() => {
      expect(mockAdminLogin).toHaveBeenCalledWith('test@example.com', 'password');
    });

    await waitFor(() => {
      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('test@example.com');
      expect(screen.getByTestId('reset-flag')).toHaveTextContent('reset-enabled');
    });
  });

  it('[REQ:ADMIN-AUTH] should handle login failure', async () => {
    mockCheckAdminSession.mockResolvedValue({ authenticated: false, reset_enabled: false });
    mockAdminLogin.mockRejectedValue(new Error('login failed'));

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
      expect(mockAdminLogin).toHaveBeenCalledWith('test@example.com', 'password');
    });

    // Should remain unauthenticated after failed login
    expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
  });

  it('[REQ:ADMIN-AUTH] should handle logout successfully', async () => {
    const mockUserData = { email: 'test@example.com', reset_enabled: true };

    mockCheckAdminSession.mockResolvedValue({ authenticated: true, ...mockUserData });
    mockAdminLogout.mockResolvedValue({});

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
      expect(mockAdminLogout).toHaveBeenCalled();
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
