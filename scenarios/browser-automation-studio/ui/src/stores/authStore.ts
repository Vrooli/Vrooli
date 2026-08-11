import { create } from 'zustand';

// Get landing page URL from environment or use default
const landingPageEnv = (import.meta.env as { VITE_LANDING_PAGE_URL?: unknown }).VITE_LANDING_PAGE_URL;
const LPBS_URL =
  typeof landingPageEnv === 'string' && landingPageEnv.length > 0
    ? landingPageEnv
    : 'https://vrooli.com';

const WEB_ACCESS_TOKEN_KEY = 'vrooli.web.access-token';
const AUTH_STATE_KEY = 'auth_state';

interface WebAccessToken {
  accessToken: string;
  expiresAt: string;
}

function readWebAccessToken(): WebAccessToken | null {
  try {
    const raw = sessionStorage.getItem(WEB_ACCESS_TOKEN_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<WebAccessToken>;
    if (typeof parsed.accessToken !== 'string' || typeof parsed.expiresAt !== 'string') return null;
    if (Date.now() >= new Date(parsed.expiresAt).getTime() - 30_000) {
      sessionStorage.removeItem(WEB_ACCESS_TOKEN_KEY);
      return null;
    }
    return { accessToken: parsed.accessToken, expiresAt: parsed.expiresAt };
  } catch {
    return null;
  }
}

/** Returns only the short-lived browser access token for same-origin API calls. */
export function getWebAccessToken(): string | null {
  return readWebAccessToken()?.accessToken ?? null;
}

function clearWebAccessToken(): void {
  try {
    sessionStorage.removeItem(WEB_ACCESS_TOKEN_KEY);
  } catch {
    // Storage can be disabled; the in-memory auth state is still cleared.
  }
}

function decodeDisplayClaims(accessToken: string): AuthUser | null {
  // This is display-only data. Every API authorization decision remains on
  // LPBS/BAS and uses the signed token; the browser never treats these claims
  // as proof of entitlement.
  try {
    const [, encodedPayload] = accessToken.split('.');
    if (!encodedPayload) return null;
    const normalized = encodedPayload.replace(/-/g, '+').replace(/_/g, '/');
    const payload = JSON.parse(atob(normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '='))) as {
      sub?: unknown;
      email?: unknown;
      email_verified?: unknown;
    };
    if (typeof payload.email !== 'string' || payload.email.trim() === '') return null;
    return {
      id: typeof payload.sub === 'string' && payload.sub ? payload.sub : payload.email,
      email: payload.email.toLowerCase(),
      emailVerified: payload.email_verified === true,
    };
  } catch {
    return null;
  }
}

async function completeWebCallback(): Promise<AuthUser | null> {
  const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : '';
  const fragment = new URLSearchParams(hash);
  const accessToken = fragment.get('access_token');
  const refreshToken = fragment.get('refresh_token');
  const expiresAt = fragment.get('expires_at');
  if (!accessToken && !refreshToken && !expiresAt) return null;

  const expectedState = sessionStorage.getItem(AUTH_STATE_KEY);
  const receivedState = fragment.get('state');
  sessionStorage.removeItem(AUTH_STATE_KEY);
  // Remove secrets from browser history before doing any network work.
  window.history.replaceState({}, document.title, `${window.location.pathname}${window.location.search}`);

  if (!expectedState || !receivedState || receivedState !== expectedState) {
    throw new Error('Authentication callback state validation failed');
  }
  if (!accessToken || !refreshToken || !expiresAt || Number.isNaN(new Date(expiresAt).getTime()) || Date.now() >= new Date(expiresAt).getTime()) {
    throw new Error('Authentication callback was incomplete or expired');
  }

  // The rotating refresh token crosses only the same-origin server boundary;
  // it is never written to browser storage and never returned by BAS.
  const response = await fetch('/api/v1/auth/subscription/session', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!response.ok) throw new Error('Subscription session could not be stored');

  sessionStorage.setItem(WEB_ACCESS_TOKEN_KEY, JSON.stringify({ accessToken, expiresAt } satisfies WebAccessToken));
  return decodeDisplayClaims(accessToken);
}

