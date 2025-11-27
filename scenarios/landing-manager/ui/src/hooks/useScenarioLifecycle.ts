import { useState, useEffect, useCallback } from 'react';
import {
  startScenario,
  stopScenario,
  restartScenario,
  getScenarioStatus,
  getScenarioLogs,
  getPreviewLinks,
  type PreviewLinks,
} from '../lib/api';

export interface ScenarioStatus {
  running: boolean;
  loading: boolean;
}

export interface UseScenarioLifecycleReturn {
  statuses: Record<string, ScenarioStatus>;
  previewLinks: Record<string, PreviewLinks>;
  showLogs: Record<string, boolean>;
  logs: Record<string, string>;
  startScenario: (scenarioId: string) => Promise<void>;
  stopScenario: (scenarioId: string) => Promise<void>;
  restartScenario: (scenarioId: string) => Promise<void>;
  toggleLogs: (scenarioId: string) => Promise<void>;
  loadStatuses: (scenarioIds: string[]) => Promise<void>;
}

/**
 * Custom hook for managing scenario lifecycle operations (start, stop, restart, logs, preview links).
 * Provides centralized state management for scenario status, preview links, and logs.
 */
export function useScenarioLifecycle(): UseScenarioLifecycleReturn {
  const [statuses, setStatuses] = useState<Record<string, ScenarioStatus>>({});
  const [previewLinks, setPreviewLinks] = useState<Record<string, PreviewLinks>>({});
  const [showLogs, setShowLogs] = useState<Record<string, boolean>>({});
  const [logs, setLogs] = useState<Record<string, string>>({});

  const updateStatus = useCallback(async (scenarioId: string, operation: () => Promise<void>) => {
    try {
      setStatuses((prev) => ({
        ...prev,
        [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: true },
      }));

      await operation();

      const status = await getScenarioStatus(scenarioId);
      setStatuses((prev) => ({
        ...prev,
        [scenarioId]: { running: status.running, loading: false },
      }));

      return status.running;
    } catch (err) {
      console.error(`Lifecycle operation failed for ${scenarioId}:`, err);
      setStatuses((prev) => ({
        ...prev,
        [scenarioId]: { running: false, loading: false },
      }));
      throw err;
    }
  }, []);

  const fetchPreviewLinks = useCallback(async (scenarioId: string) => {
    try {
      const links = await getPreviewLinks(scenarioId);
      setPreviewLinks((prev) => ({ ...prev, [scenarioId]: links }));
    } catch {
      // Preview links may not be available immediately, ignore error
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
        setLogs((prev) => ({ ...prev, [scenarioId]: 'Failed to load logs' }));
      }
    }
  }, [showLogs, logs]);

  const loadStatuses = useCallback(async (scenarioIds: string[]) => {
    for (const scenarioId of scenarioIds) {
      try {
        const status = await getScenarioStatus(scenarioId);
        setStatuses((prev) => ({
          ...prev,
          [scenarioId]: { running: status.running, loading: false },
        }));

        // Load preview links if running
        if (status.running) {
          await fetchPreviewLinks(scenarioId);
        }
      } catch {
        // Set default status on error
        setStatuses((prev) => ({
          ...prev,
          [scenarioId]: { running: false, loading: false },
        }));
      }
    }
  }, [fetchPreviewLinks]);

  return {
    statuses,
    previewLinks,
    showLogs,
    logs,
    startScenario: handleStart,
    stopScenario: handleStop,
    restartScenario: handleRestart,
    toggleLogs: handleToggleLogs,
    loadStatuses,
  };
}
