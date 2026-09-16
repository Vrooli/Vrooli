import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { AlertCircle, ArrowLeft, ArrowRight, CheckCircle2, ExternalLink, Laptop, Mail, RotateCw, ShieldCheck } from 'lucide-react';
import { AuthPageLayout } from '../../../shared/ui/AuthPageLayout';
import {
  authorizeNativeApp,
  isApiError,
  requestMagicLink,
  verifySignInCode,
  type BusinessAccount,
} from '../../../shared/api';
import { useSiteIdentity } from '../../public-landing/site/useSiteIdentity';
import { CodeInput } from '../components/CodeInput';
import { SignInAside } from '../components/SignInAside';
import { AccountChooser } from '../components/AccountChooser';
import { getBrowserBinding } from '../lib/browserBinding';
import { contextProblem, describeScope, flowForContext, safeNextPath, signInContextFromSearch } from '../lib/signInContext';
import { continueDesktopLink, finishDesktopLink, redirectBrowser, type Redirect } from '../lib/completeSignIn';
import { mailShortcutFor } from '../lib/mailProviders';

const RESEND_COOLDOWN_SECONDS = 45;

export function isValidEmail(email: string): boolean {
  const trimmed = email.trim();
  const at = trimmed.lastIndexOf('@');
  if (at < 1 || /\s/.test(trimmed)) return false;
  const domain = trimmed.slice(at + 1);
  return domain.includes('.') && !domain.startsWith('.') && !domain.endsWith('.');
}

function describeRequestError(err: unknown): string {
  if (isApiError(err, 'rate_limited')) return 'Too many sign-in requests. Wait a few minutes, then try again.';
  if (isApiError(err) && err.reason === 'delivery_unavailable') return 'We couldn’t send the email right now. Please try again in a minute.';
  if (isApiError(err, 'validation')) return err.userMessage || 'Please enter a valid email address.';
  if (isApiError(err, 'network') || isApiError(err, 'timeout')) return 'We couldn’t reach the server. Check your connection and try again.';
  return 'Something went wrong on our side. Please try again.';
}


type Step = 'email' | 'code' | 'choose-account' | 'link-retry' | 'done';

function formatCountdown(seconds: number): string {
  return `${String(Math.floor(seconds / 60))}:${String(seconds % 60).padStart(2, '0')}`;
}

