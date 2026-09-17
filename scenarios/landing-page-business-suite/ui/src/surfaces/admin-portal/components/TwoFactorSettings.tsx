import { useEffect, useState, type FormEvent } from 'react';
import { CheckCircle2, Copy, Download, KeyRound, ShieldCheck, ShieldOff, Smartphone } from 'lucide-react';
import { FormSection } from './FormSection';
import { FormField } from './FormField';
import { inputClassName } from './formFieldClasses';
import { Button } from '../../../shared/ui/button';
import { QrCode } from '../../../shared/ui/QrCode';
import {
  beginAdminMFAEnrollment,
  confirmAdminMFAEnrollment,
  disableAdminMFA,
  getAdminMFAStatus,
  getApiErrorMessage,
  regenerateAdminRecoveryCodes,
  beginAdminPasskeyRegistration,
  finishAdminPasskeyRegistration,
  listAdminPasskeys,
  renameAdminPasskey,
  revokeAdminPasskey,
  type AdminMFAEnrollment,
  type AdminMFAStatus,
} from '../../../shared/api';
import { credentialToJSON, isPasskeySupported, toCreationOptions } from '../../user-auth/lib/webauthn';

type Mode = 'idle' | 'enrolling' | 'recovery' | 'disabling' | 'regenerating';

function groupSecret(secret: string): string {
  return secret.replace(/(.{4})/g, '$1 ').trim();
}

