import { useState, useCallback } from 'react';
import type { BrowserProfile } from '@/domains/recording/types/types';
import type { UseProfileSettingsReturn } from '../settings';

export interface UseSessionPersistenceProps {
  /** Settings hook instance to coordinate with */
  settings: UseProfileSettingsReturn;
  /** Callback to save the profile */
  onSave: (profile: BrowserProfile) => Promise<void>;
}

export interface UseSessionPersistenceReturn {
  /** Whether save is in progress */
  saving: boolean;
  /** Error message from last save attempt */
  error: string | null;
  /** Whether there are unsaved changes (derived from settings.hasChanges) */
  isDirty: boolean;
  /** Save current profile, returns true on success */
  handleSave: () => Promise<boolean>;
  /** Reset settings to initial and clear error */
  reset: () => void;
  /** Clear the error state */
  clearError: () => void;
}

/**
 * Manages persistence state for session settings.
 *
 * Responsibilities:
 * - Track saving state
 * - Handle save errors
 * - Reflect dirty state from settings hook
 * - Coordinate save and reset operations
 */
export function useSessionPersistence({
  settings,
  onSave,
}: UseSessionPersistenceProps): UseSessionPersistenceReturn {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // isDirty is derived directly from settings
  const isDirty = settings.hasChanges;

  const handleSave = useCallback(async (): Promise<boolean> => {
    setSaving(true);
    setError(null);

    try {
      const profile = settings.getCurrentProfile();
      await onSave(profile);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save profile';
      setError(message);
      return false;
    } finally {
      setSaving(false);
    }
  }, [settings, onSave]);

  const reset = useCallback(() => {
    settings.resetToInitial();
    setError(null);
  }, [settings]);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    saving,
    error,
    isDirty,
    handleSave,
    reset,
    clearError,
  };
}