async function checkWebSession(): Promise<AuthUser | null> {
  const access = readWebAccessToken();
  if (access) return decodeDisplayClaims(access.accessToken);

  // The durable refresh token is owned by the local credential authority.
  // This status endpoint intentionally reveals configuration only; BAS will
  // perform a real signed refresh before using the credential.
  try {
    const response = await fetch('/api/v1/auth/subscription/session');
    if (response.ok) {
      const body = await response.json() as { configured?: boolean };
      if (body.configured === true) {
        return { id: 'vrooli-subscription', email: '', emailVerified: true };
      }
    }
  } catch {
    // Offline startup remains unauthenticated until a local token is present.
  }
  return null;
}

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

/** Desktop auth API interface */
interface DesktopAuthApi {
  signIn: () => Promise<{ state: string }>;
  signOut: () => Promise<void>;
  isAuthenticated: () => Promise<boolean>;
  getUser: () => Promise<unknown>;
  getAccessToken: () => Promise<string | null>;
  onAuthChanged: (callback: (event: AuthChangeEvent) => void) => void;
}

/** Get the desktop auth API if available. */
function getDesktopAuth(): DesktopAuthApi | undefined {
  return window.desktop?.auth as DesktopAuthApi | undefined;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  isLoading: true,
  user: null,
  error: null,

  signIn: async () => {
    set({ isLoading: true, error: null });

    try {
      const desktopAuth = getDesktopAuth();
      if (desktopAuth) {
        // Desktop: Use deep link auth flow
        const result = await desktopAuth.signIn();
        // Store state for CSRF validation (optional, main process handles it)
        sessionStorage.setItem(AUTH_STATE_KEY, result.state);

        // Note: The actual authentication completion happens via protocol handler
        // The auth:changed event will be received when tokens are received
        // For now, we just wait - the UI should show a "waiting for browser" state
        set({ isLoading: false });
      } else {
        // Web: Redirect to LPBS login page
        const state = crypto.randomUUID();
        sessionStorage.setItem(AUTH_STATE_KEY, state);

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
      const desktopAuth = getDesktopAuth();
      if (desktopAuth) {
        await desktopAuth.signOut();
      } else {
        // The server owns the durable refresh token for web/local-server
        // deployments. Best-effort deletion is safe because local state is
        // cleared regardless of network availability.
        try {
          await fetch('/api/v1/auth/subscription/session', { method: 'DELETE' });
        } catch {
          // Continue clearing browser state.
        }
        clearWebAccessToken();
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
      const desktopAuth = getDesktopAuth();
      if (desktopAuth) {
        const isAuthenticated = await desktopAuth.isAuthenticated();

        if (isAuthenticated) {
          const user = await desktopAuth.getUser();
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
        const user = await checkWebSession();
        set({ isAuthenticated: user !== null, user, isLoading: false });
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
    const desktopAuth = getDesktopAuth();
    if (desktopAuth) {
      return desktopAuth.getAccessToken();
    }
    return getWebAccessToken();
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

export const setupAuthListener = (): void => {
  const desktopAuth = getDesktopAuth();
  if (desktopAuth) {
    desktopAuth.onAuthChanged((event: AuthChangeEvent) => {
      const store = useAuthStore.getState();

      switch (event) {
        case 'tokens-received':
        case 'tokens-refreshed':
          desktopAuth.getUser().then((user) => {
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

    useAuthStore.getState().checkAuth();
  } else {
    void (async () => {
      try {
        const user = await completeWebCallback();
        if (user) {
          useAuthStore.setState({ isAuthenticated: true, user, isLoading: false, error: null });
          return;
        }
        await useAuthStore.getState().checkAuth();
      } catch (error) {
        clearWebAccessToken();
        useAuthStore.setState({
          isAuthenticated: false,
          user: null,
          isLoading: false,
          error: error instanceof Error ? error.message : 'Authentication callback failed',
        });
      }
    })();
  }
};

if (typeof window !== 'undefined') {
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
