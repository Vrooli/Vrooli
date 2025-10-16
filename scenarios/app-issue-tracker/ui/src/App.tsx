import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useWebSocket } from './hooks/useWebSocket';
import type { WebSocketEvent } from './types/events';
import { Brain, CircleAlert, Clock, Cog, Filter, LayoutDashboard, Loader2, Pause, Play, Plus } from 'lucide-react';
import { IssuesBoard } from './pages/IssuesBoard';
import { IssueDetailsModal } from './components/IssueDetailsModal';
import { IssueBoardToolbar } from './components/IssueBoardToolbar';
import { CreateIssueModal } from './components/CreateIssueModal';
import { MetricsDialog } from './components/MetricsDialog';
import { SettingsDialog } from './components/SettingsDialog';
import { Snackbar, type SnackbarTone } from './components/Snackbar';
import { AppSettings, type DisplaySettings, type Issue, type IssueStatus, type ProcessorSettings } from './data/sampleData';
import './styles/app.css';
import {
  ApiIssue,
  VALID_STATUSES,
  buildDashboardStats,
  normalizeStatus,
  transformIssue,
} from './utils/issues';
import { apiJsonRequest, apiVoidRequest } from './utils/api';
import { type CreateIssueInput, type CreateIssuePrefill, type PriorityFilterValue } from './types/issueCreation';
import { prepareFollowUpPrefill } from './utils/issueHelpers';
import { useIssueTrackerData, type RateLimitStatusPayload } from './hooks/useIssueTrackerData';

const API_BASE_INPUT = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '/api/v1';
const API_BASE_URL = API_BASE_INPUT.endsWith('/') ? API_BASE_INPUT.slice(0, -1) : API_BASE_INPUT;
const ISSUE_FETCH_LIMIT = 200;

interface SnackbarState {
  message: string;
  tone: SnackbarTone;
}
function buildApiUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

function getIssueIdFromLocation(): string | null {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const params = new URLSearchParams(window.location.search);
    const value = params.get('issue');
    if (!value) {
      return null;
    }
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  } catch (error) {
    console.warn('[IssueTracker] Unable to parse issue id from URL', error);
    return null;
  }
}

// Navigation removed - single page app with dialogs

// URL routing simplified - only issues page with query params

