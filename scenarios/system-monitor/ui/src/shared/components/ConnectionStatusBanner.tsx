import { RefreshCw, WifiOff } from 'lucide-react';
import { useEffect, useState } from 'react';

interface ConnectionStatusBannerProps {
  isStale: boolean;
  lastSuccessfulFetch: Date | null;
  onRefresh: () => void;
  retryIntervalSeconds?: number;
  retryAttempt?: number;
}

function formatTimeAgo(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 10) return 'just now';
  if (seconds < 60) return `${seconds} seconds ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
  const hours = Math.floor(minutes / 60);
  return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
}

export function ConnectionStatusBanner({ isStale, lastSuccessfulFetch, onRefresh, retryIntervalSeconds = 15, retryAttempt = 0 }: ConnectionStatusBannerProps) {
  const [, setTick] = useState(0);

  // Update relative time display every 5 seconds
  useEffect(() => {
    if (!isStale) return;
    const id = setInterval(() => { setTick(t => t + 1); }, 5000);
    return () => { clearInterval(id); };
  }, [isStale]);

  if (!isStale) return null;

  const timeAgo = lastSuccessfulFetch ? formatTimeAgo(lastSuccessfulFetch) : 'never';

  return (
    <div className="connection-status-banner" role="status" aria-live="polite">
      <div className="connection-status-banner__message">
        <WifiOff size={16} aria-hidden="true" />
        <span><strong>Showing the last reading.</strong> Last successful update: {timeAgo}. Retrying every {retryIntervalSeconds} seconds (attempt {retryAttempt}); reconnecting automatically.</span>
      </div>
      <button
        type="button"
        className="btn btn-sm connection-status-banner__action"
        onClick={onRefresh}
      >
        <RefreshCw size={14} aria-hidden="true" />
        Refresh Now
      </button>
    </div>
  );
}
