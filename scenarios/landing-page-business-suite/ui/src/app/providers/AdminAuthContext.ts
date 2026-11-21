import { createContext } from 'react';

export interface AdminAuthContextValue {
  isAuthenticated: boolean;
  isSessionLoading: boolean;
  login: (email: string, password: string, totpCode?: string, passkeyAssertion?: Uint8Array, passkeyCeremonyId?: string) => Promise<void>;
  logout: () => void;
  user: { email: string } | null;
  canResetDemoData: boolean;
  assurance?: 'full' | 'enrollment_only' | null;
}

export const AdminAuthContext = createContext<AdminAuthContextValue | null>(null);
