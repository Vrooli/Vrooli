import { createContext } from 'react';
import type { UserAuthUser } from '../../shared/api';

export interface UserAuthContextValue {
  isAuthenticated: boolean;
  isSessionLoading: boolean;
  user: UserAuthUser | null;
  logout: () => Promise<void>;
  refreshSession: (options?: { requireAuthenticated?: boolean; expectedEmail?: string }) => Promise<void>;
}

export const UserAuthContext = createContext<UserAuthContextValue | null>(null);
