import { useCallback, useEffect, useRef, useState } from 'react';
import { usePolling } from '../../../shared/hooks/usePolling';
import { extractErrorMessage, toApiError } from '../../../shared/api/apiFetch';
import {
  fetchCapacityOverview,
  fetchCapacityReconciliation,
  fetchCapacityPolicy,
  setCapacityPolicy,
} from '../api';
import type { CapacityOverview, CapacityReconciliation, PolicyLever } from '../types';
import type { APIError } from '../../../types';
import { isAbortError } from '../../../shared/utils/errors';

export interface UseCapacityResult {
  overview: CapacityOverview | null;
  reconciliation: CapacityReconciliation | null;
  policy: PolicyLever[] | null;
  isLoading: boolean;
  isPolicyLoading: boolean;
  error: string | null;
  reconciliationError: string | null;
  policyLoadError: string | null;
  policyError: string | null;
  policyOutcomeUnknown: { key: string; requestedValue: string; currentValue: string | null; readbackError: string | null } | null;
  isSavingPolicy: boolean;
  refresh: () => Promise<void>;
  refreshPolicy: () => Promise<void>;
  savePolicy: (key: string, value: string) => Promise<void>;
  acknowledgePolicyReadback: () => void;
}

/**
 * Loads and polls the capacity governance surface: per-GPU contention + claims
 * (overview), unclaimed-consumer findings (reconcile), and the tunable policy
 * levers. Policy levers load once and update from the successful write response;
 * they are not polled because they change only on explicit operator action.
 */
export function useCapacity(intervalMs = 15_000): UseCapacityResult {
  const [overview, setOverview] = useState<CapacityOverview | null>(null);
  const [reconciliation, setReconciliation] = useState<CapacityReconciliation | null>(null);
  const [policy, setPolicy] = useState<PolicyLever[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isPolicyLoading, setIsPolicyLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reconciliationError, setReconciliationError] = useState<string | null>(null);
  const [policyLoadError, setPolicyLoadError] = useState<string | null>(null);
  const [policyError, setPolicyError] = useState<string | null>(null);
  const [policyOutcomeUnknown, setPolicyOutcomeUnknown] = useState<UseCapacityResult['policyOutcomeUnknown']>(null);
  const [isSavingPolicy, setIsSavingPolicy] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const policyAbortRef = useRef<AbortController | null>(null);

  const refresh = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setIsLoading(true);
    const [overviewResult, reconciliationResult] = await Promise.allSettled([
      fetchCapacityOverview(controller.signal),
      fetchCapacityReconciliation(controller.signal),
    ]);
    if (controller.signal.aborted) return;
    if (overviewResult.status === 'fulfilled') {
      setOverview(overviewResult.value);
      setError(null);
    } else if (!isAbortError(overviewResult.reason)) {
      setError(extractErrorMessage(overviewResult.reason, 'Failed to load capacity state'));
    }
    if (reconciliationResult.status === 'fulfilled') {
      setReconciliation(reconciliationResult.value);
      setReconciliationError(null);
    } else if (!isAbortError(reconciliationResult.reason)) {
      setReconciliationError(extractErrorMessage(reconciliationResult.reason, 'Failed to reconcile capacity state'));
    }
    setIsLoading(false);
  }, []);

  const refreshPolicy = useCallback(async () => {
    policyAbortRef.current?.abort();
    const controller = new AbortController();
    policyAbortRef.current = controller;
    setIsPolicyLoading(true);
    setPolicyLoadError(null);
    try {
      const policyData = await fetchCapacityPolicy(controller.signal);
      if (controller.signal.aborted) return;
      setPolicy(policyData);
      setPolicyOutcomeUnknown(current => current ? {
        ...current,
        currentValue: policyData.find(lever => lever.key === current.key)?.value ?? null,
        readbackError: policyData.some(lever => lever.key === current.key)
          ? null
          : 'The policy response did not include this setting.',
      } : current);
      setPolicyLoadError(null);
    } catch (err) {
      if (isAbortError(err)) return;
      const message = extractErrorMessage(err, 'Failed to load capacity policy');
      setPolicyLoadError(message);
      setPolicyOutcomeUnknown(current => current ? { ...current, readbackError: message } : current);
    } finally {
      if (!controller.signal.aborted) setIsPolicyLoading(false);
    }
  }, []);

  const savePolicy = useCallback(async (key: string, value: string) => {
    if (policyOutcomeUnknown) return;
    setIsSavingPolicy(true);
    setPolicyError(null);
    try {
      const updated = await setCapacityPolicy(key, value);
      setPolicy(updated);
      setPolicyOutcomeUnknown(null);
    } catch (err) {
      const failure: APIError = toApiError(err);
      const message = extractErrorMessage(failure, `Failed to update ${key}`);
      if (failure.detail?.code === 'validation') {
        setPolicyError(message);
      } else {
        setPolicyOutcomeUnknown({ key, requestedValue: value, currentValue: null, readbackError: 'Checking current policy…' });
        setPolicyError(`The update response was not confirmed: ${message}`);
        await refreshPolicy();
      }
    } finally {
      setIsSavingPolicy(false);
    }
  }, [policyOutcomeUnknown, refreshPolicy]);

  const acknowledgePolicyReadback = useCallback(() => {
    if (!policyOutcomeUnknown || policyOutcomeUnknown.readbackError) return;
    setPolicyOutcomeUnknown(null);
    setPolicyError(null);
  }, [policyOutcomeUnknown]);

  useEffect(() => {
    void refresh();
    void refreshPolicy();
    return () => {
      abortRef.current?.abort();
      policyAbortRef.current?.abort();
    };
  }, [refresh, refreshPolicy]);

  usePolling(refresh, intervalMs, true);

  return {
    overview,
    reconciliation,
    policy,
    isLoading,
    isPolicyLoading,
    error,
    reconciliationError,
    policyLoadError,
    policyError,
    policyOutcomeUnknown,
    isSavingPolicy,
    refresh,
    refreshPolicy,
    savePolicy,
    acknowledgePolicyReadback,
  };
}
