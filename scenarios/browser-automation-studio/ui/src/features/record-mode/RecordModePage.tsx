/**
 * RecordModePage Component
 *
 * Main page for Record Mode - allows users to record browser actions
 * and generate workflows from them.
 *
 * Phase 2 Features:
 * - Confidence warnings for unstable selectors
 * - Action editing (selector and payload)
 * - Improved error states with recovery guidance
 * - Visual polish (loading states, confirmation dialogs)
 */

import { useCallback, useEffect, useState } from 'react';
import { RecordActionsPanel } from './components/RecordActionsPanel';
import { RecordingHeader } from './components/RecordingHeader';
import { ErrorBanner, UnstableSelectorsBanner } from './components/RecordModeBanners';
import {
  BlockedDialog,
  ClearActionsModal,
  GenerateWorkflowModal,
  ReplayResultsModal,
} from './components/RecordModeModals';
import type { ReplayPreviewResponse } from './types';
import { usePreviewMode } from './hooks/usePreviewMode';
import { useRecordingSession } from './hooks/useRecordingSession';
import { useRecordMode } from './hooks/useRecordMode';
import { useSnapshotPreview } from './hooks/useSnapshotPreview';
import { useTimelinePanel } from './hooks/useTimelinePanel';
import { RecordPreviewPanel } from './RecordPreviewPanel';

interface RecordModePageProps {
  /** Browser session ID */
  sessionId: string | null;
  /** Callback when workflow is generated */
  onWorkflowGenerated?: (workflowId: string, projectId: string) => void;
  /** Callback when a live session is created */
  onSessionReady?: (sessionId: string) => void;
  /** Callback to close record mode */
  onClose?: () => void;
}

