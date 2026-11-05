/**
 * ScenarioStore Test Suite
 *
 * Tests scenario fetching and caching functionality
 *
 * Requirements validated:
 * - Scenario list retrieval via ScenariosService Connect-RPC
 * - Response caching (30 second TTL)
 * - Concurrent request prevention
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act } from '@testing-library/react';

const listMock = vi.fn();

vi.mock('../../api/scenarios', () => ({
  scenariosClient: {
    list: (...args: unknown[]) => listMock(...args),
  },
}));

vi.mock('../../utils/logger', () => ({
  logger: {
    error: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    debug: vi.fn(),
  },
}));

// Import store AFTER mocks
import { useScenarioStore } from '../scenarioStore';

const resp = (scenarios: Array<Partial<{ name: string; description: string; status: string }>>) => ({
  scenarios: scenarios.map((s) => ({
    name: s.name ?? '',
    description: s.description ?? '',
    status: s.status ?? '',
  })),
});

describe('scenarioStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listMock.mockReset();
    useScenarioStore.setState({
      scenarios: [],
      isLoading: false,
      error: null,
      lastFetchTime: null,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Initial State', () => {
    it('has empty scenarios array', () => {
      expect(useScenarioStore.getState().scenarios).toEqual([]);
    });
    it('is not loading', () => {
      expect(useScenarioStore.getState().isLoading).toBe(false);
    });
    it('has no error', () => {
      expect(useScenarioStore.getState().error).toBeNull();
    });
    it('has no last fetch time', () => {
      expect(useScenarioStore.getState().lastFetchTime).toBeNull();
    });
  });

  describe('fetchScenarios', () => {
    it('fetches scenarios successfully', async () => {
      const mock = [
        { name: 'calendar', description: 'Calendar app', status: 'running' },
        { name: 'notes', description: 'Notes app', status: 'stopped' },
      ];
      listMock.mockResolvedValueOnce(resp(mock));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, isLoading, error } = useScenarioStore.getState();
      expect(scenarios).toEqual(mock);
      expect(isLoading).toBe(false);
      expect(error).toBeNull();
    });

    it('sets loading state during fetch', async () => {
      let resolveListPromise: ((value: unknown) => void) | null = null;
      const listPromise = new Promise((resolve) => {
        resolveListPromise = resolve;
      });
      listMock.mockReturnValueOnce(listPromise);

      const fetchCall = useScenarioStore.getState().fetchScenarios();
      expect(useScenarioStore.getState().isLoading).toBe(true);

      if (!resolveListPromise) throw new Error('expected resolver');
      resolveListPromise(resp([]));

      await act(async () => {
        await fetchCall;
      });
      expect(useScenarioStore.getState().isLoading).toBe(false);
    });

    it('updates lastFetchTime on successful fetch', async () => {
      const beforeFetch = Date.now();
      listMock.mockResolvedValueOnce(resp([]));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { lastFetchTime } = useScenarioStore.getState();
      const afterFetch = Date.now();
      expect(lastFetchTime).toBeGreaterThanOrEqual(beforeFetch);
      expect(lastFetchTime).toBeLessThanOrEqual(afterFetch);
    });

    it('handles empty scenarios array', async () => {
      listMock.mockResolvedValueOnce(resp([]));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, error } = useScenarioStore.getState();
      expect(scenarios).toEqual([]);
      expect(error).toBe('No scenarios found. Install or start a scenario, then refresh.');
    });

    it('filters out scenarios with empty names', async () => {
      listMock.mockResolvedValueOnce(resp([
        { name: 'calendar', description: 'Calendar app', status: 'running' },
        { name: '', description: 'Invalid', status: 'stopped' },
        { name: 'notes', description: 'Notes app', status: 'running' },
      ]));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(useScenarioStore.getState().scenarios).toEqual([
        { name: 'calendar', description: 'Calendar app', status: 'running' },
        { name: 'notes', description: 'Notes app', status: 'running' },
      ]);
    });
  });

  describe('Caching', () => {
    it('uses cache when fetched within 30 seconds', async () => {
      listMock.mockResolvedValueOnce(resp([
        { name: 'calendar', description: 'Calendar', status: 'running' },
      ]));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });
      expect(listMock).toHaveBeenCalledTimes(1);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });
      expect(listMock).toHaveBeenCalledTimes(1);
    });

    it('fetches again after cache expires (30 seconds)', async () => {
      vi.useFakeTimers();
      listMock.mockResolvedValue(resp([
        { name: 'calendar', description: 'Calendar', status: 'running' },
      ]));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });
      expect(listMock).toHaveBeenCalledTimes(1);

      vi.advanceTimersByTime(31000);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });
      expect(listMock).toHaveBeenCalledTimes(2);

      vi.useRealTimers();
    });
  });

  describe('Concurrent Request Prevention', () => {
    it('prevents concurrent fetches', async () => {
      let resolveFirst: ((value: unknown) => void) | null = null;
      const firstPromise = new Promise((resolve) => {
        resolveFirst = resolve;
      });
      listMock.mockReturnValueOnce(firstPromise);

      const firstFetch = useScenarioStore.getState().fetchScenarios();
      const secondFetch = useScenarioStore.getState().fetchScenarios();

      if (!resolveFirst) throw new Error('expected resolver');
      resolveFirst(resp([]));

      await act(async () => {
        await Promise.all([firstFetch, secondFetch]);
      });

      expect(listMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('Error Handling', () => {
    it('handles RPC errors', async () => {
      listMock.mockRejectedValueOnce(new Error('Internal server error'));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, error, isLoading } = useScenarioStore.getState();
      expect(scenarios).toEqual([]);
      expect(error).toBe('Internal server error');
      expect(isLoading).toBe(false);
    });

    it('updates lastFetchTime even on error to avoid hammering', async () => {
      const beforeFetch = Date.now();
      listMock.mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { lastFetchTime } = useScenarioStore.getState();
      const afterFetch = Date.now();
      expect(lastFetchTime).toBeGreaterThanOrEqual(beforeFetch);
      expect(lastFetchTime).toBeLessThanOrEqual(afterFetch);
    });

    it('uses default error message for non-Error exceptions', async () => {
      listMock.mockRejectedValueOnce('String error');

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(useScenarioStore.getState().error).toBe('Unable to load scenarios');
    });
  });

  describe('clearError', () => {
    it('clears error state', async () => {
      listMock.mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });
      expect(useScenarioStore.getState().error).toBe('Network error');

      act(() => {
        useScenarioStore.getState().clearError();
      });

      expect(useScenarioStore.getState().error).toBeNull();
    });
  });

  describe('API Integration', () => {
    it('calls ScenariosService.List', async () => {
      listMock.mockResolvedValueOnce(resp([]));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(listMock).toHaveBeenCalledTimes(1);
    });
  });
});
