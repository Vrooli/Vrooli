import { useEffect, useState } from 'react';
import { useAppState } from '../../contexts/AppStateContext';
import { Clock } from 'lucide-react';

export function RefreshCountdown() {
  const { cachedSettings } = useAppState();
  const refreshInterval = (cachedSettings?.processor?.refresh_interval ?? 10) * 1000; // Convert to ms

  const [secondsRemaining, setSecondsRemaining] = useState(Math.floor(refreshInterval / 1000));

  useEffect(() => {
    // Reset countdown when interval changes
    setSecondsRemaining(Math.floor(refreshInterval / 1000));

    const interval = setInterval(() => {
      setSecondsRemaining((prev) => {
        if (prev <= 1) {
          return Math.floor(refreshInterval / 1000); // Reset
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [refreshInterval]);

  return (
    <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
      <Clock className="h-3.5 w-3.5" />
      <span className="tabular-nums">{secondsRemaining}s</span>
    </div>
  );
}
