import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAPIKeysForm } from './useAPIKeysForm';
import * as apiKeysService from '../services/apiKeys.service';
import type { APIKey } from '../../../shared/api';

// Mock the service module
vi.mock('../services/apiKeys.service', async () => {
  const actual = await vi.importActual('../services/apiKeys.service');
  return {
    ...actual,
    fetchAPIKeys: vi.fn(),
    createAPIKey: vi.fn(),
    deleteAPIKey: vi.fn(),
    testAPIKey: vi.fn(),
    toggleAPIKey: vi.fn(),
  };
});

const mockFetchAPIKeys = vi.mocked(apiKeysService.fetchAPIKeys);
const mockCreateAPIKey = vi.mocked(apiKeysService.createAPIKey);
const mockDeleteAPIKey = vi.mocked(apiKeysService.deleteAPIKey);
const mockTestAPIKey = vi.mocked(apiKeysService.testAPIKey);
const mockToggleAPIKey = vi.mocked(apiKeysService.toggleAPIKey);

const createMockAPIKey = (overrides: Partial<APIKey> = {}): APIKey => ({
  id: 1,
  provider: 'openai',
  key_hint: 'sk-...abc',
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

describe('useAPIKeysForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchAPIKeys.mockResolvedValue([]);
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      expect(result.current.loading).toBe(true);
      await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it('has empty keys initially', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.keys).toEqual([]);
    });

    it('has modal closed initially', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.showAddModal).toBe(false);
    });

    it('has empty form state initially', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));
      expect(result.current.newKeyProvider).toBe('');
      expect(result.current.newKeyValue).toBe('');
    });
  });

  describe('loading data', () => {
    it('fetches keys on mount', async () => {
      const mockKeys = [
        createMockAPIKey({ id: 1, provider: 'openai' }),
        createMockAPIKey({ id: 2, provider: 'anthropic' }),
      ];
      mockFetchAPIKeys.mockResolvedValue(mockKeys);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.keys).toHaveLength(2);
      expect(mockFetchAPIKeys).toHaveBeenCalledTimes(1);
    });

    it('can refresh data', async () => {
      mockFetchAPIKeys.mockResolvedValue([]);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(mockFetchAPIKeys).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.fetchKeys();
      });

      expect(mockFetchAPIKeys).toHaveBeenCalledTimes(2);
    });
  });

  describe('available providers', () => {
    it('excludes already configured providers', async () => {
      const mockKeys = [createMockAPIKey({ provider: 'openai' })];
      mockFetchAPIKeys.mockResolvedValue(mockKeys);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      const openaiAvailable = result.current.availableProviders.find((p) => p.value === 'openai');
      expect(openaiAvailable).toBeUndefined();
    });
  });

  describe('add modal', () => {
    it('opens and closes modal', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.showAddModal).toBe(false);

      act(() => {
        result.current.setShowAddModal(true);
      });

      expect(result.current.showAddModal).toBe(true);

      act(() => {
        result.current.setShowAddModal(false);
      });

      expect(result.current.showAddModal).toBe(false);
    });

    it('updates form state', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setNewKeyProvider('openai');
        result.current.setNewKeyValue('sk-test-key');
      });

      expect(result.current.newKeyProvider).toBe('openai');
      expect(result.current.newKeyValue).toBe('sk-test-key');
    });

    it('clears form on clear', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setShowAddModal(true);
        result.current.setNewKeyProvider('openai');
        result.current.setNewKeyValue('sk-test-key');
      });

      act(() => {
        result.current.clearAddForm();
      });

      expect(result.current.showAddModal).toBe(false);
      expect(result.current.newKeyProvider).toBe('');
      expect(result.current.newKeyValue).toBe('');
    });
  });

  describe('handleAddKey', () => {
    it('validates required fields', async () => {
      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let addResult: { success: boolean; message?: string };
      await act(async () => {
        addResult = await result.current.handleAddKey();
      });

      expect(addResult!.success).toBe(false);
      expect(addResult!.message).toBe('Provider and key are required');
    });

    it('adds key successfully', async () => {
      mockCreateAPIKey.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setNewKeyProvider('openai');
        result.current.setNewKeyValue('sk-test-key');
      });

      let addResult: { success: boolean; message?: string };
      await act(async () => {
        addResult = await result.current.handleAddKey();
      });

      expect(addResult!.success).toBe(true);
      expect(mockCreateAPIKey).toHaveBeenCalledWith('openai', 'sk-test-key');
      expect(result.current.showAddModal).toBe(false);
      expect(result.current.newKeyProvider).toBe('');
    });

    it('handles add error', async () => {
      mockCreateAPIKey.mockRejectedValue(new Error('API error'));

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setNewKeyProvider('openai');
        result.current.setNewKeyValue('sk-test-key');
      });

      let addResult: { success: boolean; message?: string };
      await act(async () => {
        addResult = await result.current.handleAddKey();
      });

      expect(addResult!.success).toBe(false);
      expect(addResult!.message).toBe('API error');
    });

    it('sets adding state during add', async () => {
      let resolvePromise: () => void;
      mockCreateAPIKey.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => resolve(undefined);
        })
      );

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.setNewKeyProvider('openai');
        result.current.setNewKeyValue('sk-test-key');
      });

      let addPromise: Promise<{ success: boolean; message?: string }>;
      act(() => {
        addPromise = result.current.handleAddKey();
      });

      expect(result.current.addingKey).toBe(true);

      await act(async () => {
        resolvePromise!();
        await addPromise;
      });

      expect(result.current.addingKey).toBe(false);
    });
  });

  describe('handleDeleteKey', () => {
    it('deletes key successfully', async () => {
      mockFetchAPIKeys.mockResolvedValue([createMockAPIKey({ provider: 'openai' })]);
      mockDeleteAPIKey.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleDeleteKey('openai');
      });

      expect(deleteResult!.success).toBe(true);
      expect(mockDeleteAPIKey).toHaveBeenCalledWith('openai');
    });

    it('handles delete error', async () => {
      mockDeleteAPIKey.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let deleteResult: { success: boolean; message?: string };
      await act(async () => {
        deleteResult = await result.current.handleDeleteKey('openai');
      });

      expect(deleteResult!.success).toBe(false);
      expect(deleteResult!.message).toBe('Delete failed');
    });

    it('removes test result for deleted provider', async () => {
      mockFetchAPIKeys.mockResolvedValue([createMockAPIKey({ provider: 'openai' })]);
      mockTestAPIKey.mockResolvedValue({ success: true, message: 'Key is valid' });
      mockDeleteAPIKey.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      // First test the key
      await act(async () => {
        await result.current.handleTestKey('openai');
      });

      expect(result.current.testResults['openai']).toBeDefined();

      // Then delete it
      await act(async () => {
        await result.current.handleDeleteKey('openai');
      });

      expect(result.current.testResults['openai']).toBeUndefined();
    });
  });

  describe('handleTestKey', () => {
    it('tests key successfully', async () => {
      mockFetchAPIKeys.mockResolvedValue([createMockAPIKey({ provider: 'openai' })]);
      mockTestAPIKey.mockResolvedValue({ success: true, message: 'Key is valid' });

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let testResult: { success: boolean; message: string };
      await act(async () => {
        testResult = await result.current.handleTestKey('openai');
      });

      expect(testResult!.success).toBe(true);
      expect(testResult!.message).toBe('Key is valid');
      expect(result.current.testResults['openai']).toEqual({
        success: true,
        message: 'Key is valid',
      });
    });

    it('handles test failure', async () => {
      mockFetchAPIKeys.mockResolvedValue([createMockAPIKey({ provider: 'openai' })]);
      mockTestAPIKey.mockResolvedValue({ success: false, message: 'Invalid key' });

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let testResult: { success: boolean; message: string };
      await act(async () => {
        testResult = await result.current.handleTestKey('openai');
      });

      expect(testResult!.success).toBe(false);
      expect(result.current.testResults['openai']?.success).toBe(false);
    });

    it('handles test error', async () => {
      mockTestAPIKey.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let testResult: { success: boolean; message: string };
      await act(async () => {
        testResult = await result.current.handleTestKey('openai');
      });

      expect(testResult!.success).toBe(false);
      expect(testResult!.message).toBe('Network error');
    });

    it('sets testing state during test', async () => {
      let resolvePromise: (value: { success: boolean; message: string }) => void;
      mockTestAPIKey.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve;
        })
      );

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let testPromise: Promise<{ success: boolean; message: string }>;
      act(() => {
        testPromise = result.current.handleTestKey('openai');
      });

      expect(result.current.testingProvider).toBe('openai');

      await act(async () => {
        resolvePromise!({ success: true, message: 'OK' });
        await testPromise;
      });

      expect(result.current.testingProvider).toBeNull();
    });
  });

  describe('handleToggleKey', () => {
    it('toggles key successfully', async () => {
      mockFetchAPIKeys.mockResolvedValue([createMockAPIKey({ provider: 'openai', is_active: true })]);
      mockToggleAPIKey.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let toggleResult: { success: boolean; message?: string };
      await act(async () => {
        toggleResult = await result.current.handleToggleKey('openai', true);
      });

      expect(toggleResult!.success).toBe(true);
      expect(mockToggleAPIKey).toHaveBeenCalledWith('openai', true);
    });

    it('handles toggle error', async () => {
      mockToggleAPIKey.mockRejectedValue(new Error('Toggle failed'));

      const { result } = renderHook(() => useAPIKeysForm());
      await waitFor(() => expect(result.current.loading).toBe(false));

      let toggleResult: { success: boolean; message?: string };
      await act(async () => {
        toggleResult = await result.current.handleToggleKey('openai', true);
      });

      expect(toggleResult!.success).toBe(false);
      expect(toggleResult!.message).toBe('Toggle failed');
    });
  });
});
