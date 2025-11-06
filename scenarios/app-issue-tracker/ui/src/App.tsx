import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Brain, Cog, Filter, LayoutDashboard, Pause, Play, Plus } from 'lucide-react';
import { IssuesBoard } from './pages/IssuesBoard';
import { IssueDetailsModal } from './components/IssueDetailsModal';
import { IssueBoardToolbar } from './components/IssueBoardToolbar';
import { CreateIssueModal } from './components/CreateIssueModal';
import { MetricsDialog } from './components/MetricsDialog';
import { SettingsDialog } from './components/SettingsDialog';
import {
  AppSettings,
  type AgentSettings,
  type DisplaySettings,
  type Issue,
  type IssueStatus,
  type ProcessorSettings,
} from './data/sampleData';
import './styles/app.css';
import { type CreateIssueInput, type CreateIssuePrefill, type UpdateIssueInput } from './types/issueCreation';
import { prepareFollowUpPrefill } from './utils/issueHelpers';
import { IssueTrackerDataProvider, useIssueTrackerData } from './hooks/useIssueTrackerData';
import { useIssueFilters } from './hooks/useIssueFilters';
import { useThemeClass } from './hooks/useThemeClass';
import { useBodyScrollLock } from './hooks/useBodyScrollLock';
import { useDialogState } from './hooks/useDialogState';
import { useSearchParamSync } from './hooks/useSearchParamSync';
import { useIssueFocus } from './hooks/useIssueFocus';
import { SnackStackProvider, useSnackPublisher } from './notifications/SnackStackProvider';
import type { SnackVariant } from './notifications/snackBus';
import { resolveApiBase } from '@vrooli/api-base';
import { useComponentStore } from './stores/componentStore';

// Automatically resolves API base URL for all deployment contexts
// Works in localhost, direct tunnel, and proxied/embedded scenarios
const API_BASE_URL = resolveApiBase({ appendSuffix: true });
const ISSUE_FETCH_LIMIT = 200;
const SNACK_IDS = {
  loading: 'snack:data-loading',
  loadError: 'snack:data-error',
  connection: 'snack:connection-status',
  rateLimit: 'snack:rate-limit',
};

function IssueAppProviders({ children }: { children: ReactNode }) {
  return (
    <SnackStackProvider>
      <SnackAwareIssueTracker>{children}</SnackAwareIssueTracker>
    </SnackStackProvider>
  );
}

function SnackAwareIssueTracker({ children }: { children: ReactNode }) {
  const { publish } = useSnackPublisher();

  const showSnackbar = useCallback(
    (message: string, tone: SnackVariant = 'info') => {
      publish({
        message,
        variant: tone,
        durationMs: tone === 'error' ? 7000 : undefined,
      });
    },
    [publish],
  );

  return (
    <IssueTrackerDataProvider
      apiBaseUrl={API_BASE_URL}
      issueFetchLimit={ISSUE_FETCH_LIMIT}
      showSnackbar={showSnackbar}
    >
      {children}
    </IssueTrackerDataProvider>
  );
}

