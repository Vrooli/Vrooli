import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAISettings } from './useAISettings';
import { DEFAULT_AI_SETTINGS, STORAGE_KEYS } from './types';

describe('useAISettings', () => {
  const localStorageMock = (() => {
    let store: Record<string, string> = {};
    return {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { store[key] = value; }),
      clear: vi.fn(() => { store = {}; }),
    };
  })();

  beforeEach(() => {
    Object.defineProperty(window, 'localStorage', { value: localStorageMock, writable: true });
    localStorageMock.clear();
    localStorageMock.getItem.mockImplementation((key: string) => {
      const values: Record<string, string> = {};
      return values[key] ?? null;
    });
    vi.clearAllMocks();
  });

  it('defaults to the local-first gateway profile', () => {
    const { result } = renderHook(() => useAISettings());
    expect(result.current.settings).toEqual(DEFAULT_AI_SETTINGS);
    expect(result.current.selectedModel).toMatchObject({
      id: 'local_first',
      profile: 'local_first',
      tier: 'local',
    });
  });

  it('loads only supported profiles from storage', () => {
    localStorageMock.getItem.mockImplementation((key: string) =>
      key === STORAGE_KEYS.AI_MODEL ? 'remote_only' : key === STORAGE_KEYS.AI_MAX_STEPS ? '25' : null);
    const { result } = renderHook(() => useAISettings());
    expect(result.current.settings).toEqual({ model: 'remote_only', maxSteps: 25 });
    expect(result.current.isValidModel('remote_only')).toBe(true);
    expect(result.current.isValidModel('gpt-4o')).toBe(false);
  });

  it('persists profile and bounded step settings without pricing metadata', () => {
    const { result } = renderHook(() => useAISettings());
    act(() => result.current.updateSettings({ model: 'remote_only', maxSteps: 100 }));
    expect(result.current.settings).toEqual({ model: 'remote_only', maxSteps: 50 });
    expect(localStorageMock.setItem).toHaveBeenCalledWith(STORAGE_KEYS.AI_MODEL, 'remote_only');
    expect(localStorageMock.setItem).toHaveBeenCalledWith(STORAGE_KEYS.AI_MAX_STEPS, '50');
    expect(result.current.selectedModel).not.toHaveProperty('inputCostPer1MTokens');
    expect(result.current.selectedModel).not.toHaveProperty('outputCostPer1MTokens');
  });

  it('rejects unknown profiles and resets to defaults', () => {
    const { result } = renderHook(() => useAISettings());
    act(() => result.current.updateSettings({ model: 'gpt-4o' }));
    expect(result.current.settings.model).toBe(DEFAULT_AI_SETTINGS.model);
    act(() => result.current.updateSettings({ model: 'remote_only', maxSteps: 45 }));
    act(() => result.current.resetToDefaults());
    expect(result.current.settings).toEqual(DEFAULT_AI_SETTINGS);
  });
});
