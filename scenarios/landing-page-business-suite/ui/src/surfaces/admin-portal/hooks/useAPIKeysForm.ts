import { useState, useEffect, useCallback, useMemo } from 'react';
import type { APIKey } from '../../../shared/api';
import {
  type KeyTestResult,
  type ProviderOption,
  fetchAPIKeys,
  createAPIKey,
  deleteAPIKey,
  testAPIKey,
  toggleAPIKey,
  getAvailableProviders,
  removeTestResult,
  addTestResult,
} from '../services/apiKeys.service';

export interface UseAPIKeysFormReturn {
  // Data state
  keys: APIKey[];
  testResults: Record<string, KeyTestResult>;
  availableProviders: ProviderOption[];

  // Form state
  showAddModal: boolean;
  newKeyProvider: string;
  newKeyValue: string;

  // UI state
  loading: boolean;
  testingProvider: string | null;
  addingKey: boolean;

  // Actions
  fetchKeys: () => Promise<void>;
  handleAddKey: () => Promise<{ success: boolean; message?: string }>;
  handleDeleteKey: (provider: string) => Promise<{ success: boolean; message?: string }>;
  handleTestKey: (provider: string) => Promise<KeyTestResult>;
  handleToggleKey: (provider: string, currentActive: boolean) => Promise<{ success: boolean; message?: string }>;

  // Form actions
  setShowAddModal: (show: boolean) => void;
  setNewKeyProvider: (provider: string) => void;
  setNewKeyValue: (value: string) => void;
  clearAddForm: () => void;
}

/**
 * Reactive hook for API keys management
 *
 * Provides state and handlers for:
 * - Loading API keys list
 * - Adding, deleting, testing, and toggling keys
 * - Tracking test results
 * - Managing add modal state
 */
export function useAPIKeysForm(): UseAPIKeysFormReturn {
  // Data state
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [testResults, setTestResults] = useState<Record<string, KeyTestResult>>({});

  // Form state
  const [showAddModal, setShowAddModal] = useState(false);
  const [newKeyProvider, setNewKeyProvider] = useState('');
  const [newKeyValue, setNewKeyValue] = useState('');

  // UI state
  const [loading, setLoading] = useState(true);
  const [testingProvider, setTestingProvider] = useState<string | null>(null);
  const [addingKey, setAddingKey] = useState(false);

  /**
   * Fetch all API keys
   */
  const fetchKeys = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAPIKeys();
      setKeys(data);
    } catch (err) {
      console.error('Failed to fetch API keys:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void fetchKeys();
  }, [fetchKeys]);

  /**
   * Clear add form
   */
  const clearAddForm = useCallback(() => {
    setShowAddModal(false);
    setNewKeyProvider('');
    setNewKeyValue('');
  }, []);

  /**
   * Add a new API key
   */
  const handleAddKey = useCallback(async (): Promise<{ success: boolean; message?: string }> => {
    if (!newKeyProvider || !newKeyValue) {
      return { success: false, message: 'Provider and key are required' };
    }

    setAddingKey(true);
    try {
      await createAPIKey(newKeyProvider, newKeyValue);
      clearAddForm();
      await fetchKeys();
      return { success: true, message: `API key for ${newKeyProvider} added successfully` };
    } catch (err) {
      return {
        success: false,
        message: err instanceof Error ? err.message : 'Failed to add API key',
      };
    } finally {
      setAddingKey(false);
    }
  }, [newKeyProvider, newKeyValue, clearAddForm, fetchKeys]);

  /**
   * Delete an API key
   */
  const handleDeleteKey = useCallback(
    async (provider: string): Promise<{ success: boolean; message?: string }> => {
      try {
        await deleteAPIKey(provider);
        setTestResults((prev) => removeTestResult(prev, provider));
        await fetchKeys();
        return { success: true, message: `API key for ${provider} deleted` };
      } catch (err) {
        return {
          success: false,
          message: err instanceof Error ? err.message : 'Failed to delete API key',
        };
      }
    },
    [fetchKeys]
  );

  /**
   * Test an API key
   */
  const handleTestKey = useCallback(async (provider: string): Promise<KeyTestResult> => {
    setTestingProvider(provider);
    try {
      const result = await testAPIKey(provider);
      setTestResults((prev) => addTestResult(prev, provider, result));
      return result;
    } catch (err) {
      const result: KeyTestResult = {
        success: false,
        message: err instanceof Error ? err.message : 'Test failed',
      };
      setTestResults((prev) => addTestResult(prev, provider, result));
      return result;
    } finally {
      setTestingProvider(null);
    }
  }, []);

  /**
   * Toggle API key active status
   */
  const handleToggleKey = useCallback(
    async (provider: string, currentActive: boolean): Promise<{ success: boolean; message?: string }> => {
      try {
        await toggleAPIKey(provider, currentActive);
        await fetchKeys();
        return {
          success: true,
          message: `API key for ${provider} ${!currentActive ? 'enabled' : 'disabled'}`,
        };
      } catch (err) {
        return {
          success: false,
          message: err instanceof Error ? err.message : 'Failed to toggle API key',
        };
      }
    },
    [fetchKeys]
  );

  // Computed values
  const availableProviders = useMemo(() => getAvailableProviders(keys), [keys]);

  return {
    // Data state
    keys,
    testResults,
    availableProviders,

    // Form state
    showAddModal,
    newKeyProvider,
    newKeyValue,

    // UI state
    loading,
    testingProvider,
    addingKey,

    // Actions
    fetchKeys,
    handleAddKey,
    handleDeleteKey,
    handleTestKey,
    handleToggleKey,

    // Form actions
    setShowAddModal,
    setNewKeyProvider,
    setNewKeyValue,
    clearAddForm,
  };
}
