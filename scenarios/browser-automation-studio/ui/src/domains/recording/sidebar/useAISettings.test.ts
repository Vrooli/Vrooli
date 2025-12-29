/**
 * useAISettings Hook Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAISettings, estimateNavigationCost, formatCost } from './useAISettings';
import { DEFAULT_AI_SETTINGS, STORAGE_KEYS } from './types';
import { VISION_MODELS, type VisionModelSpec } from '../ai-navigation/types';

describe('useAISettings', () => {
  // Mock localStorage
  const localStorageMock = (() => {
    let store: Record<string, string> = {};
    return {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: vi.fn(() => {
        store = {};
      }),
    };
  })();

  beforeEach(() => {
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true,
    });
    localStorageMock.clear();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initial state', () => {
    it('should initialize with default values', () => {
      const { result } = renderHook(() => useAISettings());

      expect(result.current.settings.model).toBe(DEFAULT_AI_SETTINGS.model);
      expect(result.current.settings.maxSteps).toBe(DEFAULT_AI_SETTINGS.maxSteps);
    });

    it('should respect initialSettings option', () => {
      const { result } = renderHook(() =>
        useAISettings({
          initialSettings: { model: 'gpt-4o', maxSteps: 30 },
        })
      );

      expect(result.current.settings.model).toBe('gpt-4o');
      expect(result.current.settings.maxSteps).toBe(30);
    });

    it('should read from localStorage if no initial settings', () => {
      localStorageMock.getItem.mockImplementation((key: string) => {
        if (key === STORAGE_KEYS.AI_MODEL) return 'gpt-4o';
        if (key === STORAGE_KEYS.AI_MAX_STEPS) return '25';
        return null;
      });

      const { result } = renderHook(() => useAISettings());

      expect(result.current.settings.model).toBe('gpt-4o');
      expect(result.current.settings.maxSteps).toBe(25);
    });

    it('should fallback to default if stored model is invalid', () => {
      localStorageMock.getItem.mockImplementation((key: string) => {
        if (key === STORAGE_KEYS.AI_MODEL) return 'invalid-model';
        return null;
      });

      const { result } = renderHook(() => useAISettings());

      expect(result.current.settings.model).toBe(DEFAULT_AI_SETTINGS.model);
    });

    it('should clamp max steps to valid range', () => {
      localStorageMock.getItem.mockImplementation((key: string) => {
        if (key === STORAGE_KEYS.AI_MAX_STEPS) return '100'; // Over max
        return null;
      });

      const { result } = renderHook(() => useAISettings());

      expect(result.current.settings.maxSteps).toBe(50); // Clamped to max
    });
  });

  describe('updateSettings', () => {
    it('should update model', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ model: 'gpt-4o' });
      });

      expect(result.current.settings.model).toBe('gpt-4o');
    });

    it('should persist model to localStorage', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ model: 'gpt-4o' });
      });

      expect(localStorageMock.setItem).toHaveBeenCalledWith(STORAGE_KEYS.AI_MODEL, 'gpt-4o');
    });

    it('should update maxSteps', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ maxSteps: 35 });
      });

      expect(result.current.settings.maxSteps).toBe(35);
    });

    it('should persist maxSteps to localStorage', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ maxSteps: 35 });
      });

      expect(localStorageMock.setItem).toHaveBeenCalledWith(STORAGE_KEYS.AI_MAX_STEPS, '35');
    });

    it('should clamp maxSteps to min 5', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ maxSteps: 2 });
      });

      expect(result.current.settings.maxSteps).toBe(5);
    });

    it('should clamp maxSteps to max 50', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ maxSteps: 100 });
      });

      expect(result.current.settings.maxSteps).toBe(50);
    });

    it('should reject invalid model', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ model: 'invalid-model' });
      });

      expect(result.current.settings.model).toBe(DEFAULT_AI_SETTINGS.model);
    });
  });

  describe('resetToDefaults', () => {
    it('should reset to default values', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ model: 'gpt-4o', maxSteps: 45 });
      });

      act(() => {
        result.current.resetToDefaults();
      });

      expect(result.current.settings.model).toBe(DEFAULT_AI_SETTINGS.model);
      expect(result.current.settings.maxSteps).toBe(DEFAULT_AI_SETTINGS.maxSteps);
    });

    it('should persist defaults to localStorage', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.resetToDefaults();
      });

      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        STORAGE_KEYS.AI_MODEL,
        DEFAULT_AI_SETTINGS.model
      );
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        STORAGE_KEYS.AI_MAX_STEPS,
        String(DEFAULT_AI_SETTINGS.maxSteps)
      );
    });
  });

  describe('selectedModel', () => {
    it('should return the selected model spec', () => {
      const { result } = renderHook(() => useAISettings());

      expect(result.current.selectedModel.id).toBe(DEFAULT_AI_SETTINGS.model);
    });

    it('should update when model changes', () => {
      const { result } = renderHook(() => useAISettings());

      act(() => {
        result.current.updateSettings({ model: 'gpt-4o' });
      });

      expect(result.current.selectedModel.id).toBe('gpt-4o');
      expect(result.current.selectedModel.displayName).toBe('GPT-4o');
    });
  });

  describe('isValidModel', () => {
    it('should return true for valid models', () => {
      const { result } = renderHook(() => useAISettings());

      expect(result.current.isValidModel('gpt-4o')).toBe(true);
      expect(result.current.isValidModel('qwen3-vl-30b')).toBe(true);
    });

    it('should return false for invalid models', () => {
      const { result } = renderHook(() => useAISettings());

      expect(result.current.isValidModel('invalid-model')).toBe(false);
      expect(result.current.isValidModel('')).toBe(false);
    });
  });

  describe('cost estimation', () => {
    it('should estimate cost with current settings', () => {
      const { result } = renderHook(() => useAISettings());

      const cost = result.current.estimateCost();

      // Cost should be positive
      expect(cost).toBeGreaterThan(0);
    });

    it('should estimate cost with overrides', () => {
      const { result } = renderHook(() => useAISettings());

      const baseCost = result.current.estimateCost();
      const doubleCost = result.current.estimateCostWith({ maxSteps: 40 });

      // Double steps should roughly double cost
      expect(doubleCost).toBeGreaterThan(baseCost);
    });

    it('should format cost correctly', () => {
      const { result } = renderHook(() => useAISettings());

      expect(result.current.formatCost(0.005)).toBe('$0.0050');
      expect(result.current.formatCost(1.23)).toBe('$1.23');
      expect(result.current.formatCost(10)).toBe('$10.00');
    });
  });
});

describe('estimateNavigationCost', () => {
  const testModel: VisionModelSpec = {
    id: 'test-model',
    displayName: 'Test Model',
    provider: 'openrouter',
    inputCostPer1MTokens: 1.0,
    outputCostPer1MTokens: 2.0,
    tier: 'standard',
    recommended: false,
  };

  it('should calculate cost based on tokens and model pricing', () => {
    const cost = estimateNavigationCost(testModel, 10);

    // 10 steps * 2000 input tokens = 20000 tokens
    // 10 steps * 100 output tokens = 1000 tokens
    // Input cost: 20000 / 1M * 1.0 = 0.02
    // Output cost: 1000 / 1M * 2.0 = 0.002
    // Total: 0.022
    expect(cost).toBeCloseTo(0.022, 4);
  });

  it('should scale with max steps', () => {
    const cost10 = estimateNavigationCost(testModel, 10);
    const cost20 = estimateNavigationCost(testModel, 20);

    expect(cost20).toBeCloseTo(cost10 * 2, 4);
  });
});

describe('formatCost', () => {
  it('should format small costs with 4 decimal places', () => {
    expect(formatCost(0.001)).toBe('$0.0010');
    expect(formatCost(0.0099)).toBe('$0.0099');
  });

  it('should format larger costs with 2 decimal places', () => {
    expect(formatCost(0.01)).toBe('$0.01');
    expect(formatCost(1.5)).toBe('$1.50');
    expect(formatCost(10)).toBe('$10.00');
  });
});
