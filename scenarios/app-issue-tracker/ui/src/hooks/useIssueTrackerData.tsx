import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import type { AgentSettings, DashboardStats, Issue, IssueStatus, ProcessorSettings } from '../data/sampleData';
import {
  fetchIssueStats,
  fetchProcessorState,
  fetchRateLimitStatus,
  fetchIssueStatuses,
  fetchRunningProcesses,
  createIssue as createIssueRequest,
  deleteIssue as deleteIssueRequest,
  listIssues,
  updateIssueStatus as updateIssueStatusRequest,
  updateIssue as updateIssueRequest,
  stopRunningProcess,
  type RateLimitStatusPayload,
  type IssueStatusMetadata,
  type RunningProcessPayload,
} from '../services/issues';
export type { RateLimitStatusPayload } from '../services/issues';
import {
  type ApiIssue,
  type ApiStatsPayload,
  buildDashboardStats,
  transformIssue,
  getFallbackStatuses,
  setValidStatuses,
  formatStatusLabel,
} from '../utils/issues';
import type { SnackVariant } from '../notifications/snackBus';
import type { CreateIssueInput, UpdateIssueInput } from '../types/issueCreation';
import { useProcessorSettingsManager } from './useProcessorSettingsManager';
import { useAgentSettingsManager, type SettingsConstraints } from './useAgentSettingsManager';
import {
  useIssueRealtimeSync,
  type RunningProcessMap,
  type ConnectionStatus,
  type RunningProcessSeed,
} from './useIssueRealtimeSync';

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

const DEFAULT_STATUS_CATALOG: IssueStatusMetadata[] = getFallbackStatuses().map((status) => ({
  id: status,
  label: formatStatusLabel(status),
}));

interface IssueTrackerDataProviderProps {
  apiBaseUrl: string;
  issueFetchLimit: number;
  showSnackbar: (message: string, tone?: SnackVariant) => void;
  children: ReactNode;
}

interface IssueTrackerDataContextValue {
  issues: Issue[];
  dashboardStats: DashboardStats;
  loading: boolean;
  loadError: string | null;
  statusCatalog: IssueStatusMetadata[];
  validStatuses: IssueStatus[];
  processorError: string | null;
  processorSettings: ProcessorSettings;
  updateProcessorSettings: (updater: React.SetStateAction<ProcessorSettings>) => void;
  issuesProcessed: number;
  issuesRemaining: number | string;
  rateLimitStatus: RateLimitStatusPayload | null;
  agentSettings: AgentSettings;
  updateAgentSettings: (updater: React.SetStateAction<AgentSettings>) => void;
  agentConstraints: SettingsConstraints | null;
  fetchAllData: () => Promise<void>;
  toggleProcessorActive: () => void;
  createIssue: (input: CreateIssueInput) => Promise<string | null>;
  updateIssueStatus: (issueId: string, nextStatus: IssueStatus) => Promise<void>;
  updateIssueDetails: (input: UpdateIssueInput) => Promise<void>;
  deleteIssue: (issueId: string) => Promise<void>;
  stopAgent: (issueId: string) => Promise<void>;
  runningProcesses: RunningProcessMap;
  connectionStatus: ConnectionStatus;
  websocketError: Error | null;
  reconnectAttempts: number;
}

const IssueTrackerDataContext = createContext<IssueTrackerDataContextValue | null>(null);