/** Authenticator-app two-factor sign-in for the administrator account. */
export function TwoFactorSettings() {
  const [status, setStatus] = useState<AdminMFAStatus | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [mode, setMode] = useState<Mode>('idle');
  const [enrollment, setEnrollment] = useState<AdminMFAEnrollment | null>(null);
  const [code, setCode] = useState('');
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [passkeys, setPasskeys] = useState<Array<{ id: string; nickname: string }>>([]);

  const refresh = async () => {
    try {
      setStatus(await getAdminMFAStatus());
      const passkeyResponse = await listAdminPasskeys();
      setPasskeys(passkeyResponse.passkeys.map((passkey) => ({ id: passkey.id, nickname: passkey.nickname })));
      setLoadError(null);
    } catch (err) {
      setLoadError(getApiErrorMessage(err, 'Unable to load two-factor settings.'));
    }
  };

  const addPasskey = () => run(async () => {
    if (!isPasskeySupported()) throw new Error('This browser does not support passkeys.');
    const started = await beginAdminPasskeyRegistration();
    const credential = await navigator.credentials.create({ publicKey: toCreationOptions(JSON.parse(started.optionsJson)) as PublicKeyCredentialCreationOptions });
    if (!(credential instanceof PublicKeyCredential)) throw new Error('No passkey was created.');
    const nickname = window.prompt('Name this passkey', 'Administrator passkey')?.trim() || 'Administrator passkey';
    const saved = await finishAdminPasskeyRegistration(JSON.stringify(credentialToJSON(credential)), started.ceremonyId, nickname);
    setPasskeys((current) => [...current, { id: saved.id, nickname: saved.nickname }]);
  });

  const removePasskey = (id: string) => run(async () => {
    await revokeAdminPasskey(id);
    setPasskeys((current) => current.filter((passkey) => passkey.id !== id));
  });

  useEffect(() => {
    void refresh();
  }, []);

  const run = async (task: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await task();
    } catch (err) {
      setError(getApiErrorMessage(err, 'That didn’t work. Please try again.'));
    } finally {
      setBusy(false);
    }
  };

  const start = () => run(async () => {
    setEnrollment(await beginAdminMFAEnrollment());
    setCode('');
    setMode('enrolling');
  });

  const confirm = (event: FormEvent) => {
    event.preventDefault();
    void run(async () => {
      const result = await confirmAdminMFAEnrollment(code.replace(/\s/g, ''));
      setRecoveryCodes(result.recovery_codes);
      setEnrollment(null);
      setCode('');
      setMode('recovery');
      await refresh();
    });
  };

  const disable = (event: FormEvent) => {
    event.preventDefault();
    void run(async () => {
      await disableAdminMFA(code.trim());
      setCode('');
      setMode('idle');
      await refresh();
    });
  };

  const regenerate = (event: FormEvent) => {
    event.preventDefault();
    void run(async () => {
      const result = await regenerateAdminRecoveryCodes(code.trim());
      setRecoveryCodes(result.recovery_codes);
      setCode('');
      setMode('recovery');
      await refresh();
    });
  };

  const copyCodes = async () => {
    try {
      await navigator.clipboard.writeText(recoveryCodes.join('\n'));
      setCopied(true);
      window.setTimeout(() => { setCopied(false); }, 2000);
    } catch {
      setError('Copy failed. Select the codes and copy them manually.');
    }
  };

  const downloadCodes = () => {
    const blob = new Blob([`Admin recovery codes\nEach code works once.\n\n${recoveryCodes.join('\n')}\n`], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'admin-recovery-codes.txt';
    anchor.click();
    URL.revokeObjectURL(url);
  };

  const errorBox = error && (
    <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-100" role="alert" data-testid="mfa-error">{error}</div>
  );

  return (
    <FormSection
      title="Two-factor authentication"
      description="Require a code from an authenticator app (1Password, Google Authenticator, Authy…) in addition to your password."
      icon={Smartphone}
      iconColorClass="text-cyan-300"
      testId="profile-mfa-section"
      actions={status && (
        <span
          className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium ${status.enabled ? 'border-emerald-400/40 bg-emerald-500/10 text-emerald-200' : 'border-amber-400/40 bg-amber-500/10 text-amber-100'}`}
          data-testid="mfa-status"
        >
          {status.enabled ? <ShieldCheck className="h-3.5 w-3.5" /> : <ShieldOff className="h-3.5 w-3.5" />}
          {status.enabled ? 'On' : 'Off'}
        </span>
      )}
    >
      {loadError && <div className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-100">{loadError}</div>}

      <div className="space-y-3 rounded-lg border border-white/10 bg-slate-950/30 p-4" data-testid="admin-passkeys">
        <div>
          <h3 className="font-semibold text-slate-100">Passkeys</h3>
          <p className="text-sm text-slate-400">Use a device passkey instead of an authenticator code when signing in.</p>
        </div>
        {passkeys.length > 0 && <ul aria-label="Administrator passkeys" className="space-y-2">{passkeys.map((passkey) => <li key={passkey.id} className="flex items-center gap-2 text-sm text-slate-200"><span className="flex-1">{passkey.nickname}</span><Button type="button" variant="outline" disabled={busy} onClick={() => { const next = window.prompt('Passkey name', passkey.nickname)?.trim(); if (next) void run(async () => { await renameAdminPasskey(passkey.id, next); setPasskeys((current) => current.map((item) => item.id === passkey.id ? { ...item, nickname: next } : item)); }); }}>Rename</Button><Button type="button" variant="outline" disabled={busy} onClick={() => { void removePasskey(passkey.id); }}>Remove</Button></li>)}</ul>}
        {isPasskeySupported() && <Button type="button" variant="outline" className="gap-2" disabled={busy} onClick={() => { void addPasskey(); }} data-testid="admin-passkey-add"><KeyRound className="h-4 w-4" />{busy ? 'Working…' : 'Add a passkey'}</Button>}
      </div>

      {mode === 'recovery' && (
        <div className="space-y-4" data-testid="mfa-recovery-codes">
          <div className="flex items-start gap-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-50">
            <CheckCircle2 className="mt-0.5 h-4 w-4 flex-none" />
            <p>Save these recovery codes somewhere safe, such as your password manager. Each works once if you lose your authenticator. They won’t be shown again.</p>
          </div>
          <ol className="grid grid-cols-2 gap-2 rounded-lg border border-white/10 bg-slate-950/60 p-4 font-mono text-sm text-slate-100">
            {recoveryCodes.map((recovery) => <li key={recovery} className="tracking-wider">{recovery}</li>)}
          </ol>
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" className="gap-2" onClick={() => { void copyCodes(); }}>
              <Copy className="h-4 w-4" />{copied ? 'Copied' : 'Copy codes'}
            </Button>
            <Button type="button" variant="outline" className="gap-2" onClick={downloadCodes}>
              <Download className="h-4 w-4" />Download
            </Button>
            <Button type="button" className="ml-auto" onClick={() => { setRecoveryCodes([]); setMode('idle'); }} data-testid="mfa-recovery-done">
              I’ve saved them
            </Button>
          </div>
        </div>
      )}

      {mode === 'enrolling' && enrollment && (
        <form className="space-y-5" onSubmit={confirm} data-testid="mfa-enroll-form">
          <div className="grid gap-5 sm:grid-cols-[auto_minmax(0,1fr)] sm:items-center">
            <div className="justify-self-center rounded-xl bg-white p-2 shadow-lg shadow-cyan-500/10">
              <QrCode value={enrollment.otpauth_uri} size={176} label="QR code for your authenticator app" />
            </div>
            <div className="space-y-3 text-sm text-slate-300">
              <p><span className="font-semibold text-slate-100">1.</span> Scan the QR code with your authenticator app.</p>
              <p className="text-slate-400">Can’t scan? Enter this key manually:</p>
              <code className="block break-all rounded-md border border-white/10 bg-slate-950/70 px-3 py-2 font-mono text-xs tracking-wider text-cyan-100" data-testid="mfa-secret">
                {groupSecret(enrollment.secret)}
              </code>
              <p><span className="font-semibold text-slate-100">2.</span> Enter the 6-digit code it shows.</p>
            </div>
          </div>
          <FormField label="Authenticator code">
            <input
              value={code}
              onChange={(event) => { setCode(event.target.value.replace(/[^\d\s]/g, '')); }}
              inputMode="numeric"
              autoComplete="one-time-code"
              placeholder="123 456"
              className={`${inputClassName} font-mono tracking-[0.3em]`}
              data-testid="mfa-enroll-code"
            />
          </FormField>
          {errorBox}
          <div className="flex flex-wrap justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => { setMode('idle'); setEnrollment(null); setError(null); }}>Cancel</Button>
            <Button type="submit" disabled={busy || code.replace(/\s/g, '').length !== 6} data-testid="mfa-enroll-confirm">
              {busy ? 'Verifying…' : 'Turn on two-factor'}
            </Button>
          </div>
        </form>
      )}

      {(mode === 'disabling' || mode === 'regenerating') && (
        <form className="space-y-4" onSubmit={mode === 'disabling' ? disable : regenerate} data-testid="mfa-verify-form">
          <p className="text-sm text-slate-300">
            {mode === 'disabling'
              ? 'Confirm with a current authenticator code (or a recovery code) to turn two-factor authentication off.'
              : 'Confirm with a current authenticator code to replace all recovery codes. Old codes stop working.'}
          </p>
          <FormField label="Code">
            <input
              value={code}
              onChange={(event) => { setCode(event.target.value); }}
              autoComplete="one-time-code"
              className={`${inputClassName} font-mono tracking-widest`}
              data-testid="mfa-verify-code"
            />
          </FormField>
          {errorBox}
          <div className="flex flex-wrap justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => { setMode('idle'); setCode(''); setError(null); }}>Cancel</Button>
            <Button type="submit" variant={mode === 'disabling' ? 'destructive' : 'default'} disabled={busy || code.trim().length < 6}>
              {busy ? 'Working…' : mode === 'disabling' ? 'Turn off two-factor' : 'Replace recovery codes'}
            </Button>
          </div>
        </form>
      )}

      {mode === 'idle' && status && (
        <div className="space-y-4">
          {status.enabled ? (
            <>
              <p className="text-sm text-slate-300">
                Signing in requires your password and a code from your authenticator app.
                <span className="block text-slate-400">{status.recovery_codes_left} of 10 recovery codes left.</span>
              </p>
              {status.recovery_codes_left <= 3 && (
                <div className="rounded-md border border-amber-400/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-100">You’re running low on recovery codes. Generate a new set.</div>
              )}
              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="outline" className="gap-2" onClick={() => { setCode(''); setMode('regenerating'); }}>
                  <KeyRound className="h-4 w-4" />New recovery codes
                </Button>
                <Button type="button" variant="outline" className="gap-2 text-red-200" onClick={() => { setCode(''); setMode('disabling'); }} data-testid="mfa-disable">
                  <ShieldOff className="h-4 w-4" />Turn off
                </Button>
              </div>
            </>
          ) : (
            <>
              <p className="text-sm text-slate-300">Without two-factor authentication, anyone with your password can manage this site and its customers.</p>
              {errorBox}
              <Button type="button" className="gap-2" disabled={busy} onClick={() => { void start(); }} data-testid="mfa-enable">
                <ShieldCheck className="h-4 w-4" />{busy ? 'Starting…' : 'Set up two-factor'}
              </Button>
            </>
          )}
        </div>
      )}
    </FormSection>
  );
}
