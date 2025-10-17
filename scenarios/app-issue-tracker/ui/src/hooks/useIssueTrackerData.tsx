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
  createIssue as createIssueRequest,
  deleteIssue as deleteIssueRequest,
  listIssues,
  updateIssueStatus as updateIssueStatusRequest,
  type RateLimitStatusPayload,
} from '../services/issues';
export type { RateLimitStatusPayload } from '../services/issues';
import {
  type ApiIssue,
  type ApiStatsPayload,
  buildDashboardStats,
  transformIssue,
} from '../utils/issues';
import type { SnackVariant } from '../notifications/snackBus';
import type { CreateIssueInput } from '../types/issueCreation';
import { useProcessorSettingsManager } from './useProcessorSettingsManager';
import { useAgentSettingsManager } from './useAgentSettingsManager';
import {
  useIssueRealtimeSync,
  type RunningProcessMap,
  type ConnectionStatus,
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
  processorError: string | null;
  processorSettings: ProcessorSettings;
  updateProcessorSettings: (updater: React.SetStateAction<ProcessorSettings>) => void;
  issuesProcessed: number;
  issuesRemaining: number | string;
  rateLimitStatus: RateLimitStatusPayload | null;
  agentSettings: AgentSettings;
  updateAgentSettings: (updater: React.SetStateAction<AgentSettings>) => void;
  fetchAllData: () => Promise<void>;
  toggleProcessorActive: () => void;
  createIssue: (input: CreateIssueInput) => Promise<string | null>;
  updateIssueStatus: (issueId: string, nextStatus: IssueStatus) => Promise<void>;
  deleteIssue: (issueId: string) => Promise<void>;
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

  const { agentSettings, updateAgentSettings } = useAgentSettingsManager({
    apiBaseUrl,
    onSaveError: (message) => showSnackbar(message, 'error'),
  });

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

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
    }

    let mappedIssues: Issue[] = [];

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
      console.warn('Failed to load stats', error);
    }

    if (isMountedRef.current) {
      lastStatsFromApiRef.current = statsFromApi;
      updateIssuesState(() => mappedIssues, { invalidateRemoteStats: false });
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
      console.warn('Failed to load processor settings', error);
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
      console.warn('Failed to load rate limit status', error);
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
  });

  const contextValue = useMemo<IssueTrackerDataContextValue>(
    () => ({
      issues,
      dashboardStats,
      loading,
      loadError,
      processorError,
      processorSettings,
      updateProcessorSettings,
      issuesProcessed,
      issuesRemaining,
      rateLimitStatus,
      agentSettings,
      updateAgentSettings,
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
