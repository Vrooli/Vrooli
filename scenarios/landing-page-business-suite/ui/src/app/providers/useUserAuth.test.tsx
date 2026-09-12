import { Component, type ReactNode } from 'react';
import { render, renderHook, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { UserAuthContext, type UserAuthContextValue } from './UserAuthContext';
import { useUserAuth } from './useUserAuth';

// provider-free-exception: this contract intentionally mounts the hook without
// UserAuthProvider to verify the actionable missing-provider error.

const authContextValue: UserAuthContextValue = {
  isAuthenticated: true,
  isSessionLoading: false,
  user: {
    id: 'user-123',
    email: 'customer@example.com',
    email_verified: true,
  },
  logout: async () => {},
  refreshSession: async () => {},
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

function UserAuthConsumer() {
  useUserAuth();
  return null;
}

describe('useUserAuth', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('returns the user authentication context supplied by its provider', () => {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <UserAuthContext.Provider value={authContextValue}>{children}</UserAuthContext.Provider>
    );

    const { result } = renderHook(() => useUserAuth(), { wrapper });

    expect(result.current).toBe(authContextValue);
  });

  it('fails fast when used outside the user authentication provider', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(
      <HookErrorBoundary>
        <UserAuthConsumer />
      </HookErrorBoundary>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('useUserAuth must be used within UserAuthProvider');
  });
});