export function RecordModePage({
  sessionId: initialSessionId,
  onWorkflowGenerated,
  onSessionReady,
  onClose,
}: RecordModePageProps) {
  const {
    sessionId,
    isCreatingSession,
    sessionError,
    ensureSession,
    resetSessionError,
  } = useRecordingSession({ initialSessionId, onSessionReady });
  const [pendingStart, setPendingStart] = useState(false);
  const [workflowName, setWorkflowName] = useState('');
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [generateError, setGenerateError] = useState<string | null>(null);
  const [showReplayResults, setShowReplayResults] = useState(false);
  const [replayResults, setReplayResults] = useState<ReplayPreviewResponse | null>(null);
  const [replayError, setReplayError] = useState<string | null>(null);
  const [previewUrl, setPreviewUrl] = useState('');
  const [liveBlocked, setLiveBlocked] = useState(false);
  const [previewMode, setPreviewMode] = usePreviewMode();
  const {
    isSidebarOpen,
    timelineWidth,
    isResizingSidebar,
    handleSidebarToggle,
    handleSidebarResizeStart,
  } = useTimelinePanel();
  const [showBlockedDialog, setShowBlockedDialog] = useState(false);
  const [suppressBlockedDialog, setSuppressBlockedDialog] = useState(false);
  const [autoSwitchBlocked, setAutoSwitchBlocked] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    try {
      const stored = window.localStorage.getItem('record-mode-auto-switch-blocked');
      return stored === 'true';
    } catch {
      return false;
    }
  });

  const {
    isRecording,
    actions,
    isLoading,
    error,
    startRecording,
    stopRecording,
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

  const lastActionUrl = actions.length > 0 ? actions[actions.length - 1]?.url ?? '' : '';

  useEffect(() => {
    if (!previewUrl && lastActionUrl) {
      setPreviewUrl(lastActionUrl);
    }
  }, [lastActionUrl, previewUrl]);

  const snapshotState = useSnapshotPreview({
    url: previewUrl || lastActionUrl || null,
    enabled: previewMode === 'snapshot',
  });

  const handleStartRecording = useCallback(async () => {
    setPendingStart(true);
    resetSessionError();
    const ensuredSession = await ensureSession();
    if (!ensuredSession) {
      setPendingStart(false);
      return;
    }
    try {
      await startRecording();
    } catch (err) {
      console.error('Failed to start recording:', err);
    } finally {
      setPendingStart(false);
    }
  }, [ensureSession, resetSessionError, startRecording]);

  const handleStopRecording = useCallback(async () => {
    try {
      await stopRecording();
    } catch (err) {
      console.error('Failed to stop recording:', err);
    }
  }, [stopRecording]);

  const handleClearActions = useCallback(() => {
    clearActions();
    setShowClearConfirm(false);
  }, [clearActions]);

  const handleGenerateWorkflow = useCallback(async () => {
    if (!workflowName.trim()) {
      return;
    }

    setGenerateError(null);

    try {
      const result = await generateWorkflow(workflowName.trim());
      setShowGenerateModal(false);
      setWorkflowName('');

      if (onWorkflowGenerated) {
        onWorkflowGenerated(result.workflow_id, result.project_id);
      }
    } catch (err) {
      console.error('Failed to generate workflow:', err);
      setGenerateError(err instanceof Error ? err.message : 'Failed to generate workflow');
    }
  }, [workflowName, generateWorkflow, onWorkflowGenerated]);

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

  const handleTestRecording = useCallback(async () => {
    setReplayError(null);
    setReplayResults(null);
    setShowReplayResults(true);

    try {
      const results = await replayPreview({ stopOnFailure: true });
      setReplayResults(results);
    } catch (err) {
      console.error('Failed to test recording:', err);
      setReplayError(err instanceof Error ? err.message : 'Failed to test recording');
    }
  }, [replayPreview]);

  const hasUnstableSelectors = lowConfidenceCount > 0 || mediumConfidenceCount > 0;
  const displayError = sessionError ?? error;
  const isStartingRecording = isLoading || isCreatingSession || pendingStart;

  useEffect(() => {
    try {
      window.localStorage.setItem('record-mode-auto-switch-blocked', autoSwitchBlocked ? 'true' : 'false');
    } catch {
      // ignore
    }
  }, [autoSwitchBlocked]);

  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900">
      <RecordingHeader
        isRecording={isRecording}
        actionCount={actions.length}
        isSidebarOpen={isSidebarOpen}
        onToggleTimeline={handleSidebarToggle}
        onClose={onClose}
      />

      {/* Error display */}
      {displayError && <ErrorBanner message={displayError} />}

      {/* Unstable selectors warning banner */}
      {!isRecording && lowConfidenceCount > 0 && (
        <UnstableSelectorsBanner lowConfidenceCount={lowConfidenceCount} />
      )}

      {/* Main content split: timeline + preview */}
      <div className="flex-1 overflow-hidden flex">
        {isSidebarOpen && (
          <RecordActionsPanel
            actions={actions}
            isRecording={isRecording}
            isStartingRecording={isStartingRecording}
            isLoading={isLoading}
            isReplaying={isReplaying}
            hasUnstableSelectors={hasUnstableSelectors}
            timelineWidth={timelineWidth}
            isResizingSidebar={isResizingSidebar}
            onResizeStart={handleSidebarResizeStart}
            onStartRecording={handleStartRecording}
            onStopRecording={handleStopRecording}
            onClearRequested={() => setShowClearConfirm(true)}
            onTestRecording={handleTestRecording}
            onGenerateWorkflow={() => setShowGenerateModal(true)}
            onDeleteAction={deleteAction}
            onValidateSelector={validateSelector}
            onEditSelector={handleEditSelector}
            onEditPayload={handleEditPayload}
          />
        )}

        <div className="flex-1 h-full">
          <RecordPreviewPanel
            mode={previewMode}
            onModeChange={(mode) => setPreviewMode(mode)}
            previewUrl={previewUrl}
            onPreviewUrlChange={(url) => {
              setPreviewUrl(url);
              setLiveBlocked(false);
            }}
            snapshot={snapshotState}
            onLiveBlocked={() => {
              setLiveBlocked(true);
              if (autoSwitchBlocked) {
                setPreviewMode('snapshot');
              } else {
                if (!suppressBlockedDialog) {
                  setShowBlockedDialog(true);
                }
              }
            }}
            actions={actions}
          />
          {liveBlocked && previewMode === 'live' && (
            <div className="px-4 py-2 bg-yellow-50 dark:bg-yellow-900/20 text-xs text-yellow-800 dark:text-yellow-200 border-t border-yellow-200 dark:border-yellow-800">
              This site may block embedding in an iframe. Use the banner above to switch modes.
            </div>
          )}
        </div>
      </div>

      {/* Footer removed (actions now inside timeline footer) */}

      {/* Clear confirmation modal */}
      <ClearActionsModal
        open={showClearConfirm}
        actionCount={actions.length}
        onCancel={() => setShowClearConfirm(false)}
        onConfirm={handleClearActions}
      />

      {/* Generate workflow modal */}
      <GenerateWorkflowModal
        open={showGenerateModal}
        workflowName={workflowName}
        onWorkflowNameChange={setWorkflowName}
        hasUnstableSelectors={hasUnstableSelectors}
        actionCount={actions.length}
        generateError={generateError}
        isLoading={isLoading}
        onCancel={() => {
          setShowGenerateModal(false);
          setWorkflowName('');
          setGenerateError(null);
        }}
        onGenerate={handleGenerateWorkflow}
      />

      {/* Replay results modal */}
      <ReplayResultsModal
        open={showReplayResults}
        isReplaying={isReplaying}
        replayResults={replayResults}
        replayError={replayError}
        onClose={() => {
          setShowReplayResults(false);
          setReplayResults(null);
          setReplayError(null);
        }}
      />

      {/* Blocked dialog */}
      <BlockedDialog
        open={showBlockedDialog}
        autoSwitchBlocked={autoSwitchBlocked}
        onStay={() => {
          setSuppressBlockedDialog(true);
          setShowBlockedDialog(false);
        }}
        onSwitch={() => {
          setShowBlockedDialog(false);
          setSuppressBlockedDialog(true);
          setPreviewMode('snapshot');
        }}
        onAutoSwitchChange={(value: boolean) => setAutoSwitchBlocked(value)}
      />
    </div>
  );
}
