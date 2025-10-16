import { useCallback, useEffect, useRef, useState, type SetStateAction } from 'react';
import { SampleData, type ProcessorSettings } from '../data/sampleData';
import { patchProcessorState, type ProcessorEndpointPayload } from '../services/issues';

interface UseProcessorSettingsManagerOptions {
  apiBaseUrl: string;
  onError?: (message: string) => void;
}

interface UseProcessorSettingsManagerResult {
  processorSettings: ProcessorSettings;
  processorError: string | null;
  updateProcessorSettings: (updater: SetStateAction<ProcessorSettings>) => void;
  toggleProcessorActive: () => void;
  clearProcessorError: () => void;
  reportProcessorError: (message: string) => void;
  applyServerSnapshot: (response: ProcessorEndpointPayload) => void;
  applyRealtimeUpdate: (payload: Record<string, unknown>) => void;
  issuesProcessed: number;
  issuesRemaining: number | string;
}

function mapProcessorState(
  source: Record<string, unknown>,
  fallback: ProcessorSettings,
): ProcessorSettings {
  return {
    active: typeof source.active === 'boolean' ? (source.active as boolean) : fallback.active,
    concurrentSlots:
      typeof source.concurrent_slots === 'number'
        ? (source.concurrent_slots as number)
        : fallback.concurrentSlots,
    refreshInterval:
      typeof source.refresh_interval === 'number'
        ? (source.refresh_interval as number)
        : fallback.refreshInterval,
    maxIssues: typeof source.max_issues === 'number' ? (source.max_issues as number) : fallback.maxIssues,
    maxIssuesDisabled:
      typeof source.max_issues_disabled === 'boolean'
        ? (source.max_issues_disabled as boolean)
        : fallback.maxIssuesDisabled,
  } satisfies ProcessorSettings;
}

export function useProcessorSettingsManager({
  apiBaseUrl,
  onError,
}: UseProcessorSettingsManagerOptions): UseProcessorSettingsManagerResult {
  const [processorSettings, setProcessorSettings] = useState<ProcessorSettings>(SampleData.processor);
  const [processorError, setProcessorError] = useState<string | null>(null);
  const [issuesProcessed, setIssuesProcessed] = useState<number>(0);
  const [issuesRemaining, setIssuesRemaining] = useState<number | string>('unlimited');

  const isMountedRef = useRef(true);
  const lastSyncedRef = useRef<ProcessorSettings | null>(null);
  const initializedRef = useRef(false);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const reportProcessorError = useCallback(
    (message: string) => {
      setProcessorError(message);
      onError?.(message);
    },
    [onError],
  );

  const clearProcessorError = useCallback(() => {
    setProcessorError(null);
  }, []);

  const applyServerSnapshot = useCallback(
    (response: ProcessorEndpointPayload) => {
      const state = response.processor;
      if (!state) {
        reportProcessorError('Failed to load automation status.');
        return;
      }

      setProcessorSettings((previous) => {
        const next = mapProcessorState(state, previous);
        lastSyncedRef.current = next;
        return next;
      });

      if (typeof response.issuesProcessed === 'number') {
        setIssuesProcessed(response.issuesProcessed);
      }
      if (response.issuesRemaining !== null && response.issuesRemaining !== undefined) {
        setIssuesRemaining(response.issuesRemaining ?? 'unlimited');
      }
      clearProcessorError();
    },
    [clearProcessorError, reportProcessorError],
  );

  const applyRealtimeUpdate = useCallback(
    (payload: Record<string, unknown>) => {
      setProcessorSettings((previous) => {
        const next = mapProcessorState(payload, previous);
        lastSyncedRef.current = next;
        return next;
      });

      if (typeof payload.issues_processed === 'number') {
        setIssuesProcessed(payload.issues_processed as number);
      }
      if (payload.issues_remaining !== undefined) {
        setIssuesRemaining(payload.issues_remaining ?? 'unlimited');
      }
      clearProcessorError();
    },
    [clearProcessorError],
  );

  useEffect(() => {
    if (!initializedRef.current) {
      initializedRef.current = true;
      if (!lastSyncedRef.current) {
        lastSyncedRef.current = SampleData.processor;
      }
      return;
    }

    const baseline = lastSyncedRef.current ?? SampleData.processor;
    const isUnchanged =
      baseline.active === processorSettings.active &&
      baseline.concurrentSlots === processorSettings.concurrentSlots &&
      baseline.refreshInterval === processorSettings.refreshInterval &&
      baseline.maxIssues === processorSettings.maxIssues &&
      baseline.maxIssuesDisabled === processorSettings.maxIssuesDisabled;

    if (isUnchanged) {
      return;
    }

    const snapshot: ProcessorSettings = { ...processorSettings };
    let cancelled = false;

    const saveProcessorSettings = async () => {
      try {
        const response = await patchProcessorState(apiBaseUrl, {
          active: snapshot.active,
          concurrent_slots: snapshot.concurrentSlots,
          refresh_interval: snapshot.refreshInterval,
          max_issues: snapshot.maxIssues,
          max_issues_disabled: snapshot.maxIssuesDisabled,
        });

        if (!isMountedRef.current || cancelled) {
          return;
        }

        const state = response.processor;
        if (state) {
          setProcessorSettings((previous) => {
            const next = mapProcessorState(state, previous);
            lastSyncedRef.current = next;
            return next;
          });
        } else {
          lastSyncedRef.current = snapshot;
        }

        if (typeof response.issuesProcessed === 'number') {
          setIssuesProcessed(response.issuesProcessed);
        }
        if (response.issuesRemaining !== null && response.issuesRemaining !== undefined) {
          setIssuesRemaining(response.issuesRemaining ?? 'unlimited');
        }

        clearProcessorError();
      } catch (error) {
        if (!isMountedRef.current || cancelled) {
          return;
        }

        console.error('Failed to save processor settings', error);
        const fallback: ProcessorSettings = lastSyncedRef.current
          ? { ...lastSyncedRef.current }
          : { ...SampleData.processor };
        lastSyncedRef.current = fallback;
        setProcessorSettings(fallback);
        reportProcessorError('Failed to save processor settings');
      }
    };

    void saveProcessorSettings();

    return () => {
      cancelled = true;
    };
  }, [
    apiBaseUrl,
    processorSettings.active,
    processorSettings.concurrentSlots,
    processorSettings.maxIssues,
    processorSettings.maxIssuesDisabled,
    processorSettings.refreshInterval,
    clearProcessorError,
    reportProcessorError,
  ]);

  const updateProcessorSettings = useCallback((updater: SetStateAction<ProcessorSettings>) => {
    setProcessorSettings((previous) =>
      typeof updater === 'function'
        ? (updater as (value: ProcessorSettings) => ProcessorSettings)(previous)
        : updater,
    );
  }, []);

  const toggleProcessorActive = useCallback(() => {
    setProcessorSettings((prev) => ({
      ...prev,
      active: !prev.active,
    }));
    clearProcessorError();
  }, [clearProcessorError]);

  return {
    processorSettings,
    processorError,
    updateProcessorSettings,
    toggleProcessorActive,
    clearProcessorError,
    reportProcessorError,
    applyServerSnapshot,
    applyRealtimeUpdate,
    issuesProcessed,
    issuesRemaining,
  };
}
