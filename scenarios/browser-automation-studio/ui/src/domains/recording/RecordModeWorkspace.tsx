/**
 * RecordModePage Component
 *
 * Main page for Record Mode - allows users to record browser actions
 * and generate workflows from them.
 *
 * UX Flow (redesigned):
 * 1. Recording starts automatically when page opens (no Record/Stop buttons)
 * 2. Timeline shows all recorded actions
 * 3. User can select steps using checkboxes (shift+click for range)
 * 4. "Create Workflow →" button switches right panel to workflow creation form
 * 5. Form allows naming workflow, selecting project, testing, and generating
 * 6. Back button returns to Live Preview
 *
 * Features:
 * - Auto-recording (continuous while page is open)
 * - Step selection with range support (shift+click)
 * - Workflow creation form in right panel
 * - Confidence warnings for unstable selectors
 * - Action editing (selector and payload)
 */

import { Profiler, useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { RecordingHeader } from './capture/RecordingHeader';
import { TabBar } from './capture/TabBar';
import { ErrorBanner, UnstableSelectorsBanner, type ErrorDetails } from './capture/RecordModeBanners';
import { ClearActionsModal, ErrorDetailsModal } from './capture/RecordModeModals';
import { WorkflowCreationForm } from './conversion/WorkflowCreationForm';
import { WorkflowPickerModal } from './conversion/WorkflowPickerModal';
import { WorkflowInfoCard } from './timeline/WorkflowInfoCard';
import type { ReplayPreviewResponse, RecordedAction } from './types/types';
import type { WorkflowSettingsTyped } from '@/types/workflow';
import { SessionManager } from '@/views/SettingsView/sections/sessions/SessionManager/SessionManager';
import { useRecordingSession } from './hooks/useRecordingSession';
import { useSessionProfiles } from './hooks/useSessionProfiles';
import { useBrowserNavigation } from './hooks/useBrowserNavigation';
import { useSessionProfileSelection } from './hooks/useSessionProfileSelection';
import type { InsertedAction } from './InsertNodeModal';
import { RecordPreviewPanel } from './timeline/RecordPreviewPanel';
import { ExecutionPreviewPanel } from './timeline/ExecutionPreviewPanel';
import { PreviewContainer } from './shared';
import { PreviewSettingsPanel } from '@/domains/preview-settings';
import { ViewportProvider } from './context';
import { getConfig } from '@/config';
import { useStreamSettings } from './capture/streamSettingsState';
import type { StreamSettingsValues } from './capture/StreamSettings';
import { DEFAULT_STREAM_FPS } from './constants';
import { useExecutionModeState, useRecordingModeState } from './session';
import type { TimelineMode } from './types/timeline-unified';
import { UnifiedSidebar } from './sidebar';
import { HumanInterventionOverlay } from './ai-navigation';
import { useSessionStore } from './stores/sessionStore';
import { ExportDialog } from '@/domains/executions/export/components/ExportDialog';

interface InsertActionData {
  actionType: RecordedAction['actionType'];
  payload?: Record<string, unknown>;
  selector?: string;
}
import { ExportDialogProvider } from '@/domains/executions/export/context/ExportDialogProvider';
import { buildExportDialogContextValue } from '@/domains/executions/export/context/ExportDialogContext';
import { ExportSuccessPanel } from '@/domains/exports/ExportSuccessPanel';
import { useConfirmDialog } from '@/hooks/useConfirmDialog';
import { ConfirmDialog } from '@shared/ui/ConfirmDialog';
import { extractConsoleLogs, extractNetworkEvents, extractDomSnapshots } from './utils/artifact-extraction';
import { onProfilerRender } from '@/lib/profiler';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const parseWorkflowGenerationResult = (
  value: unknown
): { workflowId?: string; projectId?: string } => {
  if (!isRecord(value)) return {};
  const workflowId =
    typeof value.workflow_id === 'string'
      ? value.workflow_id
      : typeof value.workflowId === 'string'
        ? value.workflowId
        : undefined;
  const projectId =
    typeof value.project_id === 'string'
      ? value.project_id
      : typeof value.projectId === 'string'
        ? value.projectId
        : undefined;
  return { workflowId, projectId };
};

/** Workflow type being created (from AI modal or template) */
export type WorkflowTypeParam = 'action' | 'flow' | 'case';


export interface RecordModePageProps {
  /** Browser session ID */
  sessionId: string | null;
  /** Mode: 'recording' for live recording, 'execution' for workflow playback */
  mode?: TimelineMode;
  /** Execution ID for execution mode (required when mode is 'execution') */
  executionId?: string | null;
  /** Initial workflow ID to select (for execution mode from Build page) */
  initialWorkflowId?: string;
  /** Initial project ID (for execution mode from Build page) */
  initialProjectId?: string;
  /** Callback when workflow is generated */
  onWorkflowGenerated?: (workflowId: string, projectId: string) => void;
  /** Callback when a live session is created */
  onSessionReady?: (sessionId: string) => void;
  /** Callback to close record mode */
  onClose?: () => void;
  /** Initial URL to navigate to (from template) */
  initialUrl?: string;
  /** AI prompt to auto-start with (from template) */
  aiPrompt?: string;
  /** AI model to use (from template) */
  aiModel?: string;
  /** Max AI steps (from template) */
  aiMaxSteps?: number;
  /** Whether to auto-start AI navigation with the prompt */
  autoStartAI?: boolean;
  /** Type of workflow being created (from AI modal) */
  workflowType?: WorkflowTypeParam;
  /** Initial folder for saving the workflow */
  initialFolder?: string;
}

/** Right panel view state */
type RightPanelView = 'preview' | 'create-workflow';

export function RecordModeWorkspace({
  sessionId: initialSessionId,
  mode: initialMode = 'recording',
  executionId,
  initialWorkflowId,
  initialProjectId,
  onWorkflowGenerated,
  onSessionReady,
  onClose,
  initialUrl,
  aiPrompt,
  aiModel,
  aiMaxSteps,
  autoStartAI,
  workflowType,
  initialFolder: _initialFolder,
}: RecordModePageProps) {
  const navigate = useNavigate();

  // Track current mode - can switch between recording and execution
  const [mode, setMode] = useState<TimelineMode>(initialMode);

  const sessionProfiles = useSessionProfiles();

  const {
    sessionId,
    sessionProfileId,
    sessionError,
    actualViewport: sessionActualViewport,
    ensureSession,
    setSessionProfileId,
    retryState,
    retrySession,
  } = useRecordingSession({ initialSessionId, onSessionReady });

  // Session profile selection state and handlers (extracted to hook)
  const {
    selectedProfileId,
    handleSelectSessionProfile,
    configuringProfile,
    configuringSection,
    handleCreateSessionProfile,
    handleConfigureSession,
    handleOpenHistorySettings,
    handleNavigateToSessionSettings,
    handleSaveBrowserProfile,
    closeConfigureModal,
  } = useSessionProfileSelection({
    sessionProfileId,
    setSessionProfileId,
    sessionProfiles,
  });

  // Right panel view state
  const [rightPanelView, setRightPanelView] = useState<RightPanelView>('preview');
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [previewViewport, setPreviewViewport] = useState<{ width: number; height: number } | null>(null);

  // Browser navigation state and handlers (extracted to hook)
  const navigationPage = useSessionStore(s => {
    if (!s.isValidated || s.sessionId !== sessionId) return undefined;
    const page = s.activePageId ? s.pages.get(s.activePageId) : undefined;
    return page?.sessionId === sessionId && page.status === 'active' ? page : null;
  });
  const {
    previewUrl,
    setPreviewUrl,
    canGoBack,
    canGoForward,
    refreshToken,
    handleGoBack,
    handleGoForward,
    handleRefresh,
    handleFetchNavigationStack,
    refreshNavigationState,
    handleNavigateToIndex,
    handleNavigate,
    isInitialNavigationComplete,
    navigationError,
  } = useBrowserNavigation({
    sessionId,
    pageId: navigationPage?.id ?? null,
    observedUrl: navigationPage === undefined ? undefined : navigationPage?.url ?? '',
    initialUrl,
  });

  useEffect(() => {
    if (navigationPage) void refreshNavigationState();
  }, [navigationPage, refreshNavigationState]);

  // Handler for PreviewContainer's browser viewport changes (for session creation)
  const handleBrowserViewportChange = useCallback((viewport: { width: number; height: number }) => {
    setPreviewViewport(viewport);
  }, []);

  const autoStartedRef = useRef(false);

  // Stream settings for session creation (from shared context)
  const { settings: streamSettings, showStats } = useStreamSettings();
  const streamSettingsRef = useRef<StreamSettingsValues | null>(null);
  streamSettingsRef.current = streamSettings;

  const setConnectionStatus = useSessionStore(s => s.setConnectionStatus);

  const executionMode = useExecutionModeState({
    initialExecutionId: executionId,
    initialWorkflowId,
    initialProjectId,
    sessionProfileId: selectedProfileId,
    streamSettingsRef,
    enabled: mode === 'execution',
    onSessionProfileSelect: handleSelectSessionProfile,
  });
  const {
    selectedWorkflowId, selectedProjectId, selectedWorkflowName,
    setSelectedWorkflowId, setSelectedProjectId, setSelectedWorkflowName,
    localExecutionId, setLocalExecutionId, executionStatus, currentExecution,
    isExecuting, canRun, isReadOnly, workflowNodes, workflowEdges,
    handleWorkflowSelect: selectWorkflow, handleRun, handleStop,
    handleRerunExecution, handleEditWorkflow, exportController,
    exportDialogTitleId, exportDialogDescriptionId, showWorkflowPicker,
    setShowWorkflowPicker, selectedScreenshotIndex, setSelectedScreenshotIndex,
    logsFilter, setLogsFilter,
  } = executionMode;

  const recordingMode = useRecordingModeState({
    sessionId, mode, executionId, autoStartAI, aiModel, aiMaxSteps,
    workflowNodes, workflowEdges,
  });
  const {
    isSidebarOpen, setSidebarOpen, handleSidebarToggle, sidebarActiveTab, setSidebarTab,
    aiSettings, updateAISettings, aiMessages, aiSendMessage, aiAbortNavigation, aiResumeNavigation,
    aiClearConversation, aiIsNavigating, aiAvailableModels, aiHumanIntervention,
    isRecording, isLoading, error, startRecording, clearActions, insertAction,
    generateWorkflow, validateSelector, replayPreview, isReplaying,
    lowConfidenceCount, mediumConfidenceCount, openPages, activePageId, switchToPage, closePage,
    createPage, isPagesLoading, recentActivityPageId, isTimelineLive,
    identifiedActions, timelineDisplayItems, pageColorMap, timelineItemCount, selectedActionIds,
    selectedIndices, selectedIndicesArray, isSelectionMode, toggleSelectionMode, handleActionClick,
    selectAll, selectNone, exitSelectionMode, handleDeleteAction, handleEditSelector, handleEditPayload,
  } = recordingMode;
  // Legacy form APIs consume indexes; resolve those indexes from canonical IDs at this boundary.
  const selectedActionIndices = identifiedActions.flatMap((action, index) => selectedActionIds.includes(action.id) ? [index] : []);

  const handleModeChange = useCallback((newMode: TimelineMode) => {
    if (newMode === mode) return;
    setMode(newMode);
    if (newMode === 'recording') {
      setSelectedWorkflowId(null);
      setSelectedProjectId(null);
      setSelectedWorkflowName(null);
      setLocalExecutionId(null);
    }
  }, [mode, setLocalExecutionId, setSelectedProjectId, setSelectedWorkflowId, setSelectedWorkflowName]);

  const handleWorkflowSelect = useCallback((
    workflowId: string,
    projectId: string,
    name: string,
    defaultSessionId?: string | null,
  ) => {
    selectWorkflow(workflowId, projectId, name, defaultSessionId);
    setMode('execution');
  }, [selectWorkflow]);

  // Unified replay style and settings state (shared across all preview panels)
  const [showReplayStyle, setShowReplayStyle] = useState(false);
  const [showPreviewSettings, setShowPreviewSettings] = useState(false);

  // Live recording metadata is stored in sessionStore so stream updates do not rerender this whole page.
  const setRecordingPageTitle = useSessionStore(s => s.setRecordingPageTitle);
  const setRecordingFrameStats = useSessionStore(s => s.setRecordingFrameStats);
  const clearLivePreviewMetadata = useSessionStore(s => s.clearLivePreviewMetadata);

  useEffect(() => {
    clearLivePreviewMetadata();
  }, [clearLivePreviewMetadata, sessionId]);
  const [executionWorkflowName, setExecutionWorkflowName] = useState<string | null>(null);
  const [executionCurrentUrl, setExecutionCurrentUrl] = useState<string>('');
  const [executionFooter, setExecutionFooter] = useState<ReactNode>(null);

  // Confirmation dialog for unsaved actions
  const { dialogState: confirmDialogState, confirm, close: closeConfirmDialog } = useConfirmDialog();

  // Error management state for enhanced error banner
  const [errorDetails, setErrorDetails] = useState<ErrorDetails | null>(null);
  const [showErrorDetailsModal, setShowErrorDetailsModal] = useState(false);
  const [dismissedErrors, setDismissedErrors] = useState<Set<string>>(new Set());

  // Show error details modal
  const handleShowErrorDetails = useCallback(() => {
    setShowErrorDetailsModal(true);
  }, []);

  // Close error details modal
  const handleCloseErrorDetails = useCallback(() => {
    setShowErrorDetailsModal(false);
  }, []);

  // Helper to create error details from session or recording errors
  const createErrorDetails = useCallback((
    message: string,
    source: ErrorDetails['source'],
    rawError?: unknown
  ): ErrorDetails => ({
    message,
    timestamp: new Date(),
    source,
    sessionId: sessionId ?? undefined,
    url: undefined,
    code: typeof rawError === 'object' && rawError !== null && 'code' in rawError
      ? String((rawError as { code: unknown }).code)
      : undefined,
    stackTrace: rawError instanceof Error ? rawError.stack : undefined,
    rawError,
  }), [sessionId]);

  // Dismiss error handler - clear error details and add to dismissed set
  const handleDismissError = useCallback(() => {
    const errorMessage = sessionError ?? navigationError ?? error;
    if (errorMessage && errorDetails) {
      const errorKey = `${errorDetails.source}-${errorMessage}`;
      setDismissedErrors(prev => new Set(prev).add(errorKey));
    }
    setErrorDetails(null);
  }, [sessionError, navigationError, error, errorDetails]);

  // Update error details when sessionError or error changes
  useEffect(() => {
    const errorMessage = sessionError ?? navigationError ?? error;
    if (errorMessage) {
      const source: ErrorDetails['source'] = sessionError ? 'session' : navigationError ? 'api' : 'recording';
      const errorKey = `${source}-${errorMessage}`;
      if (!dismissedErrors.has(errorKey)) {
        setErrorDetails(createErrorDetails(errorMessage, source));
      }
    }
  }, [sessionError, navigationError, error, createErrorDetails, dismissedErrors]);

  // Handle Execute button click - opens workflow picker with optional confirmation
  const handleExecuteClick = useCallback(async () => {
    // If we have unsaved recorded actions, confirm before switching
    if (identifiedActions.length > 0 && !isRecording) {
      const confirmed = await confirm({
        title: 'Unsaved Recording',
        message: 'You have recorded actions that have not been saved as a workflow. Switching to execution mode will not save these actions. Continue?',
        confirmLabel: 'Continue',
        cancelLabel: 'Go Back',
        danger: false,
      });
      if (!confirmed) return;
    }
    setShowWorkflowPicker(true);
  }, [identifiedActions.length, isRecording, confirm]);

  // Create session when we have a URL or when in recording mode with a profile
  // (profile may have saved tabs to restore, which provides the initial URL)
  // This is separate from navigation to avoid race conditions
  useEffect(() => {
    // Already have a session - nothing to do
    if (sessionId) return;

    // In recording mode with a profile, create session immediately
    // (backend restores tabs; the canonical page snapshot supplies their URLs)
    const profileId = selectedProfileId ?? sessionProfileId;
    const shouldCreateWithoutUrl = mode === 'recording' && profileId;

    // Need either a URL or the ability to create without one
    if (!previewUrl && !shouldCreateWithoutUrl) {
      return;
    }

    let cancelled = false;

    const createSessionForUrl = async () => {
      try {
        // Restore tabs only in recording mode, not in execution mode (clean start for workflows)
        const shouldRestoreTabs = mode === 'recording';
        const newSessionId = await ensureSession(
          previewViewport,
          profileId,
          streamSettingsRef.current,
          shouldRestoreTabs
        );
        if (cancelled || !newSessionId) return;
        // Session is now created, the navigation effect will handle navigation
        // Restored tabs are observed through the canonical page snapshot.
      } catch (err) {
        if (cancelled) return;
        console.warn('Failed to create session for URL', err);
      }
    };

    void createSessionForUrl();

    return () => {
      cancelled = true;
    };
  }, [previewUrl, sessionId, ensureSession, previewViewport, selectedProfileId, sessionProfileId, mode]);

  // NOTE: Viewport sync is centralized in RecordingSession via ViewportSyncManager.
  // PreviewContainer measures bounds and calls handleBrowserViewportChange.
  // The manager handles debouncing, resize detection, and CDP screencast restart.
  // The browser viewport is decoupled from replay style - toggling replay style
  // does NOT change the actual browser viewport, preventing flickering.

  // Auto-start recording AFTER initial navigation completes.
  // CRITICAL: This prevents frame streaming from capturing about:blank
  // before the page has navigated to the target URL. Without this guard,
  // startRecording races with navigate, causing flickering/white screen.
  useEffect(() => {
    if (sessionId && !isRecording && !autoStartedRef.current && isInitialNavigationComplete) {
      autoStartedRef.current = true;
      startRecording(sessionId).catch((err) => {
        console.error('Failed to auto-start recording:', err);
      });
    }
  }, [sessionId, isRecording, startRecording, isInitialNavigationComplete]);

  // Auto-start AI navigation when requested (e.g., from template)
  const aiAutoStartedRef = useRef(false);
  useEffect(() => {
    if (
      autoStartAI &&
      aiPrompt &&
      isInitialNavigationComplete &&
      sessionId &&
      !aiAutoStartedRef.current &&
      !aiIsNavigating
    ) {
      aiAutoStartedRef.current = true;
      // Send the initial prompt to start AI navigation
      aiSendMessage(aiPrompt).catch((err) => {
        console.error('Failed to auto-start AI navigation:', err);
      });
    }
  }, [autoStartAI, aiPrompt, isInitialNavigationComplete, sessionId, aiIsNavigating, aiSendMessage]);

  // Reset state when session changes
  useEffect(() => {
    autoStartedRef.current = false;
    aiAutoStartedRef.current = false;
    exitSelectionMode(); // Clear selection when switching sessions
    setRightPanelView('preview');
  }, [sessionId, exitSelectionMode]);

  // Flush session state when leaving the page or closing the tab
  useEffect(() => {
    if (!sessionId) return;

    const persist = async () => {
      try {
        const config = await getConfig();
        const url = `${config.API_URL}/recordings/live/${sessionId}/persist`;
        if (navigator.sendBeacon) {
          const blob = new Blob([], { type: 'application/json' });
          navigator.sendBeacon(url, blob);
        } else {
          await fetch(url, { method: 'POST', keepalive: true });
        }
      } catch (err) {
        console.warn('Failed to persist session before unload', err);
      }
    };

    const handleBeforeUnload = () => {
      void persist();
    };

    window.addEventListener('beforeunload', handleBeforeUnload);

    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
      void persist();
    };
  }, [sessionId]);

  const handleClearActions = useCallback(() => {
    clearActions();
    exitSelectionMode();
    setShowClearConfirm(false);
  }, [clearActions, exitSelectionMode]);

  // Navigate to workflow creation form
  const handleCreateWorkflow = useCallback(() => {
    // If not in selection mode and clicking "Create Workflow",
    // select all actions by default
    if (!isSelectionMode || selectedIndicesArray.length === 0) {
      selectAll();
    }
    setRightPanelView('create-workflow');
  }, [isSelectionMode, selectedIndicesArray.length, selectAll]);

  // Handle back from workflow creation form
  const handleBackToPreview = useCallback(() => {
    setRightPanelView('preview');
  }, []);

  // Navigate to AI navigation mode - switch to Auto tab in sidebar
  const handleAINavigation = useCallback(() => {
    setSidebarTab('auto');
    // Ensure sidebar is open
    setSidebarOpen(true);
  }, [setSidebarTab, setSidebarOpen]);

  // Handle inserting a new step from the InsertNodeModal
  const handleInsertStep = useCallback(
    (action: InsertedAction) => {
      // Convert InsertedAction to InsertActionData format
      const actionData: InsertActionData = {
        actionType: action.type as InsertActionData['actionType'],
        payload: action.params,
        selector: action.params.selector as string | undefined,
      };
      insertAction(actionData);
    },
    [insertAction]
  );

  // Test selected actions
  const handleTestSelectedActions = useCallback(
    async (actionIndices: number[]): Promise<ReplayPreviewResponse> => {
      const selected = mode === 'recording'
        ? actionIndices
          .map((index) => timelineDisplayItems[index]?.id)
          .filter((id): id is string => Boolean(id))
          .map((id) => identifiedActions.find((action) => action.id === id))
          .filter((action): action is NonNullable<typeof action> => action !== undefined)
        : [];
      const actionsToReplay = selected.length > 0 ? selected : identifiedActions;
      const results = await replayPreview(
        { stopOnFailure: true },
        actionsToReplay,
      );
      return results;
    },
    [identifiedActions, timelineDisplayItems, mode, replayPreview]
  );

  // Generate workflow from selected actions
  const handleGenerateFromSelection = useCallback(
    async (params: {
      name: string;
      projectId: string;
      defaultSessionId: string | null;
      actionIndices: number[];
      workflowType?: 'action' | 'flow' | 'case';
      path?: string;
      referenceWorkflowId?: string;
      compositionMode?: 'inline' | 'reference';
      settings?: WorkflowSettingsTyped;
    }) => {
      setIsGenerating(true);
      try {
        // For reference mode, we create a workflow with a single subflow node
        // that references the existing workflow
        if (params.compositionMode === 'reference' && params.referenceWorkflowId) {
          // Create workflow via project files API with subflow node
          const apiUrl = (await import('@/config')).getApiBase();
          const subflowNode = {
            id: `subflow-${params.referenceWorkflowId.slice(0, 8)}`,
            type: 'subflow',
            data: {
              label: `Reference: ${params.name}`,
              workflowId: params.referenceWorkflowId,
            },
            position: { x: 250, y: 100 },
          };

          const flowDefinition = {
            nodes: [subflowNode],
            edges: [],
          };

          const response = await fetch(`${apiUrl}/projects/${params.projectId}/files`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              path: params.path || `${params.name.toLowerCase().replace(/\s+/g, '-')}.${params.workflowType || 'flow'}.json`,
              workflow: {
                name: params.name,
                type: params.workflowType || 'flow',
                flow_definition: flowDefinition,
                settings: params.settings,
              },
            }),
          });

          if (!response.ok) {
            const errorPayload: unknown = await response.json().catch(() => null);
            const message =
              isRecord(errorPayload) && typeof errorPayload.error === 'string'
                ? errorPayload.error
                : `Failed to create workflow: ${response.statusText}`;
            throw new Error(message);
          }

          const resultPayload: unknown = await response.json();
          const { workflowId } = parseWorkflowGenerationResult(resultPayload);

          // Reset state
          setRightPanelView('preview');
          exitSelectionMode();

          if (onWorkflowGenerated && workflowId) {
            onWorkflowGenerated(workflowId, params.projectId);
          }
        } else {
          // Inline mode: use existing generate workflow API
          // TODO: API should support generating from subset of actions
          // For now, generate from all actions
          const selectedActions = mode === 'recording'
            ? params.actionIndices.map((index) => identifiedActions[index]).filter((action): action is NonNullable<typeof action> => action !== undefined)
            : [];
          const actionsToGenerate = selectedActions.length > 0 ? selectedActions : identifiedActions;
          const resultPayload: unknown = await generateWorkflow(
            params.name,
            params.projectId,
            actionsToGenerate,
            params.settings
          );
          const { workflowId, projectId } = parseWorkflowGenerationResult(resultPayload);

          // Reset state
          setRightPanelView('preview');
          exitSelectionMode();

          if (onWorkflowGenerated && workflowId && projectId) {
            onWorkflowGenerated(workflowId, projectId);
          }
        }
      } finally {
        setIsGenerating(false);
      }
    },
    [generateWorkflow, exitSelectionMode, identifiedActions, mode, onWorkflowGenerated]
  );

  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;
  const displayError = sessionError ?? navigationError ?? error;

  return (
    <Profiler id="RecordingSession" onRender={onProfilerRender}>
    <ViewportProvider sessionId={sessionId} pageId={navigationPage?.id ?? null} actualViewport={sessionActualViewport}>
    <div className="flex flex-col h-full bg-flow-bg text-flow-text">
      <RecordingHeader
        isRecording={mode === 'recording' && isRecording}
        onClose={onClose}
        mode={mode}
        onModeChange={handleModeChange}
        showModeToggle={true}
        onExecuteClick={handleExecuteClick}
        selectedWorkflowName={selectedWorkflowName}
        showRunButton={!!canRun}
        onRun={handleRun}
        isExecuting={isExecuting}
        onStop={handleStop}
        sessionReadOnly={isReadOnly}
        sessionProfiles={sessionProfiles.profiles}
        sessionProfilesLoading={sessionProfiles.loading}
        selectedSessionProfileId={selectedProfileId}
        onSelectSessionProfile={handleSelectSessionProfile}
        onCreateSessionProfile={handleCreateSessionProfile}
        onConfigureSession={handleConfigureSession}
        onNavigateToSessionSettings={handleNavigateToSessionSettings}
        workflowType={workflowType}
      />

      {/* Tab bar - always visible for consistent layout */}
      <TabBar
        pages={openPages}
        activePageId={activePageId}
        onTabClick={switchToPage}
        onTabClose={closePage}
        onCreateTab={() => createPage()}
        isLoading={isPagesLoading}
        recentActivityPageId={recentActivityPageId}
      />

      {/* Error display - pass retry state for session errors */}
      {displayError && !dismissedErrors.has(`${errorDetails?.source ?? 'unknown'}-${displayError}`) && (
        <ErrorBanner
          message={displayError}
          retryState={sessionError ? retryState : undefined}
          onRetry={sessionError ? retrySession : undefined}
          onDismiss={handleDismissError}
          onShowDetails={handleShowErrorDetails}
          hasDetails={!!errorDetails}
        />
      )}

      {/* Error details modal */}
      <ErrorDetailsModal
        open={showErrorDetailsModal}
        error={errorDetails}
        onClose={handleCloseErrorDetails}
      />

      {/* Unstable selectors warning banner */}
      {!isRecording && lowConfidenceCount > 0 && (
        <UnstableSelectorsBanner lowConfidenceCount={lowConfidenceCount} />
      )}

      {/* Main content split: sidebar + right panel (preview or workflow form) */}
      <div className="flex-1 overflow-hidden flex">
        <UnifiedSidebar
          mode={mode}
          isOpen={isSidebarOpen}
          onOpenChange={setSidebarOpen}
          activeTab={sidebarActiveTab}
          onTabChange={setSidebarTab}
          timelineProps={{
            actions: identifiedActions,
            timelineItems: timelineDisplayItems,
            itemCountOverride: timelineItemCount,
            mode,
            isRecording,
            isLoading,
            isReplaying,
            isLive: isTimelineLive,
            hasUnstableSelectors,
            onClearRequested: () => setShowClearConfirm(true),
            onCreateWorkflow: handleCreateWorkflow,
            onDeleteAction: handleDeleteAction,
            onValidateSelector: validateSelector,
            onEditSelector: handleEditSelector,
            onEditPayload: handleEditPayload,
            isSelectionMode,
            selectedIndices,
            onToggleSelectionMode: toggleSelectionMode,
            onActionClick: handleActionClick,
            onSelectAll: selectAll,
            onSelectNone: selectNone,
            onAINavigation: handleAINavigation,
            onInsertStep: handleInsertStep,
            pages: openPages,
            pageColorMap,
          }}
          autoProps={{
            messages: aiMessages,
            isNavigating: aiIsNavigating,
            settings: aiSettings,
            availableModels: aiAvailableModels,
            onSendMessage: aiSendMessage,
            onAbort: aiAbortNavigation,
            onHumanDone: aiResumeNavigation,
            onSettingsChange: updateAISettings,
            onClear: aiClearConversation,
          }}
          artifactsProps={{
            screenshots: currentExecution?.screenshots ?? [],
            selectedScreenshotIndex: selectedScreenshotIndex,
            onSelectScreenshot: setSelectedScreenshotIndex,
            executionLogs: currentExecution?.logs ?? [],
            logFilter: logsFilter,
            onLogFilterChange: setLogsFilter,
            consoleLogs: extractConsoleLogs(currentExecution?.timeline ?? []),
            networkEvents: extractNetworkEvents(currentExecution?.timeline ?? []),
            domSnapshots: extractDomSnapshots(currentExecution?.timeline ?? []),
            executionStatus: executionStatus ?? undefined,
          }}
          historyProps={mode === 'execution' && selectedWorkflowId ? {
            workflowId: selectedWorkflowId,
            currentExecutionId: localExecutionId ?? undefined,
            onSelectExecution: (execId) => setLocalExecutionId(execId),
          } : undefined}
        />

        <div className="flex-1 h-full transition-all duration-300">
          {rightPanelView === 'preview' && (
            <div className="relative h-full">
              {mode === 'execution' && localExecutionId && executionStatus ? (
                // Show execution viewer when execution exists (pending/running/completed/failed)
                <PreviewContainer
                  showReplayStyle={showReplayStyle}
                  onReplayStyleToggle={() => setShowReplayStyle((prev) => !prev)}
                  onSettingsClick={() => setShowPreviewSettings(true)}
                  isSettingsPanelOpen={showPreviewSettings}
                  isSidebarOpen={isSidebarOpen}
                  onToggleSidebar={handleSidebarToggle}
                  actionCount={timelineItemCount}
                  previewUrl={executionCurrentUrl}
                  onPreviewUrlChange={() => {}}
                  pageTitle={executionWorkflowName ?? undefined}
                  readOnly={true}
                  mode="execution"
                  executionStatus={executionStatus}
                  footer={executionFooter}
                >
                  <ExecutionPreviewPanel
                    executionId={localExecutionId}
                    onWorkflowNameChange={setExecutionWorkflowName}
                    onCurrentUrlChange={setExecutionCurrentUrl}
                    renderFooter={setExecutionFooter}
                    // Completion actions
                    onExport={exportController.openExportDialog}
                    onRerun={handleRerunExecution}
                    onEditWorkflow={handleEditWorkflow}
                    isExporting={exportController.isExporting}
                    canExport={(currentExecution?.timeline?.length ?? 0) > 0}
                    canRerun={!!selectedWorkflowId}
                    canEditWorkflow={!!(selectedWorkflowId && selectedProjectId)}
                  />
                </PreviewContainer>
              ) : mode === 'execution' && selectedWorkflowId ? (
                // Show workflow info card when workflow selected but no execution started yet
                <PreviewContainer
                  showReplayStyle={showReplayStyle}
                  onReplayStyleToggle={() => setShowReplayStyle((prev) => !prev)}
                  onSettingsClick={() => setShowPreviewSettings(true)}
                  isSettingsPanelOpen={showPreviewSettings}
                  isSidebarOpen={isSidebarOpen}
                  onToggleSidebar={handleSidebarToggle}
                  actionCount={timelineItemCount}
                  previewUrl=""
                  onPreviewUrlChange={() => {}}
                  pageTitle={selectedWorkflowName ?? 'Workflow'}
                  readOnly={true}
                  mode="execution"
                >
                  <WorkflowInfoCard
                    workflowId={selectedWorkflowId}
                    workflowName={selectedWorkflowName ?? 'Workflow'}
                    onRun={handleRun}
                    onChangeWorkflow={() => setShowWorkflowPicker(true)}
                  />
                </PreviewContainer>
              ) : (
                // Show recording preview in recording mode
                <PreviewContainer
                  showReplayStyle={showReplayStyle}
                  onReplayStyleToggle={() => setShowReplayStyle((prev) => !prev)}
                  onSettingsClick={() => setShowPreviewSettings(true)}
                  isSettingsPanelOpen={showPreviewSettings}
                  isSidebarOpen={isSidebarOpen}
                  onToggleSidebar={handleSidebarToggle}
                  actionCount={timelineItemCount}
                  previewUrl={previewUrl}
                  onPreviewUrlChange={setPreviewUrl}
                  onNavigate={handleNavigate}
                  onGoBack={handleGoBack}
                  onGoForward={handleGoForward}
                  onRefresh={handleRefresh}
                  canGoBack={canGoBack}
                  canGoForward={canGoForward}
                  onFetchNavigationStack={handleFetchNavigationStack}
                  onNavigateToIndex={handleNavigateToIndex}
                  onOpenHistorySettings={handleOpenHistorySettings}
                  placeholder="Search or enter URL"
                  targetFps={streamSettings?.fps ?? DEFAULT_STREAM_FPS}
                  showStats={showStats}
                  mode="recording"
                  // Viewport props (context handles sync and actual viewport)
                  onBrowserViewportChange={handleBrowserViewportChange}
                >
                  <RecordPreviewPanel
                    previewUrl={previewUrl}
                    onPreviewUrlChange={handleNavigate}
                    sessionId={sessionId}
                    activePageId={activePageId}
                    actions={identifiedActions}
                    // Viewport state now comes from ViewportProvider context
                    onConnectionStatusChange={setConnectionStatus}
                    hideConnectionIndicator={true}
                    onPageTitleChange={setRecordingPageTitle}
                    onFrameStatsChange={setRecordingFrameStats}
                    refreshToken={refreshToken}
                  />
                  {/* Human intervention overlay - shown over browser preview for maximum visibility */}
                  {aiHumanIntervention && (
                    <HumanInterventionOverlay
                      intervention={aiHumanIntervention}
                      onComplete={aiResumeNavigation}
                      onAbort={aiAbortNavigation}
                    />
                  )}
                </PreviewContainer>
              )}
            </div>
          )}
          {rightPanelView === 'create-workflow' && (
            <WorkflowCreationForm
              actions={identifiedActions}
              selectedIndices={selectedActionIndices}
              sessionProfiles={sessionProfiles.profiles}
              sessionProfilesLoading={sessionProfiles.loading}
              isReplaying={isReplaying}
              isGenerating={isGenerating}
              onBack={handleBackToPreview}
              onTest={handleTestSelectedActions}
              onGenerate={handleGenerateFromSelection}
              lowConfidenceCount={lowConfidenceCount}
              mediumConfidenceCount={mediumConfidenceCount}
            />
          )}
        </div>

        {/* Preview settings panel (inline, pushes content) */}
        <PreviewSettingsPanel
          isOpen={showPreviewSettings}
          onClose={() => setShowPreviewSettings(false)}
          sessionId={sessionId}
        />
      </div>

      {/* Clear confirmation modal */}
      <ClearActionsModal
        open={showClearConfirm}
        actionCount={identifiedActions.length}
        onCancel={() => setShowClearConfirm(false)}
        onConfirm={handleClearActions}
      />

      {/* Session settings modal */}
      {configuringProfile && (
        <SessionManager
          profileId={configuringProfile.id}
          profileName={configuringProfile.name}
          initialProfile={configuringProfile.browser_profile}
          hasStorageState={configuringProfile.has_storage_state}
          initialSection={configuringSection}
          onSave={handleSaveBrowserProfile}
          onClose={closeConfigureModal}
        />
      )}

      {/* Workflow picker modal */}
      <WorkflowPickerModal
        isOpen={showWorkflowPicker}
        onClose={() => setShowWorkflowPicker(false)}
        onSelect={handleWorkflowSelect}
        initialProjectId={selectedProjectId}
      />

      {/* Export Dialog */}
      <ExportDialogProvider
        value={buildExportDialogContextValue({
          dialogTitleId: exportDialogTitleId,
          dialogDescriptionId: exportDialogDescriptionId,
          onClose: exportController.closeExportDialog,
          onConfirm: exportController.confirmExport,
          ...exportController.exportDialogProps,
        })}
      >
        <ExportDialog isOpen={exportController.isExportDialogOpen} />
      </ExportDialogProvider>

      {/* Export Success Panel */}
      {exportController.showExportSuccess && exportController.lastCreatedExport && (
        <ExportSuccessPanel
          export_={exportController.lastCreatedExport}
          onClose={exportController.dismissExportSuccess}
          onViewInLibrary={() => {
            exportController.dismissExportSuccess();
            navigate('/exports');
          }}
        />
      )}

      {/* Confirmation dialog for unsaved actions */}
      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
    </div>
    </ViewportProvider>
    </Profiler>
  );
}
