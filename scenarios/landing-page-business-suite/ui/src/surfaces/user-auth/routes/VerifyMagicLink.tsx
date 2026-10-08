import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { AlertCircle, ArrowRight, CheckCircle2, Clock3, Laptop, LinkIcon, RefreshCw, ShieldAlert, UserRound } from 'lucide-react';
import { AuthPageLayout } from '../../../shared/ui/AuthPageLayout';
import {
  authorizeNativeApp,
  isApiError,
  previewSignIn,
  verifyMagicLink,
  type BusinessAccount,
  type SignInContext,
  type SignInPreview,
} from '../../../shared/api';
import { useSiteIdentity } from '../../public-landing/site/useSiteIdentity';
import { SignInAside } from '../components/SignInAside';
import { AccountChooser } from '../components/AccountChooser';
import { getBrowserBinding } from '../lib/browserBinding';
import { contextProblem } from '../lib/signInContext';
import { continueDesktopLink, finishDesktopLink, redirectBrowser, type Redirect } from '../lib/completeSignIn';

export { isAllowedCallbackUrl } from '../lib/signInContext';

type Failure = 'expired' | 'used' | 'invalid' | 'network' | 'unknown';

type ViewState =
  | { status: 'loading' }
  | { status: 'confirm'; preview: SignInPreview }
  | { status: 'working'; preview: SignInPreview }
  | { status: 'choose-account'; context: SignInContext; accounts: BusinessAccount[] }
  | { status: 'done'; returning: boolean }
  | { status: 'error'; failure: Failure; message: string; retry?: () => void };

function classify(err: unknown): { failure: Failure; message: string } {
  if (isApiError(err)) {
    switch (err.reason) {
      case 'token_expired':
        return { failure: 'expired', message: 'This link has expired. Links and codes last 15 minutes.' };
      case 'token_used':
        return { failure: 'used', message: 'This link was already used. If you aren’t signed in, request a new code.' };
      case 'token_invalid':
        return { failure: 'invalid', message: 'This link isn’t valid. It may be incomplete if it was copied from the email.' };
    }
    if (err.type === 'network' || err.type === 'timeout') {
      return { failure: 'network', message: 'We couldn’t reach the server. Check your connection and try again.' };
    }
  }
  if (err instanceof Error && !isApiError(err)) {
    return { failure: 'unknown', message: err.message };
  }
  return { failure: 'unknown', message: 'We couldn’t finish signing you in. Please try again.' };
}

const FAILURE_TITLES: Record<Failure, string> = {
  expired: 'This link has expired',
  used: 'This link was already used',
  invalid: 'This link doesn’t work',
  network: 'Connection problem',
  unknown: 'Sign-in didn’t finish',
};

