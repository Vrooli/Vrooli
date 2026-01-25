import { useCallback } from 'react';
import { LogIn, LogOut, Loader2, User, Check, AlertCircle, Crown, ArrowUpRight, ExternalLink } from 'lucide-react';
import { useAuthStore, useIsAuthenticated, useAuthUser, useAuthLoading } from '@stores/authStore';
import { useEntitlementStore } from '@stores/entitlementStore';
import toast from 'react-hot-toast';

// Get landing page URL from environment or use default
const landingPageEnv = (import.meta.env as { VITE_LANDING_PAGE_URL?: unknown }).VITE_LANDING_PAGE_URL;
const LANDING_PAGE_URL =
  typeof landingPageEnv === 'string' && landingPageEnv.length > 0
    ? landingPageEnv
    : 'https://vrooli.com';

// Check if running in desktop environment
function isDesktopEnvironment(): boolean {
  return typeof window !== 'undefined' && typeof window.desktop?.auth !== 'undefined';
}

export function AuthSection() {
  const { signIn, signOut, error: authError } = useAuthStore();
  const isAuthenticated = useIsAuthenticated();
  const user = useAuthUser();
  const isLoading = useAuthLoading();
  const { status, setUserEmail } = useEntitlementStore();

  // Show "Get Subscription" link when status is inactive
  const showGetSubscription = status?.status === 'inactive' || status?.status === 'canceled';

  // Build checkout URL with email pre-filled
  const checkoutUrl = user?.email
    ? `${LANDING_PAGE_URL}/checkout?plan=pro&email=${encodeURIComponent(user.email)}`
    : `${LANDING_PAGE_URL}/checkout?plan=pro`;

  const handleSignIn = useCallback(async () => {
    try {
      await signIn();
      if (isDesktopEnvironment()) {
        toast.success('Opening browser for sign in...');
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to sign in');
    }
  }, [signIn]);

  const handleSignOut = useCallback(async () => {
    try {
      await signOut();
      toast.success('Signed out successfully');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to sign out');
    }
  }, [signOut]);

  // Sync user email with entitlement store when authenticated
  const syncEntitlements = useCallback(async () => {
    if (user?.email) {
      try {
        await setUserEmail(user.email);
        toast.success('Subscription status updated');
      } catch (_err) {
        toast.error('Failed to sync subscription');
      }
    }
  }, [user?.email, setUserEmail]);

  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-4">
        <div>
          <h3 className="text-lg font-medium text-surface flex items-center gap-2">
            <User size={20} className="text-flow-accent" />
            Vrooli Account
          </h3>
          <p className="text-sm text-gray-400 mt-1">
            Sign in to access your subscription and premium features.
          </p>
        </div>
        <div className="flex items-center gap-2 py-4">
          <Loader2 size={16} className="animate-spin text-gray-400" />
          <span className="text-gray-400">Checking authentication...</span>
        </div>
      </div>
    );
  }

  // Authenticated state
  if (isAuthenticated && user) {
    return (
      <div className="space-y-4">
        <div>
          <h3 className="text-lg font-medium text-surface flex items-center gap-2">
            <User size={20} className="text-flow-accent" />
            Vrooli Account
          </h3>
          <p className="text-sm text-gray-400 mt-1">
            Manage your account and subscription.
          </p>
        </div>

        {/* Signed in state */}
        <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-flow-accent/20 flex items-center justify-center">
                <User size={20} className="text-flow-accent" />
              </div>
              <div>
                <p className="text-surface font-medium">{user.email}</p>
                <div className="flex items-center gap-1.5 text-xs">
                  {user.emailVerified ? (
                    <>
                      <Check size={12} className="text-green-400" />
                      <span className="text-green-400">Verified</span>
                    </>
                  ) : (
                    <>
                      <AlertCircle size={12} className="text-amber-400" />
                      <span className="text-amber-400">Not verified</span>
                    </>
                  )}
                </div>
              </div>
            </div>

            <button
              onClick={handleSignOut}
              className="
                flex items-center gap-2 px-3 py-1.5 rounded-lg
                bg-gray-700 hover:bg-gray-600
                text-gray-300 text-sm font-medium
                transition-colors
              "
            >
              <LogOut size={14} />
              Sign Out
            </button>
          </div>

          {/* Subscription status */}
          {status && (
            <div className="pt-3 border-t border-gray-700">
              {status.is_active ? (
                <div className="flex items-center gap-2 text-sm text-green-400">
                  <Check size={14} />
                  <span>Active {status.tier !== 'free' ? status.tier.charAt(0).toUpperCase() + status.tier.slice(1) : ''} subscription</span>
                </div>
              ) : showGetSubscription ? (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-400">No active subscription</span>
                  <a
                    href={checkoutUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1.5 text-sm text-purple-400 hover:text-purple-300 transition-colors"
                  >
                    <Crown size={14} />
                    Get Subscription
                    <ArrowUpRight size={12} />
                  </a>
                </div>
              ) : null}
            </div>
          )}

          {/* Sync button if entitlements not yet fetched */}
          {!status && user.email && (
            <div className="pt-3 border-t border-gray-700">
              <button
                onClick={syncEntitlements}
                className="
                  text-sm text-blue-400 hover:text-blue-300
                  transition-colors
                "
              >
                Check subscription status
              </button>
            </div>
          )}
        </div>

        {authError && (
          <p className="text-sm text-red-400 flex items-center gap-1">
            <AlertCircle size={14} />
            {authError}
          </p>
        )}
      </div>
    );
  }

  // Not authenticated state
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <User size={20} className="text-flow-accent" />
          Vrooli Account
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          Sign in with your Vrooli account to access your subscription and premium features.
        </p>
      </div>

      <div className="p-4 rounded-lg bg-gray-800/50 border border-gray-700">
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
          <div className="flex-1">
            <p className="text-surface font-medium mb-1">
              Sign in to unlock premium features
            </p>
            <p className="text-sm text-gray-400">
              {isDesktopEnvironment()
                ? 'Click the button to sign in via your browser.'
                : 'You will be redirected to Vrooli to sign in.'}
            </p>
          </div>

          <button
            onClick={handleSignIn}
            disabled={isLoading}
            className="
              flex items-center gap-2 px-4 py-2.5 rounded-lg
              bg-flow-accent hover:bg-blue-600
              text-white font-medium
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors whitespace-nowrap
            "
          >
            {isDesktopEnvironment() ? (
              <>
                <ExternalLink size={18} />
                Sign In with Vrooli
              </>
            ) : (
              <>
                <LogIn size={18} />
                Sign In
              </>
            )}
          </button>
        </div>
      </div>

      {authError && (
        <p className="text-sm text-red-400 flex items-center gap-1">
          <AlertCircle size={14} />
          {authError}
        </p>
      )}

      {/* Link to create account */}
      <p className="text-sm text-gray-500">
        Don't have an account?{' '}
        <a
          href={`${LANDING_PAGE_URL}/checkout?plan=pro`}
          target="_blank"
          rel="noopener noreferrer"
          className="text-blue-400 hover:text-blue-300 transition-colors"
        >
          Get started with a subscription
        </a>
      </p>
    </div>
  );
}

export default AuthSection;
