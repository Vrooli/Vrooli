import { useCallback, useEffect, useRef, useState } from 'react';
import { usePolling } from '../../../shared/hooks/usePolling';
import { extractErrorMessage } from '../../../shared/api/apiFetch';
import {
  fetchCapacityOverview,
  fetchCapacityReconciliation,
  fetchCapacityPolicy,
  setCapacityPolicy,
} from '../api';
import type { CapacityOverview, CapacityReconciliation, PolicyLever } from '../types';

export interface UseCapacityResult {
  overview: CapacityOverview | null;
  reconciliation: CapacityReconciliation | null;
  policy: PolicyLever[] | null;
  isLoading: boolean;
  error: string | null;
  policyError: string | null;
  isSavingPolicy: boolean;
  refresh: () => Promise<void>;
  savePolicy: (key: string, value: string) => Promise<void>;
}

/**
 * Loads and polls the capacity governance surface: per-GPU contention + claims
 * (overview), unclaimed-consumer findings (reconcile), and the tunable policy
 * levers. Policy levers are fetched once and re-fetched after a successful save
 * rather than polled, since they change only on explicit operator action.
 */
export function useCapacity(intervalMs = 15_000): UseCapacityResult {
  const [overview, setOverview] = useState<CapacityOverview | null>(null);
  const [reconciliation, setReconciliation] = useState<CapacityReconciliation | null>(null);
  const [policy, setPolicy] = useState<PolicyLever[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [policyError, setPolicyError] = useState<string | null>(null);
  const [isSavingPolicy, setIsSavingPolicy] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const refresh = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setIsLoading(true);
    try {
      const [overviewData, reconData, policyData] = await Promise.all([
        fetchCapacityOverview(controller.signal),
        fetchCapacityReconciliation(controller.signal).catch(() => null),
        fetchCapacityPolicy(controller.signal),
      ]);
      if (controller.signal.aborted) return;
      setOverview(overviewData);
      setReconciliation(reconData);
      setPolicy(policyData);
      setError(null);
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      setError(extractErrorMessage(err, 'Failed to load capacity state'));
    } finally {
      if (!controller.signal.aborted) setIsLoading(false);
    }
  }, []);

  const savePolicy = useCallback(async (key: string, value: string) => {
    setIsSavingPolicy(true);
    setPolicyError(null);
    try {
      const updated = await setCapacityPolicy(key, value);
      setPolicy(updated);
    } catch (err) {
      setPolicyError(extractErrorMessage(err, `Failed to update ${key}`));
    } finally {
      setIsSavingPolicy(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    return () => abortRef.current?.abort();
  }, [refresh]);

  usePolling(refresh, intervalMs, true);

  return {
    overview,
    reconciliation,
    policy,
    isLoading,
    error,
    policyError,
    isSavingPolicy,
    refresh,
    savePolicy,
  };
}
