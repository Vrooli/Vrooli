// Hook for managing model registry editing state and operations.
// Preset values are ordered chains (string[]): primary first, optional runner-default
// sentinel (empty string) at the end. See types.ts#PresetChain.

import { useCallback, useEffect, useState } from "react";
import type { ModelRegistry, PresetChain } from "../types";
import { normalizeModelOptions, sanitizeModelRegistry } from "../lib/modelRegistry";

interface UseModelRegistryEditorOptions {
  /** Current model registry data from API */
  data: ModelRegistry | null | undefined;
  /** Whether the editor is active (e.g., dialog is open) */
  isActive: boolean;
  /** Function to update the registry via API */
  updateRegistry: (registry: ModelRegistry) => Promise<ModelRegistry>;
}

const RUNNER_DEFAULT_SENTINEL = "";

function cloneChain(chain: PresetChain | undefined): PresetChain {
  return chain ? [...chain] : [];
}

export function useModelRegistryEditor({
  data,
  isActive,
  updateRegistry,
}: UseModelRegistryEditorOptions) {
  const [draft, setDraft] = useState<ModelRegistry | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [newRunnerKey, setNewRunnerKey] = useState("");

  // Sync draft with data when editor becomes active
  useEffect(() => {
    if (!isActive) return;
    if (data) {
      setDraft(JSON.parse(JSON.stringify(data)) as ModelRegistry);
      setError(null);
    }
  }, [isActive, data]);

  const updateDraft = useCallback(
    (updater: (draft: ModelRegistry) => ModelRegistry) => {
      setDraft((prev) => (prev ? updater(prev) : prev));
    },
    []
  );

  const save = useCallback(async () => {
    if (!draft) return;
    setSaving(true);
    setError(null);
    try {
      const sanitized = sanitizeModelRegistry(draft);
      const updated = await updateRegistry(sanitized);
      setDraft(JSON.parse(JSON.stringify(updated)) as ModelRegistry);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  }, [updateRegistry, draft]);

  const reset = useCallback(() => {
    if (!data) return;
    setDraft(JSON.parse(JSON.stringify(data)) as ModelRegistry);
    setError(null);
  }, [data]);

  const addRunner = useCallback(() => {
    const trimmedKey = newRunnerKey.trim();
    if (!trimmedKey) {
      setError("Runner key is required.");
      return;
    }
    updateDraft((d) => {
      if (d.runners[trimmedKey]) {
        setError("Runner already exists.");
        return d;
      }
      return {
        ...d,
        runners: {
          ...d.runners,
          [trimmedKey]: {
            models: [],
            presets: {},
          },
        },
      };
    });
    setError(null);
    setNewRunnerKey("");
  }, [newRunnerKey, updateDraft]);

  const removeRunner = useCallback(
    (runnerKey: string) => {
      updateDraft((d) => {
        const { [runnerKey]: _, ...rest } = d.runners;
        return { ...d, runners: rest };
      });
    },
    [updateDraft]
  );

  const addFallbackRunner = useCallback(() => {
    updateDraft((d) => {
      const fallback = d.fallbackRunnerTypes ? [...d.fallbackRunnerTypes] : [];
      fallback.push("");
      return { ...d, fallbackRunnerTypes: fallback };
    });
  }, [updateDraft]);

  const updateFallbackRunner = useCallback(
    (index: number, value: string) => {
      updateDraft((d) => {
        const fallback = d.fallbackRunnerTypes ? [...d.fallbackRunnerTypes] : [];
        fallback[index] = value;
        return { ...d, fallbackRunnerTypes: fallback };
      });
    },
    [updateDraft]
  );

  const removeFallbackRunner = useCallback(
    (index: number) => {
      updateDraft((d) => {
        const fallback = d.fallbackRunnerTypes ? [...d.fallbackRunnerTypes] : [];
        fallback.splice(index, 1);
        return { ...d, fallbackRunnerTypes: fallback };
      });
    },
    [updateDraft]
  );

  const addModel = useCallback(
    (runnerKey: string) => {
      updateDraft((d) => {
        const runner = d.runners[runnerKey];
        if (!runner) return d;
        const models = normalizeModelOptions(runner.models);
        return {
          ...d,
          runners: {
            ...d.runners,
            [runnerKey]: {
              ...runner,
              models: [...models, { id: "", description: "" }],
            },
          },
        };
      });
    },
    [updateDraft]
  );

  const removeModel = useCallback(
    (runnerKey: string, index: number) => {
      updateDraft((d) => {
        const runner = d.runners[runnerKey];
        if (!runner) return d;
        const models = normalizeModelOptions(runner.models).filter((_, idx) => idx !== index);
        return {
          ...d,
          runners: {
            ...d.runners,
            [runnerKey]: {
              ...runner,
              models,
            },
          },
        };
      });
    },
    [updateDraft]
  );

  const updateModel = useCallback(
    (runnerKey: string, index: number, field: "id" | "description", value: string) => {
      updateDraft((d) => {
        const runner = d.runners[runnerKey];
        if (!runner) return d;
        const models = normalizeModelOptions(runner.models);
        const next = models.map((model, idx) =>
          idx === index ? { ...model, [field]: value } : model
        );
        return {
          ...d,
          runners: {
            ...d.runners,
            [runnerKey]: {
              ...runner,
              models: next,
            },
          },
        };
      });
    },
    [updateDraft]
  );

  // --- Preset chain editors ---

  const writeChain = useCallback(
    (runnerKey: string, presetKey: string, chain: PresetChain) => {
      updateDraft((d) => {
        const runner = d.runners[runnerKey];
        if (!runner) return d;
        const presets = { ...(runner.presets ?? {}) };
        if (chain.length === 0) {
          delete presets[presetKey];
        } else {
          presets[presetKey] = chain;
        }
        return {
          ...d,
          runners: {
            ...d.runners,
            [runnerKey]: { ...runner, presets },
          },
        };
      });
    },
    [updateDraft]
  );

  const setPresetEntry = useCallback(
    (runnerKey: string, presetKey: string, index: number, modelID: string) => {
      setDraft((prev) => {
        if (!prev) return prev;
        const runner = prev.runners[runnerKey];
        if (!runner) return prev;
        const existing = cloneChain(runner.presets?.[presetKey]);
        while (existing.length <= index) existing.push("");
        existing[index] = modelID;
        return {
          ...prev,
          runners: {
            ...prev.runners,
            [runnerKey]: {
              ...runner,
              presets: { ...(runner.presets ?? {}), [presetKey]: existing },
            },
          },
        };
      });
    },
    []
  );

  const addPresetEntry = useCallback(
    (runnerKey: string, presetKey: string) => {
      setDraft((prev) => {
        if (!prev) return prev;
        const runner = prev.runners[runnerKey];
        if (!runner) return prev;
        const existing = cloneChain(runner.presets?.[presetKey]);
        // Insert a new blank entry *before* the runner-default sentinel so it
        // stays at the tail.
        if (existing.length > 0 && existing[existing.length - 1] === RUNNER_DEFAULT_SENTINEL) {
          existing.splice(existing.length - 1, 0, "");
        } else {
          existing.push("");
        }
        return {
          ...prev,
          runners: {
            ...prev.runners,
            [runnerKey]: {
              ...runner,
              presets: { ...(runner.presets ?? {}), [presetKey]: existing },
            },
          },
        };
      });
    },
    []
  );

  const removePresetEntry = useCallback(
    (runnerKey: string, presetKey: string, index: number) => {
      setDraft((prev) => {
        if (!prev) return prev;
        const runner = prev.runners[runnerKey];
        if (!runner) return prev;
        const existing = cloneChain(runner.presets?.[presetKey]);
        existing.splice(index, 1);
        const presets = { ...(runner.presets ?? {}) };
        if (existing.length === 0) {
          delete presets[presetKey];
        } else {
          presets[presetKey] = existing;
        }
        return {
          ...prev,
          runners: {
            ...prev.runners,
            [runnerKey]: { ...runner, presets },
          },
        };
      });
    },
    []
  );

  const toggleRunnerDefault = useCallback(
    (runnerKey: string, presetKey: string, enabled: boolean) => {
      setDraft((prev) => {
        if (!prev) return prev;
        const runner = prev.runners[runnerKey];
        if (!runner) return prev;
        const existing = cloneChain(runner.presets?.[presetKey]);
        const hasSentinel =
          existing.length > 0 && existing[existing.length - 1] === RUNNER_DEFAULT_SENTINEL;
        if (enabled && !hasSentinel) {
          existing.push(RUNNER_DEFAULT_SENTINEL);
        } else if (!enabled && hasSentinel) {
          existing.pop();
        } else {
          return prev;
        }
        return {
          ...prev,
          runners: {
            ...prev.runners,
            [runnerKey]: {
              ...runner,
              presets: { ...(runner.presets ?? {}), [presetKey]: existing },
            },
          },
        };
      });
    },
    []
  );

  return {
    draft,
    error,
    saving,
    newRunnerKey,
    setNewRunnerKey,
    setError,
    save,
    reset,
    addRunner,
    removeRunner,
    addFallbackRunner,
    updateFallbackRunner,
    removeFallbackRunner,
    addModel,
    removeModel,
    updateModel,
    // Preset-chain API (replaces legacy updatePreset).
    setPresetEntry,
    addPresetEntry,
    removePresetEntry,
    toggleRunnerDefault,
    writeChain,
  };
}
