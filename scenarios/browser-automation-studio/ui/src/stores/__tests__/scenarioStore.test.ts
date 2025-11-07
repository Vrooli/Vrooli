/**
 * ScenarioStore Test Suite
 *
 * Tests scenario fetching and caching functionality
 *
 * Requirements validated:
 * - Scenario list retrieval
 * - Response caching (30 second TTL)
 * - Concurrent request prevention
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act } from '@testing-library/react';

// Mock dependencies using proper factory functions
vi.mock('../../config', () => ({
  getConfig: vi.fn(() => Promise.resolve({
    API_URL: 'http://localhost:8080/api/v1',
  })),
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

describe('scenarioStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();

    // Reset store state
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
      const { scenarios } = useScenarioStore.getState();
      expect(scenarios).toEqual([]);
    });

    it('is not loading', () => {
      const { isLoading } = useScenarioStore.getState();
      expect(isLoading).toBe(false);
    });

    it('has no error', () => {
      const { error } = useScenarioStore.getState();
      expect(error).toBeNull();
    });

    it('has no last fetch time', () => {
      const { lastFetchTime } = useScenarioStore.getState();
      expect(lastFetchTime).toBeNull();
    });
  });

  describe('fetchScenarios', () => {
    it('fetches scenarios successfully', async () => {
      const mockScenarios = [
        { name: 'calendar', description: 'Calendar app', status: 'running' },
        { name: 'notes', description: 'Notes app', status: 'stopped' },
      ];

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: mockScenarios }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, isLoading, error } = useScenarioStore.getState();

      expect(scenarios).toEqual(mockScenarios);
      expect(isLoading).toBe(false);
      expect(error).toBeNull();
    });

    it('sets loading state during fetch', async () => {
      let resolvePromise: (value: any) => void;
      const fetchPromise = new Promise((resolve) => {
        resolvePromise = resolve;
      });

      vi.mocked(global.fetch).mockReturnValueOnce(fetchPromise as any);

      const fetchCall = useScenarioStore.getState().fetchScenarios();

      // Check loading state immediately
      expect(useScenarioStore.getState().isLoading).toBe(true);

      // Resolve fetch
      resolvePromise!({
        ok: true,
        json: async () => ({ scenarios: [] }),
      });

      await act(async () => {
        await fetchCall;
      });

      expect(useScenarioStore.getState().isLoading).toBe(false);
    });

    it('updates lastFetchTime on successful fetch', async () => {
      const beforeFetch = Date.now();

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: [] }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { lastFetchTime } = useScenarioStore.getState();
      const afterFetch = Date.now();

      expect(lastFetchTime).toBeGreaterThanOrEqual(beforeFetch);
      expect(lastFetchTime).toBeLessThanOrEqual(afterFetch);
    });

    it('handles empty scenarios array', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: [] }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, error } = useScenarioStore.getState();

      expect(scenarios).toEqual([]);
      expect(error).toBe('No scenarios found. Install or start a scenario, then refresh.');
    });

    it('filters out scenarios with empty names', async () => {
      const mockScenarios = [
        { name: 'calendar', description: 'Calendar app', status: 'running' },
        { name: '', description: 'Invalid', status: 'stopped' },
        { name: 'notes', description: 'Notes app', status: 'running' },
      ];

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: mockScenarios }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios } = useScenarioStore.getState();

      expect(scenarios).toHaveLength(2);
      expect(scenarios).toEqual([
        { name: 'calendar', description: 'Calendar app', status: 'running' },
        { name: 'notes', description: 'Notes app', status: 'running' },
      ]);
    });

    it('normalizes scenario data with missing fields', async () => {
      const mockScenarios = [
        { name: 'calendar' }, // Missing description and status
        { name: 'notes', description: 'Notes app' }, // Missing status
      ];

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: mockScenarios }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios } = useScenarioStore.getState();

      expect(scenarios).toEqual([
        { name: 'calendar', description: '', status: '' },
        { name: 'notes', description: 'Notes app', status: '' },
      ]);
    });
  });

  describe('Caching', () => {
    it('uses cache when fetched within 30 seconds', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          scenarios: [{ name: 'calendar', description: 'Calendar', status: 'running' }],
        }),
      } as Response);

      // First fetch
      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(global.fetch).toHaveBeenCalledTimes(1);

      // Second fetch immediately after (should use cache)
      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(global.fetch).toHaveBeenCalledTimes(1); // No additional fetch
    });

    it('fetches again after cache expires (30 seconds)', async () => {
      vi.useFakeTimers();

      vi.mocked(global.fetch).mockResolvedValue({
        ok: true,
        json: async () => ({
          scenarios: [{ name: 'calendar', description: 'Calendar', status: 'running' }],
        }),
      } as Response);

      // First fetch
      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(global.fetch).toHaveBeenCalledTimes(1);

      // Advance time by 31 seconds
      vi.advanceTimersByTime(31000);

      // Second fetch (cache expired)
      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(global.fetch).toHaveBeenCalledTimes(2);

      vi.useRealTimers();
    });
  });

  describe('Concurrent Request Prevention', () => {
    it('prevents concurrent fetches', async () => {
      let resolveFirstFetch: (value: any) => void;
      const firstFetchPromise = new Promise((resolve) => {
        resolveFirstFetch = resolve;
      });

      vi.mocked(global.fetch).mockReturnValueOnce(firstFetchPromise as any);

      // Start first fetch
      const firstFetch = useScenarioStore.getState().fetchScenarios();

      // Start second fetch while first is in progress
      const secondFetch = useScenarioStore.getState().fetchScenarios();

      // Resolve first fetch
      resolveFirstFetch!({
        ok: true,
        json: async () => ({ scenarios: [] }),
      });

      await act(async () => {
        await Promise.all([firstFetch, secondFetch]);
      });

      // Should only have called fetch once
      expect(global.fetch).toHaveBeenCalledTimes(1);
    });
  });

  describe('Error Handling', () => {
    it('handles HTTP error responses', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        text: async () => 'Internal server error',
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, error, isLoading } = useScenarioStore.getState();

      expect(scenarios).toEqual([]);
      expect(error).toBe('Internal server error');
      expect(isLoading).toBe(false);
    });

    it('handles network errors', async () => {
      vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { scenarios, error, isLoading } = useScenarioStore.getState();

      expect(scenarios).toEqual([]);
      expect(error).toBe('Network error');
      expect(isLoading).toBe(false);
    });

    it('updates lastFetchTime even on error to avoid hammering', async () => {
      const beforeFetch = Date.now();

      vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { lastFetchTime } = useScenarioStore.getState();
      const afterFetch = Date.now();

      expect(lastFetchTime).toBeGreaterThanOrEqual(beforeFetch);
      expect(lastFetchTime).toBeLessThanOrEqual(afterFetch);
    });

    it('uses default error message for non-Error exceptions', async () => {
      vi.mocked(global.fetch).mockRejectedValueOnce('String error');

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const { error } = useScenarioStore.getState();

      expect(error).toBe('Unable to load scenarios');
    });
  });

  describe('clearError', () => {
    it('clears error state', async () => {
      // Set an error first
      vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(useScenarioStore.getState().error).toBe('Network error');

      // Clear the error
      act(() => {
        useScenarioStore.getState().clearError();
      });

      expect(useScenarioStore.getState().error).toBeNull();
    });

    it('does not affect other state', async () => {
      const mockScenarios = [
        { name: 'calendar', description: 'Calendar', status: 'running' },
      ];

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: mockScenarios }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      const beforeClear = useScenarioStore.getState();

      act(() => {
        useScenarioStore.getState().clearError();
      });

      const afterClear = useScenarioStore.getState();

      expect(afterClear.scenarios).toEqual(beforeClear.scenarios);
      expect(afterClear.isLoading).toBe(beforeClear.isLoading);
      expect(afterClear.lastFetchTime).toBe(beforeClear.lastFetchTime);
      expect(afterClear.error).toBeNull();
    });
  });

  describe('API Integration', () => {
    it('calls correct API endpoint', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ scenarios: [] }),
      } as Response);

      await act(async () => {
        await useScenarioStore.getState().fetchScenarios();
      });

      expect(global.fetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/scenarios');
    });
  });
});