export function VerifyMagicLink({ redirectTo = redirectBrowser }: { redirectTo?: Redirect }) {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const identity = useSiteIdentity();
  const token = searchParams.get('token')?.trim() ?? '';
  const siteName = identity.brandName || 'your account';
  const [state, setState] = useState<ViewState>({ status: 'loading' });
  const [choosingId, setChoosingId] = useState<string | null>(null);
  const [chooseError, setChooseError] = useState<string | null>(null);

  const loadPreview = useCallback(async () => {
    if (!token) {
      setState({ status: 'error', failure: 'invalid', message: 'This link is missing its sign-in token. Open the link from the email again, or request a new code.' });
      return;
    }
    setState({ status: 'loading' });
    try {
      const preview = await previewSignIn(token, getBrowserBinding());
      setState({ status: 'confirm', preview });
    } catch (err) {
      const { failure, message } = classify(err);
      setState({ status: 'error', failure, message, retry: failure === 'network' ? () => { void loadPreview(); } : undefined });
    }
  }, [token]);

  useEffect(() => {
    void loadPreview();
  }, [loadPreview]);

  const appContext = state.status === 'confirm' || state.status === 'working' ? state.preview.context : undefined;
  const appName = appContext?.app || siteName;

  const confirm = useCallback(async (preview: SignInPreview, websiteOnly: boolean) => {
    setState({ status: 'working', preview });
    const binding = getBrowserBinding();
    const context = websiteOnly ? undefined : preview.context;
    try {
      if (context) {
        const problem = contextProblem(context);
        if (problem) throw new Error(problem);
      }
      if (context && preview.flow === 'native_app') {
        const target = await authorizeNativeApp({ token, browserBinding: binding }, context);
        setState({ status: 'done', returning: true });
        redirectTo(target);
        return;
      }
      await verifyMagicLink(token, binding);
      if (context && preview.flow === 'desktop_link') {
        try {
          const outcome = await continueDesktopLink(context, redirectTo);
          if (outcome.kind === 'choose-account') {
            setState({ status: 'choose-account', context, accounts: outcome.accounts });
            return;
          }
          setState({ status: 'done', returning: true });
        } catch (err) {
          // The browser is signed in now; retrying only repeats the connection.
          const retry = () => {
            void continueDesktopLink(context, redirectTo)
              .then((outcome) => {
                setState(outcome.kind === 'choose-account'
                  ? { status: 'choose-account', context, accounts: outcome.accounts }
                  : { status: 'done', returning: true });
              })
              .catch((retryErr: unknown) => { setState({ status: 'error', failure: 'unknown', message: classify(retryErr).message, retry }); });
          };
          setState({ status: 'error', failure: 'unknown', message: `You’re signed in, but we couldn’t connect ${context.app || 'the app'}: ${classify(err).message}`, retry });
        }
        return;
      }
      setState({ status: 'done', returning: false });
      window.setTimeout(() => { navigate('/', { replace: true }); }, 900);
    } catch (err) {
      const { failure, message } = classify(err);
      setState({ status: 'error', failure, message, retry: failure === 'network' ? () => { void confirm(preview, websiteOnly); } : undefined });
    }
  }, [navigate, redirectTo, token]);

  const chooseAccount = async (context: SignInContext, accountId: string) => {
    setChoosingId(accountId);
    setChooseError(null);
    try {
      await finishDesktopLink(context, accountId, redirectTo);
      setState({ status: 'done', returning: true });
    } catch (err) {
      setChooseError(classify(err).message);
    } finally {
      setChoosingId(null);
    }
  };

  const aside = useMemo(() => <SignInAside appName={siteName} />, [siteName]);

  if (state.status === 'loading') {
    return (
      <AuthPageLayout pageTitle="Checking your link" aside={aside} stepKey="loading">
        <div className="auth-step auth-status" role="status" aria-live="polite" data-testid="verify-loading">
          <span className="auth-status-mark auth-status-pending" aria-hidden="true"><span className="site-spinner" /></span>
          <h1>Checking your link…</h1>
          <p>This only takes a moment.</p>
        </div>
      </AuthPageLayout>
    );
  }

  if (state.status === 'confirm' || state.status === 'working') {
    const { preview } = state;
    const working = state.status === 'working';
    const otherBrowserApp = preview.flow !== 'browser' && !preview.same_browser;
    return (
      <AuthPageLayout pageTitle="Confirm sign-in" aside={aside} stepKey="confirm">
        <div className="auth-step" aria-busy={working}>
          <span className="auth-badge" aria-hidden="true"><LinkIcon /></span>
          <header className="auth-head">
            <h1>{preview.flow === 'browser' || otherBrowserApp ? `Sign in to ${siteName}` : `Continue to ${appName}`}</h1>
            <p>Confirm it’s you to finish signing in.</p>
          </header>

          <div className="auth-identity" data-testid="verify-identity">
            <span className="auth-identity-avatar" aria-hidden="true"><UserRound /></span>
            <span className="auth-identity-text">
              <small>Signing in as</small>
              <strong>{preview.email_hint}</strong>
            </span>
          </div>

          {otherBrowserApp && (
            <div className="auth-callout" role="note">
              <ShieldAlert aria-hidden="true" />
              <p>
                This sign-in started in an app in a different browser or on another device.
                To finish there, <strong>type the 6-digit code from the email into that window</strong>.
                Signing in to the website here instead ends that app’s sign-in.
              </p>
            </div>
          )}
          {!otherBrowserApp && preview.flow !== 'browser' && (
            <div className="auth-callout auth-callout-info" role="note">
              <Laptop aria-hidden="true" />
              <p>After you confirm, you’ll return to <strong>{appName}</strong> on this computer.</p>
            </div>
          )}

          <button
            type="button"
            className="button button-primary auth-submit"
            disabled={working}
            onClick={() => { void confirm(preview, otherBrowserApp); }}
            data-testid="confirm-sign-in"
          >
            {working ? <><span className="site-spinner" aria-hidden="true" />Signing in…</> : <>{otherBrowserApp ? 'Sign in to the website' : 'Confirm and sign in'}<ArrowRight aria-hidden="true" /></>}
          </button>
          <p className="auth-fineprint">
            <Clock3 aria-hidden="true" />Link expires {new Date(preview.expires_at).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}. Not you? <Link to="/auth/login">Use a different email</Link>
          </p>
        </div>
      </AuthPageLayout>
    );
  }

  if (state.status === 'choose-account') {
    return (
      <AuthPageLayout pageTitle="Choose an account" aside={aside} stepKey="choose">
        <AccountChooser appName={state.context.app || 'the app'} accounts={state.accounts} busyId={choosingId} onChoose={(id) => { void chooseAccount(state.context, id); }} />
        {chooseError && <p className="auth-alert" role="alert"><AlertCircle aria-hidden="true" />{chooseError}</p>}
      </AuthPageLayout>
    );
  }

  if (state.status === 'done') {
    return (
      <AuthPageLayout pageTitle="Signed in" aside={aside} stepKey="done">
        <div className="auth-step auth-status" role="status" aria-live="polite">
          <span className="auth-status-mark auth-status-success" aria-hidden="true"><CheckCircle2 /></span>
          <h1>You’re signed in</h1>
          <p>{state.returning ? 'Returning you to the app. You can close this tab once it opens.' : 'Taking you there now…'}</p>
        </div>
      </AuthPageLayout>
    );
  }

  return (
    <AuthPageLayout pageTitle={FAILURE_TITLES[state.failure]} aside={aside} stepKey={`error-${state.failure}`}>
      <div className="auth-step auth-status" role="alert" data-testid="verify-error">
        <span className={`auth-status-mark ${state.failure === 'network' ? 'auth-status-warning' : 'auth-status-danger'}`} aria-hidden="true">
          {state.failure === 'expired' ? <Clock3 /> : <AlertCircle />}
        </span>
        <h1>{FAILURE_TITLES[state.failure]}</h1>
        <p>{state.message}</p>
        <div className="auth-status-actions">
          {state.retry && (
            <button type="button" className="button button-primary" onClick={state.retry}>
              <RefreshCw aria-hidden="true" />Try again
            </button>
          )}
          <Link className={`button ${state.retry ? 'button-secondary' : 'button-primary'}`} to="/auth/login">
            Get a new code<ArrowRight aria-hidden="true" />
          </Link>
        </div>
      </div>
    </AuthPageLayout>
  );
}

export default VerifyMagicLink;
