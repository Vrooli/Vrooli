import { useState, useCallback } from 'react';
import { Mail, Loader2, Check, X, RefreshCw } from 'lucide-react';
import { useEntitlementStore, isValidEmail } from '@stores/entitlementStore';
import toast from 'react-hot-toast';

export function EmailInputSection() {
  const { userEmail, setUserEmail, clearUserEmail, isLoading, error } = useEntitlementStore();
  const [inputEmail, setInputEmail] = useState(userEmail);
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setLocalError(null);

    const trimmed = inputEmail.trim();
    if (!trimmed) {
      setLocalError('Email is required');
      return;
    }
    if (!isValidEmail(trimmed)) {
      setLocalError('Please enter a valid email address');
      return;
    }

    try {
      await setUserEmail(trimmed);
      toast.success('Email verified successfully');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to verify email');
    }
  }, [inputEmail, setUserEmail]);

  const handleClear = useCallback(async () => {
    try {
      await clearUserEmail();
      setInputEmail('');
      toast.success('Email cleared');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to clear email');
    }
  }, [clearUserEmail]);

  const displayError = localError || error;
  const hasStoredEmail = Boolean(userEmail);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <Mail size={20} className="text-flow-accent" />
          Subscription Email
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          Enter the email address associated with your subscription to unlock premium features.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="relative">
          <input
            type="email"
            value={inputEmail}
            onChange={(e) => {
              setInputEmail(e.target.value);
              setLocalError(null);
            }}
            placeholder="you@example.com"
            disabled={isLoading}
            className={`
              w-full px-4 py-3 pr-12 rounded-lg bg-gray-800 border
              text-surface placeholder-gray-500
              focus:outline-none focus:ring-2 focus:ring-flow-accent/50
              disabled:opacity-50 disabled:cursor-not-allowed
              ${displayError ? 'border-red-500' : 'border-gray-700 focus:border-flow-accent'}
            `}
          />
          {hasStoredEmail && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2">
              <Check size={18} className="text-green-400" />
            </div>
          )}
        </div>

        {displayError && (
          <p className="text-sm text-red-400 flex items-center gap-1">
            <X size={14} />
            {displayError}
          </p>
        )}

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={isLoading || !inputEmail.trim() || inputEmail.trim() === userEmail}
            className="
              flex items-center gap-2 px-4 py-2 rounded-lg
              bg-flow-accent hover:bg-blue-600
              text-white font-medium
              disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors
            "
          >
            {isLoading ? (
              <>
                <Loader2 size={16} className="animate-spin" />
                Verifying...
              </>
            ) : (
              <>
                <RefreshCw size={16} />
                {hasStoredEmail ? 'Update Email' : 'Verify Subscription'}
              </>
            )}
          </button>

          {hasStoredEmail && (
            <button
              type="button"
              onClick={handleClear}
              disabled={isLoading}
              className="
                flex items-center gap-2 px-4 py-2 rounded-lg
                bg-gray-700 hover:bg-gray-600
                text-gray-300 font-medium
                disabled:opacity-50 disabled:cursor-not-allowed
                transition-colors
              "
            >
              <X size={16} />
              Clear Email
            </button>
          )}
        </div>
      </form>
    </div>
  );
}

export default EmailInputSection;
