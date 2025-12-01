import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { adminLogin, adminLogout, checkAdminSession } from '../services/api';

interface AuthContextValue {
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  user: { email: string } | null;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<{ email: string } | null>(null);

  // Check for existing session only on admin routes (lazy check)
  // This avoids triggering 401 errors during UI smoke tests on public pages
  useEffect(() => {
    // Only check session if we're on an admin route
    if (!window.location.pathname.startsWith('/admin')) {
      return;
    }

    const checkSession = async () => {
      try {
        const session = await checkAdminSession();
        setIsAuthenticated(session.authenticated);
        setUser(session.authenticated && session.email ? { email: session.email } : null);
      } catch (e) {
        console.error('Session check failed:', e);
        setIsAuthenticated(false);
        setUser(null);
      }
    };
    checkSession();
  }, []);

  const login = async (email: string, password: string) => {
    await adminLogin(email, password);
    setIsAuthenticated(true);
    setUser({ email });
  };

  const logout = async () => {
    try {
      await adminLogout();
    } catch (e) {
      console.error('Logout failed:', e);
    }
    setIsAuthenticated(false);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, login, logout, user }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};