function App() {
  const isMountedRef = useRef(true);
  const [metricsDialogOpen, setMetricsDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [wsConnectionStatus, setWsConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
  const [displaySettings, setDisplaySettings] = useState(AppSettings.display);
  const [priorityFilter, setPriorityFilter] = useState<PriorityFilterValue>('all');
  const [appFilter, setAppFilter] = useState<string>('all');
  const [searchFilter, setSearchFilter] = useState<string>('');
  const [hiddenColumns, setHiddenColumns] = useState<IssueStatus[]>(['archived']);
  const [createIssueOpen, setCreateIssueOpen] = useState(false);
  const [createIssuePrefill, setCreateIssuePrefill] = useState<CreateIssuePrefill | null>(null);
  const [issueDetailOpen, setIssueDetailOpen] = useState(false);
  const [focusedIssueId, setFocusedIssueId] = useState<string | null>(() => getIssueIdFromLocation());
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [snackbar, setSnackbar] = useState<SnackbarState | null>(null);
  const [runningProcesses, setRunningProcesses] = useState<Map<string, { agent_id: string; start_time: string; duration?: string }>>(new Map());
  const [followUpLoadingId, setFollowUpLoadingId] = useState<string | null>(null);

  const showSnackbar = useCallback((message: string, tone: SnackbarTone = 'info') => {
    setSnackbar({ message, tone });
  }, []);

  const {
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
  } = useIssueTrackerData({
    apiBaseUrl: API_BASE_URL,
    issueFetchLimit: ISSUE_FETCH_LIMIT,
    isMountedRef,
    showSnackbar,
  });

  // WebSocket event handler
  const handleWebSocketEvent = useCallback((event: WebSocketEvent) => {
    switch (event.type) {
      case 'issue.created':
      case 'issue.updated': {
        const updatedIssue = transformIssue(event.data.issue, { apiBaseUrl: API_BASE_URL });
        setIssues(prev => {
          const index = prev.findIndex(i => i.id === updatedIssue.id);
          if (index >= 0) {
            return prev.map((i, idx) => idx === index ? updatedIssue : i);
          }
          return [...prev, updatedIssue];
        });
        break;
      }

      case 'issue.status_changed': {
        const { issue_id, new_status } = event.data;
        setIssues(prev => prev.map(i =>
          i.id === issue_id ? { ...i, status: normalizeStatus(new_status) } : i
        ));
        break;
      }

      case 'issue.deleted': {
        const { issue_id } = event.data;
        setIssues(prev => prev.filter(i => i.id !== issue_id));
        if (focusedIssueId === issue_id) {
          setFocusedIssueId(null);
          setIssueDetailOpen(false);
        }
        break;
      }

      case 'agent.started': {
        const { issue_id, agent_id, start_time } = event.data;
        setRunningProcesses(prev => {
          const next = new Map(prev);
          next.set(issue_id, { agent_id, start_time });
          return next;
        });
        break;
      }

      case 'agent.completed':
      case 'agent.failed': {
        const { issue_id } = event.data;
        setRunningProcesses(prev => {
          const next = new Map(prev);
          next.delete(issue_id);
          return next;
        });
        break;
      }

      default:
        break;
    }
  }, [focusedIssueId]);

  // WebSocket connection
  const wsUrl = useMemo(() => {
    if (typeof window === 'undefined') return '';

    // If API_BASE_URL is absolute (starts with http/https), extract host from it
    if (API_BASE_INPUT.startsWith('http://') || API_BASE_INPUT.startsWith('https://')) {
      const url = new URL(API_BASE_INPUT);
      const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      const finalUrl = `${wsProtocol}//${url.host}${API_BASE_URL}/ws`;
      console.log('[WebSocket] URL (absolute):', finalUrl);
      return finalUrl;
    }

    // Check if we're behind a proxy/tunnel (non-localhost hostname)
    const hostname = window.location.hostname;
    const isProxied = hostname !== 'localhost' && hostname !== '127.0.0.1' && !hostname.match(/^192\.168\./);

    if (isProxied) {
      // Use the proxy - Vite will forward WebSocket connections
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const finalUrl = `${wsProtocol}//${window.location.host}${API_BASE_URL}/ws`;
      console.log('[WebSocket] URL (proxied):', finalUrl);
      return finalUrl;
    }

    // For local development, connect directly to API port
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const apiPort = typeof __API_PORT__ !== 'undefined' ? __API_PORT__ : '8090';
    const finalUrl = `${wsProtocol}//${hostname}:${apiPort}${API_BASE_URL}/ws`;
    console.log('[WebSocket] URL (direct):', finalUrl);
    return finalUrl;
  }, []);

  const { status: wsStatus, error: wsError } = useWebSocket({
    url: wsUrl,
    onEvent: handleWebSocketEvent,
    enabled: typeof window !== 'undefined',
  });

  useEffect(() => {
    setWsConnectionStatus(wsStatus);
  }, [wsStatus]);

  useEffect(() => {
    if (wsError) {
      console.error('[App] WebSocket error:', wsError);
    }
  }, [wsError]);

  const selectedIssue = useMemo(() => {
    return issues.find((issue) => issue.id === focusedIssueId) ?? null;
  }, [issues, focusedIssueId]);

  const updateStatsFromIssues = useCallback(
    (nextIssues: Issue[]) => {
      setDashboardStats(buildDashboardStats(nextIssues, null));
    },
    [],
  );

  const availableApps = useMemo(() => {
    const apps = new Set<string>();
    issues.forEach((issue) => {
      if (issue.app) {
        apps.add(issue.app);
      }
    });
    if (appFilter !== 'all' && appFilter.trim()) {
      apps.add(appFilter);
    }
    return Array.from(apps).sort((first, second) => first.localeCompare(second));
  }, [issues, appFilter]);

  const filteredIssues = useMemo(() => {
    return issues.filter((issue) => {
      const matchesPriority = priorityFilter === 'all' || issue.priority === priorityFilter;
      const matchesApp = appFilter === 'all' || issue.app === appFilter;

      let matchesSearch = true;
      if (searchFilter.trim()) {
        const query = searchFilter.toLowerCase();
        matchesSearch =
          issue.id.toLowerCase().includes(query) ||
          issue.title.toLowerCase().includes(query) ||
          issue.description.toLowerCase().includes(query) ||
          issue.assignee.toLowerCase().includes(query) ||
          issue.tags.some(tag => tag.toLowerCase().includes(query)) ||
          (issue.reporterName?.toLowerCase().includes(query) ?? false) ||
          (issue.reporterEmail?.toLowerCase().includes(query) ?? false);
      }

      return matchesPriority && matchesApp && matchesSearch;
    });
  }, [issues, priorityFilter, appFilter, searchFilter]);

  useEffect(() => {
    if (appFilter === 'all') {
      return;
    }
    if (availableApps.length === 0) {
      return;
    }
    if (!availableApps.includes(appFilter)) {
      setAppFilter('all');
    }
  }, [appFilter, availableApps]);

  useEffect(() => {
    if (!snackbar || typeof window === 'undefined') {
      return;
    }
    const timeoutId = window.setTimeout(() => {
      setSnackbar((current) => (current === snackbar ? null : current));
    }, 5000);
    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [snackbar]);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const params = new URLSearchParams(window.location.search);
    const appParam = params.get('app_id');
    if (appParam && appParam.trim()) {
      setAppFilter(appParam.trim());
    }
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const nextIssueId = params.get('issue');
      const nextApp = params.get('app_id');

      setFocusedIssueId(nextIssueId);
      setIssueDetailOpen(Boolean(nextIssueId));
      setAppFilter(nextApp && nextApp.trim() ? nextApp.trim() : 'all');
    };

    window.addEventListener('popstate', handlePopState);
    return () => {
      window.removeEventListener('popstate', handlePopState);
    };
  }, []);

  // Removed navigation initialization - single page app

  useEffect(() => {
    void fetchAllData();
  }, [fetchAllData]);

  // Running processes duration calculation (updates every second for display)
  useEffect(() => {
    const updateDurations = () => {
      setRunningProcesses(prev => {
        if (prev.size === 0) return prev;

        const next = new Map(prev);
        let changed = false;

        for (const [issueId, proc] of next.entries()) {
          const startTime = new Date(proc.start_time);
          const now = new Date();
          const durationMs = now.getTime() - startTime.getTime();
          const seconds = Math.floor(durationMs / 1000);
          const minutes = Math.floor(seconds / 60);
          const hours = Math.floor(minutes / 60);

          let duration = '';
          if (hours > 0) {
            duration = `${hours}h ${minutes % 60}m ${seconds % 60}s`;
          } else if (minutes > 0) {
            duration = `${minutes}m ${seconds % 60}s`;
          } else {
            duration = `${seconds}s`;
          }

          if (proc.duration !== duration) {
            next.set(issueId, { ...proc, duration });
            changed = true;
          }
        }

        return changed ? next : prev;
      });
    };

    const intervalId = setInterval(updateDurations, 1000);
    return () => clearInterval(intervalId);
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const url = new URL(window.location.href);
    if (focusedIssueId) {
      url.searchParams.set('issue', focusedIssueId);
    } else {
      url.searchParams.delete('issue');
    }

    if (appFilter && appFilter !== 'all') {
      url.searchParams.set('app_id', appFilter);
    } else {
      url.searchParams.delete('app_id');
    }

    const nextSearch = url.searchParams.toString();
    const nextUrl = `${url.pathname}${nextSearch ? `?${nextSearch}` : ''}${url.hash}`;
    window.history.replaceState({}, '', nextUrl);
  }, [appFilter, focusedIssueId]);

  useEffect(() => {
    setIssueDetailOpen(Boolean(focusedIssueId));
  }, [focusedIssueId]);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const themeClass = displaySettings.theme === 'dark' ? 'dark' : 'light';
    const body = document.body;
    const root = document.documentElement;

    body.classList.remove('light', 'dark');
    body.classList.add(themeClass);

    root.classList.remove('light', 'dark');
    root.classList.add(themeClass);
    root.style.setProperty('color-scheme', themeClass === 'dark' ? 'dark' : 'light');

    return () => {
      body.classList.remove(themeClass);
      root.classList.remove(themeClass);
    };
  }, [displaySettings.theme]);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    if (createIssueOpen || (issueDetailOpen && selectedIssue)) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [createIssueOpen, issueDetailOpen, selectedIssue]);

  useEffect(() => {
    if (!focusedIssueId) {
      return;
    }

    const issueExists = issues.some((issue) => issue.id === focusedIssueId);
    if (!issueExists) {
      return;
    }

    if (typeof window === 'undefined' || typeof document === 'undefined') {
      return;
    }

    const element = document.querySelector<HTMLElement>(`[data-issue-id="${focusedIssueId}"]`);
    if (element) {
      element.focus({ preventScroll: true });
      window.setTimeout(() => {
        element.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }, 50);
    }
  }, [focusedIssueId, issues]);

  // Removed navigation sync - single page app

  const createIssue = useCallback(
    async (input: CreateIssueInput): Promise<string | null> => {
      const payload = {
        title: input.title.trim(),
        description: input.description.trim() || input.title.trim(),
        priority: input.priority.toLowerCase(),
        status: input.status.toLowerCase(),
        app_id: input.appId.trim() || 'unknown',
        tags: input.tags,
        reporter_name: input.reporterName?.trim(),
        reporter_email: input.reporterEmail?.trim(),
        artifacts: input.attachments.map((attachment) => ({
          name: attachment.name,
          category: attachment.category ?? 'attachment',
          content: attachment.content,
          encoding: attachment.encoding,
          content_type: attachment.contentType,
        })),
      };

      const responseBody = await apiJsonRequest<Record<string, unknown>>(buildApiUrl('/issues'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });
      await fetchAllData();
      const responseData = (responseBody?.data ?? responseBody) as Record<string, unknown> | undefined;
      const issueId = typeof responseData?.issue_id === 'string' ? (responseData.issue_id as string) : null;
      return issueId;
    },
    [fetchAllData],
  );

  const deleteIssue = useCallback(async (issueId: string) => {
    await apiVoidRequest(buildApiUrl(`/issues/${encodeURIComponent(issueId)}`), {
      method: 'DELETE',
    });
  }, []);

  const updateIssueStatus = useCallback(
    async (issueId: string, nextStatus: IssueStatus) => {
      const responseBody = await apiJsonRequest<Record<string, unknown>>(buildApiUrl(`/issues/${encodeURIComponent(issueId)}`), {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: nextStatus }),
      });
      const responseData = (responseBody?.data ?? responseBody) as Record<string, unknown> | undefined;
      const updatedIssue = (responseData?.issue ?? null) as ApiIssue | null;

      setIssues((prev) => {
        let nextIssues: Issue[];
        if (updatedIssue) {
          const transformed = transformIssue(updatedIssue, { apiBaseUrl: API_BASE_URL });
          nextIssues = prev.map((issue) => (issue.id === issueId ? transformed : issue));
        } else {
          nextIssues = prev.map((issue) =>
            issue.id === issueId
              ? {
                  ...issue,
                  status: nextStatus,
                }
              : issue,
          );
        }
        updateStatsFromIssues(nextIssues);
        return nextIssues;
      });
    },
    [updateStatsFromIssues],
  );

  const handleCreateIssue = useCallback(() => {
    setFiltersOpen(false);
    setCreateIssuePrefill(null);
    setCreateIssueOpen(true);
  }, []);

  const handleCreateIssueClose = useCallback(() => {
    setCreateIssuePrefill(null);
    setCreateIssueOpen(false);
  }, []);

  const handleIssueSelect = useCallback((issueId: string) => {
    setFocusedIssueId((current) => {
      if (current === issueId) {
        setIssueDetailOpen(false);
        return null;
      }
      setIssueDetailOpen(true);
      return issueId;
    });
  }, []);

  const handleCreateFollowUp = useCallback(
    async (issue: Issue) => {
      setFollowUpLoadingId(issue.id);
      try {
        const prefill = await prepareFollowUpPrefill(issue);
        setCreateIssuePrefill(prefill);
        setCreateIssueOpen(true);
      } catch (error) {
        console.error('[IssueTracker] Failed to prepare follow-up issue', error);
        const message = error instanceof Error ? error.message : 'Failed to prepare follow-up issue.';
        showSnackbar(message, 'error');
      } finally {
        setFollowUpLoadingId(null);
      }
    },
    [showSnackbar],
  );

  const handleSubmitNewIssue = useCallback(
    async (input: CreateIssueInput) => {
      const newIssueId = await createIssue(input);
      if (newIssueId) {
        setFocusedIssueId(newIssueId);
        setIssueDetailOpen(true);
      }
      setCreateIssuePrefill(null);
    },
    [createIssue],
  );

  const handlePriorityFilterChange = useCallback((value: PriorityFilterValue) => {
    setPriorityFilter(value);
  }, []);

  const handleAppFilterChange = useCallback((value: string) => {
    setAppFilter(value);
  }, []);

  const handleSearchFilterChange = useCallback((value: string) => {
    setSearchFilter(value);
  }, []);

  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (priorityFilter !== 'all') count++;
    if (appFilter !== 'all') count++;
    if (searchFilter.trim()) count++;
    return count;
  }, [priorityFilter, appFilter, searchFilter]);

  const handleIssueArchive = useCallback(
    async (issue: Issue) => {
      try {
        await updateIssueStatus(issue.id, 'archived');
      } catch (error) {
        console.error('Failed to archive issue', error);
        if (typeof window !== 'undefined' && typeof window.alert === 'function') {
          window.alert('Failed to archive issue. Please try again.');
        }
      }
    },
    [updateIssueStatus],
  );

  const handleIssueDelete = useCallback(
    async (issue: Issue) => {
      const issueId = issue.id;
      try {
        await deleteIssue(issueId);
        setIssues((prev) => {
          const nextIssues = prev.filter((item) => item.id !== issueId);
          updateStatsFromIssues(nextIssues);
          return nextIssues;
        });
        if (focusedIssueId === issueId) {
          setIssueDetailOpen(false);
          setFocusedIssueId(null);
        }
      } catch (error) {
        console.error('Failed to delete issue', error);
        if (typeof window !== 'undefined' && typeof window.alert === 'function') {
          window.alert('Failed to delete issue. Please try again.');
        }
      }
    },
    [deleteIssue, focusedIssueId, updateStatsFromIssues],
  );

  const handleIssueStatusChange = useCallback(
    async (issueId: string, _fromStatus: IssueStatus, targetStatus: IssueStatus) => {
      if (_fromStatus === targetStatus) {
        return;
      }
      try {
        await updateIssueStatus(issueId, targetStatus);
      } catch (error) {
        console.error('Failed to update issue status', error);
        if (typeof window !== 'undefined' && typeof window.alert === 'function') {
          window.alert('Failed to move issue. Please try again.');
        }
      }
    },
    [updateIssueStatus],
  );

  const handleHideColumn = useCallback((status: IssueStatus) => {
    setHiddenColumns((current) => (current.includes(status) ? current : [...current, status]));
  }, []);

  const handleToggleColumn = useCallback((status: IssueStatus) => {
    setHiddenColumns((current) =>
      current.includes(status)
        ? current.filter((value) => value !== status)
        : [...current, status],
    );
  }, []);

  const handleResetHiddenColumns = useCallback(() => {
    setHiddenColumns([]);
  }, []);

  const handleIssueDetailClose = useCallback(() => {
    setIssueDetailOpen(false);
    setFocusedIssueId(null);
  }, []);

  const handleOpenMetrics = useCallback(() => {
    setMetricsDialogOpen(true);
  }, []);

  const handleCloseMetrics = useCallback(() => {
    setMetricsDialogOpen(false);
  }, []);

  const handleOpenSettings = useCallback(() => {
    setSettingsDialogOpen(true);
  }, []);

  const handleCloseSettings = useCallback(() => {
    setSettingsDialogOpen(false);
  }, []);

  const handleMetricsOpenIssue = useCallback(
    (issueId: string) => {
      setMetricsDialogOpen(false);
      setFocusedIssueId(issueId);
      setIssueDetailOpen(true);
    },
    [],
  );

  const processorSetter = (settings: ProcessorSettings) => {
    setProcessorSettings(settings);
  };

  // Single page - always render issues board

  const showIssueDetailModal = issueDetailOpen && selectedIssue;

  return (
    <>
      <div className={`app-shell ${displaySettings.theme}`}>
        <div className="main-panel">
          {wsConnectionStatus === 'connecting' && (
            <div className="connection-banner connection-banner--connecting">
              <Loader2 size={18} className="spin" style={{ marginRight: '8px' }} />
              <span>Connecting to live updates...</span>
            </div>
          )}
          {wsConnectionStatus === 'error' && (
            <div className="connection-banner connection-banner--error">
              <CircleAlert size={18} style={{ marginRight: '8px' }} />
              <span>Connection error - live updates unavailable</span>
            </div>
          )}
          {rateLimitStatus?.rate_limited && rateLimitStatus.seconds_until_reset > 0 && (
            <div className="rate-limit-banner">
              <Clock size={18} style={{ marginRight: '8px' }} />
              <span>
                Rate limit active ({rateLimitStatus.rate_limited_count} issue{rateLimitStatus.rate_limited_count !== 1 ? 's' : ''} waiting) - Resets in{' '}
                {Math.floor(rateLimitStatus.seconds_until_reset / 60)}m {rateLimitStatus.seconds_until_reset % 60}s
                {rateLimitStatus.rate_limit_agent && ` (${rateLimitStatus.rate_limit_agent})`}
              </span>
            </div>
          )}
          <main className="page-container page-container--issues">
            {loading && <div className="data-banner loading">Loading latest data…</div>}
            {loadError && (
              <div className="data-banner error">
                <span>{loadError}</span>
                <button type="button" onClick={() => void fetchAllData()}>
                  Retry
                </button>
              </div>
            )}
            <IssuesBoard
              issues={filteredIssues}
              focusedIssueId={focusedIssueId}
              runningProcesses={runningProcesses}
              onIssueSelect={handleIssueSelect}
              onIssueDelete={handleIssueDelete}
              onIssueArchive={handleIssueArchive}
              onIssueDrop={handleIssueStatusChange}
              hiddenColumns={hiddenColumns}
              onHideColumn={handleHideColumn}
            />

            <div className="issues-floating-controls">
              <div className="issues-floating-panel">
                <button
                  type="button"
                  className={`icon-button icon-button--status ${processorSettings.active ? 'is-active' : ''}`.trim()}
                  onClick={toggleProcessorActive}
                  aria-label={
                    processorSettings.active ? 'Pause issue automation' : 'Activate issue automation'
                  }
                >
                  {processorSettings.active ? <Pause size={18} /> : <Play size={18} />}
                </button>
                {runningProcesses.size > 0 && (
                  <div className="running-agents-indicator">
                    <Brain size={16} className="running-agents-icon" />
                    <span className="running-agents-count">{runningProcesses.size}</span>
                  </div>
                )}
                <button
                  type="button"
                  className="icon-button icon-button--primary"
                  onClick={handleCreateIssue}
                  aria-label="Create new issue"
                >
                  <Plus size={18} />
                </button>
                <button
                  type="button"
                  className={`icon-button icon-button--filter ${filtersOpen ? 'is-active' : ''}`.trim()}
                  onClick={() => setFiltersOpen((state) => !state)}
                  aria-label="Toggle filters"
                  aria-expanded={filtersOpen}
                  style={{ position: 'relative' }}
                >
                  <Filter size={18} />
                  {activeFilterCount > 0 && (
                    <span className="filter-badge" aria-label={`${activeFilterCount} active filters`}>
                      {activeFilterCount}
                    </span>
                  )}
                </button>
                <button
                  type="button"
                  className="icon-button icon-button--secondary"
                  onClick={handleOpenMetrics}
                  aria-label="Open metrics"
                >
                  <LayoutDashboard size={18} />
                </button>
                <button
                  type="button"
                  className="icon-button icon-button--secondary"
                  onClick={handleOpenSettings}
                  aria-label="Open settings"
                >
                  <Cog size={18} />
                </button>
                <IssueBoardToolbar
                  open={filtersOpen}
                  onRequestClose={() => setFiltersOpen(false)}
                  priorityFilter={priorityFilter}
                  onPriorityFilterChange={handlePriorityFilterChange}
                  appFilter={appFilter}
                  onAppFilterChange={handleAppFilterChange}
                  searchFilter={searchFilter}
                  onSearchFilterChange={handleSearchFilterChange}
                  appOptions={availableApps}
                  hiddenColumns={hiddenColumns}
                  onToggleColumn={handleToggleColumn}
                  onResetColumns={handleResetHiddenColumns}
                />
              </div>
            </div>
          </main>
        </div>
      </div>

      {createIssueOpen && (
        <CreateIssueModal
          key={createIssuePrefill?.key ?? 'create-issue'}
          onClose={handleCreateIssueClose}
          onSubmit={handleSubmitNewIssue}
          initialData={createIssuePrefill?.initial}
          lockedAttachments={createIssuePrefill?.lockedAttachments}
          followUpInfo={createIssuePrefill?.followUpOf}
        />
      )}

      {showIssueDetailModal && selectedIssue && (
        <IssueDetailsModal
          issue={selectedIssue}
          apiBaseUrl={API_BASE_URL}
          onClose={handleIssueDetailClose}
          onStatusChange={updateIssueStatus}
          onFollowUp={handleCreateFollowUp}
          followUpLoadingId={followUpLoadingId}
          validStatuses={VALID_STATUSES}
        />
      )}

      {metricsDialogOpen && (
        <MetricsDialog
          stats={dashboardStats}
          issues={issues}
          processor={processorSettings}
          agentSettings={agentSettings}
          onClose={handleCloseMetrics}
          onOpenIssue={handleMetricsOpenIssue}
        />
      )}

      {settingsDialogOpen && (
        <SettingsDialog
          apiBaseUrl={API_BASE_URL}
          processor={processorSettings}
          agent={agentSettings}
          display={displaySettings}
          onProcessorChange={processorSetter}
          onAgentChange={setAgentSettings}
          onDisplayChange={setDisplaySettings}
          onClose={handleCloseSettings}
          issuesProcessed={issuesProcessed}
          issuesRemaining={issuesRemaining}
        />
      )}

      {snackbar && (
        <Snackbar message={snackbar.message} tone={snackbar.tone} onClose={() => setSnackbar(null)} />
      )}
    </>
  );
}
export default App;
