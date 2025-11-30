import { useState, useCallback } from 'react';
import {
  startScenario,
  stopScenario,
  restartScenario,
  getScenarioStatus,
  getScenarioLogs,
  getPreviewLinks,
  type PreviewLinks,
} from '../lib/api';
import { parseApiError, type StructuredError } from '../components/ErrorDisplay';

export interface ScenarioStatus {
  running: boolean;
  loading: boolean;
  /** Last error that occurred for this scenario */
  error?: StructuredError | null;
}

export interface UseScenarioLifecycleReturn {
  statuses: Record<string, ScenarioStatus>;
  previewLinks: Record<string, PreviewLinks>;
  showLogs: Record<string, boolean>;
  logs: Record<string, string>;
  /** Global error for lifecycle operations */
  lifecycleError: StructuredError | null;
  /** Clear the global lifecycle error */
  clearError: () => void;
  startScenario: (scenarioId: string) => Promise<void>;
  stopScenario: (scenarioId: string) => Promise<void>;
  restartScenario: (scenarioId: string) => Promise<void>;
  toggleLogs: (scenarioId: string) => Promise<void>;
  loadStatuses: (scenarioIds: string[]) => Promise<void>;
}

/**
 * Custom hook for managing scenario lifecycle operations (start, stop, restart, logs, preview links).
 * Provides centralized state management for scenario status, preview links, logs, and errors.
 *
 * Error handling:
 * - Per-scenario errors are stored in the status object for that scenario
 * - A global lifecycleError is set for the most recent operation failure
 * - Errors include structured information like codes, suggestions, and recoverability
 */
export function useScenarioLifecycle(): UseScenarioLifecycleReturn {
  const [statuses, setStatuses] = useState<Record<string, ScenarioStatus>>({});
  const [previewLinks, setPreviewLinks] = useState<Record<string, PreviewLinks>>({});
  const [showLogs, setShowLogs] = useState<Record<string, boolean>>({});
  const [logs, setLogs] = useState<Record<string, string>>({});
  const [lifecycleError, setLifecycleError] = useState<StructuredError | null>(null);

  const clearError = useCallback(() => {
    setLifecycleError(null);
  }, []);

  const updateStatus = useCallback(async (scenarioId: string, operation: () => Promise<void>) => {
    try {
      // Clear any previous error for this scenario
      setStatuses((prev) => ({
        ...prev,
        [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: true, error: null },
      }));
      // Clear global error when starting a new operation
      setLifecycleError(null);

      await operation();

      const status = await getScenarioStatus(scenarioId);
      setStatuses((prev) => ({
        ...prev,
        [scenarioId]: { running: status.running, loading: false, error: null },
      }));

      return status.running;
    } catch (err) {
      console.error(`Lifecycle operation failed for ${scenarioId}:`, err);

      // Parse the error into structured format
      const structuredError = parseApiError(err);

      // Update both per-scenario error and global error
      setStatuses((prev) => ({
        ...prev,
        [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: false, error: structuredError },
      }));
      setLifecycleError(structuredError);

      throw err;
    }
  }, []);

  const fetchPreviewLinks = useCallback(async (scenarioId: string, retries = 2) => {
    try {
      const links = await getPreviewLinks(scenarioId);
      setPreviewLinks((prev) => ({ ...prev, [scenarioId]: links }));
    } catch (err) {
      // Preview links may not be available immediately after start
      // Retry a few times with delay before giving up
      if (retries > 0) {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        return fetchPreviewLinks(scenarioId, retries - 1);
      }
      // After retries exhausted, log but don't surface error to user
      // The scenario may still be initializing
      console.warn(`Preview links unavailable for ${scenarioId}:`, err instanceof Error ? err.message : err);
    }
  }, []);

  const handleStart = useCallback(
    async (scenarioId: string) => {
      const isRunning = await updateStatus(scenarioId, () => startScenario(scenarioId));
      if (isRunning) {
        await fetchPreviewLinks(scenarioId);
      }
    },
    [updateStatus, fetchPreviewLinks],
  );

  const handleStop = useCallback(
    async (scenarioId: string) => {
      await updateStatus(scenarioId, () => stopScenario(scenarioId));
    },
    [updateStatus],
  );

  const handleRestart = useCallback(
    async (scenarioId: string) => {
      const isRunning = await updateStatus(scenarioId, () => restartScenario(scenarioId));
      if (isRunning) {
        await fetchPreviewLinks(scenarioId);
      }
    },
    [updateStatus, fetchPreviewLinks],
  );

  const handleToggleLogs = useCallback(async (scenarioId: string) => {
    const show = !showLogs[scenarioId];
    setShowLogs((prev) => ({ ...prev, [scenarioId]: show }));

    if (show && !logs[scenarioId]) {
      try {
        const result = await getScenarioLogs(scenarioId, 100);
        setLogs((prev) => ({ ...prev, [scenarioId]: result.logs }));
      } catch (err) {
        console.error('Failed to load logs:', err);
        // Provide actionable feedback instead of generic error
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        let logError = '--- Failed to load logs ---\n\n';
        if (errorMessage.includes('not found')) {
          logError += 'The scenario was not found. It may have been deleted or moved.\n\nTry refreshing the generated scenarios list.';
        } else if (errorMessage.includes('not running')) {
          logError += 'The scenario is not currently running.\n\nStart the scenario first to view logs.';
        } else {
          logError += `Error: ${errorMessage}\n\nTry restarting the scenario or check the API logs.`;
        }
        setLogs((prev) => ({ ...prev, [scenarioId]: logError }));
      }
    }
  }, [showLogs, logs]);

  const loadStatuses = useCallback(async (scenarioIds: string[]) => {
    // Parallelize all status checks instead of sequential loop
    const results = await Promise.allSettled(
      scenarioIds.map(async (scenarioId) => {
        const status = await getScenarioStatus(scenarioId);
        return { scenarioId, status };
      })
    );

    // Update all statuses in a single batch
    const statusUpdates: Record<string, ScenarioStatus> = {};
    const runningScenarios: string[] = [];

    results.forEach((result) => {
      if (result.status === 'fulfilled') {
        const { scenarioId, status } = result.value;
        statusUpdates[scenarioId] = { running: status.running, loading: false };
        if (status.running) {
          runningScenarios.push(scenarioId);
        }
      } else {
        // Extract scenarioId from rejected promise (fallback to index)
        const index = results.indexOf(result);
        if (index >= 0 && index < scenarioIds.length) {
          statusUpdates[scenarioIds[index]] = { running: false, loading: false };
        }
      }
    });

    // Single state update for all statuses
    setStatuses((prev) => ({ ...prev, ...statusUpdates }));

    // Parallelize preview link fetches for running scenarios
    if (runningScenarios.length > 0) {
      await Promise.allSettled(
        runningScenarios.map((scenarioId) => fetchPreviewLinks(scenarioId))
      );
    }
  }, [fetchPreviewLinks]);

  return {
    statuses,
    previewLinks,
    showLogs,
    logs,
    lifecycleError,
    clearError,
    startScenario: handleStart,
    stopScenario: handleStop,
    restartScenario: handleRestart,
    toggleLogs: handleToggleLogs,
    loadStatuses,
  };
}
