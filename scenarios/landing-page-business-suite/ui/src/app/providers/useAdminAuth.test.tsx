import { Component, type ReactNode } from 'react';
import { render, renderHook, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AdminAuthContext, type AdminAuthContextValue } from './AdminAuthContext';
import { useAdminAuth } from './useAdminAuth';

// provider-free-exception: this contract intentionally mounts the hook without
// AdminAuthProvider to verify the actionable missing-provider error.

const authContextValue: AdminAuthContextValue = {
  isAuthenticated: true,
  isSessionLoading: false,
  login: async () => {},
  logout: () => {},
  user: { email: 'admin@example.com' },
  canResetDemoData: true,
};

class HookErrorBoundary extends Component<{ children: ReactNode }, { message: string }> {
  state = { message: '' };

  static getDerivedStateFromError(error: Error) {
    return { message: error.message };
  }

  render() {
    if (this.state.message) {
      return <p role="alert">{this.state.message}</p>;
    }
    return this.props.children;
  }
}

function AdminAuthConsumer() {
  useAdminAuth();
  return null;
}

describe('useAdminAuth', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns the administrator authentication context supplied by its provider', () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <AdminAuthContext.Provider value={authContextValue}>{children}</AdminAuthContext.Provider>
    );

    const { result } = renderHook(() => useAdminAuth(), { wrapper });

    expect(result.current).toBe(authContextValue);
  });

  it('fails fast when used outside the administrator authentication provider', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(
      <HookErrorBoundary>
        <AdminAuthConsumer />
      </HookErrorBoundary>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('useAdminAuth must be used within AdminAuthProvider');
  });
});
