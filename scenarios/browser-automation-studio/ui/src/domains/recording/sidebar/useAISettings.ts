/**
 * useAISettings Hook
 *
 * Manages AI navigation settings including:
 * - Model selection
 * - Max steps configuration
 * - Provider-neutral AI Gateway routing profile selection
 *
 * Settings are persisted to localStorage.
 */

import { useState, useCallback, useMemo } from 'react';
import { type AISettings, DEFAULT_AI_SETTINGS, STORAGE_KEYS } from './types';
import { type VisionModelSpec, VISION_MODELS } from '../ai-navigation/types';

// ============================================================================
// LocalStorage Helpers
// ============================================================================

function getStoredString(key: string, defaultValue: string): string {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    return stored ?? defaultValue;
  } catch {
    return defaultValue;
  }
}

function getStoredNumber(key: string, defaultValue: number, min?: number, max?: number): number {
  if (typeof window === 'undefined') return defaultValue;
  try {
    const stored = localStorage.getItem(key);
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!isNaN(parsed)) {
        if (min !== undefined && parsed < min) return min;
        if (max !== undefined && parsed > max) return max;
        return parsed;
      }
    }
  } catch {
    // Ignore storage errors
  }
  return defaultValue;
}

function setStoredValue(key: string, value: string | number): void {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(key, String(value));
  } catch {
    // Ignore storage errors
  }
}

// ============================================================================
// Hook Options
// ============================================================================

export interface UseAISettingsOptions {
  /** Available models (defaults to VISION_MODELS) */
  availableModels?: VisionModelSpec[];
  /** Initial settings (overrides localStorage) */
  initialSettings?: Partial<AISettings>;
}

// ============================================================================
// Hook Return Type
// ============================================================================

export interface UseAISettingsReturn {
  /** Current settings */
  settings: AISettings;
  /** Update one or more settings */
  updateSettings: (updates: Partial<AISettings>) => void;
  /** Reset to defaults */
  resetToDefaults: () => void;
  /** Currently selected model spec */
  selectedModel: VisionModelSpec;
  /** All available models */
  availableModels: VisionModelSpec[];
  /** Check if a model ID is valid */
  isValidModel: (modelId: string) => boolean;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useAISettings(options: UseAISettingsOptions = {}): UseAISettingsReturn {
  const { availableModels = VISION_MODELS, initialSettings } = options;

  // Initialize from localStorage or defaults
  const [settings, setSettings] = useState<AISettings>(() => {
    const storedModel = getStoredString(STORAGE_KEYS.AI_MODEL, DEFAULT_AI_SETTINGS.model);
    const storedMaxSteps = getStoredNumber(
      STORAGE_KEYS.AI_MAX_STEPS,
      DEFAULT_AI_SETTINGS.maxSteps,
      5,
      50
    );

    // Validate stored model exists
    const validModel = availableModels.some((m) => m.id === storedModel)
      ? storedModel
      : DEFAULT_AI_SETTINGS.model;

    return {
      model: initialSettings?.model ?? validModel,
      maxSteps: initialSettings?.maxSteps ?? storedMaxSteps,
    };
  });

  // Get the selected model spec
  const selectedModel = useMemo(() => {
    return (
      availableModels.find((m) => m.id === settings.model) ??
      availableModels[0] ?? {
        id: 'unknown',
        displayName: 'No models available',
        profile: 'local_first' as const,
        tier: 'local' as const,
        recommended: false,
      }
    );
  }, [availableModels, settings.model]);

  // Update settings and persist
  const updateSettings = useCallback(
    (updates: Partial<AISettings>) => {
      setSettings((prev) => {
        const next = { ...prev, ...updates };

        // Validate and persist
        if (updates.model !== undefined) {
          const validModel = availableModels.some((m) => m.id === updates.model)
            ? updates.model
            : prev.model;
          next.model = validModel;
          setStoredValue(STORAGE_KEYS.AI_MODEL, validModel);
        }

        if (updates.maxSteps !== undefined) {
          const clampedSteps = Math.max(5, Math.min(50, updates.maxSteps));
          next.maxSteps = clampedSteps;
          setStoredValue(STORAGE_KEYS.AI_MAX_STEPS, clampedSteps);
        }

        return next;
      });
    },
    [availableModels]
  );

  // Reset to defaults
  const resetToDefaults = useCallback(() => {
    setSettings(DEFAULT_AI_SETTINGS);
    setStoredValue(STORAGE_KEYS.AI_MODEL, DEFAULT_AI_SETTINGS.model);
    setStoredValue(STORAGE_KEYS.AI_MAX_STEPS, DEFAULT_AI_SETTINGS.maxSteps);
  }, []);

  // Model validation
  const isValidModel = useCallback(
    (modelId: string) => availableModels.some((m) => m.id === modelId),
    [availableModels]
  );

  return {
    settings,
    updateSettings,
    resetToDefaults,
    selectedModel,
    availableModels,
    isValidModel,
  };
}
