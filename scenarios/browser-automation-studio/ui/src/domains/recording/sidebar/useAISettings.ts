/**
 * useAISettings Hook
 *
 * Manages AI navigation settings including:
 * - Model selection
 * - Max steps configuration
 * - Cost estimation
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
// Cost Estimation
// ============================================================================

/**
 * Estimate cost for a navigation session.
 * Based on approximate token usage per step:
 * - ~2000 input tokens (screenshot + prompt)
 * - ~100 output tokens (action + reasoning)
 */
export function estimateNavigationCost(model: VisionModelSpec, maxSteps: number): number {
  const avgInputTokensPerStep = 2000;
  const avgOutputTokensPerStep = 100;
  const totalInputTokens = avgInputTokensPerStep * maxSteps;
  const totalOutputTokens = avgOutputTokensPerStep * maxSteps;

  const inputCost = (totalInputTokens / 1_000_000) * model.inputCostPer1MTokens;
  const outputCost = (totalOutputTokens / 1_000_000) * model.outputCostPer1MTokens;

  return inputCost + outputCost;
}

/**
 * Format cost for display.
 */
export function formatCost(cost: number): string {
  if (cost < 0.01) {
    return `$${cost.toFixed(4)}`;
  }
  return `$${cost.toFixed(2)}`;
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
  /** Estimate cost with current settings */
  estimateCost: () => number;
  /** Estimate cost with custom settings */
  estimateCostWith: (overrides?: Partial<AISettings>) => number;
  /** Format cost for display */
  formatCost: (cost: number) => string;
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
        tier: 'standard' as const,
        inputCostPer1MTokens: 0,
        outputCostPer1MTokens: 0,
        provider: 'openrouter' as const,
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

  // Cost estimation
  const estimateCostWithSettings = useCallback(
    (overrides?: Partial<AISettings>) => {
      const effectiveSettings = { ...settings, ...overrides };
      const model =
        availableModels.find((m) => m.id === effectiveSettings.model) ?? selectedModel;
      return estimateNavigationCost(model, effectiveSettings.maxSteps);
    },
    [availableModels, selectedModel, settings]
  );

  const estimateCostCurrent = useCallback(() => {
    return estimateNavigationCost(selectedModel, settings.maxSteps);
  }, [selectedModel, settings.maxSteps]);

  // Model validation
  const isValidModel = useCallback(
    (modelId: string) => availableModels.some((m) => m.id === modelId),
    [availableModels]
  );

  return {
    settings,
    updateSettings,
    resetToDefaults,
    estimateCost: estimateCostCurrent,
    estimateCostWith: estimateCostWithSettings,
    formatCost,
    selectedModel,
    availableModels,
    isValidModel,
  };
}
