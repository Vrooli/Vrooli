import { useEffect, useState } from 'react';

interface ConnectionStatusBannerProps {
  isStale: boolean;
  lastSuccessfulFetch: Date | null;
  onRefresh: () => void;
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

export function ConnectionStatusBanner({ isStale, lastSuccessfulFetch, onRefresh }: ConnectionStatusBannerProps) {
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
    <div data-sm-style="sm-style-53f1cf7a58">
      <span>Data may be outdated. Last successful update: {timeAgo}. Retrying...</span>
      <button
        onClick={onRefresh}
        data-sm-style="sm-style-db72597db4"
      >
        Refresh Now
      </button>
    </div>
  );
}
