import { useState, useEffect, useCallback, ReactNode } from 'react';
import { getUserMe, userLogout, refreshUserTokens, isApiError, type UserAuthUser } from '../../shared/api';
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
      if (response.user) {
        setIsAuthenticated(true);
        setUser(response.user);
      } else {
        setIsAuthenticated(false);
        setUser(null);
      }
    } catch (err) {
      // Not authenticated or error
      if (isApiError(err, 'unauthorized')) {
        // Try to refresh the session
        try {
          await refreshUserTokens(''); // Uses cookie
          const retryResponse = await getUserMe();
          if (retryResponse.user) {
            setIsAuthenticated(true);
            setUser(retryResponse.user);
            return;
          }
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
    // Only check session if we have cookies set (access_token cookie exists)
    // This avoids unnecessary 401 errors on initial page load
    const hasAuthCookie = document.cookie.includes('access_token=');
    if (hasAuthCookie) {
      checkSession();
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
