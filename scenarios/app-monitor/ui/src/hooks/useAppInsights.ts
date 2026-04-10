import { useEffect, useMemo } from 'react';
import type {
  AppProxyMetadata,
  CompleteDiagnostics,
  CompletenessScore,
  LighthouseHistory,
  LocalhostUsageReport,
} from '@/types';
import {
  APP_INSIGHTS_DEFAULT_STALE_TIME_MS,
  type AppInsightsDataset,
  useAppInsightsStore,
} from '@/state/appInsightsStore';

interface UseAppInsightsOptions {
  preload?: boolean;
  staleTimeMs?: number;
  datasets?: AppInsightsDataset[];
}

interface UseAppInsightsResult {
  diagnostics: CompleteDiagnostics | null;
  diagnosticsLoading: boolean;
  diagnosticsError: string | null;
  lighthouseHistory: LighthouseHistory | null;
  lighthouseLoading: boolean;
  lighthouseError: string | null;
  completeness: CompletenessScore | null;
  completenessLoading: boolean;
  completenessError: string | null;
  proxyMetadata: AppProxyMetadata | null;
  localhostReport: LocalhostUsageReport | null;
  proxyLoading: boolean;
  proxyError: string | null;
  refetchDiagnostics: () => Promise<void>;
  refetchLighthouse: () => Promise<void>;
  refetchCompleteness: () => Promise<void>;
  refetchProxy: () => Promise<void>;
  prefetch: () => Promise<void>;
}

const DEFAULT_DATASETS: AppInsightsDataset[] = ['diagnostics', 'lighthouse', 'completeness', 'proxy'];

const normalizeAppId = (appId: string | null | undefined): string | null => {
  if (!appId) {
    return null;
  }
  const trimmed = appId.trim();
  return trimmed.length > 0 ? trimmed : null;
};

const createNoopAsync = async (): Promise<void> => {};

export function useAppInsights(
  appId: string | null | undefined,
  options: UseAppInsightsOptions = {},
): UseAppInsightsResult {
  const {
    preload = true,
    staleTimeMs = APP_INSIGHTS_DEFAULT_STALE_TIME_MS,
    datasets,
  } = options;
  const normalizedDatasets = useMemo(
    () => datasets ?? DEFAULT_DATASETS,
    [datasets],
  );

  const normalizedAppId = useMemo(() => normalizeAppId(appId), [appId]);
  const entry = useAppInsightsStore((state) => (
    normalizedAppId ? state.byAppId[normalizedAppId] : undefined
  ));
  const prefetchFromStore = useAppInsightsStore((state) => state.prefetch);
  const fetchDiagnostics = useAppInsightsStore((state) => state.fetchDiagnostics);
  const fetchLighthouse = useAppInsightsStore((state) => state.fetchLighthouse);
  const fetchCompleteness = useAppInsightsStore((state) => state.fetchCompleteness);
  const fetchProxy = useAppInsightsStore((state) => state.fetchProxy);

  useEffect(() => {
    if (!preload || !normalizedAppId) {
      return;
    }
    void prefetchFromStore(normalizedAppId, { datasets: normalizedDatasets, staleTimeMs });
  }, [normalizedDatasets, normalizedAppId, preload, prefetchFromStore, staleTimeMs]);

  const callForApp = (fn: (nextAppId: string, opts?: { force?: boolean; staleTimeMs?: number }) => Promise<void>) => (
    normalizedAppId
      ? fn(normalizedAppId, { force: true, staleTimeMs })
      : createNoopAsync()
  );

  return {
    diagnostics: entry?.diagnostics.data ?? null,
    diagnosticsLoading: entry?.diagnostics.loading ?? false,
    diagnosticsError: entry?.diagnostics.error ?? null,
    lighthouseHistory: entry?.lighthouse.data ?? null,
    lighthouseLoading: entry?.lighthouse.loading ?? false,
    lighthouseError: entry?.lighthouse.error ?? null,
    completeness: entry?.completeness.data ?? null,
    completenessLoading: entry?.completeness.loading ?? false,
    completenessError: entry?.completeness.error ?? null,
    proxyMetadata: entry?.proxy.data?.proxyMetadata ?? null,
    localhostReport: entry?.proxy.data?.localhostReport ?? null,
    proxyLoading: entry?.proxy.loading ?? false,
    proxyError: entry?.proxy.error ?? null,
    refetchDiagnostics: () => callForApp(fetchDiagnostics),
    refetchLighthouse: () => callForApp(fetchLighthouse),
    refetchCompleteness: () => callForApp(fetchCompleteness),
    refetchProxy: () => callForApp(fetchProxy),
    prefetch: () => (
      normalizedAppId
        ? prefetchFromStore(normalizedAppId, { datasets: normalizedDatasets, force: true, staleTimeMs })
        : createNoopAsync()
    ),
  };
}
