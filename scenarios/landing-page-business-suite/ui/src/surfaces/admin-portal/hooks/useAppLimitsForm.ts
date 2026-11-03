import { useState, useEffect, useCallback, useMemo } from 'react';
import type { TierLimit } from '../../../shared/api';
import {
  getEditKey,
  parseEditedValue,
  buildTierLimitUpdate,
} from '../../../shared/lib/tierUtils';
import {
  APP_OPTIONS,
  DEFAULT_NEW_LIMIT,
  validateNewLimitForm,
  buildCreateLimitPayload,
  collectLimitKeys,
  getAppMeterKind,
  fetchAppLimits,
  saveTierLimit,
  createNewTierLimit,
  removeTierLimit,
  type NewLimitFormState,
} from '../services/appLimits.service';

export interface UseAppLimitsFormReturn {
  // Data state
  selectedApp: string;
  limits: Record<string, TierLimit[]>;
  editedValues: Record<string, string>;
  newLimit: NewLimitFormState;
  limitKeys: Set<string>;

  // UI state
  loading: boolean;
  saving: string | null;
  showAddLimit: boolean;

  // Actions
  setSelectedApp: (app: string) => void;
  setEditedValue: (editKey: string, value: string) => void;
  setNewLimit: (update: Partial<NewLimitFormState>) => void;
  setShowAddLimit: (show: boolean) => void;
  handleSave: (tierID: string, limit: TierLimit) => Promise<{ success: boolean; message: string }>;
  handleAddLimit: () => Promise<{ success: boolean; message: string }>;
  handleDeleteLimit: (tierID: string, limitKey: string) => Promise<{ success: boolean; message: string }>;
  refreshLimits: () => Promise<void>;
  resetNewLimitForm: () => void;
  clearEditedValue: (editKey: string) => void;
  getEditedOrDisplayValue: (editKey: string, limit: TierLimit) => string;
  isEdited: (editKey: string) => boolean;
}

/**
 * Reactive hook for app limits form management
 *
 * Provides state and handlers for:
 * - Loading app limits by app bundle key
 * - Editing limit values inline
 * - Creating new limits
 * - Deleting limits
 */
