import { useState, useEffect, ReactNode } from 'react';
import { adminLogin, adminLogout, beginAdminSecondFactor, checkAdminSession, reauthenticateAdmin } from '../../shared/api';
import { registerAdminReauthRequester } from '../../shared/api/adminReauthentication';
import { AdminReauthDialog } from '../../surfaces/admin-portal/components/AdminReauthDialog';
import { AdminAuthContext } from './AdminAuthContext';
import { credentialToJSON, isPasskeySupported, toRequestOptions } from '../../surfaces/user-auth/lib/webauthn';

export function AdminAuthProvider({ children }: { children: ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<{ email: string } | null>(null);
  const [isSessionLoading, setIsSessionLoading] = useState(() => (typeof window !== 'undefined' ? window.location.pathname.startsWith('/admin') : false));
  const [canResetDemoData, setCanResetDemoData] = useState(false);
  const [assurance, setAssurance] = useState<'full' | 'enrollment_only' | null>(null);
  const [reauthPending, setReauthPending] = useState<((ok: boolean) => void) | null>(null);

  useEffect(() => registerAdminReauthRequester(() => new Promise<void>((resolve, reject) => {
    setReauthPending(() => (ok: boolean) => { setReauthPending(null); ok ? resolve() : reject(new Error('Administrator reauthentication cancelled')); });
  })), []);

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
        setAssurance(session.assurance ?? (session.authenticated ? 'full' : null));
      } catch (e) {
        console.error('Session check failed:', e);
        setIsAuthenticated(false);
        setUser(null);
        setCanResetDemoData(false);
        setAssurance(null);
      } finally {
        if (isMounted) {
          setIsSessionLoading(false);
        }
      }
    };
    void checkSession();

    return () => {
      isMounted = false;
    };
  }, []);

  const login = async (email: string, password: string, totpCode?: string, passkeyAssertion?: Uint8Array, passkeyCeremonyId?: string) => {
    const response = passkeyAssertion && passkeyCeremonyId
      ? await adminLogin(email, password, totpCode, passkeyAssertion, passkeyCeremonyId)
      : await adminLogin(email, password, totpCode);
    setIsAuthenticated(response.authenticated);
    setUser(response.authenticated && response.email ? { email: response.email } : { email });
    setIsSessionLoading(false);
    setCanResetDemoData(Boolean(response.reset_enabled));
    setAssurance(response.assurance ?? 'full');
  };

  const logout = (): void => {
    void adminLogout().catch((error: unknown) => {
      console.error('Logout failed:', error);
    });
    setIsAuthenticated(false);
    setUser(null);
    setIsSessionLoading(false);
    setCanResetDemoData(false);
    setAssurance(null);
  };

  return (
    <AdminAuthContext.Provider value={{ isAuthenticated, isSessionLoading, login, logout, user, canResetDemoData, assurance }}>
      {children}
      {reauthPending && <AdminReauthDialog onCancel={() => reauthPending(false)} onSubmit={async (password, totp, recovery) => { await reauthenticateAdmin(password, totp, recovery); reauthPending(true); }} onPasskey={user && isPasskeySupported() ? async (password) => { const started = await beginAdminSecondFactor(user.email); const credential = await navigator.credentials.get({ publicKey: toRequestOptions(JSON.parse(started.optionsJson)) as PublicKeyCredentialRequestOptions }); if (!(credential instanceof PublicKeyCredential)) throw new Error('No passkey assertion was returned.'); await reauthenticateAdmin(password, '', '', new TextEncoder().encode(JSON.stringify(credentialToJSON(credential))), started.ceremonyId); reauthPending(true); } : undefined} />}
    </AdminAuthContext.Provider>
  );
}
