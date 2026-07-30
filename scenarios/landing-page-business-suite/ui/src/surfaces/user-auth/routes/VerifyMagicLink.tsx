import { useEffect, useState, useCallback } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { Loader2, CheckCircle, XCircle, RefreshCw } from 'lucide-react';
import { AuthPageLayout } from '../../../shared/ui/AuthPageLayout';
import { verifyMagicLink, isApiError, VerifyMagicLinkResponse } from '../../../shared/api';
import { isRecord, safeParseJson } from '../../../shared/lib/utils';

// Session storage key for auth callback params (set in UserLogin)
const AUTH_CALLBACK_PARAMS_KEY = 'auth_callback_params';

// Allowed callback URL schemes for security
const ALLOWED_CALLBACK_SCHEMES = ['vrooli', 'http', 'https'];
const ALLOWED_LOCALHOST_HOSTS = ['localhost', '127.0.0.1'];

interface AuthCallbackParams {
  redirect_uri: string;
  app: string;
  state: string;
}

type VerifyStatus = 'verifying' | 'success' | 'error';

interface VerifyState {
  status: VerifyStatus;
  error?: string;
  errorCode?: 'expired' | 'used' | 'invalid' | 'network' | 'unknown';
}

function redirectBrowser(url: string): void {
  window.location.href = url;
}

function parseAuthCallbackParams(raw: string): AuthCallbackParams | null {
  const parsed = safeParseJson(raw);
  if (!isRecord(parsed)) {
    return null;
  }
  const redirect = parsed.redirect_uri;
  const app = parsed.app;
  const state = parsed.state;
  if (typeof redirect !== 'string' || typeof app !== 'string' || typeof state !== 'string') {
    return null;
  }
  return { redirect_uri: redirect, app, state };
}

/**
 * Validate that a callback URL is allowed for security.
 * Allows:
 * - vrooli:// scheme (for desktop apps)
 * - localhost/127.0.0.1 (for development)
 */
export function isAllowedCallbackUrl(urlString: string): boolean {
  try {
    const url = new URL(urlString);

    // Allow vrooli:// scheme for desktop deep links
    if (url.protocol === 'vrooli:') {
      return true;
    }

    // For http/https, only allow localhost (development)
    if (url.protocol === 'http:' || url.protocol === 'https:') {
      return ALLOWED_LOCALHOST_HOSTS.includes(url.hostname);
    }

    // Check if scheme is in allowed list
    const scheme = url.protocol.replace(':', '');
    return ALLOWED_CALLBACK_SCHEMES.includes(scheme);
  } catch {
    return false;
  }
}

/**
 * Build the redirect URL with tokens in the URL fragment.
 * Uses fragment (#) so tokens aren't sent to any servers in HTTP headers.
 */
function buildRedirectUrl(baseUrl: string, tokens: VerifyMagicLinkResponse, state: string): string {
  const url = new URL(baseUrl);

  // Build fragment with tokens
  const fragment = new URLSearchParams();
  fragment.set('access_token', tokens.access_token);
  fragment.set('refresh_token', tokens.refresh_token);
  fragment.set('expires_at', tokens.expires_at);
  fragment.set('token_type', tokens.token_type);
  if (state) {
    fragment.set('state', state);
  }

  url.hash = fragment.toString();
  return url.toString();
}

