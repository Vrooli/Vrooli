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
    const id = setInterval(() => setTick(t => t + 1), 5000);
    return () => clearInterval(id);
  }, [isStale]);

  if (!isStale) return null;

  const timeAgo = lastSuccessfulFetch ? formatTimeAgo(lastSuccessfulFetch) : 'never';

  return (
    <div style={{
      background: 'var(--color-warning-bg, rgba(255, 193, 7, 0.15))',
      borderBottom: '1px solid var(--color-warning-border, rgba(255, 193, 7, 0.4))',
      padding: '0.5rem 1rem',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      gap: '1rem',
      fontSize: '0.875rem',
      color: 'var(--color-warning, #ffc107)',
    }}>
      <span>Data may be outdated. Last successful update: {timeAgo}. Retrying...</span>
      <button
        onClick={onRefresh}
        style={{
          background: 'var(--color-warning-bg-hover, rgba(255, 193, 7, 0.2))',
          border: '1px solid var(--color-warning-border, rgba(255, 193, 7, 0.5))',
          color: 'var(--color-warning, #ffc107)',
          padding: '0.25rem 0.75rem',
          borderRadius: '4px',
          cursor: 'pointer',
          fontSize: '0.8rem',
        }}
      >
        Refresh Now
      </button>
    </div>
  );
}