export function IssueTrackerDataProvider({
  apiBaseUrl,
  issueFetchLimit,
  showSnackbar,
  children,
}: IssueTrackerDataProviderProps) {
  const isMountedRef = useRef(true);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [dashboardStats, setDashboardStats] = useState<DashboardStats>(DEFAULT_STATS);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [rateLimitStatus, setRateLimitStatus] = useState<RateLimitStatusPayload | null>(null);
  const [realtimeReady, setRealtimeReady] = useState(false);
  const [statusCatalog, setStatusCatalog] = useState<IssueStatusMetadata[]>(DEFAULT_STATUS_CATALOG);
  const [initialRunningProcesses, setInitialRunningProcesses] = useState<RunningProcessSeed | null>(null);

  const validStatuses = useMemo<IssueStatus[]>(
    () => statusCatalog.map((status) => status.id as IssueStatus),
    [statusCatalog],
  );

  const lastStatsFromApiRef = useRef<ApiStatsPayload | null>(null);

  const {
    processorSettings,
    processorError,
    updateProcessorSettings,
    toggleProcessorActive,
    reportProcessorError,
    clearProcessorError,
    applyServerSnapshot: applyProcessorSnapshot,
    applyRealtimeUpdate: applyProcessorRealtimeUpdate,
    issuesProcessed,
    issuesRemaining,
  } = useProcessorSettingsManager({
    apiBaseUrl,
    onError: (message) => showSnackbar(message, 'error'),
  });

  const { agentSettings, updateAgentSettings, constraints: agentConstraints } = useAgentSettingsManager({
    apiBaseUrl,
    onSaveError: (message) => showSnackbar(message, 'error'),
  });

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadStatuses() {
      try {
        const fetched = await fetchIssueStatuses(apiBaseUrl);
        const nextCatalog = fetched.length > 0 ? fetched : DEFAULT_STATUS_CATALOG;
        if (!cancelled) {
          setStatusCatalog(nextCatalog);
          setValidStatuses(nextCatalog.map((status) => status.id));
        }
      } catch (error) {
        // Silently fall back to default status catalog
        if (!cancelled) {
          setStatusCatalog(DEFAULT_STATUS_CATALOG);
          setValidStatuses(DEFAULT_STATUS_CATALOG.map((status) => status.id));
        }
      }
    }

    loadStatuses();

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl]);

  const updateIssuesState = useCallback(
    (
      updater: (previous: Issue[]) => Issue[],
      options: { invalidateRemoteStats?: boolean } = {},
    ) => {
      const { invalidateRemoteStats = true } = options;
      setIssues((previous) => {
        const next = updater(previous);
        if (invalidateRemoteStats) {
          lastStatsFromApiRef.current = null;
        }
        setDashboardStats(buildDashboardStats(next, lastStatsFromApiRef.current));
        return next;
      });
    },
    [setDashboardStats],
  );

  const fetchAllData = useCallback(async () => {
    if (isMountedRef.current) {
      setLoading(true);
      setLoadError(null);
      clearProcessorError();
      setRealtimeReady(false);
      setInitialRunningProcesses(null);
    }

    let mappedIssues: Issue[] = [];
    let runningProcessesSnapshot: RunningProcessPayload[] = [];

    try {
      const rawIssues = await listIssues(apiBaseUrl, issueFetchLimit);
      mappedIssues = rawIssues.map((issue) => transformIssue(issue, { apiBaseUrl }));
    } catch (error) {
      console.error('Failed to load issues', error);
      if (isMountedRef.current) {
        setLoadError('Failed to load issues from the API.');
        setLoading(false);
        setRealtimeReady(false);
      }
      return;
    }

    let statsFromApi: ApiStatsPayload | null = null;
    try {
      const stats = await fetchIssueStats(apiBaseUrl);
      if (stats) {
        statsFromApi = stats;
      }
    } catch (error) {
      // Stats are supplementary, continue without them
    }

    try {
      runningProcessesSnapshot = await fetchRunningProcesses(apiBaseUrl);
    } catch (error) {
      // Running processes are supplementary, continue without them
      runningProcessesSnapshot = [];
    }

    if (isMountedRef.current) {
      lastStatsFromApiRef.current = statsFromApi;
      updateIssuesState(() => mappedIssues, { invalidateRemoteStats: false });
      setInitialRunningProcesses({
        processes: runningProcessesSnapshot,
        version: Date.now(),
      });
      setRealtimeReady(true);
    }

    try {
      const processorResponse = await fetchProcessorState(apiBaseUrl);
      if (isMountedRef.current) {
        if (processorResponse.processor) {
          applyProcessorSnapshot(processorResponse);
        } else {
          reportProcessorError('Failed to load automation status.');
        }
      }
    } catch (error) {
      // Report processor error to UI without console noise
      if (isMountedRef.current) {
        reportProcessorError('Failed to load automation status.');
      }
    }

    try {
      const status = await fetchRateLimitStatus(apiBaseUrl);
      if (isMountedRef.current) {
        setRateLimitStatus(status ?? null);
      }
    } catch (error) {
      // Rate limit status is optional, continue without it
      if (isMountedRef.current) {
        setRateLimitStatus(null);
      }
    }

    if (isMountedRef.current) {
      setLoading(false);
    }
  }, [
    apiBaseUrl,
    applyProcessorSnapshot,
    clearProcessorError,
    issueFetchLimit,
    reportProcessorError,
    updateIssuesState,
  ]);

  const {
    runningProcesses,
    removeProcess,
    connectionStatus,
    websocketError,
    reconnectAttempts,
  } = useIssueRealtimeSync({
    apiBaseUrl,
    updateIssuesState,
    applyProcessorRealtimeUpdate,
    setRateLimitStatus,
    reportProcessorError,
    enabled: realtimeReady,
    initialRunningProcesses,
  });

  const contextValue = useMemo<IssueTrackerDataContextValue>(
    () => ({
      issues,
      dashboardStats,
      loading,
      loadError,
      statusCatalog,
      validStatuses,
      processorError,
      processorSettings,
      updateProcessorSettings,
      issuesProcessed,
      issuesRemaining,
      rateLimitStatus,
      agentSettings,
      updateAgentSettings,
      agentConstraints,
      fetchAllData,
      toggleProcessorActive,
      createIssue: async (input: CreateIssueInput) => {
        const { issueId, issue } = await createIssueRequest(apiBaseUrl, input);
        if (issue) {
          const transformed = transformIssue(issue, { apiBaseUrl });
          updateIssuesState((previous) => {
            const index = previous.findIndex((candidate) => candidate.id === transformed.id);
            if (index >= 0) {
              const next = [...previous];
              next[index] = transformed;
              return next;
            }
            return [...previous, transformed];
          });
        } else {
          await fetchAllData();
        }
        return issueId;
      },
      updateIssueDetails: async (input: UpdateIssueInput) => {
        const { issue: updatedIssue } = await updateIssueRequest(apiBaseUrl, input);
        if (updatedIssue) {
          const transformed = transformIssue(updatedIssue, { apiBaseUrl });
          updateIssuesState((previous) =>
            previous.map((issue) => (issue.id === transformed.id ? transformed : issue)),
          );
        } else {
          await fetchAllData();
        }
      },
      updateIssueStatus: async (issueId: string, nextStatus: IssueStatus) => {
        const { issue: updatedIssue } = await updateIssueStatusRequest(apiBaseUrl, issueId, nextStatus);
        if (updatedIssue) {
          const transformed = transformIssue(updatedIssue, { apiBaseUrl });
          updateIssuesState((previous) =>
            previous.map((issue) => (issue.id === transformed.id ? transformed : issue)),
          );
        } else {
          updateIssuesState((previous) =>
            previous.map((issue) =>
              issue.id === issueId
                ? {
                    ...issue,
                    status: nextStatus,
                  }
                : issue,
            ),
          );
        }
      },
      deleteIssue: async (issueId: string) => {
        await deleteIssueRequest(apiBaseUrl, issueId);
        updateIssuesState((previous) => previous.filter((issue) => issue.id !== issueId));
        removeProcess(issueId);
      },
      stopAgent: async (issueId: string) => {
        try {
          await stopRunningProcess(apiBaseUrl, issueId);
          removeProcess(issueId);
        } catch (error) {
          console.error('Failed to stop running agent', error);
          showSnackbar('Failed to stop agent. Please try again.', 'error');
        }
      },
      runningProcesses,
      connectionStatus,
      websocketError,
      reconnectAttempts,
    }),
    [
      issues,
      dashboardStats,
      loading,
      loadError,
      statusCatalog,
      validStatuses,
      processorError,
      processorSettings,
      issuesProcessed,
      issuesRemaining,
      rateLimitStatus,
      agentSettings,
      fetchAllData,
      toggleProcessorActive,
      runningProcesses,
      connectionStatus,
      websocketError,
      reconnectAttempts,
      updateIssuesState,
      removeProcess,
      apiBaseUrl,
      showSnackbar,
    ],
  );

  return (
    <IssueTrackerDataContext.Provider value={contextValue}>
      {children}
    </IssueTrackerDataContext.Provider>
  );
}

export function useIssueTrackerData(): IssueTrackerDataContextValue {
  const context = useContext(IssueTrackerDataContext);
  if (!context) {
    throw new Error('useIssueTrackerData must be used within an IssueTrackerDataProvider');
  }
  return context;
}