export function useAppLimitsForm(): UseAppLimitsFormReturn {
  // Data state
  const [selectedApp, setSelectedApp] = useState<string>(APP_OPTIONS[0].value);
  const [limits, setLimits] = useState<Record<string, TierLimit[]>>({});
  const [editedValues, setEditedValues] = useState<Record<string, string>>({});
  const [newLimit, setNewLimitState] = useState<NewLimitFormState>(DEFAULT_NEW_LIMIT);

  // UI state
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [showAddLimit, setShowAddLimit] = useState(false);

  /**
   * Fetch limits from API
   */
  const refreshLimits = useCallback(async () => {
    setLoading(true);
    try {
      const fetchedLimits = await fetchAppLimits(selectedApp);
      setLimits(fetchedLimits);
      setEditedValues({});
    } finally {
      setLoading(false);
    }
  }, [selectedApp]);

  // Initial load and refresh when app changes
  useEffect(() => {
    refreshLimits().catch(() => {
      // Swallow error - component will show error state via limits being empty
    });
  }, [refreshLimits]);

  /**
   * Update a single edited value
   */
  const setEditedValue = useCallback((editKey: string, value: string) => {
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
      Reflect.deleteProperty(next, editKey);
      return next;
    });
  }, []);

  /**
   * Update new limit form
   */
  const setNewLimit = useCallback((update: Partial<NewLimitFormState>) => {
    setNewLimitState((prev) => ({ ...prev, ...update }));
  }, []);

  /**
   * Reset new limit form to defaults
   */
  const resetNewLimitForm = useCallback(() => {
    setNewLimitState(DEFAULT_NEW_LIMIT);
  }, []);

  /**
   * Get the edited value or display value for a limit
   */
  const getEditedOrDisplayValue = useCallback(
    (editKey: string, limit: TierLimit): string => {
      if (editedValues[editKey] !== undefined) {
        return editedValues[editKey];
      }
      const isUnlimited = limit.limit_value < 0;
      if (isUnlimited) {
        return 'unlimited';
      }
      const kind = getAppMeterKind(limit);
      if (kind === 'count_based') return String(limit.limit_value);
      if (kind === null) return '';
      return (limit.display_dollars ?? limit.limit_value / limit.cost_multiplier / 100).toFixed(2);
    },
    [editedValues]
  );

  /**
   * Check if a value has been edited
   */
  const isEdited = useCallback(
    (editKey: string): boolean => {
      return editedValues[editKey] !== undefined;
    },
    [editedValues]
  );

  /**
   * Save an edited limit value
   */
  const handleSave = useCallback(
    async (tierID: string, limit: TierLimit): Promise<{ success: boolean; message: string }> => {
      const editKey = getEditKey(tierID, limit.limit_key);
      const editedValue = editedValues[editKey];

      if (editedValue === undefined) {
        return { success: false, message: 'No changes to save' };
      }

      const kind = getAppMeterKind(limit);
      if (kind === null) return { success: false, message: 'This limit needs an explicit meter kind before editing' };
      const parsedValue = parseEditedValue(editedValue);
      if (parsedValue === null) {
        return { success: false, message: kind === 'count_based' ? 'Please enter a non-negative whole count' : 'Please enter a valid dollar amount' };
      }
      const numeric = Number(editedValue.trim());
      if (kind === 'count_based' && !('isUnlimited' in parsedValue) && (!Number.isSafeInteger(numeric) || numeric < 0)) {
        return { success: false, message: 'Please enter a non-negative whole count' };
      }

      try {
        setSaving(editKey);
        const update = 'isUnlimited' in parsedValue ? { is_unlimited: true } : kind === 'count_based' ? { limit_value: numeric } : buildTierLimitUpdate(parsedValue);
        await saveTierLimit(tierID, limit.limit_key, update, selectedApp);

        clearEditedValue(editKey);
        await refreshLimits();

        return {
          success: true,
          message: `Limit for ${tierID}/${limit.limit_key} updated`,
        };
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to update limit',
        };
      } finally {
        setSaving(null);
      }
    },
    [editedValues, selectedApp, clearEditedValue, refreshLimits]
  );

  /**
   * Create a new limit
   */
  const handleAddLimit = useCallback(async (): Promise<{ success: boolean; message: string }> => {
    const validationError = validateNewLimitForm(newLimit);
    if (validationError) {
      return { success: false, message: validationError };
    }

    try {
      setSaving('new');
      const payload = buildCreateLimitPayload(newLimit, selectedApp);
      await createNewTierLimit(payload);

      setShowAddLimit(false);
      resetNewLimitForm();
      await refreshLimits();

      return { success: true, message: 'New limit created' };
    } catch (error) {
      return {
        success: false,
        message: error instanceof Error ? error.message : 'Failed to create limit',
      };
    } finally {
      setSaving(null);
    }
  }, [newLimit, selectedApp, resetNewLimitForm, refreshLimits]);

  /**
   * Delete a limit
   */
  const handleDeleteLimit = useCallback(
    async (tierID: string, limitKey: string): Promise<{ success: boolean; message: string }> => {
      try {
        setSaving(`delete:${tierID}:${limitKey}`);
        await removeTierLimit(tierID, limitKey, selectedApp);
        await refreshLimits();

        return { success: true, message: 'Limit deleted' };
      } catch (error) {
        return {
          success: false,
          message: error instanceof Error ? error.message : 'Failed to delete limit',
        };
      } finally {
        setSaving(null);
      }
    },
    [selectedApp, refreshLimits]
  );

  // Computed values
  const limitKeys = useMemo(() => collectLimitKeys(limits), [limits]);

  return {
    // Data state
    selectedApp,
    limits,
    editedValues,
    newLimit,
    limitKeys,

    // UI state
    loading,
    saving,
    showAddLimit,

    // Actions
    setSelectedApp,
    setEditedValue,
    setNewLimit,
    setShowAddLimit,
    handleSave,
    handleAddLimit,
    handleDeleteLimit,
    refreshLimits,
    resetNewLimitForm,
    clearEditedValue,
    getEditedOrDisplayValue,
    isEdited,
  };
}
