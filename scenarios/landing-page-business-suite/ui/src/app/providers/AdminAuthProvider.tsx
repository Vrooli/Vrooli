import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { adminLogin, adminLogout, checkAdminSession } from '../../shared/api';

interface AdminAuthContextValue {
  isAuthenticated: boolean;
  isSessionLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  user: { email: string } | null;
  canResetDemoData: boolean;
}

const AdminAuthContext = createContext<AdminAuthContextValue | null>(null);

export function AdminAuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<{ email: string } | null>(null);
  const [isSessionLoading, setIsSessionLoading] = useState(() => (typeof window !== 'undefined' ? window.location.pathname.startsWith('/admin') : false));
  const [canResetDemoData, setCanResetDemoData] = useState(false);

  // Check for existing session only on admin routes (lazy check)
  // This avoids triggering 401 errors during UI smoke tests on public pages
  useEffect(() => {
    // Only check session if we're on an admin route
    if (typeof window === 'undefined') {
      return;
    }

    if (!window.location.pathname.startsWith('/admin')) {
      setIsSessionLoading(false);
      return;
    }

    let isMounted = true;
    setIsSessionLoading(true);

    const checkSession = async () => {
      try {
        const session = await checkAdminSession();
        if (!isMounted) {
          return;
        }
        setIsAuthenticated(session.authenticated);
        setUser(session.authenticated && session.email ? { email: session.email } : null);
        setCanResetDemoData(Boolean(session.reset_enabled));
      } catch (e) {
        console.error('Session check failed:', e);
        setIsAuthenticated(false);
        setUser(null);
        setCanResetDemoData(false);
      } finally {
        if (isMounted) {
          setIsSessionLoading(false);
        }
      }
    };
    checkSession();

    return () => {
      isMounted = false;
    };
  }, []);

  const login = async (email: string, password: string) => {
    const response = await adminLogin(email, password);
    setIsAuthenticated(Boolean(response?.authenticated));
    setUser(response?.authenticated && response.email ? { email: response.email } : { email });
    setIsSessionLoading(false);
    setCanResetDemoData(Boolean(response?.reset_enabled));
  };

  const logout = async () => {
    try {
      await adminLogout();
    } catch (e) {
      console.error('Logout failed:', e);
    }
    setIsAuthenticated(false);
    setUser(null);
    setIsSessionLoading(false);
    setCanResetDemoData(false);
  };

  return (
    <AdminAuthContext.Provider value={{ isAuthenticated, isSessionLoading, login, logout, user, canResetDemoData }}>
      {children}
    </AdminAuthContext.Provider>
  );
}

export const useAdminAuth = () => {
  const context = useContext(AdminAuthContext);
  if (!context) {
    throw new Error('useAdminAuth must be used within AdminAuthProvider');
  }
  return context;
};
