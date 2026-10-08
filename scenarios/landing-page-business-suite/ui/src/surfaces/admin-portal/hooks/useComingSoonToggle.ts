import { useState, useCallback, useEffect, useRef } from 'react';
import { fetchBranding, toggleComingSoonMode } from '../services/waitlist.service';

export interface UseComingSoonToggleReturn {
  comingSoonEnabled: boolean;
  loading: boolean;
  error?: string;
  toggling: boolean;
  reload: () => void;
  handleToggle: () => Promise<{ success: boolean; message?: string }>;
}

/** Private admin workflow: branding is read from its owner, never public config. */
export function useComingSoonToggle(): UseComingSoonToggleReturn {
  const [enabled, setEnabled] = useState<boolean>();
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState(false);
  const [version, setVersion] = useState(0);
  const live = useRef(false);
  const pending = useRef(false);
  useEffect(() => {
    live.current = true;
    let current = true;
    setLoading(true); setEnabled(undefined); setError(undefined);
    void fetchBranding().then(branding => {
      if (current) setEnabled(branding.coming_soon_enabled === true);
    }).catch(() => { if (current) setError('Coming soon status is unavailable. Reload its configuration before changing it.'); })
      .finally(() => { if (current) setLoading(false); });
    return () => { current = false; live.current = false; };
  }, [version]);

  const handleToggle = useCallback(async (): Promise<{ success: boolean; message?: string }> => {
    if (enabled === undefined || loading || pending.current) return { success: false, message: 'Coming soon status is not ready.' };
    pending.current = true; setToggling(true); setError(undefined);
    try {
      const next = await toggleComingSoonMode(enabled);
      if (live.current) setEnabled(next);
      return { success: true };
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to toggle coming soon mode';
      if (live.current) { setEnabled(undefined); setError(message); }
      return { success: false, message };
    } finally {
      pending.current = false;
      if (live.current) setToggling(false);
    }
  }, [enabled, loading]);

  return { comingSoonEnabled: enabled === true, loading, error, toggling,
    reload: () => { if (!pending.current) setVersion(current => current + 1); }, handleToggle };
}
