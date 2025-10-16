import { useCallback, useEffect, useMemo, useRef, useState, type MutableRefObject } from 'react';
import { AppSettings, SampleData } from '../data/sampleData';
import type { AgentSettings, DashboardStats, Issue, ProcessorSettings } from '../data/sampleData';
import { apiJsonRequest, apiVoidRequest } from '../utils/api';
import {
  type ApiIssue,
  type ApiStatsPayload,
  buildDashboardStats,
  transformIssue,
} from '../utils/issues';
import type { SnackbarTone } from '../components/Snackbar';
import {
  agentSettingsSnapshotsEqual,
  buildAgentSettingsSnapshot,
  type AgentSettingsSnapshot,
} from '../utils/issueHelpers';

const DEFAULT_STATS: DashboardStats = {
  totalIssues: 0,
  openIssues: 0,
  inProgress: 0,
  completedToday: 0,
  priorityBreakdown: {
    Critical: 0,
    High: 0,
    Medium: 0,
    Low: 0,
  },
  statusTrend: buildDashboardStats([], null).statusTrend,
};

export interface RateLimitStatusPayload {
  rate_limited: boolean;
  rate_limited_count: number;
  reset_time: string;
  seconds_until_reset: number;
  rate_limit_agent: string;
}

interface UseIssueTrackerDataOptions {
  apiBaseUrl: string;
  issueFetchLimit: number;
  isMountedRef: MutableRefObject<boolean>;
  showSnackbar: (message: string, tone?: SnackbarTone) => void;
}

