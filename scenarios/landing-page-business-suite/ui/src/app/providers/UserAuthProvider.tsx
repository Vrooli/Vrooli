import { useState, useEffect, useCallback, ReactNode } from 'react';
import { getUserMe, userLogout, refreshSessionOnce, isApiError, type UserAuthUser } from '../../shared/api';
import { UserAuthContext } from './UserAuthContext';

export function UserAuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<UserAuthUser | null>(null);
  const [isSessionLoading, setIsSessionLoading] = useState(false);

  // Check for existing session
  const checkSession = useCallback(async () => {
    setIsSessionLoading(true);
    try {
      const response = await getUserMe();
      setIsAuthenticated(true);
      setUser(response.user);
    } catch (err) {
      // Not authenticated or error
      if (isApiError(err, 'unauthorized')) {
        // Try to refresh the session
        try {
          await refreshSessionOnce();
          const retryResponse = await getUserMe();
          setIsAuthenticated(true);
          setUser(retryResponse.user);
          return;
        } catch {
          // Refresh failed
        }
      }
      setIsAuthenticated(false);
      setUser(null);
    } finally {
      setIsSessionLoading(false);
    }
  }, []);

  // Lazy session check - only runs when explicitly called
  // or when needed by components
  useEffect(() => {
    // Only check session if the non-authoritative presence hint exists.
    // This avoids unnecessary 401 errors on initial page load
	const hasSessionHint = document.cookie.split(';').some((cookie) => {
		const name = cookie.trim().split('=', 1)[0];
		return name === 'lpbs_session_hint' || name === '__Host-lpbs_session_hint';
	});
	if (hasSessionHint) {
      void checkSession();
    }
  }, [checkSession]);

  const logout = useCallback(async () => {
    try {
      await userLogout();
    } catch (err) {
      console.error('Logout failed:', err);
    }
    setIsAuthenticated(false);
    setUser(null);
  }, []);

  const refreshSession = useCallback(async () => {
    await checkSession();
  }, [checkSession]);

  return (
    <UserAuthContext.Provider
      value={{
        isAuthenticated,
        isSessionLoading,
        user,
        logout,
        refreshSession,
      }}
    >
      {children}
    </UserAuthContext.Provider>
  );
}