export function UserLogin({ redirectTo = redirectBrowser }: { redirectTo?: Redirect }) {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const identity = useSiteIdentity();
  const context = useMemo(() => signInContextFromSearch(searchParams), [searchParams]);
  const flow = flowForContext(context);
  const problem = context ? contextProblem(context) : null;
  const nextPath = safeNextPath(searchParams.get('next'));
  const siteName = identity.brandName || 'your account';
  const appName = context?.app || siteName;

  const [step, setStep] = useState<Step>('email');
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [accounts, setAccounts] = useState<BusinessAccount[]>([]);
  const [choosingId, setChoosingId] = useState<string | null>(null);
  const emailRef = useRef<HTMLInputElement>(null);
  const submittingCode = useRef(false);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = window.setTimeout(() => { setCooldown((value) => value - 1); }, 1000);
    return () => { window.clearTimeout(timer); };
  }, [cooldown]);

  const sendCode = useCallback(async (address: string, isResend: boolean) => {
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const started = await requestMagicLink(address, { browserBinding: getBrowserBinding(), context });
      setExpiresAt(started.expires_at ?? null);
      setCooldown(RESEND_COOLDOWN_SECONDS);
      setCode('');
      setStep('code');
      if (isResend) setNotice('We sent a new code. Earlier codes still work until they expire.');
    } catch (err) {
      setError(describeRequestError(err));
    } finally {
      setBusy(false);
    }
  }, [context]);

  const handleEmailSubmit = (event: FormEvent) => {
    event.preventDefault();
    const address = email.trim().toLowerCase();
    if (!address) {
      setError('Email is required');
      emailRef.current?.focus();
      return;
    }
    if (!isValidEmail(address)) {
      setError('Please enter a valid email address');
      emailRef.current?.focus();
      return;
    }
    setEmail(address);
    void sendCode(address, false);
  };

  const submitCode = useCallback(async (value: string) => {
    if (submittingCode.current || value.length !== 6) return;
    submittingCode.current = true;
    setBusy(true);
    setError(null);
    setNotice(null);
    const binding = getBrowserBinding();
    try {
      if (flow === 'native_app' && context) {
        const target = await authorizeNativeApp({ email, code: value, browserBinding: binding }, context);
        setStep('done');
        redirectTo(target);
        return;
      }
      await verifySignInCode(email, value, binding);
      if (flow === 'desktop_link' && context) {
        await connectDesktop();
        return;
      }
      setStep('done');
      window.setTimeout(() => { navigate(nextPath, { replace: true }); }, 900);
    } catch (err) {
      setCode('');
      if (isApiError(err) && err.reason === 'code_invalid') {
        setError('That code isn’t right or has expired. Check your latest email and try again.');
      } else if (isApiError(err, 'rate_limited')) {
        setError('Too many incorrect codes. Request a new code in a few minutes.');
      } else if (isApiError(err, 'network') || isApiError(err, 'timeout')) {
        setError('We couldn’t reach the server. Check your connection and try again.');
      } else if (err instanceof Error && !isApiError(err)) {
        setError(err.message);
      } else {
        setError('We couldn’t finish signing you in. Please try again.');
      }
    } finally {
      submittingCode.current = false;
      setBusy(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- connectDesktop reads the same context and redirect.
  }, [context, email, flow, navigate, nextPath, redirectTo]);

  // Runs after the browser is signed in; a failure here is retried without a
  // new code because the code has already been used.
  async function connectDesktop() {
    if (!context) return;
    try {
      const outcome = await continueDesktopLink(context, redirectTo);
      if (outcome.kind === 'choose-account') {
        setAccounts(outcome.accounts);
        setStep('choose-account');
        return;
      }
      setStep('done');
    } catch (err) {
      setError(err instanceof Error && !isApiError(err) ? err.message : 'We couldn’t connect the app. Please try again.');
      setStep('link-retry');
    }
  }

  const chooseAccount = async (accountId: string) => {
    if (!context) return;
    setChoosingId(accountId);
    setError(null);
    try {
      await finishDesktopLink(context, accountId, redirectTo);
      setStep('done');
    } catch (err) {
      setError(err instanceof Error && !isApiError(err) ? err.message : 'We couldn’t connect that account. Please try again.');
    } finally {
      setChoosingId(null);
    }
  };

  const expiryText = useMemo(() => {
    if (!expiresAt) return 'The code expires in 15 minutes.';
    const time = new Date(expiresAt);
    if (Number.isNaN(time.getTime())) return 'The code expires in 15 minutes.';
    return `The code expires at ${time.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}.`;
  }, [expiresAt]);

  const aside = <SignInAside appName={siteName} />;
  const mailShortcut = mailShortcutFor(email);

  if (problem) {
    return (
      <AuthPageLayout pageTitle="Sign in" aside={aside} stepKey="problem">
        <div className="auth-step auth-status" role="alert">
          <span className="auth-status-mark auth-status-danger" aria-hidden="true"><AlertCircle /></span>
          <h1>We can’t continue this sign-in</h1>
          <p>{problem}</p>
          <Link className="button button-secondary" to="/auth/login">Sign in to the website instead</Link>
        </div>
      </AuthPageLayout>
    );
  }

  if (step === 'done') {
    return (
      <AuthPageLayout pageTitle="Signed in" aside={aside} stepKey="done">
        <div className="auth-step auth-status" role="status" aria-live="polite">
          <span className="auth-status-mark auth-status-success" aria-hidden="true"><CheckCircle2 /></span>
          <h1>You’re signed in</h1>
          <p>{flow === 'browser' ? 'Taking you there now…' : `Returning you to ${appName}. You can close this tab once it opens.`}</p>
        </div>
      </AuthPageLayout>
    );
  }

  if (step === 'link-retry') {
    return (
      <AuthPageLayout pageTitle={`Connect ${appName}`} aside={aside} stepKey="link-retry">
        <div className="auth-step auth-status" role="alert">
          <span className="auth-status-mark auth-status-warning" aria-hidden="true"><AlertCircle /></span>
          <h1>You’re signed in, but {appName} isn’t connected yet</h1>
          <p>{error}</p>
          <button type="button" className="button button-primary" disabled={busy} onClick={() => { setBusy(true); setError(null); void connectDesktop().finally(() => { setBusy(false); }); }}>
            <RotateCw aria-hidden="true" />Try connecting again
          </button>
        </div>
      </AuthPageLayout>
    );
  }

  if (step === 'choose-account') {
    return (
      <AuthPageLayout pageTitle="Choose an account" aside={aside} stepKey="choose">
        <AccountChooser appName={appName} accounts={accounts} busyId={choosingId} onChoose={(id) => { void chooseAccount(id); }} />
        {error && <p className="auth-alert" role="alert"><AlertCircle aria-hidden="true" />{error}</p>}
      </AuthPageLayout>
    );
  }

  if (step === 'code') {
    return (
      <AuthPageLayout pageTitle="Check your email" aside={aside} stepKey="code">
        <form
          className="auth-step"
          noValidate
          aria-busy={busy}
          onSubmit={(event) => { event.preventDefault(); void submitCode(code); }}
        >
          <button type="button" className="auth-back" onClick={() => { setStep('email'); setError(null); setNotice(null); }}>
            <ArrowLeft aria-hidden="true" />Use a different email
          </button>
          <span className="auth-badge" aria-hidden="true"><Mail /></span>
          <header className="auth-head">
            <h1>Check your email</h1>
            <p>Enter the 6-digit code we sent to <strong className="auth-email">{email}</strong></p>
          </header>

          <CodeInput
            value={code}
            onChange={(value) => { setCode(value); if (error) setError(null); }}
            onComplete={(value) => { void submitCode(value); }}
            disabled={busy}
            invalid={Boolean(error)}
            autoFocus
            label="6-digit sign-in code"
            describedBy={error ? 'auth-code-error' : 'auth-code-hint'}
            testId="code-input"
          />

          <div className="auth-feedback" aria-live="polite">
            {error
              ? <p className="auth-alert" id="auth-code-error" role="alert"><AlertCircle aria-hidden="true" />{error}</p>
              : notice
                ? <p className="auth-notice"><CheckCircle2 aria-hidden="true" />{notice}</p>
                : <p className="auth-hint" id="auth-code-hint">{expiryText} You can also open the link in the email on this device.</p>}
          </div>

          <button type="submit" className="button button-primary auth-submit" disabled={busy || code.length !== 6} data-testid="verify-code-button">
            {busy ? <><span className="site-spinner" aria-hidden="true" />Checking…</> : <>Continue<ArrowRight aria-hidden="true" /></>}
          </button>

          <div className="auth-secondary-actions">
            {mailShortcut && (
              <a className="auth-link" href={mailShortcut.href} target="_blank" rel="noreferrer noopener">
                {mailShortcut.label}<ExternalLink aria-hidden="true" />
              </a>
            )}
            <button
              type="button"
              className="auth-link"
              disabled={busy || cooldown > 0}
              onClick={() => { void sendCode(email, true); }}
              data-testid="resend-button"
            >
              <RotateCw aria-hidden="true" />
              {cooldown > 0 ? <>Resend code in <span className="auth-tabular">{formatCountdown(cooldown)}</span></> : 'Resend code'}
            </button>
          </div>
          <p className="auth-fineprint">No email? Check spam or promotions, or make sure the address above is right.</p>
        </form>
      </AuthPageLayout>
    );
  }

  return (
    <AuthPageLayout pageTitle={flow === 'desktop_link' ? `Connect ${appName}` : 'Sign in'} aside={aside} stepKey="email">
      <form className="auth-step" onSubmit={handleEmailSubmit} noValidate aria-busy={busy}>
        <header className="auth-head">
          <h1>{flow === 'browser' ? `Sign in to ${siteName}` : `Continue to ${appName}`}</h1>
          <p>{flow === 'browser' ? 'Enter your email. We’ll send a code. New here? This creates your account.' : `Sign in with your ${siteName} account to continue.`}</p>
        </header>

        {context && flow === 'desktop_link' && (
          <div className="auth-request" data-testid="desktop-link-consent">
            <div className="auth-request-head">
              <span className="auth-request-icon" aria-hidden="true"><Laptop /></span>
              <p><strong>{appName}</strong> wants to connect to your account on this computer.</p>
            </div>
            <p className="auth-request-label">It will be able to:</p>
            <ul className="auth-scopes">
              {(context.scopes ?? []).map((scope) => (
                <li key={scope}><ShieldCheck aria-hidden="true" />{describeScope(scope)}<code>{scope}</code></li>
              ))}
            </ul>
          </div>
        )}
        {context && flow === 'native_app' && (
          <div className="auth-request auth-request-compact">
            <div className="auth-request-head">
              <span className="auth-request-icon" aria-hidden="true"><Laptop /></span>
              <p>After you sign in, you’ll return to <strong>{appName}</strong> on this computer.</p>
            </div>
          </div>
        )}

        <div className="site-field auth-field" data-invalid={Boolean(error)}>
          <label htmlFor="email">Email address</label>
          <div className="site-input-icon">
            <Mail aria-hidden="true" />
            <input
              ref={emailRef}
              id="email"
              type="email"
              name="email"
              value={email}
              onChange={(event) => { setEmail(event.target.value); if (error) setError(null); }}
              placeholder="you@example.com"
              disabled={busy}
              autoComplete="email"
              inputMode="email"
              autoCapitalize="none"
              spellCheck={false}
              autoFocus
              aria-invalid={Boolean(error)}
              aria-describedby={error ? 'email-error' : undefined}
              data-testid="email-input"
            />
          </div>
          {error && <p className="site-field-error" id="email-error" role="alert"><AlertCircle aria-hidden="true" />{error}</p>}
        </div>

        <button type="submit" disabled={busy} data-testid="submit-button" className="button button-primary auth-submit">
          {busy ? <><span className="site-spinner" aria-hidden="true" />Sending code…</> : <>Continue with email<ArrowRight aria-hidden="true" /></>}
        </button>

        <p className="auth-fineprint">
          By continuing you agree to the <Link to="/terms">Terms</Link> and <Link to="/privacy">Privacy Policy</Link>.
        </p>
      </form>
    </AuthPageLayout>
  );
}

export default UserLogin;
