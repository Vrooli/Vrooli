import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  listDownloadAppsAdmin,
  createDownloadAppAdmin,
  saveDownloadAppAdmin,
  deleteDownloadAppAdmin,
  type DownloadApp,
} from '../../../shared/api';
import {
  buildDefaultAppValues,
  deserializeApp,
  serializeApp,
  isFormDirty,
  computeDownloadHealthFromForms,
  type AppFormValues,
  type PlatformKey,
  type PlatformFormValues,
} from '../services/downloads.service';

/**
 * State for a single app form
 */
export interface AppFormState {
  key: string;
  values: AppFormValues;
  original: AppFormValues;
  saving: boolean;
  error?: string;
  isNew?: boolean;
  lastSavedAt?: number;
}

/**
 * Return type for the useDownloadsForm hook
 */
export interface UseDownloadsFormReturn {
  /** Array of app form states */
  forms: AppFormState[];
  /** Whether the forms are loading */
  loading: boolean;
  /** Global error message */
  error: string | null;
  /** Count of dirty (unsaved) forms */
  dirtyCount: number;
  /** Download health metrics */
  downloadHealth: ReturnType<typeof computeDownloadHealthFromForms>;

  /** Load/reload apps from the API */
  loadApps: () => Promise<void>;
  /** Handle field change for an app */
  handleFieldChange: (key: string, field: keyof AppFormValues, value: string | number | boolean) => void;
  /** Handle platform-specific field change */
  handlePlatformChange: (
    key: string,
    platformKey: PlatformKey,
    field: keyof PlatformFormValues,
    value: string | boolean
  ) => void;
  /** Add a new app form */
  handleAddApp: () => void;
  /** Reset an app form to original values */
  handleReset: (key: string) => void;
  /** Delete an app */
  handleDelete: (key: string) => Promise<void>;
  /** Save a single app */
  handleSave: (key: string) => Promise<void>;
  /** Save all dirty forms */
  handleSaveAll: () => Promise<void>;

  /** Whether a bulk save is in progress */
  savingAll: boolean;

  // Drag-and-drop state and handlers
  draggingKey: string | null;
  dragOverKey: string | null;
  handleDragStart: (key: string) => (e: React.DragEvent) => void;
  handleDragOver: (key: string) => (e: React.DragEvent) => void;
  handleDragLeave: () => void;
  handleDrop: (targetKey: string) => (e: React.DragEvent) => void;
  handleDragEnd: () => void;
}

/**
 * Custom hook for managing download app forms
 *
 * Encapsulates all state management for the download settings page,
 * including loading, saving, drag-and-drop reordering, and form validation.
 *
 * @returns Object containing form state and handlers
 */
