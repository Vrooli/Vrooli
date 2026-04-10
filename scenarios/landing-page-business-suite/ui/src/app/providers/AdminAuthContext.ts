import { createContext } from 'react';

export interface AdminAuthContextValue {
  isAuthenticated: boolean;
  isSessionLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  user: { email: string } | null;
  canResetDemoData: boolean;
}

export const AdminAuthContext = createContext<AdminAuthContextValue | null>(null);
