import { useEffect, useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { sessionApi, type BrowserSession } from "../api/session";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/** The browser retains only display state. Canonical HttpOnly cookies carry credentials. */
export function AccountSession() {
  const { t } = useTranslation();
  const copy = strings.lightDesk.account;
  const queryClient = useQueryClient();
  const [session, setSession] = useState<BrowserSession | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [factor, setFactor] = useState("");
  const [recovery, setRecovery] = useState(false);
  const [challenge, setChallenge] = useState("");
  const [mfaRequired, setMFARequired] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    let live = true;
    void sessionApi.status().then((value) => { if (live) setSession(value); }).catch(() => { if (live) setError(t(copy.statusFailed)); });
    return () => { live = false; };
  }, [t, copy.statusFailed]);
  useEffect(() => {
    if (!session?.authenticated || !session.expiresAt) return;
    const delay = new Date(session.expiresAt).getTime() - Date.now();
    const timer = setTimeout(() => setSession({ authenticated: false }), Math.max(0, delay));
    return () => clearTimeout(timer);
  }, [session]);
  const perform = async (operation: () => Promise<BrowserSession>, expectSession = true) => {
    setBusy(true); setError("");
    try {
      const result = await operation();
      if (result.mfaRequired) { setChallenge(result.mfaChallenge || ""); setMFARequired(true); return; }
      setPassword(""); setFactor(""); setChallenge(""); setMFARequired(false);
      const verified = await sessionApi.status();
      setSession(verified);
      if (expectSession && !verified.authenticated) setError(t(copy.verificationFailed));
      await queryClient.invalidateQueries({ queryKey: ["lighting-control"] });
    } catch (failure) {
      if (!expectSession) setSession(await sessionApi.status().catch(() => ({ authenticated: false })));
      setError(failure instanceof Error ? failure.message : t(copy.failed));
    }
    finally { setBusy(false); }
  };
  const login = (event: FormEvent) => {
    event.preventDefault();
    void perform(() => sessionApi.login({ email, password, mfaChallenge: challenge, ...(recovery ? { recoveryCode: factor } : { totpCode: factor }) }));
  };
  return <section className="lightdesk-panel" aria-labelledby="account-heading" aria-busy={busy}>
    <h2 id="account-heading">{t(copy.title)}</h2>
    <p className="lightdesk-muted">{t(copy.intro)}</p>
    {session?.authenticated ? <>
      <p role="status">{t(copy.signedIn, { account: session.email || session.subject })}</p>
      {session.expiresAt && <p>{t(copy.expires, { time: new Date(session.expiresAt).toLocaleString() })}</p>}
      <button className="lightdesk-button" disabled={busy} onClick={() => void perform(sessionApi.refresh)}>{t(copy.renew)}</button>
      <button className="lightdesk-button" disabled={busy} onClick={() => void perform(sessionApi.logout, false)}>{t(copy.signOut)}</button>
    </> : <form className="lightdesk-settings-form" onSubmit={login}>
      <p role="status">{t(session ? copy.signedOut : copy.checking)}</p>
      <label className="lightdesk-field">{t(copy.email)}<input type="email" autoComplete="username" required value={email} onChange={(event) => setEmail(event.target.value)} disabled={busy}/></label>
      <label className="lightdesk-field">{t(copy.password)}<input type="password" autoComplete="current-password" required value={password} onChange={(event) => setPassword(event.target.value)} disabled={busy}/></label>
      {mfaRequired && <>
        <p role="status">{t(copy.mfaRequired)}</p>
        <label className="lightdesk-check"><input type="checkbox" checked={recovery} onChange={(event) => { setRecovery(event.target.checked); setFactor(""); }}/>{t(copy.useRecovery)}</label>
        <label className="lightdesk-field">{t(recovery ? copy.recoveryCode : copy.totpCode)}<input autoComplete="one-time-code" required value={factor} onChange={(event) => setFactor(event.target.value)} disabled={busy}/></label>
      </>}
      <button type="submit" className="lightdesk-button lightdesk-button--primary" disabled={busy}>{t(copy.signIn)}</button>
      <button type="button" className="lightdesk-button" disabled={busy} onClick={() => void perform(sessionApi.refresh)}>{t(copy.renew)}</button>
    </form>}
    {error && <p role="alert" className="lightdesk-alert">{error}</p>}
  </section>;
}
