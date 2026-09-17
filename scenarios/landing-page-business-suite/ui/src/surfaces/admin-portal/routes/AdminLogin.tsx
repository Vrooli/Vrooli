import { useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import { AlertCircle, ArrowLeft, ArrowRight, Eye, EyeOff, KeyRound, Lock, Mail, ShieldCheck, WifiOff } from 'lucide-react';
import { AuthPageLayout } from '../../../shared/ui/AuthPageLayout';
import { useAdminAuth } from '../../../app/providers/useAdminAuth';
import { isApiError } from '../../../shared/api';
import { CodeInput } from '../../user-auth/components/CodeInput';
import { beginAdminSecondFactor } from '../../../shared/api';
import { credentialToJSON, isPasskeySupported, toRequestOptions } from '../../user-auth/lib/webauthn';

type LoginFailure = { message: string; kind: 'auth' | 'locked' | 'network' | 'server' };

function describeLoginError(err: unknown, step: 'credentials' | 'second-factor'): LoginFailure | 'mfa-required' {
  if (err instanceof ConnectError) {
    switch (err.code) {
      case Code.FailedPrecondition:
        return 'mfa-required';
      case Code.ResourceExhausted:
        return { kind: 'locked', message: 'Too many failed attempts. For your security, sign-in is paused for 15 minutes.' };
      case Code.Unauthenticated:
        return step === 'second-factor'
          ? { kind: 'auth', message: 'That code isn’t right. Use the current code from your app, or a recovery code.' }
          : { kind: 'auth', message: 'That email and password don’t match an administrator account.' };
      case Code.Unavailable:
        return { kind: 'server', message: err.rawMessage || 'Sign-in is temporarily unavailable. Please try again shortly.' };
      case Code.Unknown:
      case Code.Canceled:
      case Code.DeadlineExceeded:
        return { kind: 'network', message: 'We couldn’t reach the server. Check your connection and try again.' };
      default:
        return { kind: 'server', message: 'The server couldn’t complete sign-in. Please try again.' };
    }
  }
  if (isApiError(err, 'network') || isApiError(err, 'timeout') || err instanceof TypeError) {
    return { kind: 'network', message: 'We couldn’t reach the server. Check your connection and try again.' };
  }
  if (isApiError(err, 'server_error')) {
    return { kind: 'server', message: 'The server couldn’t complete sign-in. Please try again.' };
  }
  return { kind: 'auth', message: 'That email and password don’t match an administrator account.' };
}

/**
 * Operator sign-in: password, then an authenticator code when two-factor
 * authentication is on. Not linked from public pages.
 *
 * [REQ:ADMIN-AUTH] [REQ:ADMIN-SECURITY]
 */
export function AdminLogin() {
  const navigate = useNavigate();
  const { login } = useAdminAuth();
  const [step, setStep] = useState<'credentials' | 'second-factor'>('credentials');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [code, setCode] = useState('');
  const [useRecovery, setUseRecovery] = useState(false);
  const [recoveryCode, setRecoveryCode] = useState('');
  const [failure, setFailure] = useState<LoginFailure | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [passkeyLoading, setPasskeyLoading] = useState(false);
  const passwordRef = useRef<HTMLInputElement>(null);

  const attempt = async (secondFactor = '', passkeyAssertion?: Uint8Array, passkeyCeremonyId = '') => {
    setFailure(null);
    setIsLoading(true);
    try {
      if (passkeyAssertion && passkeyCeremonyId) {
        await login(email.trim(), password, secondFactor, passkeyAssertion, passkeyCeremonyId);
      } else {
        await login(email.trim(), password, secondFactor);
      }
      navigate('/admin');
    } catch (err) {
      const outcome = describeLoginError(err, secondFactor ? 'second-factor' : 'credentials');
      if (outcome === 'mfa-required') {
        setStep('second-factor');
        setCode('');
        return;
      }
      setFailure(outcome);
      if (secondFactor) setCode('');
    } finally {
      setIsLoading(false);
    }
  };

  const signInWithPasskey = async () => {
    if (!isPasskeySupported()) {
      setFailure({ kind: 'auth', message: 'This browser does not support passkeys.' });
      return;
    }
    setFailure(null);
    setPasskeyLoading(true);
    try {
      const started = await beginAdminSecondFactor(email.trim());
      const credential = await navigator.credentials.get({ publicKey: toRequestOptions(JSON.parse(started.optionsJson)) as PublicKeyCredentialRequestOptions });
      if (!(credential instanceof PublicKeyCredential)) throw new Error('No passkey assertion was returned.');
      await attempt('', new TextEncoder().encode(JSON.stringify(credentialToJSON(credential))), started.ceremonyId);
    } catch (err) {
      if (err instanceof DOMException && err.name === 'NotAllowedError') return;
      setFailure({ kind: 'auth', message: 'That passkey was not accepted. Try again or use another second factor.' });
    } finally {
      setPasskeyLoading(false);
    }
  };

  const onCredentials = (event: FormEvent) => {
    event.preventDefault();
    if (!email.trim() || !password) {
      setFailure({ kind: 'auth', message: 'Enter your administrator email and password.' });
      return;
    }
    void attempt();
  };

  const onSecondFactor = (event: FormEvent) => {
    event.preventDefault();
    const value = useRecovery ? recoveryCode.trim() : code;
    if (useRecovery ? value.length < 10 : value.length !== 6) return;
    void attempt(value);
  };

  const alert = failure && (
    <div className={`auth-alert-box auth-alert-${failure.kind}`} role="alert" data-testid="admin-login-error">
      {failure.kind === 'network' ? <WifiOff aria-hidden="true" /> : <AlertCircle aria-hidden="true" />}
      <p>{failure.message}</p>
      {failure.kind === 'network' && (
        <button type="button" className="auth-link" onClick={() => { void attempt(step === 'second-factor' ? (useRecovery ? recoveryCode.trim() : code) : ''); }}>
          Retry
        </button>
      )}
    </div>
  );

  if (step === 'second-factor') {
    return (
      <AuthPageLayout pageTitle="Two-factor verification" chrome="minimal" stepKey="admin-mfa">
        <form className="auth-step" onSubmit={onSecondFactor} noValidate aria-busy={isLoading}>
          <button type="button" className="auth-back" onClick={() => { setStep('credentials'); setFailure(null); setPassword(''); }}>
            <ArrowLeft aria-hidden="true" />Back
          </button>
          <span className="auth-badge" aria-hidden="true"><ShieldCheck /></span>
          <header className="auth-head">
            <h1>Two-factor verification</h1>
            <p>{useRecovery ? 'Enter one of the recovery codes you saved when you turned on two-factor authentication.' : 'Enter the 6-digit code from your authenticator app.'}</p>
          </header>

          {useRecovery ? (
            <div className="site-field auth-field">
              <label htmlFor="recovery-code">Recovery code</label>
              <div className="site-input-icon">
                <KeyRound aria-hidden="true" />
                <input
                  id="recovery-code"
                  value={recoveryCode}
                  onChange={(event) => { setRecoveryCode(event.target.value); setFailure(null); }}
                  placeholder="xxxxx-xxxxx"
                  autoComplete="one-time-code"
                  autoCapitalize="none"
                  spellCheck={false}
                  autoFocus
                  disabled={isLoading}
                  data-testid="admin-recovery-code"
                />
              </div>
            </div>
          ) : (
            <CodeInput
              value={code}
              onChange={(value) => { setCode(value); setFailure(null); }}
              onComplete={(value) => { void attempt(value); }}
              disabled={isLoading}
              invalid={failure?.kind === 'auth'}
              autoFocus
              label="Authenticator code"
              testId="admin-totp-code"
            />
          )}

          {alert}

          <button
            type="submit"
            className="button button-primary auth-submit"
            disabled={isLoading || (useRecovery ? recoveryCode.trim().length < 10 : code.length !== 6)}
            data-testid="admin-mfa-submit"
          >
            {isLoading ? <><span className="site-spinner" aria-hidden="true" />Verifying…</> : <>Verify and sign in<ArrowRight aria-hidden="true" /></>}
          </button>
          {isPasskeySupported() && !useRecovery && (
            <button type="button" className="auth-link auth-toggle" onClick={() => { void signInWithPasskey(); }} disabled={isLoading || passkeyLoading} data-testid="admin-passkey-sign-in">
              <KeyRound aria-hidden="true" />{passkeyLoading ? 'Checking passkey…' : 'Use a passkey instead'}
            </button>
          )}
          <button
            type="button"
            className="auth-link auth-toggle"
            onClick={() => { setUseRecovery(!useRecovery); setFailure(null); }}
          >
            <KeyRound aria-hidden="true" />{useRecovery ? 'Use authenticator app instead' : 'Use a recovery code'}
          </button>
        </form>
      </AuthPageLayout>
    );
  }

  return (
    <AuthPageLayout pageTitle="Admin sign-in" chrome="minimal" stepKey="admin-credentials">
      <form className="auth-step" onSubmit={onCredentials} noValidate aria-busy={isLoading}>
        <span className="auth-badge" aria-hidden="true"><Lock /></span>
        <header className="auth-head">
          <h1>Admin sign-in</h1>
          <p>Manage your site, pricing, and customers.</p>
        </header>

        <div className="site-field auth-field">
          <label htmlFor="email">Email address</label>
          <div className="site-input-icon">
            <Mail aria-hidden="true" />
            <input
              id="email"
              type="email"
              value={email}
              onChange={(event) => { setEmail(event.target.value); setFailure(null); }}
              placeholder="admin@example.com"
              autoComplete="username"
              autoCapitalize="none"
              spellCheck={false}
              autoFocus
              required
              disabled={isLoading}
              data-testid="admin-login-email"
            />
          </div>
        </div>

        <div className="site-field auth-field">
          <label htmlFor="password">Password</label>
          <div className="site-input-icon auth-password">
            <Lock aria-hidden="true" />
            <input
              ref={passwordRef}
              id="password"
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(event) => { setPassword(event.target.value); setFailure(null); }}
              autoComplete="current-password"
              required
              disabled={isLoading}
              data-testid="admin-login-password"
            />
            <button
              type="button"
              className="auth-reveal"
              aria-label={showPassword ? 'Hide password' : 'Show password'}
              aria-pressed={showPassword}
              onClick={() => { setShowPassword(!showPassword); passwordRef.current?.focus(); }}
            >
              {showPassword ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
            </button>
          </div>
        </div>

        {alert}

        <button type="submit" className="button button-primary auth-submit" data-testid="admin-login-submit" disabled={isLoading || failure?.kind === 'locked'}>
          {isLoading ? <><span className="site-spinner" aria-hidden="true" />Signing in…</> : <>Sign in<ArrowRight aria-hidden="true" /></>}
        </button>
        <p className="auth-fineprint auth-fineprint-center"><ShieldCheck aria-hidden="true" />Administrator access only. Sign-in attempts are rate-limited and logged.</p>
      </form>
    </AuthPageLayout>
  );
}
