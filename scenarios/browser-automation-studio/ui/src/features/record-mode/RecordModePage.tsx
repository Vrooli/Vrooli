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

import { useState, useCallback, useEffect } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import { useRecordMode } from './hooks/useRecordMode';
import { ActionTimeline } from './ActionTimeline';
import type { ReplayPreviewResponse } from './types';

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
  const [sessionId, setSessionId] = useState<string | null>(initialSessionId ?? null);
  const [isCreatingSession, setIsCreatingSession] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [pendingStart, setPendingStart] = useState(false);
  const [workflowName, setWorkflowName] = useState('');
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [showClearConfirm, setShowClearConfirm] = useState(false);
  const [generateError, setGenerateError] = useState<string | null>(null);
  const [showReplayResults, setShowReplayResults] = useState(false);
  const [replayResults, setReplayResults] = useState<ReplayPreviewResponse | null>(null);
  const [replayError, setReplayError] = useState<string | null>(null);

  useEffect(() => {
    setSessionId(initialSessionId ?? null);
    setSessionError(null);
    setPendingStart(false);
  }, [initialSessionId]);

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

  const ensureSession = useCallback(async (): Promise<string | null> => {
    if (sessionId) {
      return sessionId;
    }

    setIsCreatingSession(true);
    setSessionError(null);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/session`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          viewport_width: 1280,
          viewport_height: 720,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to create recording session: ${response.statusText}`);
      }

      const data = await response.json();
      const newSessionId = data.session_id;

      if (!newSessionId) {
        throw new Error('No session ID returned from server');
      }

      setSessionId(newSessionId);
      if (onSessionReady) {
        onSessionReady(newSessionId);
      }
      return newSessionId;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create recording session';
      setSessionError(message);
      logger.error('Failed to create recording session', { component: 'RecordModePage', action: 'ensureSession' }, err);
      return null;
    } finally {
      setIsCreatingSession(false);
    }
  }, [sessionId, onSessionReady]);

  useEffect(() => {
    if (!pendingStart || !sessionId) {
      return;
    }

    startRecording()
      .catch((err) => {
        logger.error('Failed to start recording', { component: 'RecordModePage', action: 'startRecording' }, err);
      })
      .finally(() => {
        setPendingStart(false);
      });
  }, [pendingStart, sessionId, startRecording]);

  const handleStartRecording = useCallback(async () => {
    setPendingStart(true);
    const ensuredSession = await ensureSession();
    if (!ensuredSession) {
      setPendingStart(false);
    }
  }, [ensureSession]);

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

  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-white">
            Record Mode
          </h1>
          {isRecording && (
            <span className="flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400 rounded-full">
              <span className="w-1.5 h-1.5 bg-red-500 rounded-full animate-pulse" />
              Recording
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {onClose && (
            <button
              onClick={onClose}
              className="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              title="Close Record Mode"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Control bar */}
      <div className="flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          {!isRecording ? (
            <button
              onClick={handleStartRecording}
              disabled={isStartingRecording}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-red-500 rounded-lg hover:bg-red-600 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isStartingRecording ? (
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="8" />
                </svg>
              )}
              Start Recording
            </button>
          ) : (
            <button
              onClick={handleStopRecording}
              disabled={isLoading}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-gray-600 rounded-lg hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isLoading ? (
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                  <rect x="6" y="6" width="12" height="12" rx="1" />
                </svg>
              )}
              Stop Recording
            </button>
          )}

          {actions.length > 0 && !isRecording && (
            <button
              onClick={() => setShowClearConfirm(true)}
              className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Clear
            </button>
          )}
        </div>

        <div className="flex items-center gap-3">
          {/* Confidence summary */}
          {actions.length > 0 && hasUnstableSelectors && !isRecording && (
            <div className="flex items-center gap-2 text-xs">
              {lowConfidenceCount > 0 && (
                <span className="flex items-center gap-1 px-2 py-1 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-full">
                  <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                  {lowConfidenceCount} unstable
                </span>
              )}
              {mediumConfidenceCount > 0 && (
                <span className="flex items-center gap-1 px-2 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 rounded-full">
                  <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                  {mediumConfidenceCount} needs review
                </span>
              )}
            </div>
          )}
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {actions.length} action{actions.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      {/* Error display */}
      {displayError && (
        <div className="px-4 py-3 bg-red-50 dark:bg-red-900/20 border-b border-red-200 dark:border-red-800">
          <div className="flex items-start gap-3">
            <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            <div>
              <p className="text-sm font-medium text-red-800 dark:text-red-200">{displayError}</p>
              <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                {displayError.includes('driver') || displayError.includes('unavailable')
                  ? 'Make sure the browser session is active and try again.'
                  : displayError.includes('recording')
                  ? 'Try stopping and restarting the recording.'
                  : 'Please try again. If the problem persists, refresh the page.'}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Unstable selectors warning banner */}
      {!isRecording && lowConfidenceCount > 0 && (
        <div className="px-4 py-3 bg-orange-50 dark:bg-orange-900/20 border-b border-orange-200 dark:border-orange-800">
          <div className="flex items-start gap-3">
            <svg className="w-5 h-5 text-orange-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            <div>
              <p className="text-sm font-medium text-orange-800 dark:text-orange-200">
                {lowConfidenceCount} action{lowConfidenceCount !== 1 ? 's have' : ' has'} unstable selectors
              </p>
              <p className="text-xs text-orange-600 dark:text-orange-400 mt-1">
                Actions with red warning icons may fail when replayed. Click on them to edit the selectors
                and choose more stable alternatives.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Actions timeline */}
      <div className="flex-1 overflow-y-auto">
        <ActionTimeline
          actions={actions}
          isRecording={isRecording}
          onDeleteAction={deleteAction}
          onValidateSelector={validateSelector}
          onEditSelector={handleEditSelector}
          onEditPayload={handleEditPayload}
        />
      </div>

      {/* Footer with test and generate buttons */}
      {actions.length > 0 && !isRecording && (
        <div className="px-4 py-3 bg-gray-50 dark:bg-gray-800/50 border-t border-gray-200 dark:border-gray-700">
          <div className="flex gap-2">
            <button
              onClick={handleTestRecording}
              disabled={isReplaying || isLoading}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isReplaying ? (
                <>
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Testing...
                </>
              ) : (
                <>
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  Test Recording
                </>
              )}
            </button>
            <button
              onClick={() => setShowGenerateModal(true)}
              disabled={isReplaying || isLoading}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
              </svg>
              Generate Workflow
            </button>
          </div>
        </div>
      )}

      {/* Clear confirmation modal */}
      {showClearConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-sm mx-4">
            <div className="p-6">
              <div className="flex items-center justify-center w-12 h-12 mx-auto bg-red-100 dark:bg-red-900/30 rounded-full mb-4">
                <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-center text-gray-900 dark:text-white mb-2">
                Clear all actions?
              </h3>
              <p className="text-sm text-center text-gray-600 dark:text-gray-400 mb-6">
                This will permanently delete all {actions.length} recorded actions. This action cannot be undone.
              </p>
              <div className="flex gap-3">
                <button
                  onClick={() => setShowClearConfirm(false)}
                  className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleClearActions}
                  className="flex-1 px-4 py-2 text-sm font-medium text-white bg-red-500 rounded-lg hover:bg-red-600 transition-colors"
                >
                  Clear All
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Generate workflow modal */}
      {showGenerateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Generate Workflow
              </h2>
            </div>
            <div className="px-6 py-4">
              {/* Warning if unstable selectors */}
              {hasUnstableSelectors && (
                <div className="mb-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
                  <p className="text-xs text-yellow-700 dark:text-yellow-300">
                    <strong>Note:</strong> Some actions have unstable selectors. The workflow may fail on replay.
                    Consider editing selectors before generating.
                  </p>
                </div>
              )}

              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Create a reusable workflow from {actions.length} recorded action{actions.length !== 1 ? 's' : ''}.
              </p>

              <label className="block">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Workflow Name
                </span>
                <input
                  type="text"
                  value={workflowName}
                  onChange={(e) => setWorkflowName(e.target.value)}
                  placeholder="e.g., Login Flow, Submit Form, etc."
                  className="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  autoFocus
                />
              </label>

              {/* Generation error */}
              {generateError && (
                <div className="mt-3 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                  <p className="text-xs text-red-700 dark:text-red-300">{generateError}</p>
                </div>
              )}
            </div>
            <div className="px-6 py-4 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg flex justify-end gap-2">
              <button
                onClick={() => {
                  setShowGenerateModal(false);
                  setWorkflowName('');
                  setGenerateError(null);
                }}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleGenerateWorkflow}
                disabled={!workflowName.trim() || isLoading}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? (
                  <>
                    <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    Generating...
                  </>
                ) : (
                  'Generate'
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Replay results modal */}
      {showReplayResults && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
                Test Results
              </h2>
              {replayResults && (
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                  replayResults.success
                    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
                    : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
                }`}>
                  {replayResults.success ? 'All Passed' : 'Failed'}
                </span>
              )}
            </div>

            <div className="flex-1 overflow-y-auto px-6 py-4">
              {/* Loading state */}
              {isReplaying && !replayResults && (
                <div className="flex flex-col items-center justify-center py-8">
                  <svg className="w-8 h-8 animate-spin text-blue-500 mb-3" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Running test...</p>
                  <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">Replaying recorded actions</p>
                </div>
              )}

              {/* Error state */}
              {replayError && (
                <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                  <div className="flex items-start gap-3">
                    <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                    </svg>
                    <div>
                      <p className="text-sm font-medium text-red-800 dark:text-red-200">Test Failed</p>
                      <p className="text-xs text-red-600 dark:text-red-400 mt-1">{replayError}</p>
                    </div>
                  </div>
                </div>
              )}

              {/* Results */}
              {replayResults && (
                <div className="space-y-4">
                  {/* Summary */}
                  <div className="flex items-center gap-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                    <div className="text-center">
                      <p className="text-2xl font-bold text-green-600 dark:text-green-400">{replayResults.passed_actions}</p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">Passed</p>
                    </div>
                    <div className="text-center">
                      <p className="text-2xl font-bold text-red-600 dark:text-red-400">{replayResults.failed_actions}</p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">Failed</p>
                    </div>
                    <div className="flex-1 text-right">
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        {(replayResults.total_duration_ms / 1000).toFixed(1)}s total
                      </p>
                      {replayResults.stopped_early && (
                        <p className="text-xs text-yellow-600 dark:text-yellow-400">Stopped on first failure</p>
                      )}
                    </div>
                  </div>

                  {/* Action results */}
                  <div className="space-y-2">
                    {replayResults.results.map((result, index) => (
                      <div
                        key={result.action_id}
                        className={`p-3 rounded-lg border ${
                          result.success
                            ? 'bg-green-50 dark:bg-green-900/10 border-green-200 dark:border-green-800'
                            : 'bg-red-50 dark:bg-red-900/10 border-red-200 dark:border-red-800'
                        }`}
                      >
                        <div className="flex items-center gap-2">
                          {result.success ? (
                            <svg className="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                            </svg>
                          ) : (
                            <svg className="w-4 h-4 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                            </svg>
                          )}
                          <span className="text-sm font-medium text-gray-900 dark:text-white">
                            {index + 1}. {result.action_type}
                          </span>
                          <span className="text-xs text-gray-500 dark:text-gray-400 ml-auto">
                            {result.duration_ms}ms
                          </span>
                        </div>
                        {result.error && (
                          <div className="mt-2 pl-6">
                            <p className="text-xs text-red-700 dark:text-red-300">{result.error.message}</p>
                            {result.error.selector && (
                              <code className="block mt-1 text-xs font-mono text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30 px-2 py-1 rounded truncate">
                                {result.error.selector}
                              </code>
                            )}
                            {result.error.code === 'SELECTOR_NOT_FOUND' && (
                              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                💡 Try editing the selector to use a more stable attribute
                              </p>
                            )}
                            {result.error.code === 'SELECTOR_AMBIGUOUS' && result.error.match_count && (
                              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                💡 Found {result.error.match_count} elements. Make the selector more specific.
                              </p>
                            )}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="px-6 py-4 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg flex justify-end">
              <button
                onClick={() => {
                  setShowReplayResults(false);
                  setReplayResults(null);
                  setReplayError(null);
                }}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
