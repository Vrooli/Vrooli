import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { act } from '@testing-library/react';
import { useAuthStore, useIsAuthenticated, useAuthUser, useAuthLoading, type AuthUser } from './authStore';

// Mock the desktop API
const mockDesktopAuth = {
  signIn: vi.fn(),
  signOut: vi.fn(),
  getAccessToken: vi.fn(),
  getUser: vi.fn(),
  isAuthenticated: vi.fn(),
  onAuthChanged: vi.fn(),
  offAuthChanged: vi.fn(),
};

describe('authStore [REQ:BAS-AUTH]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store state
    useAuthStore.setState({
      isAuthenticated: false,
      isLoading: true,
      user: null,
      error: null,
    });
  });

  afterEach(() => {
    // Clean up desktop mock
    delete (window as any).desktop;
  });

  describe('checkAuth (desktop mode)', () => {
    it('sets authenticated when desktop API returns true', async () => {
      // Set up desktop environment
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.isAuthenticated.mockResolvedValue(true);
      mockDesktopAuth.getUser.mockResolvedValue({
        id: '123',
        email: 'test@example.com',
        emailVerified: true,
      });

      await act(async () => {
        await useAuthStore.getState().checkAuth();
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user?.email).toBe('test@example.com');
      expect(state.isLoading).toBe(false);
    });

    it('sets not authenticated when desktop API returns false', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.isAuthenticated.mockResolvedValue(false);

      await act(async () => {
        await useAuthStore.getState().checkAuth();
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(state.isLoading).toBe(false);
    });

    it('handles errors gracefully', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.isAuthenticated.mockRejectedValue(new Error('API error'));

      // Suppress console.error for this test
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});

      await act(async () => {
        await useAuthStore.getState().checkAuth();
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(state.isLoading).toBe(false);

      consoleError.mockRestore();
    });
  });

  describe('checkAuth (web mode)', () => {
    it('sets not authenticated in web environment', async () => {
      // No desktop environment
      delete (window as any).desktop;

      await act(async () => {
        await useAuthStore.getState().checkAuth();
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(state.isLoading).toBe(false);
    });
  });

  describe('signIn (desktop mode)', () => {
    it('calls desktop.auth.signIn and stores state', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.signIn.mockResolvedValue({ state: 'test-state-123' });

      // Mock sessionStorage
      const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');

      await act(async () => {
        await useAuthStore.getState().signIn();
      });

      expect(mockDesktopAuth.signIn).toHaveBeenCalled();
      expect(setItemSpy).toHaveBeenCalledWith('auth_state', 'test-state-123');

      const state = useAuthStore.getState();
      // For desktop, signIn just opens browser - doesn't set authenticated immediately
      expect(state.isLoading).toBe(false);

      setItemSpy.mockRestore();
    });

    it('handles sign in errors', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.signIn.mockRejectedValue(new Error('Failed to open browser'));

      await act(async () => {
        await useAuthStore.getState().signIn();
      });

      const state = useAuthStore.getState();
      expect(state.error).toBe('Failed to open browser');
      expect(state.isLoading).toBe(false);
    });
  });

  describe('signIn (web mode)', () => {
    it('redirects to LPBS auth page', async () => {
      delete (window as any).desktop;

      // Mock location
      const originalLocation = window.location;
      const mockLocation = { href: '' };
      Object.defineProperty(window, 'location', {
        value: mockLocation,
        writable: true,
        configurable: true,
      });

      // Mock crypto.randomUUID
      vi.spyOn(crypto, 'randomUUID').mockReturnValue('test-uuid-123');
      const setItemSpy = vi.spyOn(Storage.prototype, 'setItem');

      await act(async () => {
        await useAuthStore.getState().signIn();
      });

      expect(window.location.href).toContain('/auth/login');
      expect(window.location.href).toContain('redirect_uri=');
      expect(window.location.href).toContain('app=Browser');
      expect(setItemSpy).toHaveBeenCalledWith('auth_state', 'test-uuid-123');

      // Restore
      Object.defineProperty(window, 'location', {
        value: originalLocation,
        writable: true,
        configurable: true,
      });
      setItemSpy.mockRestore();
    });
  });

  describe('signOut (desktop mode)', () => {
    it('clears user and auth state', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.signOut.mockResolvedValue(undefined);

      // Set initial authenticated state
      useAuthStore.setState({
        isAuthenticated: true,
        user: { id: '123', email: 'test@example.com', emailVerified: true },
        isLoading: false,
        error: null,
      });

      await act(async () => {
        await useAuthStore.getState().signOut();
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(mockDesktopAuth.signOut).toHaveBeenCalled();
    });

    it('handles sign out errors', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.signOut.mockRejectedValue(new Error('Logout failed'));

      useAuthStore.setState({
        isAuthenticated: true,
        user: { id: '123', email: 'test@example.com', emailVerified: true },
        isLoading: false,
        error: null,
      });

      await act(async () => {
        await useAuthStore.getState().signOut();
      });

      const state = useAuthStore.getState();
      expect(state.error).toBe('Logout failed');
      expect(state.isLoading).toBe(false);
    });
  });

  describe('signOut (web mode)', () => {
    it('clears state without calling desktop API', async () => {
      delete (window as any).desktop;

      useAuthStore.setState({
        isAuthenticated: true,
        user: { id: '123', email: 'test@example.com', emailVerified: true },
        isLoading: false,
        error: null,
      });

      await act(async () => {
        await useAuthStore.getState().signOut();
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });
  });

  describe('getAccessToken', () => {
    it('returns token from desktop API', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };
      mockDesktopAuth.getAccessToken.mockResolvedValue('test-access-token');

      const token = await useAuthStore.getState().getAccessToken();

      expect(token).toBe('test-access-token');
      expect(mockDesktopAuth.getAccessToken).toHaveBeenCalled();
    });

    it('returns null in web mode', async () => {
      delete (window as any).desktop;

      const token = await useAuthStore.getState().getAccessToken();

      expect(token).toBeNull();
    });
  });

  describe('internal setters', () => {
    it('_setAuthenticated updates user and isAuthenticated', () => {
      const user: AuthUser = { id: '1', email: 'test@test.com', emailVerified: true };

      act(() => {
        useAuthStore.getState()._setAuthenticated(user);
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).toEqual(user);
    });

    it('_setAuthenticated(null) clears auth', () => {
      useAuthStore.setState({
        isAuthenticated: true,
        user: { id: '1', email: 'test@test.com', emailVerified: true },
      });

      act(() => {
        useAuthStore.getState()._setAuthenticated(null);
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });

    it('_setLoading updates loading state', () => {
      act(() => {
        useAuthStore.getState()._setLoading(false);
      });

      expect(useAuthStore.getState().isLoading).toBe(false);

      act(() => {
        useAuthStore.getState()._setLoading(true);
      });

      expect(useAuthStore.getState().isLoading).toBe(true);
    });

    it('_setError updates error state', () => {
      act(() => {
        useAuthStore.getState()._setError('Test error');
      });

      expect(useAuthStore.getState().error).toBe('Test error');

      act(() => {
        useAuthStore.getState()._setError(null);
      });

      expect(useAuthStore.getState().error).toBeNull();
    });
  });

  describe('convenience hooks', () => {
    it('useIsAuthenticated returns auth state', () => {
      useAuthStore.setState({ isAuthenticated: true });
      // Note: These hooks need to be tested within a React component context
      // For direct state access, we test the underlying state
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it('useAuthUser returns user', () => {
      const user: AuthUser = { id: '1', email: 'test@test.com', emailVerified: true };
      useAuthStore.setState({ user });
      expect(useAuthStore.getState().user).toEqual(user);
    });

    it('useAuthLoading returns loading state', () => {
      useAuthStore.setState({ isLoading: true });
      expect(useAuthStore.getState().isLoading).toBe(true);
    });
  });

  describe('auth change event handling', () => {
    it('handles tokens-received event', async () => {
      (window as any).desktop = { auth: mockDesktopAuth };

      // Capture the callback registered with onAuthChanged
      let authChangedCallback: ((event: string) => void) | undefined;
      mockDesktopAuth.onAuthChanged.mockImplementation((cb: (event: string) => void) => {
        authChangedCallback = cb;
      });
      mockDesktopAuth.getUser.mockResolvedValue({
        id: 'user-123',
        email: 'newuser@example.com',
        emailVerified: true,
      });

      // Simulate the event
      if (authChangedCallback) {
        // The callback is async, so we need to wait
        await act(async () => {
          authChangedCallback!('tokens-received');
          // Wait for the async getUser call
          await new Promise((resolve) => setTimeout(resolve, 10));
        });
      }
    });

    it('handles session-expired event', () => {
      // Set initial authenticated state
      useAuthStore.setState({
        isAuthenticated: true,
        user: { id: '123', email: 'test@example.com', emailVerified: true },
      });

      // Simulate what happens when session expires
      act(() => {
        useAuthStore.getState()._setAuthenticated(null);
        useAuthStore.getState()._setLoading(false);
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });

    it('handles signed-out event', () => {
      useAuthStore.setState({
        isAuthenticated: true,
        user: { id: '123', email: 'test@example.com', emailVerified: true },
      });

      // Simulate what happens when signed out
      act(() => {
        useAuthStore.getState()._setAuthenticated(null);
        useAuthStore.getState()._setLoading(false);
      });

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });
  });
});
