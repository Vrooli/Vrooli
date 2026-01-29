import { useState, useCallback } from 'react';
import { toggleComingSoonMode } from '../services/waitlist.service';
import { useLandingVariant } from '../../../app/providers/useLandingVariant';

export interface UseComingSoonToggleReturn {
  comingSoonEnabled: boolean;
  toggling: boolean;
  handleToggle: () => Promise<{ success: boolean; message?: string }>;
}

/**
 * Hook for toggling coming soon mode from any admin component.
 * Reads state from the landing config and refreshes after toggle.
 */
export function useComingSoonToggle(): UseComingSoonToggleReturn {
  const { config, refresh } = useLandingVariant();
  const [toggling, setToggling] = useState(false);

  const comingSoonEnabled = Boolean(config?.branding?.coming_soon_enabled);

  const handleToggle = useCallback(async (): Promise<{ success: boolean; message?: string }> => {
    setToggling(true);
    try {
      await toggleComingSoonMode(comingSoonEnabled);
      // Refresh landing config to get updated state
      await refresh();
      return { success: true };
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to toggle coming soon mode';
      return { success: false, message };
    } finally {
      setToggling(false);
    }
  }, [comingSoonEnabled, refresh]);

  return {
    comingSoonEnabled,
    toggling,
    handleToggle,
  };
}
