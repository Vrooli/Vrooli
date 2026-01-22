import { useCallback, useEffect, useState } from 'react';
import type { TierLimit } from '../../../shared/api';
import {
  fetchAllTierLimits,
  saveTierLimit,
  getEditKey,
  getTierLabel,
  getTierColor,
  parseEditedValue,
  buildTierLimitUpdate,
  getDisplayValue,
  findAICreditsLimit,
  calculateDoubledLimits,
  isUnlimitedValue,
  DEFAULT_TIER_VALUES,
  TIER_OPTIONS,
} from '../services/tiers.service';

/**
 * Toast message to display
 */
export interface ToastMessage {
  type: 'success' | 'error';
  message: string;
}

/**
 * Return type for the useTierLimitsForm hook
 */
export interface UseTierLimitsFormReturn {
  /** Limits data grouped by tier ID */
  limits: Record<string, TierLimit[]>;
  /** Whether limits are loading */
  loading: boolean;
  /** Currently saving edit key (tier:limit_key) */
  saving: string | null;
  /** Edited values map (edit key -> value string) */
  editedValues: Record<string, string>;
  /** Toast messages to display */
  toasts: ToastMessage[];

  /** Fetch limits from API */
  fetchLimits: () => Promise<void>;
  /** Update an edited value */
  updateEditedValue: (editKey: string, value: string) => void;
  /** Clear an edited value */
  clearEditedValue: (editKey: string) => void;
  /** Save a limit */
  handleSave: (tierID: string, limit: TierLimit) => Promise<void>;
  /** Reset all tiers to default values */
  resetToDefaults: () => void;
  /** Double all current limits */
  doubleAllLimits: () => void;
  /** Clear toasts */
  clearToasts: () => void;

  /** Get edit key for tier and limit key */
  getEditKey: typeof getEditKey;
  /** Get tier display label */
  getTierLabel: typeof getTierLabel;
  /** Get tier color class */
  getTierColor: typeof getTierColor;
  /** Get display value for a limit */
  getDisplayValue: typeof getDisplayValue;
  /** Find AI credits limit for a tier */
  findAICreditsLimit: typeof findAICreditsLimit;
  /** Check if limit is unlimited */
  isUnlimitedValue: typeof isUnlimitedValue;
  /** Tier options */
  TIER_OPTIONS: typeof TIER_OPTIONS;
}

/**
 * Custom hook for managing tier limits form
 *
 * Encapsulates all state management for the tier limits settings page,
 * including loading, saving, and validation.
 *
 * @returns Object containing form state and handlers
 */
export function useTierLimitsForm(): UseTierLimitsFormReturn {
  const [limits, setLimits] = useState<Record<string, TierLimit[]>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [editedValues, setEditedValues] = useState<Record<string, string>>({});
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  /**
   * Add a toast message
   */
  const addToast = useCallback((toast: ToastMessage) => {
    setToasts((prev) => [...prev, toast]);
  }, []);

  /**
   * Clear all toasts
   */
  const clearToasts = useCallback(() => {
    setToasts([]);
  }, []);

  /**
   * Fetch limits from API
   */
  const fetchLimits = useCallback(async () => {
    try {
      setLoading(true);
      const data = await fetchAllTierLimits();
      setLimits(data);
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to load tier limits',
      });
    } finally {
      setLoading(false);
    }
  }, [addToast]);

  // Load limits on mount
  useEffect(() => {
    fetchLimits();
  }, [fetchLimits]);

  /**
   * Update an edited value
   */
  const updateEditedValue = useCallback((editKey: string, value: string) => {
    setEditedValues((prev) => ({
      ...prev,
      [editKey]: value.toLowerCase(),
    }));
  }, []);

  /**
   * Clear an edited value
   */
  const clearEditedValue = useCallback((editKey: string) => {
    setEditedValues((prev) => {
      const next = { ...prev };
      delete next[editKey];
      return next;
    });
  }, []);

  /**
   * Save a tier limit
   */
  const handleSave = useCallback(
    async (tierID: string, limit: TierLimit) => {
      const editKey = getEditKey(tierID, limit.limit_key);
      const editedValue = editedValues[editKey];

      if (editedValue === undefined) return;

      const parsedValue = parseEditedValue(editedValue);
      if (!parsedValue) {
        addToast({ type: 'error', message: 'Please enter a valid dollar amount' });
        return;
      }

      try {
        setSaving(editKey);
        const update = buildTierLimitUpdate(parsedValue);
        await saveTierLimit(tierID, limit.limit_key, update);
        addToast({ type: 'success', message: `Limit for ${tierID}/${limit.limit_key} updated` });

        // Clear edited value and refresh
        clearEditedValue(editKey);
        await fetchLimits();
      } catch (error) {
        addToast({
          type: 'error',
          message: error instanceof Error ? error.message : 'Failed to update limit',
        });
      } finally {
        setSaving(null);
      }
    },
    [editedValues, addToast, clearEditedValue, fetchLimits]
  );

  /**
   * Reset all tiers to default values
   */
  const resetToDefaults = useCallback(() => {
    setEditedValues(DEFAULT_TIER_VALUES);
  }, []);

  /**
   * Double all current limits
   */
  const doubleAllLimits = useCallback(() => {
    const doubled = calculateDoubledLimits(limits);
    setEditedValues((prev) => ({ ...prev, ...doubled }));
  }, [limits]);

  return {
    limits,
    loading,
    saving,
    editedValues,
    toasts,
    fetchLimits,
    updateEditedValue,
    clearEditedValue,
    handleSave,
    resetToDefaults,
    doubleAllLimits,
    clearToasts,
    getEditKey,
    getTierLabel,
    getTierColor,
    getDisplayValue,
    findAICreditsLimit,
    isUnlimitedValue,
    TIER_OPTIONS,
  };
}
