interface ErrorBannerProps {
  message: string;
}

export function ErrorBanner({ message }: ErrorBannerProps) {
  return (
    <div className="px-4 py-3 bg-red-50 dark:bg-red-900/20 border-b border-red-200 dark:border-red-800">
      <div className="flex items-start gap-3">
        <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
        <div>
          <p className="text-sm font-medium text-red-800 dark:text-red-200">{message}</p>
          <p className="text-xs text-red-600 dark:text-red-400 mt-1">
            {message.includes('driver') || message.includes('unavailable')
              ? 'Make sure the browser session is active and try again.'
              : message.includes('recording')
              ? 'Try stopping and restarting the recording.'
              : 'Please try again. If the problem persists, refresh the page.'}
          </p>
        </div>
      </div>
    </div>
  );
}

interface UnstableSelectorsBannerProps {
  lowConfidenceCount: number;
}

export function UnstableSelectorsBanner({ lowConfidenceCount }: UnstableSelectorsBannerProps) {
  if (lowConfidenceCount === 0) return null;

  return (
    <div className="px-4 py-3 bg-orange-50 dark:bg-orange-900/20 border-b border-orange-200 dark:border-orange-800">
      <div className="flex items-start gap-3">
        <svg className="w-5 h-5 text-orange-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
        <div>
          <p className="text-sm font-medium text-orange-800 dark:text-orange-200">
            {lowConfidenceCount} action{lowConfidenceCount !== 1 ? 's have' : ' has'} unstable selectors
          </p>
          <p className="text-xs text-orange-600 dark:text-orange-400 mt-1">
            Actions with red warning icons may fail when replayed. Click on them to edit the selectors and choose
            more stable alternatives.
          </p>
        </div>
      </div>
    </div>
  );
}
