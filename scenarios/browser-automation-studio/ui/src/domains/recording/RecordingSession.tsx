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

import { useCallback, useEffect, useRef, useState, useMemo, useId, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { RecordingHeader } from './capture/RecordingHeader';
import { TabBar } from './capture/TabBar';
import { ErrorBanner, UnstableSelectorsBanner } from './capture/RecordModeBanners';
import { ClearActionsModal } from './capture/RecordModeModals';
import { WorkflowCreationForm } from './conversion/WorkflowCreationForm';
import { WorkflowPickerModal } from './conversion/WorkflowPickerModal';
import { WorkflowInfoCard, type ExecutionConfigSettings } from './timeline/WorkflowInfoCard';
import type { BrowserProfile, RecordingSessionProfile, ReplayPreviewResponse } from './types/types';
import type { WorkflowSettingsTyped } from '@/types/workflow';
import { SessionManager } from '@/views/SettingsView/sections/sessions';
import { useRecordingSession } from './hooks/useRecordingSession';
import { useSessionProfiles } from './hooks/useSessionProfiles';
import { useRecordMode, type InsertActionData } from './hooks/useRecordMode';
import type { InsertedAction } from './InsertNodeModal';
import { useActionSelection } from './hooks/useActionSelection';
import { useUnifiedTimeline } from './hooks/useUnifiedTimeline';
import { usePages } from './hooks/usePages';
import { RecordPreviewPanel } from './timeline/RecordPreviewPanel';
import { ExecutionPreviewPanel } from './timeline/ExecutionPreviewPanel';
import { PreviewContainer } from './shared';
import { PreviewSettingsPanel } from '@/domains/preview-settings';
import { ViewportProvider } from './context';
import { mergeConsecutiveActions } from './utils/mergeActions';
import { recordedActionToTimelineItem } from './types/timeline-unified';
import { getConfig } from '@/config';
import { useStreamSettings, type StreamSettingsValues } from './capture/StreamSettings';
import type { StreamConnectionStatus, FrameStats } from './capture/PlaywrightView';
import { DEFAULT_STREAM_FPS, DEFAULT_STREAM_QUALITY } from './constants';
import type { TimelineMode } from './types/timeline-unified';
import { mergeActionsWithAISteps } from './types/timeline-unified';
import { UnifiedSidebar, useUnifiedSidebar, useAISettings } from './sidebar';
import { useAIConversation } from './ai-conversation';
import { HumanInterventionOverlay } from './ai-navigation';
import { useExecutionStore, useStartWorkflow, useExecutionEvents } from '@/domains/executions';
import { useExecutionExport } from '@/domains/executions/viewer/useExecutionExport';
import { useReplayCustomization } from '@/domains/executions/viewer/useReplayCustomization';
import { useExportStore } from '@/domains/exports';
import ExportDialog from '@/domains/executions/viewer/ExportDialog';
import { ExportSuccessPanel } from '@/domains/exports/ExportSuccessPanel';
import { useConfirmDialog } from '@/hooks/useConfirmDialog';
import { ConfirmDialog } from '@shared/ui/ConfirmDialog';
import toast from 'react-hot-toast';
import { extractConsoleLogs, extractNetworkEvents, extractDomSnapshots } from './utils/artifact-extraction';

/** LocalStorage key for persisting the last selected session profile ID */
const LAST_SELECTED_PROFILE_KEY = 'bas_last_selected_session_profile_id';

/** Workflow type being created (from AI modal or template) */
export type WorkflowTypeParam = 'action' | 'flow' | 'case';

interface RecordModePageProps {
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

export function RecordModePage({
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

  // Handle mode changes
  const handleModeChange = useCallback((newMode: TimelineMode) => {
    if (newMode === mode) return;
    setMode(newMode);
    // Clear timeline when switching modes
    if (newMode === 'recording') {
      // Switching to recording mode - timeline will be populated from actions
      // Clear workflow selection when going back to recording
      setSelectedWorkflowId(null);
      setSelectedProjectId(null);
      setSelectedWorkflowName(null);
      setLocalExecutionId(null);
    }
    // When switching to execution mode, the workflow selection will trigger execution
  }, [mode]);

  const sessionProfiles = useSessionProfiles();
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);

  const {
    sessionId,
    sessionProfileId,
    sessionError,
    actualViewport: sessionActualViewport,
    ensureSession,
    setSessionProfileId,
  } = useRecordingSession({ initialSessionId, onSessionReady });

  // Right panel view state
  const [rightPanelView, setRightPanelView] = useState<RightPanelView>('preview');
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  // Session settings modal state
  const [configuringProfile, setConfiguringProfile] = useState<RecordingSessionProfile | null>(null);
  // Initial section to open in session settings dialog (e.g., 'history' when opening from navigation popup)
  const [configuringSection, setConfiguringSection] = useState<'history' | undefined>(undefined);
  // Initialize previewUrl from template's initialUrl if provided
  const [previewUrl, setPreviewUrl] = useState(initialUrl || '');
  const [previewViewport, setPreviewViewport] = useState<{ width: number; height: number } | null>(null);

  // Handler for PreviewContainer's browser viewport changes (for session creation)
  const handleBrowserViewportChange = useCallback((viewport: { width: number; height: number }) => {
    setPreviewViewport(viewport);
  }, []);

  // Navigation handlers for browser back/forward/refresh
  const handleGoBack = useCallback(async () => {
    if (!sessionId) return;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/go-back`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setPreviewUrl(data.url || '');
        setCanGoBack(data.can_go_back ?? false);
        setCanGoForward(data.can_go_forward ?? false);
      }
    } catch (err) {
      console.warn('Failed to go back:', err);
    }
  }, [sessionId]);

  const handleGoForward = useCallback(async () => {
    if (!sessionId) return;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/go-forward`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setPreviewUrl(data.url || '');
        setCanGoBack(data.can_go_back ?? false);
        setCanGoForward(data.can_go_forward ?? false);
      }
    } catch (err) {
      console.warn('Failed to go forward:', err);
    }
  }, [sessionId]);

  const handleRefresh = useCallback(async () => {
    if (!sessionId) return;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/reload`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setCanGoBack(data.can_go_back ?? false);
        setCanGoForward(data.can_go_forward ?? false);
        // Trigger a refresh of the frame display
        setRefreshToken((t) => t + 1);
      }
    } catch (err) {
      console.warn('Failed to refresh:', err);
    }
  }, [sessionId]);

  // Fetch navigation stack for right-click popup
  const handleFetchNavigationStack = useCallback(async () => {
    if (!sessionId) return null;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/navigation-stack`);
      if (!response.ok) return null;
      const data = await response.json();
      return {
        backStack: data.back_stack || [],
        current: data.current || null,
        forwardStack: data.forward_stack || [],
      };
    } catch (err) {
      console.warn('Failed to fetch navigation stack:', err);
      return null;
    }
  }, [sessionId]);

  // Navigate to a specific delta (e.g., -2 for 2 steps back, +3 for 3 steps forward)
  const handleNavigateToIndex = useCallback(async (delta: number) => {
    if (!sessionId || delta === 0) return;
    try {
      const config = await getConfig();
      // For multiple steps, we call go-back or go-forward multiple times
      const endpoint = delta < 0 ? 'go-back' : 'go-forward';
      const steps = Math.abs(delta);

      let lastResponse: { url?: string; can_go_back?: boolean; can_go_forward?: boolean } | null = null;

      for (let i = 0; i < steps; i++) {
        const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/${endpoint}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({}),
        });
        if (!response.ok) break;
        lastResponse = await response.json();
      }

      if (lastResponse) {
        setPreviewUrl(lastResponse.url || '');
        setCanGoBack(lastResponse.can_go_back ?? false);
        setCanGoForward(lastResponse.can_go_forward ?? false);
      }
    } catch (err) {
      console.warn('Failed to navigate to index:', err);
    }
  }, [sessionId]);

  const autoStartedRef = useRef(false);

  // Stream settings for session creation (from shared context)
  const { settings: streamSettings, showStats } = useStreamSettings();
  const streamSettingsRef = useRef<StreamSettingsValues | null>(null);
  streamSettingsRef.current = streamSettings;

  // Connection status for header indicator
  const [connectionStatus, setConnectionStatus] = useState<StreamConnectionStatus | null>(null);

  // Workflow selection and execution state
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(initialWorkflowId ?? null);
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(initialProjectId ?? null);
  const [selectedWorkflowName, setSelectedWorkflowName] = useState<string | null>(null);
  const [localExecutionId, setLocalExecutionId] = useState<string | null>(executionId ?? null);
  // Show workflow picker only if no initial workflow provided and we're in execution mode
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false);

  // Workflow nodes/edges for timeline pre-population
  const [workflowNodes, setWorkflowNodes] = useState<Array<{ id: string; type?: string; data?: Record<string, unknown>; action?: { type: string; metadata?: { label?: string }; navigate?: { url?: string } } }>>([]);
  const [workflowEdges, setWorkflowEdges] = useState<Array<{ source: string; target: string }>>([]);

  // Execution sidebar state (screenshots and logs tabs)
  const [selectedScreenshotIndex, setSelectedScreenshotIndex] = useState(0);
  const [logsFilter, setLogsFilter] = useState<'all' | 'error' | 'warning' | 'info' | 'success'>('all');

  // Unified replay style and settings state (shared across all preview panels)
  const [showReplayStyle, setShowReplayStyle] = useState(false);
  const [showPreviewSettings, setShowPreviewSettings] = useState(false);

  // Metadata from content panels for PreviewContainer's BrowserChrome
  const [recordingPageTitle, setRecordingPageTitle] = useState<string>('');
  const [recordingFrameStats, setRecordingFrameStats] = useState<FrameStats | null>(null);
  const [executionWorkflowName, setExecutionWorkflowName] = useState<string | null>(null);
  const [executionCurrentUrl, setExecutionCurrentUrl] = useState<string>('');
  const [executionFooter, setExecutionFooter] = useState<ReactNode>(null);
  const [refreshToken, setRefreshToken] = useState(0);
  // Navigation history state for back/forward buttons
  const [canGoBack, setCanGoBack] = useState(false);
  const [canGoForward, setCanGoForward] = useState(false);

  // Confirmation dialog for unsaved actions
  const { dialogState: confirmDialogState, confirm, close: closeConfirmDialog } = useConfirmDialog();

  // Execution store for status tracking
  const currentExecution = useExecutionStore(s => s.currentExecution);
  const stopExecution = useExecutionStore(s => s.stopExecution);

  // Derived execution status
  const executionStatus = currentExecution?.id === localExecutionId ? currentExecution.status : null;
  // Can run only when workflow selected but no execution started yet
  const canRun = selectedWorkflowId && !localExecutionId;
  const isExecuting = executionStatus === 'running';
  // Session is read-only whenever viewing an execution (any status)
  const isReadOnly = !!(localExecutionId && executionStatus);

  // Subscribe to execution events via WebSocket
  // This updates the store when execution status changes (pending -> running -> completed)
  useExecutionEvents(
    localExecutionId && executionStatus
      ? { id: localExecutionId, status: executionStatus }
      : undefined
  );

  // Export infrastructure for execution completion actions
  const { createExport } = useExportStore();
  const exportDialogTitleId = useId();
  const exportDialogDescriptionId = useId();
  const replayCustomization = useReplayCustomization({
    executionId: localExecutionId ?? '',
  });

  // Create a placeholder execution for the hook when no execution is loaded
  // The hook will be effectively disabled but hooks must be called unconditionally
  const defaultExecution: import('@/domains/executions').Execution = useMemo(() => ({
    id: '',
    workflowId: '',
    status: 'pending' as const,
    startedAt: new Date(),
    screenshots: [],
    timeline: [],
    logs: [],
    progress: 0,
  }), []);

  const exportController = useExecutionExport({
    execution: currentExecution ?? defaultExecution,
    replayFrames: currentExecution?.timeline ?? [],
    workflowName: selectedWorkflowName ?? 'Workflow',
    replayCustomization,
    createExport: createExport as Parameters<typeof useExecutionExport>[0]['createExport'],
  });

  // Start workflow hook
  const { startWorkflow } = useStartWorkflow({
    onSuccess: (execId) => {
      setLocalExecutionId(execId);
    },
    onError: (error) => {
      toast.error(error.message || 'Failed to start workflow execution');
    },
  });

  // Unified sidebar state - auto-switch to Auto tab if autoStartAI
  const {
    isOpen: isSidebarOpen,
    setIsOpen: setSidebarOpen,
    toggleOpen: handleSidebarToggle,
    activeTab: sidebarActiveTab,
    setActiveTab: setSidebarTab,
    setAutoActivity,
  } = useUnifiedSidebar({
    initialTab: autoStartAI ? 'auto' : 'timeline',
  });

  // AI settings (for Auto tab)
  const {
    settings: aiSettings,
    updateSettings: updateAISettings,
  } = useAISettings({
    initialSettings: {
      model: aiModel,
      maxSteps: aiMaxSteps,
    },
  });

  // AI Conversation - wraps useAINavigation with chat message management
  const {
    messages: aiMessages,
    sendMessage: aiSendMessage,
    abortNavigation: aiAbortNavigation,
    resumeNavigation: aiResumeNavigation,
    clearConversation: aiClearConversation,
    isNavigating: aiIsNavigating,
    navigationSteps: aiSteps,
    availableModels: aiAvailableModels,
    humanIntervention: aiHumanIntervention,
  } = useAIConversation({
    sessionId,
    settings: aiSettings,
    onTimelineAction: () => {
      // Flash the timeline tab when AI performs an action
      setAutoActivity(true);
    },
  });

  const {
    isRecording,
    actions,
    isLoading,
    error,
    startRecording,
    clearActions,
    deleteAction,
    insertAction,
    updateSelector,
    updatePayload,
    generateWorkflow,
    validateSelector,
    replayPreview,
    isReplaying,
    lowConfidenceCount,
    mediumConfidenceCount,
  } = useRecordMode({
    sessionId,
    pollInterval: 500, // Poll every 500ms for faster feedback
    onActionsReceived: (newActions) => {
      console.log('New actions received:', newActions.length);
    },
  });

  // Handle Execute button click - opens workflow picker with optional confirmation
  const handleExecuteClick = useCallback(async () => {
    // If we have unsaved recorded actions, confirm before switching
    if (actions.length > 0 && !isRecording) {
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
  }, [actions.length, isRecording, confirm]);

  // Handle workflow selection from picker
  const handleWorkflowSelect = useCallback((
    workflowId: string,
    projectId: string,
    name: string,
    defaultSessionId?: string | null
  ) => {
    setSelectedWorkflowId(workflowId);
    setSelectedProjectId(projectId);
    setSelectedWorkflowName(name);
    setLocalExecutionId(null); // Clear any previous execution
    setShowWorkflowPicker(false);
    setMode('execution');

    // Auto-select the workflow's default session if provided
    if (defaultSessionId) {
      setSelectedProfileId(defaultSessionId);
      setSessionProfileId(defaultSessionId);
    }
  }, [setSessionProfileId]);

  // Handle Run button click - start execution with optional config overrides
  const handleRun = useCallback(async (overrides?: ExecutionConfigSettings) => {
    if (!selectedWorkflowId) return;
    try {
      // Use stream settings for frame streaming if available
      const currentStreamSettings = streamSettingsRef.current;
      await startWorkflow({
        workflowId: selectedWorkflowId,
        sessionProfileId: selectedProfileId,
        overrides: overrides ? {
          viewport_width: overrides.viewportWidth,
          viewport_height: overrides.viewportHeight,
          timeout_ms: overrides.actionTimeoutSeconds * 1000,
          navigation_wait_until: overrides.navigationWaitUntil,
          continue_on_error: overrides.continueOnError,
        } : undefined,
        artifactProfile: overrides?.artifactProfile,
        // Enable live frame streaming for real-time execution preview
        frameStreaming: {
          enabled: true,
          quality: currentStreamSettings?.quality ?? DEFAULT_STREAM_QUALITY,
          fps: currentStreamSettings?.fps ?? DEFAULT_STREAM_FPS,
        },
      });
    } catch {
      // Error is already handled by onError callback (shows toast)
      // Catch here to prevent unhandled promise rejection
    }
  }, [selectedWorkflowId, selectedProfileId, startWorkflow]);

  // Handle Stop button click - stop execution
  const handleStop = useCallback(async () => {
    if (localExecutionId) {
      await stopExecution(localExecutionId);
    }
  }, [localExecutionId, stopExecution]);

  // Handle Re-run button click - clear execution and start fresh
  const handleRerunExecution = useCallback(() => {
    setLocalExecutionId(null);
    // Small delay to ensure state clears before re-running
    setTimeout(() => {
      if (selectedWorkflowId) {
        void handleRun();
      }
    }, 100);
  }, [selectedWorkflowId, handleRun]);

  // Handle Edit Workflow button click - navigate to workflow editor
  const handleEditWorkflow = useCallback(() => {
    if (selectedWorkflowId && selectedProjectId) {
      navigate(`/projects/${selectedProjectId}/workflows/${selectedWorkflowId}`);
    }
  }, [selectedWorkflowId, selectedProjectId, navigate]);

  // Fetch workflow nodes/edges when a workflow is selected for execution mode
  useEffect(() => {
    console.log('[RecordingSession] Workflow fetch effect:', { selectedWorkflowId, mode });

    if (!selectedWorkflowId || mode !== 'execution') {
      console.log('[RecordingSession] Skipping fetch - conditions not met');
      setWorkflowNodes([]);
      setWorkflowEdges([]);
      return;
    }

    const fetchWorkflowDefinition = async () => {
      try {
        const config = await getConfig();
        console.log('[RecordingSession] Fetching workflow:', `${config.API_URL}/workflows/${selectedWorkflowId}`);
        const response = await fetch(`${config.API_URL}/workflows/${selectedWorkflowId}`);
        if (!response.ok) {
          console.error('[RecordingSession] Failed to fetch workflow definition, status:', response.status);
          return;
        }
        const data = await response.json();
        console.log('[RecordingSession] Raw workflow data:', JSON.stringify(data, null, 2).slice(0, 1000));

        // Extract nodes and edges from the workflow definition
        // API returns { workflow: { flow_definition: { nodes, edges } } }
        const workflow = data.workflow ?? data;
        const flowDef = workflow.flow_definition ?? workflow.flowDefinition ?? {};
        const nodes = flowDef.nodes ?? workflow.nodes ?? [];
        const edges = flowDef.edges ?? workflow.edges ?? [];

        console.log('[RecordingSession] Loaded workflow with', nodes.length, 'nodes and', edges.length, 'edges');
        console.log('[RecordingSession] First node sample:', nodes[0]);
        setWorkflowNodes(nodes);
        setWorkflowEdges(edges);
      } catch (error) {
        console.error('[RecordingSession] Error fetching workflow definition:', error);
      }
    };

    fetchWorkflowDefinition();
  }, [selectedWorkflowId, mode]);

  // Unified timeline for both recording and execution modes
  // TODO: Use clearTimelineItems, getRecordedActions, and timelineStats when fully integrated
  const {
    items: timelineItems,
    isLive: isTimelineLive,
    clearItems: _clearTimelineItems,
    getRecordedActions: _getRecordedActions,
    stats: _timelineStats,
  } = useUnifiedTimeline({
    mode,
    executionId,
    initialActions: actions,
    workflowNodes: mode === 'execution' ? workflowNodes : undefined,
    workflowEdges: mode === 'execution' ? workflowEdges : undefined,
  });

  // Page tracking for multi-tab recording
  const [recentActivityPageId, setRecentActivityPageId] = useState<string | null>(null);
  const {
    openPages,
    activePageId,
    pages: pagesMap,
    switchToPage,
    closePage,
    createPage,
    isLoading: isPagesLoading,
  } = usePages({
    sessionId,
    onPageCreated: (page) => {
      console.log('[RecordModePage] New page created:', page.id, page.url);
      // Show activity indicator briefly on new pages
      setRecentActivityPageId(page.id);
      setTimeout(() => setRecentActivityPageId(null), 2000);

      // Auto-switch to new tabs (e.g., opened by clicking target="_blank" links)
      // This gives the user immediate feedback of the new tab
      void switchToPage(page.id);
    },
    onPageClosed: (pageId) => {
      console.log('[RecordModePage] Page closed:', pageId);
      // If the closed page was the one being viewed, clear the URL
      // The openPages array will be updated after this callback, so we check
      // if we're closing the active page and there will be no pages left
      if (pageId === activePageId) {
        // Check how many open pages remain (excluding the one being closed)
        const remainingPages = openPages.filter(p => p.id !== pageId);
        if (remainingPages.length === 0) {
          // No more pages - clear the URL to show the empty state
          setPreviewUrl('');
        }
      }
    },
    onActivePageChanged: (pageId) => {
      console.log('[RecordModePage] Active page changed:', pageId);
      // Update the URL bar to show the new page's URL
      const page = pagesMap.get(pageId);
      if (page?.url) {
        setPreviewUrl(page.url);
      }
    },
    onPageNavigated: (pageId, url, _title, isActive) => {
      console.log('[RecordModePage] Page navigated:', pageId, url, 'isActive:', isActive);
      // Update the URL bar if the active page navigated
      if (isActive && url) {
        setPreviewUrl(url);
      }
    },
  });

  const mergedActions = useMemo(() => mergeConsecutiveActions(actions), [actions]);

  const mergedTimelineItems = useMemo(() => {
    if (mode !== 'recording') return timelineItems;
    // If we have AI steps, merge them with the recorded actions
    if (aiSteps.length > 0) {
      return mergeActionsWithAISteps(mergedActions, aiSteps);
    }
    return mergedActions.map((action) => recordedActionToTimelineItem(action));
  }, [mergedActions, mode, timelineItems, aiSteps]);

  // Page color palette for visual distinction in multi-tab recording
  const PAGE_COLORS = [
    'bg-blue-500',
    'bg-green-500',
    'bg-purple-500',
    'bg-orange-500',
    'bg-pink-500',
    'bg-cyan-500',
    'bg-yellow-500',
    'bg-red-500',
  ] as const;

  // Create a stable color map for pages based on creation order
  const pageColorMap = useMemo(() => {
    const map = new Map<string, typeof PAGE_COLORS[number]>();
    openPages.forEach((page, index) => {
      map.set(page.id, PAGE_COLORS[index % PAGE_COLORS.length]);
    });
    return map;
  }, [openPages]);

  const mergedIndexMap = useMemo(() => {
    const map = new Map<number, number[]>();
    mergedActions.forEach((action, index) => {
      const mergedIds = action._merged?.mergedIds ?? [action.id];
      const originalIndices = mergedIds
        .map((id) => actions.findIndex((raw) => raw.id === id))
        .filter((idx) => idx !== -1);
      map.set(index, originalIndices);
    });
    return map;
  }, [actions, mergedActions]);

  const handleDeleteMergedAction = useCallback(
    (index: number) => {
      const originalIndices = mergedIndexMap.get(index) ?? [];
      const sorted = [...originalIndices].sort((a, b) => b - a);
      for (const originalIndex of sorted) {
        deleteAction(originalIndex);
      }
    },
    [deleteAction, mergedIndexMap]
  );

  const handleEditMergedSelector = useCallback(
    (index: number, newSelector: string) => {
      const originalIndices = mergedIndexMap.get(index) ?? [];
      if (originalIndices.length > 0) {
        updateSelector(originalIndices[0], newSelector);
      }
    },
    [mergedIndexMap, updateSelector]
  );

  const handleEditMergedPayload = useCallback(
    (index: number, payload: Record<string, unknown>) => {
      const originalIndices = mergedIndexMap.get(index) ?? [];
      for (const originalIndex of originalIndices) {
        updatePayload(originalIndex, payload);
      }
    },
    [mergedIndexMap, updatePayload]
  );

  // Use unified timeline items count for selection
  const timelineItemCount = useMemo(() => {
    return mode === 'recording' ? mergedActions.length : timelineItems.length;
  }, [mergedActions.length, mode, timelineItems.length]);

  // Selection state for multi-step workflow creation
  const {
    selectedIndices,
    selectedIndicesArray,
    isSelectionMode,
    toggleSelectionMode,
    handleActionClick,
    selectAll,
    selectNone,
    exitSelectionMode,
  } = useActionSelection({ actionCount: timelineItemCount });

  const lastActionUrl = actions.length > 0 ? actions[actions.length - 1]?.url ?? '' : '';

  // Initialize profile selection - prefer localStorage, then API, then default
  useEffect(() => {
    // Skip if we already have a selection
    if (selectedProfileId) return;
    // Wait for profiles to load
    if (sessionProfiles.profiles.length === 0) return;

    // Priority 1: Check localStorage for last explicitly selected profile
    const storedProfileId = localStorage.getItem(LAST_SELECTED_PROFILE_KEY);
    if (storedProfileId && sessionProfiles.profiles.some((p) => p.id === storedProfileId)) {
      setSelectedProfileId(storedProfileId);
      setSessionProfileId(storedProfileId);
      return;
    }

    // Priority 2: Use sessionProfileId from hook or API default
    const maybeDefault = sessionProfileId ?? sessionProfiles.getDefaultProfileId();
    if (maybeDefault) {
      setSelectedProfileId(maybeDefault);
      setSessionProfileId(maybeDefault);
    }
  }, [selectedProfileId, sessionProfileId, sessionProfiles.profiles, sessionProfiles.getDefaultProfileId, setSessionProfileId]);

  useEffect(() => {
    if (sessionProfileId && sessionProfileId !== selectedProfileId) {
      setSelectedProfileId(sessionProfileId);
    }
  }, [sessionProfileId, selectedProfileId]);

  // Handle case where selected profile no longer exists (was deleted)
  useEffect(() => {
    if (
      selectedProfileId &&
      sessionProfiles.profiles.length > 0 &&
      !sessionProfiles.profiles.some((p) => p.id === selectedProfileId)
    ) {
      const fallback = sessionProfiles.getDefaultProfileId();
      setSelectedProfileId(fallback);
      setSessionProfileId(fallback);
      // Update localStorage with the fallback or clear it
      if (fallback) {
        localStorage.setItem(LAST_SELECTED_PROFILE_KEY, fallback);
      } else {
        localStorage.removeItem(LAST_SELECTED_PROFILE_KEY);
      }
    }
  }, [selectedProfileId, sessionProfiles.getDefaultProfileId, sessionProfiles.profiles, setSessionProfileId]);

  useEffect(() => {
    if (!previewUrl && lastActionUrl) {
      setPreviewUrl(lastActionUrl);
    }
  }, [lastActionUrl, previewUrl]);

  // Track whether initial URL navigation has been done to avoid double-navigation
  const initialUrlNavigatedRef = useRef(false);
  // Track whether initial navigation is complete (for AI auto-start)
  const [isInitialNavigationComplete, setIsInitialNavigationComplete] = useState(!initialUrl);
  // Ref to avoid stale closure when setting completion state
  const isInitialNavigationCompleteRef = useRef(!initialUrl);

  // Create session when we have a URL but no session yet
  // This is separate from navigation to avoid race conditions
  useEffect(() => {
    if (!previewUrl || sessionId) {
      return; // No URL needed or session already exists
    }

    let cancelled = false;

    const createSessionForUrl = async () => {
      try {
        const newSessionId = await ensureSession(
          previewViewport,
          selectedProfileId ?? sessionProfileId ?? null,
          streamSettingsRef.current
        );
        if (cancelled || !newSessionId) return;
        // Session is now created, the navigation effect will handle navigation
      } catch (err) {
        if (cancelled) return;
        console.warn('Failed to create session for URL', err);
      }
    };

    void createSessionForUrl();

    return () => {
      cancelled = true;
    };
  }, [previewUrl, sessionId, ensureSession, previewViewport, selectedProfileId, sessionProfileId]);

  useEffect(() => {
    // Don't navigate if no URL or no session yet
    // We wait for the session to be created by the effect above
    if (!previewUrl || !sessionId) {
      return;
    }

    const abortController = new AbortController();
    let cancelled = false;

    const syncPreviewToSession = async () => {
      try {
        // For initial URL from template, add a small delay to ensure session is ready
        // This prevents race conditions with playwright context initialization
        const isInitialNavigation = initialUrl && !initialUrlNavigatedRef.current;
        if (isInitialNavigation) {
          initialUrlNavigatedRef.current = true;
          await new Promise((resolve) => setTimeout(resolve, 500));
          if (cancelled) return;
        }

        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/navigate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: previewUrl }),
          signal: abortController.signal,
        });

        // Update navigation state from response
        if (response.ok && !cancelled) {
          try {
            const data = await response.json();
            setCanGoBack(data.can_go_back ?? false);
            setCanGoForward(data.can_go_forward ?? false);
          } catch {
            // Ignore JSON parse errors
          }
        }

        // Mark initial navigation as complete if this was the initial navigation
        // and it succeeded (or at least didn't throw)
        if (isInitialNavigation && !cancelled && response.ok) {
          // Add a brief delay to let the page start rendering
          await new Promise((resolve) => setTimeout(resolve, 1000));
          if (!cancelled) {
            isInitialNavigationCompleteRef.current = true;
            setIsInitialNavigationComplete(true);
          }
        }
      } catch (err) {
        if (abortController.signal.aborted || cancelled) return;
        console.warn('Failed to sync preview URL to recording session', err);
        // Still mark as complete on error so UI isn't stuck
        if (initialUrl && !isInitialNavigationCompleteRef.current) {
          isInitialNavigationCompleteRef.current = true;
          setIsInitialNavigationComplete(true);
        }
      }
    };

    void syncPreviewToSession();

    return () => {
      cancelled = true;
      abortController.abort();
    };
  }, [sessionId, previewUrl, initialUrl]); // Note: isInitialNavigationComplete intentionally NOT in deps to avoid re-triggering navigation

  // NOTE: Viewport sync is centralized in RecordingSession via ViewportSyncManager.
  // PreviewContainer measures bounds and calls handleBrowserViewportChange.
  // The manager handles debouncing, resize detection, and CDP screencast restart.
  // The browser viewport is decoupled from replay style - toggling replay style
  // does NOT change the actual browser viewport, preventing flickering.

  // Auto-start recording when session is ready
  useEffect(() => {
    if (sessionId && !isRecording && !autoStartedRef.current) {
      autoStartedRef.current = true;
      startRecording(sessionId).catch((err) => {
        console.error('Failed to auto-start recording:', err);
      });
    }
  }, [sessionId, isRecording, startRecording]);

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

  const handleSelectSessionProfile = useCallback(
    (profileId: string | null) => {
      setSelectedProfileId(profileId);
      setSessionProfileId(profileId);
      // Persist to localStorage for next visit
      if (profileId) {
        localStorage.setItem(LAST_SELECTED_PROFILE_KEY, profileId);
      } else {
        localStorage.removeItem(LAST_SELECTED_PROFILE_KEY);
      }
    },
    [setSessionProfileId]
  );

  const handleCreateSessionProfile = useCallback(async () => {
    const created = await sessionProfiles.create();
    if (created) {
      setSelectedProfileId(created.id);
      setSessionProfileId(created.id);
      // Persist newly created profile to localStorage
      localStorage.setItem(LAST_SELECTED_PROFILE_KEY, created.id);
    }
  }, [sessionProfiles, setSessionProfileId]);

  const handleConfigureSession = useCallback((profileId: string) => {
    const profile = sessionProfiles.profiles.find((p) => p.id === profileId);
    if (profile) {
      setConfiguringSection(undefined);
      setConfiguringProfile(profile);
    }
  }, [sessionProfiles.profiles]);

  // Open history settings for the current session profile
  const handleOpenHistorySettings = useCallback(() => {
    // Find the current profile from the session (use selectedProfileId as fallback
    // since sessionProfileId may not be set until a session is fully started)
    const profileId = sessionProfileId || selectedProfileId;
    const currentProfile = sessionProfiles.profiles.find((p) => p.id === profileId);
    if (currentProfile) {
      setConfiguringSection('history');
      setConfiguringProfile(currentProfile);
    }
  }, [sessionProfiles.profiles, sessionProfileId, selectedProfileId]);

  const handleNavigateToSessionSettings = useCallback(() => {
    navigate('/settings?tab=sessions');
  }, [navigate]);

  const handleSaveBrowserProfile = useCallback(
    async (browserProfile: BrowserProfile) => {
      if (!configuringProfile) return;
      await sessionProfiles.updateBrowserProfile(configuringProfile.id, browserProfile);
      // Refresh the profile in state
      await sessionProfiles.refresh();
    },
    [configuringProfile, sessionProfiles]
  );

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
        ? actionIndices.map((index) => mergedActions[index]).filter(Boolean)
        : [];
      const actionsToReplay = selected.length > 0 ? selected : mergedActions;
      const results = await replayPreview(
        { stopOnFailure: true },
        actionsToReplay,
      );
      return results;
    },
    [mergedActions, mode, replayPreview]
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
            const errorData = await response.json().catch(() => ({}));
            throw new Error(errorData.error || `Failed to create workflow: ${response.statusText}`);
          }

          const result = await response.json();

          // Reset state
          setRightPanelView('preview');
          exitSelectionMode();

          if (onWorkflowGenerated && result.workflow_id) {
            onWorkflowGenerated(result.workflow_id, params.projectId);
          }
        } else {
          // Inline mode: use existing generate workflow API
          // TODO: API should support generating from subset of actions
          // For now, generate from all actions
          const selectedActions = mode === 'recording'
            ? params.actionIndices.map((index) => mergedActions[index]).filter(Boolean)
            : [];
          const actionsToGenerate = selectedActions.length > 0 ? selectedActions : mergedActions;
          const result = await generateWorkflow(params.name, params.projectId, actionsToGenerate, params.settings);

          // Reset state
          setRightPanelView('preview');
          exitSelectionMode();

          if (onWorkflowGenerated) {
            onWorkflowGenerated(result.workflow_id, result.project_id);
          }
        }
      } finally {
        setIsGenerating(false);
      }
    },
    [generateWorkflow, exitSelectionMode, mergedActions, mode, onWorkflowGenerated]
  );

  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;
  const displayError = sessionError ?? error;

  return (
    <ViewportProvider sessionId={sessionId} actualViewport={sessionActualViewport}>
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
        connectionStatus={connectionStatus}
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

      {/* Error display */}
      {displayError && <ErrorBanner message={displayError} />}

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
            actions,
            timelineItems: mergedTimelineItems,
            itemCountOverride: timelineItemCount,
            mode,
            isRecording,
            isLoading,
            isReplaying,
            isLive: isTimelineLive,
            hasUnstableSelectors,
            onClearRequested: () => setShowClearConfirm(true),
            onCreateWorkflow: handleCreateWorkflow,
            onDeleteAction: handleDeleteMergedAction,
            onValidateSelector: validateSelector,
            onEditSelector: handleEditMergedSelector,
            onEditPayload: handleEditMergedPayload,
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
                  onNavigate={setPreviewUrl}
                  onGoBack={handleGoBack}
                  onGoForward={handleGoForward}
                  onRefresh={handleRefresh}
                  canGoBack={canGoBack}
                  canGoForward={canGoForward}
                  onFetchNavigationStack={handleFetchNavigationStack}
                  onNavigateToIndex={handleNavigateToIndex}
                  onOpenHistorySettings={handleOpenHistorySettings}
                  pageTitle={recordingPageTitle || undefined}
                  placeholder={actions[actions.length - 1]?.url || 'Search or enter URL'}
                  frameStats={recordingFrameStats}
                  targetFps={streamSettings?.fps ?? DEFAULT_STREAM_FPS}
                  showStats={showStats}
                  mode="recording"
                  // Viewport props (context handles sync and actual viewport)
                  onBrowserViewportChange={handleBrowserViewportChange}
                >
                  <RecordPreviewPanel
                    previewUrl={previewUrl}
                    onPreviewUrlChange={setPreviewUrl}
                    sessionId={sessionId}
                    activePageId={activePageId}
                    actions={actions}
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
              actions={actions}
              selectedIndices={selectedIndicesArray}
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
        actionCount={actions.length}
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
          onClose={() => {
            setConfiguringProfile(null);
            setConfiguringSection(undefined);
          }}
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
      <ExportDialog
        isOpen={exportController.isExportDialogOpen}
        onClose={exportController.closeExportDialog}
        onConfirm={exportController.confirmExport}
        dialogTitleId={exportDialogTitleId}
        dialogDescriptionId={exportDialogDescriptionId}
        {...exportController.exportDialogProps}
      />

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
  );
}
