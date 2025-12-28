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

import { useCallback, useEffect, useRef, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { RecordActionsPanel } from './capture/RecordActionsPanel';
import { RecordingHeader } from './capture/RecordingHeader';
import { TabBar } from './capture/TabBar';
import { ErrorBanner, UnstableSelectorsBanner } from './capture/RecordModeBanners';
import { ClearActionsModal } from './capture/RecordModeModals';
import { WorkflowCreationForm } from './conversion/WorkflowCreationForm';
import type { BrowserProfile, RecordingSessionProfile, ReplayPreviewResponse } from './types/types';
import { SessionManager } from '@/views/SettingsView/sections/sessions';
import { useRecordingSession } from './hooks/useRecordingSession';
import { useSessionProfiles } from './hooks/useSessionProfiles';
import { useRecordMode } from './hooks/useRecordMode';
import { useTimelinePanel } from './hooks/useTimelinePanel';
import { useActionSelection } from './hooks/useActionSelection';
import { useUnifiedTimeline } from './hooks/useUnifiedTimeline';
import { usePages } from './hooks/usePages';
import { RecordPreviewPanel } from './timeline/RecordPreviewPanel';
import { mergeConsecutiveActions } from './utils/mergeActions';
import { recordedActionToTimelineItem } from './types/timeline-unified';
import { getConfig } from '@/config';
import type { StreamSettingsValues } from './capture/StreamSettings';
import type { StreamConnectionStatus } from './capture/PlaywrightView';
import type { TimelineMode } from './types/timeline-unified';
import { mergeActionsWithAISteps } from './types/timeline-unified';
import { AINavigationView } from './ai-navigation';
import { useAINavigation } from './ai-navigation/useAINavigation';

interface RecordModePageProps {
  /** Browser session ID */
  sessionId: string | null;
  /** Mode: 'recording' for live recording, 'execution' for workflow playback */
  mode?: TimelineMode;
  /** Execution ID for execution mode (required when mode is 'execution') */
  executionId?: string | null;
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
}

/** Right panel view state */
type RightPanelView = 'preview' | 'create-workflow' | 'ai-navigation';

export function RecordModePage({
  sessionId: initialSessionId,
  mode: initialMode = 'recording',
  executionId,
  onWorkflowGenerated,
  onSessionReady,
  onClose,
  initialUrl,
  aiPrompt,
  aiModel,
  aiMaxSteps,
  autoStartAI,
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
    }
    // When switching to execution mode, the workflow selection will trigger execution
  }, [mode]);
  const sessionProfiles = useSessionProfiles();
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);

  const {
    sessionId,
    sessionProfileId,
    sessionError,
    ensureSession,
    setSessionProfileId,
  } = useRecordingSession({ initialSessionId, onSessionReady });

  // Right panel view state - auto-switch to AI view if autoStartAI is true
  const [rightPanelView, setRightPanelView] = useState<RightPanelView>(
    autoStartAI ? 'ai-navigation' : 'preview'
  );
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  // Session settings modal state
  const [configuringProfile, setConfiguringProfile] = useState<RecordingSessionProfile | null>(null);
  // Initialize previewUrl from template's initialUrl if provided
  const [previewUrl, setPreviewUrl] = useState(initialUrl || '');
  const [previewViewport, setPreviewViewport] = useState<{ width: number; height: number } | null>(null);
  const viewportSyncTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autoStartedRef = useRef(false);

  // Stream settings for session creation
  const [streamSettings, setStreamSettings] = useState<StreamSettingsValues | null>(null);
  const streamSettingsRef = useRef<StreamSettingsValues | null>(null);
  streamSettingsRef.current = streamSettings;

  // Connection status for header indicator
  const [connectionStatus, setConnectionStatus] = useState<StreamConnectionStatus | null>(null);

  const {
    isSidebarOpen,
    timelineWidth,
    isResizingSidebar,
    handleSidebarToggle,
    handleSidebarResizeStart,
  } = useTimelinePanel();

  // AI Navigation state - lifted here to merge AI steps with timeline
  const {
    state: aiState,
    startNavigation: aiStartNavigation,
    abortNavigation: aiAbortNavigation,
    resumeNavigation: aiResumeNavigation,
    reset: aiReset,
    availableModels: aiAvailableModels,
    isNavigating: aiIsNavigating,
  } = useAINavigation({ sessionId });

  const {
    isRecording,
    actions,
    isLoading,
    error,
    startRecording,
    clearActions,
    deleteAction,
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
    hasMultiplePages,
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
    if (aiState.steps.length > 0) {
      return mergeActionsWithAISteps(mergedActions, aiState.steps);
    }
    return mergedActions.map((action) => recordedActionToTimelineItem(action));
  }, [mergedActions, mode, timelineItems, aiState.steps]);

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

  useEffect(() => {
    const maybeDefault = sessionProfileId ?? sessionProfiles.getDefaultProfileId();
    if (!selectedProfileId && maybeDefault) {
      setSelectedProfileId(maybeDefault);
      setSessionProfileId(maybeDefault);
    }
  }, [selectedProfileId, sessionProfileId, sessionProfiles.getDefaultProfileId, setSessionProfileId]);

  useEffect(() => {
    if (sessionProfileId && sessionProfileId !== selectedProfileId) {
      setSelectedProfileId(sessionProfileId);
    }
  }, [sessionProfileId, selectedProfileId]);

  useEffect(() => {
    if (
      selectedProfileId &&
      sessionProfiles.profiles.length > 0 &&
      !sessionProfiles.profiles.some((p) => p.id === selectedProfileId)
    ) {
      const fallback = sessionProfiles.getDefaultProfileId();
      setSelectedProfileId(fallback);
      setSessionProfileId(fallback);
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

  useEffect(() => {
    if (!sessionId || !previewViewport) {
      return;
    }

    if (viewportSyncTimer.current) {
      clearTimeout(viewportSyncTimer.current);
    }

    viewportSyncTimer.current = setTimeout(async () => {
      try {
        const config = await getConfig();
        await fetch(`${config.API_URL}/recordings/live/${sessionId}/viewport`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            width: Math.round(previewViewport.width),
            height: Math.round(previewViewport.height),
          }),
        });
      } catch (err) {
        console.warn('Failed to sync viewport to recording session', err);
      }
    }, 200);

    return () => {
      if (viewportSyncTimer.current) {
        clearTimeout(viewportSyncTimer.current);
      }
    };
  }, [previewViewport, sessionId]);

  // Auto-start recording when session is ready
  useEffect(() => {
    if (sessionId && !isRecording && !autoStartedRef.current) {
      autoStartedRef.current = true;
      startRecording(sessionId).catch((err) => {
        console.error('Failed to auto-start recording:', err);
      });
    }
  }, [sessionId, isRecording, startRecording]);

  // Reset state when session changes (but preserve AI view if auto-starting)
  useEffect(() => {
    autoStartedRef.current = false;
    exitSelectionMode(); // Clear selection when switching sessions
    // Only reset to preview if we're not auto-starting AI navigation
    if (!autoStartAI) {
      setRightPanelView('preview');
    }
  }, [sessionId, exitSelectionMode, autoStartAI]);

  const handleSelectSessionProfile = useCallback(
    (profileId: string | null) => {
      setSelectedProfileId(profileId);
      setSessionProfileId(profileId);
    },
    [setSessionProfileId]
  );

  const handleCreateSessionProfile = useCallback(async () => {
    const created = await sessionProfiles.create();
    if (created) {
      setSelectedProfileId(created.id);
      setSessionProfileId(created.id);
    }
  }, [sessionProfiles, setSessionProfileId]);

  const handleConfigureSession = useCallback((profileId: string) => {
    const profile = sessionProfiles.profiles.find((p) => p.id === profileId);
    if (profile) {
      setConfiguringProfile(profile);
    }
  }, [sessionProfiles.profiles]);

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

  // Navigate to AI navigation mode
  const handleAINavigation = useCallback(() => {
    setRightPanelView('ai-navigation');
  }, []);

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
          const result = await generateWorkflow(params.name, params.projectId, actionsToGenerate);

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
    <div className="flex flex-col h-full bg-flow-bg text-flow-text">
      <RecordingHeader
        isRecording={mode === 'recording' && isRecording}
        actionCount={timelineItemCount}
        isSidebarOpen={isSidebarOpen}
        onToggleTimeline={handleSidebarToggle}
        onClose={onClose}
        mode={mode}
        onModeChange={handleModeChange}
        showModeToggle={true}
        canExecute={actions.length > 0}
        sessionProfiles={sessionProfiles.profiles}
        sessionProfilesLoading={sessionProfiles.loading}
        selectedSessionProfileId={selectedProfileId}
        onSelectSessionProfile={handleSelectSessionProfile}
        onCreateSessionProfile={handleCreateSessionProfile}
        connectionStatus={connectionStatus}
        onConfigureSession={handleConfigureSession}
        onNavigateToSessionSettings={handleNavigateToSessionSettings}
      />

      {/* Tab bar for multi-tab sessions */}
      <TabBar
        pages={openPages}
        activePageId={activePageId}
        onTabClick={switchToPage}
        onTabClose={closePage}
        onCreateTab={() => createPage()}
        isLoading={isPagesLoading}
        recentActivityPageId={recentActivityPageId}
        show={hasMultiplePages}
      />

      {/* Error display */}
      {displayError && <ErrorBanner message={displayError} />}

      {/* Unstable selectors warning banner */}
      {!isRecording && lowConfidenceCount > 0 && (
        <UnstableSelectorsBanner lowConfidenceCount={lowConfidenceCount} />
      )}

      {/* Main content split: timeline + right panel (preview or workflow form) */}
      <div className="flex-1 overflow-hidden flex">
        {isSidebarOpen && (
          <RecordActionsPanel
            actions={actions}
            timelineItems={mergedTimelineItems}
            itemCountOverride={timelineItemCount}
            mode={mode}
            isRecording={isRecording}
            isLoading={isLoading}
            isReplaying={isReplaying}
            isLive={isTimelineLive}
            hasUnstableSelectors={hasUnstableSelectors}
            timelineWidth={timelineWidth}
            isResizingSidebar={isResizingSidebar}
            onResizeStart={handleSidebarResizeStart}
            onClearRequested={() => setShowClearConfirm(true)}
            onCreateWorkflow={handleCreateWorkflow}
            onDeleteAction={handleDeleteMergedAction}
            onValidateSelector={validateSelector}
            onEditSelector={handleEditMergedSelector}
            onEditPayload={handleEditMergedPayload}
            isSelectionMode={isSelectionMode}
            selectedIndices={selectedIndices}
            onToggleSelectionMode={toggleSelectionMode}
            onActionClick={handleActionClick}
            onSelectAll={selectAll}
            onSelectNone={selectNone}
            onAINavigation={handleAINavigation}
            pages={openPages}
            pageColorMap={pageColorMap}
          />
        )}

        <div className="flex-1 h-full">
          {rightPanelView === 'preview' && (
            <RecordPreviewPanel
              previewUrl={previewUrl}
              sessionId={sessionId}
              activePageId={activePageId}
              onPreviewUrlChange={setPreviewUrl}
              onViewportChange={(size) => setPreviewViewport(size)}
              onStreamSettingsChange={setStreamSettings}
              onConnectionStatusChange={setConnectionStatus}
              hideConnectionIndicator={true}
              actions={actions}
            />
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
          {rightPanelView === 'ai-navigation' && (
            <AINavigationView
              sessionId={sessionId}
              previewUrl={previewUrl}
              onPreviewUrlChange={setPreviewUrl}
              actions={actions}
              streamSettings={streamSettings ?? undefined}
              initialPrompt={aiPrompt}
              initialModel={aiModel}
              initialMaxSteps={aiMaxSteps}
              autoStart={autoStartAI && isInitialNavigationComplete}
              aiState={aiState}
              aiModels={aiAvailableModels}
              aiIsNavigating={aiIsNavigating}
              onAIStart={aiStartNavigation}
              onAIAbort={aiAbortNavigation}
              onAIResume={aiResumeNavigation}
              onAIReset={aiReset}
            />
          )}
        </div>
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
          onSave={handleSaveBrowserProfile}
          onClose={() => setConfiguringProfile(null)}
        />
      )}
    </div>
  );
}
