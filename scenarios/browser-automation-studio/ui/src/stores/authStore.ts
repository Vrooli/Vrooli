import { create } from 'zustand';

// Get landing page URL from environment or use default
const LPBS_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://vrooli.com';

// User information from authentication
export interface AuthUser {
  id: string;
  email: string;
  emailVerified: boolean;
}

// Auth change event types (from desktop API)
export type AuthChangeEvent = 'tokens-received' | 'tokens-refreshed' | 'session-expired' | 'signed-out';

// Auth state
interface AuthState {
  // State
  isAuthenticated: boolean;
  isLoading: boolean;
  user: AuthUser | null;
  error: string | null;

  // Actions
  signIn: () => Promise<void>;
  signOut: () => Promise<void>;
  checkAuth: () => Promise<void>;
  getAccessToken: () => Promise<string | null>;

  // Internal
  _setAuthenticated: (user: AuthUser | null) => void;
  _setLoading: (loading: boolean) => void;
  _setError: (error: string | null) => void;
}

// Check if running in desktop environment with auth support
function isDesktopWithAuth(): boolean {
  return typeof window !== 'undefined' &&
    typeof window.desktop?.auth !== 'undefined';
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isLoading: true,
  user: null,
  error: null,

  signIn: async () => {
    set({ isLoading: true, error: null });

    try {
      if (isDesktopWithAuth()) {
        // Desktop: Use deep link auth flow
        const result = await window.desktop!.auth.signIn();
        // Store state for CSRF validation (optional, main process handles it)
        sessionStorage.setItem('auth_state', result.state);

        // Note: The actual authentication completion happens via protocol handler
        // The auth:changed event will be received when tokens are received
        // For now, we just wait - the UI should show a "waiting for browser" state
        set({ isLoading: false });
      } else {
        // Web: Redirect to LPBS login page
        const state = crypto.randomUUID();
        sessionStorage.setItem('auth_state', state);

        const currentUrl = window.location.href;
        const authUrl = new URL(`${LPBS_URL}/auth/login`);
        authUrl.searchParams.set('redirect_uri', currentUrl);
        authUrl.searchParams.set('app', 'Browser Automation Studio');
        authUrl.searchParams.set('state', state);

        window.location.href = authUrl.toString();
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to initiate sign in';
      set({ error: errorMessage, isLoading: false });
    }
  },

  signOut: async () => {
    set({ isLoading: true, error: null });

    try {
      if (isDesktopWithAuth()) {
        await window.desktop!.auth.signOut();
      }
      // Clear state
      set({
        isAuthenticated: false,
        user: null,
        isLoading: false,
      });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to sign out';
      set({ error: errorMessage, isLoading: false });
    }
  },

  checkAuth: async () => {
    set({ isLoading: true, error: null });

    try {
      if (isDesktopWithAuth()) {
        const isAuthenticated = await window.desktop!.auth.isAuthenticated();

        if (isAuthenticated) {
          const user = await window.desktop!.auth.getUser();
          set({
            isAuthenticated: true,
            user: user as AuthUser | null,
            isLoading: false,
          });
        } else {
          set({
            isAuthenticated: false,
            user: null,
            isLoading: false,
          });
        }
      } else {
        // Web environment - no persistent auth without cookies
        set({
          isAuthenticated: false,
          user: null,
          isLoading: false,
        });
      }
    } catch (err) {
      console.error('[Auth] Failed to check auth status:', err);
      set({
        isAuthenticated: false,
        user: null,
        isLoading: false,
      });
    }
  },

  getAccessToken: async (): Promise<string | null> => {
    if (isDesktopWithAuth()) {
      return window.desktop!.auth.getAccessToken();
    }
    return null;
  },

  _setAuthenticated: (user) => {
    set({
      isAuthenticated: user !== null,
      user,
    });
  },

  _setLoading: (loading) => {
    set({ isLoading: loading });
  },

  _setError: (error) => {
    set({ error });
  },
}));

// Set up desktop auth change listener
if (typeof window !== 'undefined') {
  // Wait for window.desktop to be available
  const setupAuthListener = () => {
    if (isDesktopWithAuth()) {
      window.desktop!.auth.onAuthChanged((event: AuthChangeEvent) => {
        const store = useAuthStore.getState();

        switch (event) {
          case 'tokens-received':
          case 'tokens-refreshed':
            // Fetch user info and update state
            window.desktop!.auth.getUser().then((user) => {
              store._setAuthenticated(user as AuthUser | null);
              store._setLoading(false);
            });
            break;

          case 'session-expired':
          case 'signed-out':
            store._setAuthenticated(null);
            store._setLoading(false);
            break;
        }
      });

      // Check initial auth state
      useAuthStore.getState().checkAuth();
    } else {
      // Not in desktop environment, mark as not loading
      useAuthStore.getState()._setLoading(false);
    }
  };

  // Try to set up listener immediately, or wait for DOMContentLoaded
  if (document.readyState === 'complete') {
    setupAuthListener();
  } else {
    window.addEventListener('DOMContentLoaded', setupAuthListener);
  }
}

// Convenience hooks
export const useIsAuthenticated = (): boolean => {
  return useAuthStore((state) => state.isAuthenticated);
};

export const useAuthUser = (): AuthUser | null => {
  return useAuthStore((state) => state.user);
};

export const useAuthLoading = (): boolean => {
  return useAuthStore((state) => state.isLoading);
};

export default useAuthStore;
