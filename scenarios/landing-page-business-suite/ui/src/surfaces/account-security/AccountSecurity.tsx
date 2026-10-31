import { useEffect, useState } from 'react';
import { Link, Navigate } from 'react-router-dom';
import { beginPasskeyRegistration, completeAccountReauthentication, finishPasskeyRegistration, listAccountSessions, listPasskeys, renamePasskey, revokeAccountSession, revokeOtherAccountSessions, startAccountReauthentication, type AccountSession } from '../../shared/api';
import { credentialToJSON, isPasskeySupported, toCreationOptions, toRequestOptions } from '../user-auth/lib/webauthn';
import { useUserAuth } from '../../app/providers/useUserAuth';

export function AccountSecurity() {
  const { isAuthenticated, isSessionLoading } = useUserAuth();
  const [sessions, setSessions] = useState<AccountSession[]>([]);
  const [message, setMessage] = useState('');
  const [busy, setBusy] = useState(false);
  const [reauthOpen, setReauthOpen] = useState(false);
  const [code, setCode] = useState('');
  const [passkeys, setPasskeys] = useState<Array<{ id: string; nickname: string }>>([]);
  const [reauthPasskey, setReauthPasskey] = useState<{ optionsJson: string; ceremonyId: string } | null>(null);

  useEffect(() => {
    if (!isAuthenticated) return;
    void listAccountSessions().then((response) => setSessions(response.sessions)).catch(() => setMessage('We could not load your active sessions.'));
    void listPasskeys().then((response) => setPasskeys(response.passkeys.map((p) => ({ id: p.id, nickname: p.nickname })))).catch(() => undefined);
  }, [isAuthenticated]);

  if (isSessionLoading) return <main className="auth-shell"><p>Loading your account…</p></main>;
  if (!isAuthenticated) return <Navigate to="/auth/login?next=%2Faccount%2Fsecurity" replace />;

  const revoke = async (session: AccountSession) => {
    setBusy(true);
    setMessage('');
    try {
      await revokeAccountSession(session.id);
      setSessions((current) => current.filter((item) => item.id !== session.id));
    } catch {
      setMessage('That session could not be revoked.');
    } finally {
      setBusy(false);
    }
  };

  const addPasskey = async () => {
    if (!isPasskeySupported()) { setMessage('This browser does not support passkeys.'); return; }
    setBusy(true); setMessage('');
    try {
      const begin = await beginPasskeyRegistration();
      const credential = await navigator.credentials.create({ publicKey: toCreationOptions(JSON.parse(begin.optionsJson)) as PublicKeyCredentialCreationOptions });
      if (!(credential instanceof PublicKeyCredential)) throw new Error('No passkey was created');
      const saved = await finishPasskeyRegistration(JSON.stringify(credentialToJSON(credential)), begin.ceremonyId);
      setPasskeys((current) => [...current, { id: saved.id, nickname: saved.nickname || 'Passkey' }]); setMessage('Passkey added.');
    } catch { setMessage('Passkey registration was cancelled or rejected.'); } finally { setBusy(false); }
  };

  const revokeOthers = async () => {
    setBusy(true);
    setMessage('');
    try {
      await revokeOtherAccountSessions();
      setSessions((current) => current.filter((session) => session.current));
      setMessage('Other sessions have been signed out.');
    } catch {
      setMessage('Other sessions could not be signed out.');
    } finally {
      setBusy(false);
    }
  };

  const reauthenticate = async () => {
    setBusy(true);
    setMessage('');
    try {
      await completeAccountReauthentication(code);
      setReauthOpen(false);
      setCode('');
      setMessage('Identity verified for the next few minutes.');
    } catch {
      setMessage('That verification code was not accepted.');
    } finally {
      setBusy(false);
    }
  };

  const reauthenticateWithPasskey = async () => {
    if (!reauthPasskey || !isPasskeySupported()) return;
    setBusy(true); setMessage('');
    try {
      const credential = await navigator.credentials.get({ publicKey: toRequestOptions(JSON.parse(reauthPasskey.optionsJson)) });
      if (!(credential instanceof PublicKeyCredential)) throw new Error('No passkey assertion');
      await completeAccountReauthentication('', '', new Uint8Array(new TextEncoder().encode(JSON.stringify(credentialToJSON(credential)))), reauthPasskey.ceremonyId);
      setReauthOpen(false); setReauthPasskey(null); setMessage('Identity verified for the next few minutes.');
    } catch { setMessage('That passkey verification was not accepted.'); } finally { setBusy(false); }
  };

  const openReauthentication = async () => {
    setReauthOpen(true);
    try {
      const started = await startAccountReauthentication();
      if (started.passkeyOptionsJson && started.passkeyCeremonyId) setReauthPasskey({ optionsJson: started.passkeyOptionsJson, ceremonyId: started.passkeyCeremonyId });
    } catch { setMessage('We could not start identity verification.'); }
  };

  return (
    <main className="auth-shell" aria-labelledby="account-security-title">
      <section className="auth-card account-security-card">
        <Link to="/" className="auth-back-link">Back to home</Link>
        <p className="auth-eyebrow">Account</p>
        <h1 id="account-security-title">Security</h1>
        <p className="auth-subtitle">Review where your account is signed in and sign out devices you no longer use.</p>
        {message && <p role="status" className="auth-status">{message}</p>}
        <div className="account-security-actions">
          <button type="button" className="button button-secondary" onClick={() => void revokeOthers()} disabled={busy}>Sign out elsewhere</button>
          <button type="button" className="button button-secondary" onClick={() => { void openReauthentication(); }} disabled={busy}>Verify identity</button>
        </div>
        <section aria-labelledby="passkeys-title" className="mt-8">
          <h2 id="passkeys-title">Passkeys</h2>
          <p className="auth-subtitle">Use a device passkey for faster sign-in.</p>
          {isPasskeySupported() && <button type="button" className="button button-secondary" onClick={() => void addPasskey()} disabled={busy}>Add a passkey</button>}
          <ul aria-label="Saved passkeys" className="account-session-list">{passkeys.map((passkey) => <li key={passkey.id} className="account-session-row"><strong>{passkey.nickname}</strong><button type="button" className="button button-quiet" disabled={busy} onClick={() => { const next = window.prompt('Passkey name', passkey.nickname); if (next) void renamePasskey(passkey.id, next).then(() => setPasskeys((current) => current.map((item) => item.id === passkey.id ? { ...item, nickname: next } : item))); }}>Rename</button><button type="button" className="button button-quiet" disabled={busy} onClick={() => void import('../../shared/api').then(({ revokePasskey }) => revokePasskey(passkey.id)).then(() => setPasskeys((current) => current.filter((item) => item.id !== passkey.id)))}>Remove</button></li>)}</ul>
        </section>
        <ul className="account-session-list" aria-label="Active sessions">
          {sessions.map((session) => (
            <li key={session.id} className="account-session-row">
              <div>
                <strong>{session.deviceLabel || 'Unknown device'}{session.current ? ' · This device' : ''}</strong>
                <span>{session.ipHint || 'Unknown location'} · {session.authMethod || 'Email sign-in'}</span>
              </div>
              {!session.current && <button type="button" className="button button-quiet" onClick={() => void revoke(session)} disabled={busy}>Revoke</button>}
            </li>
          ))}
        </ul>
        {sessions.length === 0 && <p>No active sessions were found.</p>}
        {reauthOpen && <div role="dialog" aria-modal="true" aria-labelledby="reauth-title" className="auth-dialog">
          <h2 id="reauth-title">Verify your identity</h2>
          <p>We sent a one-time code to your sign-in email.</p>
          <label htmlFor="reauth-code">Verification code</label>
          <input id="reauth-code" value={code} onChange={(event) => setCode(event.target.value)} inputMode="numeric" autoComplete="one-time-code" />
          <div className="account-security-actions">
            <button type="button" className="button button-primary" onClick={() => void reauthenticate()} disabled={busy || code.trim() === ''}>Continue</button>
            {reauthPasskey && isPasskeySupported() && <button type="button" className="button button-secondary" onClick={() => void reauthenticateWithPasskey()} disabled={busy}>Use a passkey</button>}
            <button type="button" className="button button-quiet" onClick={() => setReauthOpen(false)} disabled={busy}>Cancel</button>
          </div>
        </div>}
      </section>
    </main>
  );
}

export default AccountSecurity;