export function VerifyMagicLink({ redirectTo = redirectBrowser }: { redirectTo?: (url: string) => void }) {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [state, setState] = useState<VerifyState>({ status: 'verifying' });
  const [redirecting, setRedirecting] = useState(false);

  const token = searchParams.get('token');

  const performVerification = useCallback(async () => {
    if (!token) {
      setState({
        status: 'error',
        error: 'No verification token provided',
        errorCode: 'invalid',
      });
      return;
    }

    setState({ status: 'verifying' });

    try {
      const response = await verifyMagicLink(token);

      // Check for stored callback params
      const storedParams = sessionStorage.getItem(AUTH_CALLBACK_PARAMS_KEY);

      if (storedParams) {
        try {
          const params = parseAuthCallbackParams(storedParams);
          if (!params) {
            throw new Error('Invalid stored auth params');
          }

          // Clear stored params
          sessionStorage.removeItem(AUTH_CALLBACK_PARAMS_KEY);

          // Validate callback URL
          if (isAllowedCallbackUrl(params.redirect_uri)) {
            const redirectUrl = buildRedirectUrl(params.redirect_uri, response, params.state);
            setRedirecting(true);
            setState({ status: 'success' });

            // Redirect to the callback URL
            redirectTo(redirectUrl);
            return;
          } else {
            console.warn('Invalid callback URL rejected:', params.redirect_uri);
          }
        } catch (parseErr) {
          console.error('Failed to parse stored auth params:', parseErr);
        }
      }

      // No valid callback URL - show success and redirect to home
      setState({ status: 'success' });
      setTimeout(() => {
        navigate('/');
      }, 2000);

    } catch (err) {
      let errorMessage = 'Failed to verify login link. Please try again.';
      let errorCode: VerifyState['errorCode'] = 'unknown';

      if (isApiError(err)) {
        if (err.userMessage) {
          errorMessage = err.userMessage;
        }

        // Classify error type based on message content
        const msg = err.message.toLowerCase();
        if (msg.includes('expired')) {
          errorCode = 'expired';
        } else if (msg.includes('used') || msg.includes('already')) {
          errorCode = 'used';
        } else if (msg.includes('invalid')) {
          errorCode = 'invalid';
        } else if (err.type === 'network') {
          errorCode = 'network';
          errorMessage = 'Unable to reach the server. Please check your connection.';
        }
      }

      setState({
        status: 'error',
        error: errorMessage,
        errorCode,
      });
    }
  }, [token, navigate, redirectTo]);

  // Run verification on mount
  useEffect(() => {
    void performVerification();
  }, [performVerification]);

  // Verifying state
  if (state.status === 'verifying') {
    return (
      <AuthPageLayout>
        <div className="text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-blue-500/10 mb-6">
            <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Verifying...</h1>
          <p className="text-slate-400">Please wait while we verify your login link.</p>
        </div>
      </AuthPageLayout>
    );
  }

  // Success state
  if (state.status === 'success') {
    return (
      <AuthPageLayout>
        <div className="text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-emerald-500/10 mb-6">
            <CheckCircle className="w-8 h-8 text-emerald-400" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">
            {redirecting ? 'Signed in!' : 'Verification successful'}
          </h1>
          <p className="text-slate-400">
            {redirecting
              ? 'Redirecting you back to the app...'
              : 'You are now signed in. Redirecting...'}
          </p>
          {redirecting && (
            <div className="mt-6">
              <Loader2 className="w-5 h-5 text-slate-500 animate-spin mx-auto" />
            </div>
          )}
        </div>
      </AuthPageLayout>
    );
  }

  // Error state
  return (
    <AuthPageLayout>
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-red-500/10 mb-6">
          <XCircle className="w-8 h-8 text-red-400" />
        </div>
        <h1 className="text-2xl font-bold text-white mb-2">Verification failed</h1>
        <p className="text-slate-400 mb-6">{state.error}</p>

        {/* Show appropriate action based on error type */}
        {(state.errorCode === 'expired' || state.errorCode === 'used' || state.errorCode === 'invalid') && (
          <a
            href="/auth/login"
            className="
              inline-flex items-center gap-2
              px-4 py-2 rounded-lg
              bg-blue-600 hover:bg-blue-500
              text-white font-medium
              transition-colors
            "
          >
            <RefreshCw className="w-4 h-4" />
            Request new link
          </a>
        )}

        {state.errorCode === 'network' && (
          <button
            onClick={() => { void performVerification(); }}
            className="
              inline-flex items-center gap-2
              px-4 py-2 rounded-lg
              bg-blue-600 hover:bg-blue-500
              text-white font-medium
              transition-colors
            "
          >
            <RefreshCw className="w-4 h-4" />
            Try again
          </button>
        )}
      </div>
    </AuthPageLayout>
  );
}

export default VerifyMagicLink;
