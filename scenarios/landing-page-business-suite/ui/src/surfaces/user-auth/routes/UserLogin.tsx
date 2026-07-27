import { useState, useCallback, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Mail, Loader2, ArrowRight, CheckCircle } from 'lucide-react';
import { AuthPageLayout } from '../../../shared/ui/AuthPageLayout';
import { requestMagicLink, isApiError } from '../../../shared/api';
import { useToast } from '../../../shared/ui/useToast';

// Session storage key for auth callback params
const AUTH_CALLBACK_PARAMS_KEY = 'auth_callback_params';

interface AuthCallbackParams {
  redirect_uri: string;
  app: string;
  state: string;
}

function isValidEmail(email: string): boolean {
  const trimmed = email.trim();
  if (!trimmed) return false;
  const atIndex = trimmed.indexOf('@');
  if (atIndex < 1) return false;
  const domain = trimmed.slice(atIndex + 1);
  return domain.length > 0 && domain.includes('.') && !domain.endsWith('.');
}

export function UserLogin() {
  const [searchParams] = useSearchParams();
  const { addToast } = useToast();
  const [email, setEmail] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [appName, setAppName] = useState<string>('');

  // Extract and store callback params from URL
  useEffect(() => {
    const redirectUri = searchParams.get('redirect_uri');
    const app = searchParams.get('app') || 'Vrooli';
    const state = searchParams.get('state') || '';

    setAppName(app);

    // Store callback params in sessionStorage for use after verification
    if (redirectUri) {
      const params: AuthCallbackParams = {
        redirect_uri: redirectUri,
        app,
        state,
      };
      sessionStorage.setItem(AUTH_CALLBACK_PARAMS_KEY, JSON.stringify(params));
    }
  }, [searchParams]);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const trimmedEmail = email.trim().toLowerCase();
    if (!trimmedEmail) {
      setError('Email is required');
      return;
    }
    if (!isValidEmail(trimmedEmail)) {
      setError('Please enter a valid email address');
      return;
    }

    setIsLoading(true);
    try {
      await requestMagicLink(trimmedEmail);
      setSuccess(true);
      addToast({
        type: 'success',
        title: 'Check your email',
        message: `We sent a login link to ${trimmedEmail}`,
      });
    } catch (err) {
      if (isApiError(err, 'rate_limited')) {
        setError('Too many login attempts. Please wait a moment and try again.');
      } else if (isApiError(err, 'validation')) {
        setError('Please enter a valid email address.');
      } else if (isApiError(err, 'network')) {
        setError('Unable to reach the server. Please check your connection.');
      } else {
        setError('Something went wrong. Please try again.');
      }
    } finally {
      setIsLoading(false);
    }
  }, [email, addToast]);

  // Success state - show check email message
  if (success) {
    return (
      <AuthPageLayout>
        <div className="text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-emerald-500/10 mb-6">
            <CheckCircle className="w-8 h-8 text-emerald-400" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Check your email</h1>
          <p className="text-slate-400 mb-6">
            We sent a login link to <span className="text-white font-medium">{email}</span>
          </p>
          <p className="text-sm text-slate-500 mb-8">
            Click the link in the email to sign in{appName ? ` to ${appName}` : ''}.
            The link expires in 15 minutes.
          </p>
          <button
            onClick={() => {
              setSuccess(false);
              setEmail('');
            }}
            className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
          >
            Use a different email
          </button>
        </div>
      </AuthPageLayout>
    );
  }

  return (
    <AuthPageLayout
      title="Sign In"
      subtitle={appName ? `Sign in to access ${appName}` : 'Sign in with your email'}
    >
      <form onSubmit={(event) => { void handleSubmit(event); }} className="space-y-6">
        <div>
          <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
            Email address
          </label>
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Mail className="h-5 w-5 text-slate-500" />
            </div>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                setError(null);
              }}
              placeholder="you@example.com"
              disabled={isLoading}
              autoComplete="email"
              autoFocus
              data-testid="email-input"
              className={`
                w-full pl-10 pr-4 py-3 rounded-lg
                bg-slate-900/50 border
                text-white placeholder-slate-500
                focus:outline-none focus:ring-2 focus:ring-blue-500/50
                disabled:opacity-50 disabled:cursor-not-allowed
                transition-colors
                ${error ? 'border-red-500' : 'border-slate-600 focus:border-blue-500'}
              `}
            />
          </div>
          {error && (
            <p className="mt-2 text-sm text-red-400">{error}</p>
          )}
        </div>

        <button
          type="submit"
          disabled={isLoading || !email.trim()}
          data-testid="submit-button"
          className="
            w-full flex items-center justify-center gap-2
            px-4 py-3 rounded-lg
            bg-blue-600 hover:bg-blue-500
            text-white font-medium
            disabled:opacity-50 disabled:cursor-not-allowed
            transition-colors
          "
        >
          {isLoading ? (
            <>
              <Loader2 className="w-5 h-5 animate-spin" />
              Sending link...
            </>
          ) : (
            <>
              Continue with Email
              <ArrowRight className="w-5 h-5" />
            </>
          )}
        </button>

        <p className="text-xs text-center text-slate-500">
          We'll send you a magic link to sign in. No password needed.
        </p>
      </form>
    </AuthPageLayout>
  );
}

export default UserLogin;