export function useIssueTrackerData({
  apiBaseUrl,
  issueFetchLimit,
  isMountedRef,
  showSnackbar,
}: UseIssueTrackerDataOptions) {
  const [issues, setIssues] = useState<Issue[]>([]);
  const [dashboardStats, setDashboardStats] = useState<DashboardStats>(DEFAULT_STATS);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [processorError, setProcessorError] = useState<string | null>(null);
  const [processorSettings, setProcessorSettings] = useState<ProcessorSettings>(SampleData.processor);
  const [issuesProcessed, setIssuesProcessed] = useState<number>(0);
  const [issuesRemaining, setIssuesRemaining] = useState<number | string>('unlimited');
  const [rateLimitStatus, setRateLimitStatus] = useState<RateLimitStatusPayload | null>(null);
  const [agentSettings, setAgentSettings] = useState<AgentSettings>(AppSettings.agent);

  const lastSyncedProcessorSettingsRef = useRef<ProcessorSettings | null>(null);
  const lastSyncedAgentSettingsRef = useRef<AgentSettingsSnapshot | null>(null);
  const processorSettingsInitializedRef = useRef(false);
  const agentSettingsInitializedRef = useRef(false);

  const buildApiUrl = useCallback((path: string) => `${apiBaseUrl}${path}`, [apiBaseUrl]);

  const handleProcessorError = useCallback(
    (message: string) => {
      setProcessorError(message);
      showSnackbar(message, 'error');
    },
    [showSnackbar],
  );

  const fetchAllData = useCallback(async () => {
    if (isMountedRef.current) {
      setLoading(true);
      setLoadError(null);
      setProcessorError(null);
    }

    let mappedIssues: Issue[] = [];

    try {
      const rawIssues = await apiJsonRequest<ApiIssue[]>(buildApiUrl(`/issues?limit=${issueFetchLimit}`), {
        selector: (payload) => {
          const issuesPayload = (payload as { data?: { issues?: ApiIssue[] } } | null | undefined)?.data?.issues;
          return Array.isArray(issuesPayload) ? issuesPayload : [];
        },
      });
      mappedIssues = rawIssues.map((issue) => transformIssue(issue, { apiBaseUrl }));
      if (isMountedRef.current) {
        setIssues(mappedIssues);
      }
    } catch (error) {
      console.error('Failed to load issues', error);
      if (isMountedRef.current) {
        setLoadError('Failed to load issues from the API.');
        setLoading(false);
      }
      return;
    }

    let statsFromApi: ApiStatsPayload | null = null;
    try {
      const stats = await apiJsonRequest<ApiStatsPayload | null>(buildApiUrl('/stats'), {
        selector: (payload) => {
          const raw = (payload as { data?: { stats?: Record<string, unknown> } } | null | undefined)?.data?.stats;
          if (!raw) {
            return null;
          }
          return {
            totalIssues: typeof raw.total_issues === 'number' ? raw.total_issues : undefined,
            openIssues: typeof raw.open_issues === 'number' ? raw.open_issues : undefined,
            inProgress: typeof raw.in_progress === 'number' ? raw.in_progress : undefined,
            completedToday: typeof raw.completed_today === 'number' ? raw.completed_today : undefined,
          } satisfies ApiStatsPayload;
        },
      });
      if (stats) {
        statsFromApi = stats;
      }
    } catch (error) {
      console.warn('Failed to load stats', error);
    }

    const computedStats = buildDashboardStats(mappedIssues, statsFromApi);
    if (isMountedRef.current) {
      setDashboardStats(computedStats);
    }

    try {
      const payload = await apiJsonRequest<Record<string, unknown>>(buildApiUrl('/automation/processor'));
      const data = (payload?.data as Record<string, unknown> | undefined) ?? undefined;
      const state = (data?.processor ?? payload?.processor) as Record<string, unknown> | undefined;

      if (state && isMountedRef.current) {
        setProcessorSettings((prev) => {
          const next: ProcessorSettings = {
            active: typeof state.active === 'boolean' ? (state.active as boolean) : prev.active,
            concurrentSlots:
              typeof state.concurrent_slots === 'number'
                ? (state.concurrent_slots as number)
                : prev.concurrentSlots,
            refreshInterval:
              typeof state.refresh_interval === 'number'
                ? (state.refresh_interval as number)
                : prev.refreshInterval,
            maxIssues: typeof state.max_issues === 'number' ? (state.max_issues as number) : prev.maxIssues,
            maxIssuesDisabled:
              typeof state.max_issues_disabled === 'boolean'
                ? (state.max_issues_disabled as boolean)
                : prev.maxIssuesDisabled,
          };
          lastSyncedProcessorSettingsRef.current = next;
          return next;
        });

        if (typeof data?.issues_processed === 'number') {
          setIssuesProcessed(data.issues_processed as number);
        }
        if (data?.issues_remaining !== undefined) {
          setIssuesRemaining(data.issues_remaining as number | string);
        }

        setProcessorError(null);
      } else if (isMountedRef.current) {
        handleProcessorError('Failed to load automation status.');
      }
    } catch (error) {
      console.warn('Failed to load processor settings', error);
      if (isMountedRef.current) {
        handleProcessorError('Failed to load automation status.');
      }
    }

    try {
      const status = await apiJsonRequest<RateLimitStatusPayload | null>(buildApiUrl('/rate-limit-status'), {
        selector: (payload) =>
          ((payload as { data?: RateLimitStatusPayload } | null | undefined)?.data ?? null),
      });
      if (status && isMountedRef.current) {
        setRateLimitStatus(status);
      }
    } catch (error) {
      console.warn('Failed to load rate limit status', error);
    }

    if (isMountedRef.current) {
      setLoading(false);
    }
  }, [apiBaseUrl, buildApiUrl, handleProcessorError, isMountedRef, issueFetchLimit]);

  useEffect(() => {
    const fetchAgentBackendSettings = async () => {
      try {
        const payload = await apiJsonRequest<Record<string, unknown>>(buildApiUrl('/agent/settings'));
        const settingsData = (payload?.data ?? payload) as Record<string, unknown> | undefined;
        if (!settingsData || !isMountedRef.current) {
          return;
        }

        setAgentSettings((prev) => {
          const backendData = (settingsData.agent_backend ?? {}) as Record<string, unknown>;
          const providersData = (settingsData.providers ?? {}) as Record<string, any>;

          const rawProvider = typeof backendData.provider === 'string' ? backendData.provider.trim() : '';
          const provider: 'codex' | 'claude-code' =
            rawProvider === 'codex' || rawProvider === 'claude-code'
              ? rawProvider
              : prev.backend?.provider ?? 'codex';

          const autoFallback =
            typeof backendData.auto_fallback === 'boolean'
              ? (backendData.auto_fallback as boolean)
              : prev.backend?.autoFallback ?? true;

          const skipPermissions =
            typeof backendData.skip_permissions === 'boolean'
              ? (backendData.skip_permissions as boolean)
              : prev.skipPermissionChecks;

          const providerConfig = (providersData?.[provider] ?? {}) as Record<string, any>;
          const operationsConfig = (providerConfig?.operations ?? {}) as Record<string, any>;
          const investigateConfig = (operationsConfig?.investigate ?? {}) as Record<string, any>;
          const fixConfig = (operationsConfig?.fix ?? {}) as Record<string, any>;

          const resolveNumber = (value: unknown): number | undefined =>
            typeof value === 'number' && Number.isFinite(value) ? value : undefined;

          const resolveString = (value: unknown): string | undefined =>
            typeof value === 'string' && value.trim().length > 0 ? value : undefined;

          const selectedMaxTurns =
            resolveNumber(investigateConfig.max_turns) ??
            resolveNumber(fixConfig.max_turns) ??
            prev.maximumTurns;

          const timeoutSeconds =
            resolveNumber(investigateConfig.timeout_seconds) ??
            resolveNumber(fixConfig.timeout_seconds) ??
            prev.taskTimeout * 60;

          const allowedToolsRaw =
            resolveString(investigateConfig.allowed_tools) ??
            resolveString(fixConfig.allowed_tools);

          const parsedAllowedTools =
            allowedToolsRaw
              ?.split(',')
              .map((tool: string) => tool.trim())
              .filter((tool: string) => tool.length > 0) ?? prev.allowedTools;

          const next: AgentSettings = {
            ...prev,
            backend: {
              provider,
              autoFallback,
            },
            maximumTurns: selectedMaxTurns,
            taskTimeout: Math.max(5, Math.round(timeoutSeconds / 60)),
            allowedTools: parsedAllowedTools.length > 0 ? parsedAllowedTools : prev.allowedTools,
            skipPermissionChecks: skipPermissions,
          };

          const allowedToolsKey = next.allowedTools
            .map((tool) => tool.trim())
            .filter((tool) => tool.length > 0)
            .join(',');
          lastSyncedAgentSettingsRef.current = buildAgentSettingsSnapshot(next, allowedToolsKey);

          return next;
        });
      } catch (error) {
        console.warn('Failed to load agent backend settings', error);
      }
    };
    void fetchAgentBackendSettings();
  }, [buildApiUrl, isMountedRef]);

  const agentAllowedToolsSignature = useMemo(
    () =>
      agentSettings.allowedTools
        .map((tool) => tool.trim())
        .filter((tool) => tool.length > 0)
        .join(','),
    [agentSettings.allowedTools],
  );

  useEffect(() => {
    if (!processorSettingsInitializedRef.current) {
      processorSettingsInitializedRef.current = true;
      return;
    }

    const lastSynced = lastSyncedProcessorSettingsRef.current;
    if (!lastSynced) {
      return;
    }

    const isUnchanged =
      lastSynced.active === processorSettings.active &&
      lastSynced.concurrentSlots === processorSettings.concurrentSlots &&
      lastSynced.refreshInterval === processorSettings.refreshInterval &&
      lastSynced.maxIssues === processorSettings.maxIssues &&
      lastSynced.maxIssuesDisabled === processorSettings.maxIssuesDisabled;

    if (isUnchanged) {
      return;
    }

    const saveProcessorSettings = async () => {
      try {
        const snapshot: ProcessorSettings = { ...processorSettings };
        await apiVoidRequest(buildApiUrl('/automation/processor'), {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            active: processorSettings.active,
            concurrent_slots: processorSettings.concurrentSlots,
            refresh_interval: processorSettings.refreshInterval,
            max_issues: processorSettings.maxIssues,
            max_issues_disabled: processorSettings.maxIssuesDisabled,
          }),
        });

        lastSyncedProcessorSettingsRef.current = snapshot;
      } catch (error) {
        console.error('Failed to save processor settings', error);
        showSnackbar('Failed to save processor settings', 'error');
      }
    };

    void saveProcessorSettings();
  }, [
    buildApiUrl,
    processorSettings.active,
    processorSettings.concurrentSlots,
    processorSettings.refreshInterval,
    processorSettings.maxIssues,
    processorSettings.maxIssuesDisabled,
    showSnackbar,
  ]);

  useEffect(() => {
    const backend = agentSettings.backend;
    if (!backend) {
      return;
    }

    const snapshot = buildAgentSettingsSnapshot(agentSettings, agentAllowedToolsSignature);

    if (!agentSettingsInitializedRef.current) {
      agentSettingsInitializedRef.current = true;
      if (!lastSyncedAgentSettingsRef.current) {
        lastSyncedAgentSettingsRef.current = snapshot;
      }
      return;
    }

    if (agentSettingsSnapshotsEqual(lastSyncedAgentSettingsRef.current, snapshot)) {
      return;
    }

    const payload: Record<string, unknown> = {
      provider: snapshot.provider,
      auto_fallback: snapshot.autoFallback,
      max_turns: snapshot.maximumTurns,
      timeout_seconds: snapshot.taskTimeoutMinutes * 60,
      skip_permissions: snapshot.skipPermissionChecks,
    };

    if (snapshot.allowedToolsKey.length > 0 || agentSettings.allowedTools.length === 0) {
      payload.allowed_tools = snapshot.allowedToolsKey;
    }

    const saveAgentBackendSettings = async () => {
      try {
        await apiVoidRequest(buildApiUrl('/agent/settings'), {
          method: 'PATCH',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(payload),
        });

        lastSyncedAgentSettingsRef.current = snapshot;
      } catch (error) {
        console.error('Failed to save agent backend settings', error);
        showSnackbar('Failed to save agent settings', 'error');
      }
    };

    void saveAgentBackendSettings();
  }, [
    agentAllowedToolsSignature,
    agentSettings,
    buildApiUrl,
    showSnackbar,
  ]);

  const toggleProcessorActive = useCallback(async () => {
    const previousActive = processorSettings.active;
    const nextActive = !previousActive;

    setProcessorSettings((prev) => ({
      ...prev,
      active: nextActive,
    }));

    setProcessorError(null);

    try {
      const payload = await apiJsonRequest<Record<string, unknown>>(buildApiUrl('/automation/processor'), {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ active: nextActive }),
      });
      const payloadData = (payload?.data as Record<string, unknown> | undefined) ?? undefined;
      const state = (payloadData?.processor ?? payload?.processor) as Record<string, unknown> | undefined;
      if (state && isMountedRef.current) {
        setProcessorSettings((prev) => {
          const next: ProcessorSettings = {
            active: typeof state.active === 'boolean' ? state.active : prev.active,
            concurrentSlots:
              typeof state.concurrent_slots === 'number'
                ? state.concurrent_slots
                : prev.concurrentSlots,
            refreshInterval:
              typeof state.refresh_interval === 'number'
                ? state.refresh_interval
                : prev.refreshInterval,
            maxIssues:
              typeof state.max_issues === 'number'
                ? state.max_issues
                : prev.maxIssues,
            maxIssuesDisabled:
              typeof state.max_issues_disabled === 'boolean'
                ? state.max_issues_disabled
                : prev.maxIssuesDisabled,
          };
          lastSyncedProcessorSettingsRef.current = next;
          return next;
        });
        setProcessorError(null);
      }
    } catch (error) {
      console.error('Failed to update processor state', error);
      if (isMountedRef.current) {
        setProcessorSettings((prev) => ({
          ...prev,
          active: previousActive,
        }));
        handleProcessorError('Failed to update automation status.');
      }
    }
  }, [buildApiUrl, handleProcessorError, isMountedRef, processorSettings.active]);

  return {
    issues,
    setIssues,
    dashboardStats,
    setDashboardStats,
    loading,
    loadError,
    processorError,
    processorSettings,
    setProcessorSettings,
    issuesProcessed,
    issuesRemaining,
    rateLimitStatus,
    agentSettings,
    setAgentSettings,
    fetchAllData,
    handleProcessorError,
    toggleProcessorActive,
  };
}