export function useDownloadsForm(): UseDownloadsFormReturn {
  const [forms, setForms] = useState<AppFormState[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [savingAll, setSavingAll] = useState(false);

  // Drag-and-drop state
  const [draggingKey, setDraggingKey] = useState<string | null>(null);
  const [dragOverKey, setDragOverKey] = useState<string | null>(null);

  /**
   * Load apps from the API
   */
  const loadApps = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { apps } = await listDownloadAppsAdmin();
      const sorted = [...apps].sort((a, b) => (a.display_order ?? 0) - (b.display_order ?? 0));
      const nextForms = sorted.map((app: DownloadApp) => {
        const values = deserializeApp(app);
        return {
          key: app.app_key,
          values,
          original: values,
          saving: false,
        };
      });
      setForms(nextForms);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load download apps');
    } finally {
      setLoading(false);
    }
  }, []);

  // Load apps on mount
  useEffect(() => {
    void loadApps();
  }, [loadApps]);

  /**
   * Handle field change for an app
   */
  const handleFieldChange = useCallback(
    (key: string, field: keyof AppFormValues, value: string | number | boolean) => {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? {
                ...form,
                values: {
                  ...form.values,
                  [field]: field === 'displayOrder' ? Number(value) : value,
                },
              }
            : form
        )
      );
    },
    []
  );

  /**
   * Handle platform-specific field change
   */
  const handlePlatformChange = useCallback(
    (
      key: string,
      platformKey: PlatformKey,
      field: keyof PlatformFormValues,
      value: string | boolean
    ) => {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? {
                ...form,
                values: {
                  ...form.values,
                  platforms: {
                    ...form.values.platforms,
                    [platformKey]: {
                      ...form.values.platforms[platformKey],
                      [field]: value,
                    },
                  },
                },
              }
            : form
        )
      );
    },
    []
  );

  /**
   * Add a new app form
   */
  const handleAddApp = useCallback(() => {
    const tempKey = `app-${String(Date.now())}`;
    const nextValues = {
      ...buildDefaultAppValues(tempKey),
      name: 'New Bundle App',
    };
    setForms((prev) => [
      ...prev,
      {
        key: tempKey,
        values: nextValues,
        original: nextValues,
        saving: false,
        isNew: true,
      },
    ]);
  }, []);

  /**
   * Reset an app form to original values
   */
  const handleReset = useCallback((key: string) => {
    setForms((prev) =>
      prev.map((form) =>
        form.key === key ? { ...form, values: form.original, error: undefined } : form
      )
    );
  }, []);

  /**
   * Delete an app
   */
  const handleDelete = useCallback(async (key: string) => {
    const target = forms.find((form) => form.key === key);
    if (!target) return;

    // For new unsaved apps, just remove from local state
    if (target.isNew) {
      setForms((prev) => prev.filter((form) => form.key !== key));
      return;
    }

    const confirmed = window.confirm(
      `Are you sure you want to delete "${target.values.name || target.values.appKey}"? This action cannot be undone.`
    );
    if (!confirmed) return;

    setForms((prev) =>
      prev.map((form) =>
        form.key === key ? { ...form, saving: true, error: undefined } : form
      )
    );

    try {
      await deleteDownloadAppAdmin(target.values.appKey);
      setForms((prev) => prev.filter((form) => form.key !== key));
    } catch (err) {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? { ...form, saving: false, error: err instanceof Error ? err.message : 'Failed to delete app' }
            : form
        )
      );
    }
  }, [forms]);

  /**
   * Save a single app
   */
  const handleSave = useCallback(async (key: string) => {
    const target = forms.find((form) => form.key === key);
    if (!target) return;

    if (!target.values.appKey.trim()) {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key ? { ...form, error: 'App key is required before saving.' } : form
        )
      );
      return;
    }

    setForms((prev) =>
      prev.map((form) =>
        form.key === key ? { ...form, saving: true, error: undefined } : form
      )
    );

    try {
      const payload = serializeApp(target.values);
      const response = target.isNew
        ? await createDownloadAppAdmin(payload)
        : await saveDownloadAppAdmin(target.values.appKey, payload);
      const updatedValues = deserializeApp(response);
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? {
                ...form,
                key: response.app_key,
                values: updatedValues,
                original: updatedValues,
                saving: false,
                isNew: false,
                error: undefined,
                lastSavedAt: Date.now(),
              }
            : form
        )
      );
    } catch (err) {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? {
                ...form,
                saving: false,
                error: err instanceof Error ? err.message : 'Failed to save app',
              }
            : form
        )
      );
    }
  }, [forms]);

  /**
   * Save all dirty forms
   */
  const handleSaveAll = useCallback(async () => {
    const dirtyForms = forms.filter((form) => isFormDirty(form.values, form.original));
    if (dirtyForms.length === 0) return;

    setSavingAll(true);

    // Mark all dirty forms as saving
    setForms((prev) =>
      prev.map((form) =>
        isFormDirty(form.values, form.original) ? { ...form, saving: true, error: undefined } : form
      )
    );

    // Save all dirty forms in parallel
    const results = await Promise.allSettled(
      dirtyForms.map(async (form) => {
        const payload = serializeApp(form.values);
        const response = form.isNew
          ? await createDownloadAppAdmin(payload)
          : await saveDownloadAppAdmin(form.values.appKey, payload);
        return { key: form.key, response };
      })
    );

    // Update forms based on results
    setForms((prev) =>
      prev.map((form) => {
        const result = results.find((r, i) => dirtyForms[i]?.key === form.key);
        if (!result) return form;

        if (result.status === 'fulfilled') {
          const updatedValues = deserializeApp(result.value.response);
          return {
            ...form,
            key: result.value.response.app_key,
            values: updatedValues,
            original: updatedValues,
            saving: false,
            isNew: false,
            error: undefined,
            lastSavedAt: Date.now(),
          };
        } else {
          return {
            ...form,
            saving: false,
            error: result.reason instanceof Error ? result.reason.message : 'Failed to save app',
          };
        }
      })
    );

    setSavingAll(false);
  }, [forms]);

  // Drag-and-drop handlers
  const handleDragStart = useCallback(
    (key: string) => (e: React.DragEvent) => {
      setDraggingKey(key);
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', key);
    },
    []
  );

  const handleDragOver = useCallback(
    (key: string) => (e: React.DragEvent) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      if (draggingKey && key !== draggingKey) {
        setDragOverKey(key);
      }
    },
    [draggingKey]
  );

  const handleDragLeave = useCallback(() => {
    setDragOverKey(null);
  }, []);

  const handleDrop = useCallback(
    (targetKey: string) => (e: React.DragEvent) => {
      e.preventDefault();
      const sourceKey = e.dataTransfer.getData('text/plain');

      if (!sourceKey || sourceKey === targetKey) {
        setDraggingKey(null);
        setDragOverKey(null);
        return;
      }

      setForms((prev) => {
        const sourceIndex = prev.findIndex((f) => f.key === sourceKey);
        const targetIndex = prev.findIndex((f) => f.key === targetKey);

        if (sourceIndex === -1 || targetIndex === -1) return prev;

        // Create new array with reordered items
        const newForms = [...prev];
        const removed = newForms.splice(sourceIndex, 1)[0];
        if (!removed) return prev;
        newForms.splice(targetIndex, 0, removed);

        // Update displayOrder for all items based on new positions
        return newForms.map((form, index) => ({
          ...form,
          values: {
            ...form.values,
            displayOrder: index,
          },
        }));
      });

      setDraggingKey(null);
      setDragOverKey(null);
    },
    []
  );

  const handleDragEnd = useCallback(() => {
    setDraggingKey(null);
    setDragOverKey(null);
  }, []);

  // Compute derived state
  const dirtyCount = useMemo(
    () => forms.filter((form) => isFormDirty(form.values, form.original)).length,
    [forms]
  );

  const downloadHealth = useMemo(
    () => computeDownloadHealthFromForms(forms),
    [forms]
  );

  return {
    forms,
    loading,
    error,
    dirtyCount,
    downloadHealth,
    loadApps,
    handleFieldChange,
    handlePlatformChange,
    handleAddApp,
    handleReset,
    handleDelete,
    handleSave,
    handleSaveAll,
    savingAll,
    draggingKey,
    dragOverKey,
    handleDragStart,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleDragEnd,
  };
}
