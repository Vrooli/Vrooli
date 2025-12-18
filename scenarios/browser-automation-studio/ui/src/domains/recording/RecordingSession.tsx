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
import { RecordActionsPanel } from './capture/RecordActionsPanel';
import { RecordingHeader } from './capture/RecordingHeader';
import { ErrorBanner, UnstableSelectorsBanner } from './capture/RecordModeBanners';
import { ClearActionsModal } from './capture/RecordModeModals';
import { WorkflowCreationForm } from './conversion/WorkflowCreationForm';
import type { ReplayPreviewResponse } from './types/types';
import { useRecordingSession } from './hooks/useRecordingSession';
import { useSessionProfiles } from './hooks/useSessionProfiles';
import { useRecordMode } from './hooks/useRecordMode';
import { useTimelinePanel } from './hooks/useTimelinePanel';
import { useActionSelection } from './hooks/useActionSelection';
import { useUnifiedTimeline } from './hooks/useUnifiedTimeline';
import { RecordPreviewPanel } from './timeline/RecordPreviewPanel';
import { getConfig } from '@/config';
import type { StreamSettingsValues } from './capture/StreamSettings';
import type { TimelineMode } from './types/timeline-unified';

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
}

/** Right panel view state */
type RightPanelView = 'preview' | 'create-workflow';

export function RecordModePage({
  sessionId: initialSessionId,
  mode: initialMode = 'recording',
  executionId,
  onWorkflowGenerated,
  onSessionReady,
  onClose,
}: RecordModePageProps) {
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

  // Right panel view state
  const [rightPanelView, setRightPanelView] = useState<RightPanelView>('preview');
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [previewUrl, setPreviewUrl] = useState('');
  const [previewViewport, setPreviewViewport] = useState<{ width: number; height: number } | null>(null);
  const viewportSyncTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autoStartedRef = useRef(false);

  // Stream settings for session creation
  const [streamSettings, setStreamSettings] = useState<StreamSettingsValues | null>(null);
  const streamSettingsRef = useRef<StreamSettingsValues | null>(null);
  streamSettingsRef.current = streamSettings;

  const {
    isSidebarOpen,
    timelineWidth,
    isResizingSidebar,
    handleSidebarToggle,
    handleSidebarResizeStart,
  } = useTimelinePanel();

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

  // Use unified timeline items count for selection
  const timelineItemCount = useMemo(() => timelineItems.length, [timelineItems]);

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

  useEffect(() => {
    if (!previewUrl) {
      return;
    }

    const abortController = new AbortController();
    let cancelled = false;

    const syncPreviewToSession = async () => {
      try {
        const activeSessionId =
          sessionId ?? (await ensureSession(previewViewport, selectedProfileId ?? sessionProfileId ?? null, streamSettingsRef.current));
        if (!activeSessionId || cancelled) return;

        const config = await getConfig();
        await fetch(`${config.API_URL}/recordings/live/${activeSessionId}/navigate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: previewUrl }),
          signal: abortController.signal,
        });
      } catch (err) {
        if (abortController.signal.aborted || cancelled) return;
        console.warn('Failed to sync preview URL to recording session', err);
      }
    };

    void syncPreviewToSession();

    return () => {
      cancelled = true;
      abortController.abort();
    };
  }, [sessionId, previewUrl, ensureSession, previewViewport, selectedProfileId, sessionProfileId]);

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

  // Reset state when session changes
  useEffect(() => {
    autoStartedRef.current = false;
    exitSelectionMode(); // Clear selection when switching sessions
    setRightPanelView('preview'); // Return to preview mode
  }, [sessionId, exitSelectionMode]);

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

  const handleEditSelector = useCallback(
    (index: number, newSelector: string) => {
      updateSelector(index, newSelector);
    },
    [updateSelector]
  );

  const handleEditPayload = useCallback(
    (index: number, payload: Record<string, unknown>) => {
      updatePayload(index, payload);
    },
    [updatePayload]
  );

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

  // Test selected actions
  const handleTestSelectedActions = useCallback(
    async (_actionIndices: number[]): Promise<ReplayPreviewResponse> => {
      // TODO: API should support selective replay - for now replay all actions
      const results = await replayPreview({ stopOnFailure: true });
      return results;
    },
    [replayPreview]
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
          const result = await generateWorkflow(params.name, params.projectId);

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
    [generateWorkflow, exitSelectionMode, onWorkflowGenerated]
  );

  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;
  const displayError = sessionError ?? error;

  return (
    <div className="flex flex-col h-full bg-flow-bg text-flow-text">
      <RecordingHeader
        isRecording={mode === 'recording' && isRecording}
        actionCount={timelineItems.length}
        isSidebarOpen={isSidebarOpen}
        onToggleTimeline={handleSidebarToggle}
        onClose={onClose}
        mode={mode}
        onModeChange={handleModeChange}
        showModeToggle={true}
        canExecute={actions.length > 0} // Can execute if we have recorded actions
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
            timelineItems={timelineItems}
            mode={mode}
            isRecording={isRecording}
            isLoading={isLoading}
            isReplaying={isReplaying}
            isLive={isTimelineLive}
            hasUnstableSelectors={hasUnstableSelectors}
            timelineWidth={timelineWidth}
            isResizingSidebar={isResizingSidebar}
            onResizeStart={handleSidebarResizeStart}
            sessionProfiles={sessionProfiles.profiles}
            sessionProfilesLoading={sessionProfiles.loading}
            selectedSessionProfileId={selectedProfileId}
            onSelectSessionProfile={handleSelectSessionProfile}
            onCreateSessionProfile={handleCreateSessionProfile}
            onClearRequested={() => setShowClearConfirm(true)}
            onCreateWorkflow={handleCreateWorkflow}
            onDeleteAction={deleteAction}
            onValidateSelector={validateSelector}
            onEditSelector={handleEditSelector}
            onEditPayload={handleEditPayload}
            isSelectionMode={isSelectionMode}
            selectedIndices={selectedIndices}
            onToggleSelectionMode={toggleSelectionMode}
            onActionClick={handleActionClick}
            onSelectAll={selectAll}
            onSelectNone={selectNone}
          />
        )}

        <div className="flex-1 h-full">
          {rightPanelView === 'preview' ? (
            <RecordPreviewPanel
              previewUrl={previewUrl}
              sessionId={sessionId}
              onPreviewUrlChange={setPreviewUrl}
              onViewportChange={(size) => setPreviewViewport(size)}
              onStreamSettingsChange={setStreamSettings}
              actions={actions}
            />
          ) : (
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
      </div>

      {/* Clear confirmation modal */}
      <ClearActionsModal
        open={showClearConfirm}
        actionCount={actions.length}
        onCancel={() => setShowClearConfirm(false)}
        onConfirm={handleClearActions}
      />
    </div>
  );
}