function AppContent() {
  const { publish, dismiss } = useSnackPublisher();

  const showSnack = useCallback(
    (message: string, tone: SnackVariant = 'info') => {
      publish({
        message,
        variant: tone,
        durationMs: tone === 'error' ? 7000 : undefined,
      });
    },
    [publish],
  );

  const {
    issues,
    dashboardStats,
    loading,
    loadError,
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
    createIssue: createIssueAction,
    deleteIssue: deleteIssueAction,
    updateIssueStatus: updateIssueStatusAction,
    updateIssueDetails: updateIssueDetailsAction,
    stopAgent,
    runningProcesses,
    connectionStatus,
    websocketError,
    reconnectAttempts,
    statusCatalog,
    validStatuses,
  } = useIssueTrackerData();

  const {
    open: metricsDialogOpen,
    openDialog: openMetricsDialog,
    closeDialog: closeMetricsDialog,
  } = useDialogState(false);
  const {
    open: settingsDialogOpen,
    openDialog: openSettingsDialog,
    closeDialog: closeSettingsDialog,
  } = useDialogState(false);
  const [displaySettings, setDisplaySettings] = useState<DisplaySettings>(AppSettings.display);
  const [hiddenColumns, setHiddenColumns] = useState<IssueStatus[]>(['archived']);
  const statusOrder = useMemo(
    () => (statusCatalog.length > 0 ? statusCatalog.map((status) => status.id) : validStatuses),
    [statusCatalog, validStatuses],
  );

  useEffect(() => {
    setHiddenColumns((current) => {
      const filtered = current.filter((status) => statusOrder.includes(status));
      if (filtered.length === 0 && statusOrder.includes('archived')) {
        return ['archived'];
      }
      return filtered;
    });
  }, [statusOrder]);
  const {
    open: createIssueOpen,
    openDialog: openCreateIssue,
    closeDialog: closeCreateIssue,
  } = useDialogState(false);
  const [createIssuePrefill, setCreateIssuePrefill] = useState<CreateIssuePrefill | null>(null);
  const {
    open: issueDetailOpen,
    openDialog: openIssueDetail,
    closeDialog: closeIssueDetail,
    setOpen: setIssueDetailOpenState,
  } = useDialogState(false);
  const {
    open: editIssueOpen,
    openDialog: openEditIssue,
    closeDialog: closeEditIssue,
  } = useDialogState(false);
  const [issueBeingEdited, setIssueBeingEdited] = useState<Issue | null>(null);
  const { getParam, getParams, setParams, subscribe } = useSearchParamSync();
  const searchHelpers = useMemo(
    () => ({
      getParam,
      setParams,
      subscribe,
    }),
    [getParam, setParams, subscribe],
  );
  const {
    open: filtersOpen,
    openDialog: openFilters,
    closeDialog: closeFilters,
    toggleDialog: toggleFilters,
  } = useDialogState(false);
  const [followUpLoadingId, setFollowUpLoadingId] = useState<string | null>(null);

  useEffect(() => {
    if (loading) {
      publish({
        id: SNACK_IDS.loading,
        message: 'Loading latest data…',
        variant: 'loading',
        autoDismiss: false,
        dismissible: false,
      });
    } else {
      dismiss(SNACK_IDS.loading);
    }
  }, [dismiss, loading, publish]);

  useEffect(() => {
    if (!loadError) {
      dismiss(SNACK_IDS.loadError);
      return;
    }

    publish({
      id: SNACK_IDS.loadError,
      message: loadError,
      variant: 'error',
      autoDismiss: false,
      dismissible: true,
      action: {
        label: 'Retry',
        handler: () => {
          void fetchAllData();
          dismiss(SNACK_IDS.loadError);
        },
      },
    });
  }, [dismiss, fetchAllData, loadError, publish]);

  useEffect(() => {
    if (connectionStatus === 'connected') {
      dismiss(SNACK_IDS.connection);
      return;
    }

    if (connectionStatus === 'connecting') {
      publish({
        id: SNACK_IDS.connection,
        message: 'Connecting to live updates…',
        variant: 'loading',
        autoDismiss: false,
        dismissible: false,
      });
      return;
    }

    if (connectionStatus === 'error') {
      publish({
        id: SNACK_IDS.connection,
        message: 'Connection error - live updates unavailable',
        variant: 'error',
        autoDismiss: false,
      });
      return;
    }

    if (connectionStatus === 'disconnected' && reconnectAttempts > 0) {
      publish({
        id: SNACK_IDS.connection,
        message: `Reconnecting to live updates (attempt ${reconnectAttempts + 1})…`,
        variant: 'warning',
        autoDismiss: false,
      });
      return;
    }

    dismiss(SNACK_IDS.connection);
  }, [connectionStatus, dismiss, publish, reconnectAttempts]);

  useEffect(() => {
    const active = Boolean(rateLimitStatus?.rate_limited) && (rateLimitStatus?.seconds_until_reset ?? 0) > 0;
    if (!active) {
      dismiss(SNACK_IDS.rateLimit);
      return;
    }

    const count = rateLimitStatus?.rate_limited_count ?? 0;
    const secondsUntilReset = rateLimitStatus?.seconds_until_reset ?? 0;
    const minutes = Math.floor(secondsUntilReset / 60);
    const seconds = secondsUntilReset % 60;
    const agentName = rateLimitStatus?.rate_limit_agent ? ` (${rateLimitStatus.rate_limit_agent})` : '';
    const message = `Rate limit active (${count} issue${count === 1 ? '' : 's'} waiting) - resets in ${minutes}m ${seconds}s${agentName}`;

    publish({
      id: SNACK_IDS.rateLimit,
      message,
      variant: 'warning',
      autoDismiss: false,
      dismissible: false,
    });
  }, [dismiss, publish, rateLimitStatus]);

  const {
    filteredIssues,
    availableApps,
    priorityFilter,
    handlePriorityFilterChange,
    appFilter,
    handleAppFilterChange,
    searchFilter,
    handleSearchFilterChange,
    activeFilterCount,
    syncAppFilterFromParams,
  } = useIssueFilters({ issues });

  const fetchAllDataRef = useRef(fetchAllData);

  useEffect(() => {
    fetchAllDataRef.current = fetchAllData;
  }, [fetchAllData]);

  useEffect(() => {
    if (!websocketError) {
      return;
    }
    // WebSocket error is tracked in state for debugging
  }, [websocketError]);

  const {
    focusedIssueId,
    selectIssue,
    clearFocus,
    selectedIssue,
  } = useIssueFocus({
    issues,
    search: searchHelpers,
    onOpen: openIssueDetail,
    onClose: closeIssueDetail,
    setOpen: setIssueDetailOpenState,
  });

  const themeMode = displaySettings.theme === 'dark' ? 'dark' : 'light';
  useThemeClass(themeMode);
  useBodyScrollLock(createIssueOpen || editIssueOpen || (issueDetailOpen && Boolean(selectedIssue)));

  useEffect(() => {
    syncAppFilterFromParams(getParams());
  }, [getParams, syncAppFilterFromParams]);

  useEffect(() => {
    return subscribe((params) => {
      syncAppFilterFromParams(params);
    });
  }, [subscribe, syncAppFilterFromParams]);

  useEffect(() => {
    void fetchAllDataRef.current();
  }, []);

  // Load components globally on mount (for target selector)
  const fetchComponents = useComponentStore((state) => state.fetchComponents);
  useEffect(() => {
    void fetchComponents(API_BASE_URL);
  }, [fetchComponents]);

  useEffect(() => {
    setParams((params) => {
      if (appFilter && appFilter !== 'all') {
        params.set('app_id', appFilter);
      } else {
        params.delete('app_id');
      }
    });
  }, [appFilter, setParams]);

  const handleCreateIssue = useCallback(() => {
    closeFilters();
    setCreateIssuePrefill(null);
    openCreateIssue();
  }, [closeFilters, openCreateIssue]);

  const handleCreateIssueClose = useCallback(() => {
    setCreateIssuePrefill(null);
    closeCreateIssue();
  }, [closeCreateIssue]);

  const handleIssueSelect = useCallback(
    (issueId: string) => {
      selectIssue(issueId);
    },
    [selectIssue],
  );

  const handleStartEditIssue = useCallback(
    (issue: Issue) => {
      const freshIssue = issues.find((candidate) => candidate.id === issue.id) ?? issue;
      setIssueBeingEdited(freshIssue);
      clearFocus();
      openEditIssue();
    },
    [clearFocus, issues, openEditIssue],
  );

  const handleCloseEditIssue = useCallback(() => {
    setIssueBeingEdited(null);
    closeEditIssue();
  }, [closeEditIssue]);

  const handleCreateFollowUp = useCallback(
    async (issue: Issue) => {
      setFollowUpLoadingId(issue.id);
      try {
        const prefill = await prepareFollowUpPrefill(issue);
        setCreateIssuePrefill(prefill);
        openCreateIssue();
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to prepare follow-up issue.';
        showSnack(message, 'error');
      } finally {
        setFollowUpLoadingId(null);
      }
    },
    [openCreateIssue, showSnack],
  );

  const handleSubmitNewIssue = useCallback(
    async (input: CreateIssueInput) => {
      const newIssueId = await createIssueAction(input);
      if (newIssueId) {
        selectIssue(newIssueId);
      }
      setCreateIssuePrefill(null);
    },
    [createIssueAction, selectIssue],
  );

  const handleSubmitIssueUpdate = useCallback(
    async (input: UpdateIssueInput) => {
      await updateIssueDetailsAction(input);
      showSnack('Issue updated successfully.', 'success');
      selectIssue(input.issueId);
    },
    [selectIssue, showSnack, updateIssueDetailsAction],
  );

  const handleIssueArchive = useCallback(
    async (issue: Issue) => {
      try {
        await updateIssueStatusAction(issue.id, 'archived');
      } catch (error) {
        showSnack('Failed to archive issue. Please try again.', 'error');
      }
    },
    [showSnack, updateIssueStatusAction],
  );

  const handleIssueDelete = useCallback(
    async (issue: Issue) => {
      const issueId = issue.id;
      try {
        await deleteIssueAction(issueId);
        if (focusedIssueId === issueId) {
          clearFocus();
        }
      } catch (error) {
        showSnack('Failed to delete issue. Please try again.', 'error');
      }
    },
    [clearFocus, deleteIssueAction, focusedIssueId, showSnack],
  );

  const handleIssueStatusChange = useCallback(
    async (issueId: string, fromStatus: IssueStatus, targetStatus: IssueStatus) => {
      if (fromStatus === targetStatus) {
        return;
      }
      try {
        await updateIssueStatusAction(issueId, targetStatus);
      } catch (error) {
        showSnack('Failed to move issue. Please try again.', 'error');
      }
    },
    [showSnack, updateIssueStatusAction],
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
    clearFocus();
  }, [clearFocus]);

  const handleStopAgent = useCallback(
    (issueId: string) => {
      void stopAgent(issueId);
    },
    [stopAgent],
  );

  const handleOpenMetrics = useCallback(() => {
    openMetricsDialog();
  }, [openMetricsDialog]);

  const handleCloseMetrics = useCallback(() => {
    closeMetricsDialog();
  }, [closeMetricsDialog]);

  const handleOpenSettings = useCallback(() => {
    openSettingsDialog();
  }, [openSettingsDialog]);

  const handleCloseSettings = useCallback(() => {
    closeSettingsDialog();
  }, [closeSettingsDialog]);

  const handleMetricsOpenIssue = useCallback(
    (issueId: string) => {
      closeMetricsDialog();
      selectIssue(issueId);
    },
    [closeMetricsDialog, selectIssue],
  );

  const handleProcessorSettingsChange = useCallback(
    (settings: ProcessorSettings) => {
      updateProcessorSettings(settings);
    },
    [updateProcessorSettings],
  );

  const handleAgentSettingsChange = useCallback(
    (settings: AgentSettings) => {
      updateAgentSettings(settings);
    },
    [updateAgentSettings],
  );

  const showIssueDetailModal = issueDetailOpen && selectedIssue;

  return (
    <>
      <div className={`app-shell ${displaySettings.theme}`}>
        <div className="main-panel">
          <main className="page-container page-container--issues">
            <IssuesBoard
              issues={filteredIssues}
              statusOrder={statusOrder}
              focusedIssueId={focusedIssueId}
              runningProcesses={runningProcesses}
              onIssueSelect={handleIssueSelect}
              onIssueDelete={handleIssueDelete}
              onIssueArchive={handleIssueArchive}
              onIssueDrop={handleIssueStatusChange}
              onStopAgent={handleStopAgent}
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
                  onClick={toggleFilters}
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
                  onRequestClose={closeFilters}
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
                  statusOrder={statusOrder}
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
          validStatuses={validStatuses}
        />
      )}

      {showIssueDetailModal && selectedIssue && (
        <IssueDetailsModal
          issue={selectedIssue}
          apiBaseUrl={API_BASE_URL}
          onClose={handleIssueDetailClose}
          onStatusChange={updateIssueStatusAction}
          onEdit={handleStartEditIssue}
          onArchive={handleIssueArchive}
          onDelete={handleIssueDelete}
          onFollowUp={handleCreateFollowUp}
          followUpLoadingId={followUpLoadingId}
          validStatuses={validStatuses}
        />
      )}

      {editIssueOpen && issueBeingEdited && (
        <CreateIssueModal
          key={`edit-${issueBeingEdited.id}`}
          mode="edit"
          existingIssue={issueBeingEdited}
          onClose={handleCloseEditIssue}
          onSubmit={handleSubmitIssueUpdate}
          validStatuses={validStatuses}
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
          onProcessorChange={handleProcessorSettingsChange}
          onAgentChange={handleAgentSettingsChange}
          onDisplayChange={setDisplaySettings}
          onClose={handleCloseSettings}
          issuesProcessed={issuesProcessed}
          constraints={agentConstraints}
          issuesRemaining={issuesRemaining}
        />
      )}
    </>
  );
}

function App() {
  return (
    <IssueAppProviders>
      <AppContent />
    </IssueAppProviders>
  );
}

export default App;
