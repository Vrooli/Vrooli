/**
 * RecoveryDialog
 *
 * Shown when a recording session can be recovered after a driver crash.
 * The user can choose to resume from the last checkpoint or start fresh.
 */

import React, { useState } from 'react';
import { formatDuration } from '../../hooks/useDriverStatus';

/** Checkpoint data from the API */
export interface RecoveryCheckpoint {
  /** Session ID */
  sessionId: string;
  /** Workflow ID if recording was for a workflow */
  workflowId?: string;
  /** Number of recorded actions */
  actionCount: number;
  /** Last URL at checkpoint */
  currentUrl: string;
  /** When the checkpoint was created */
  createdAt: string;
  /** When the checkpoint was last updated */
  updatedAt: string;
  /** Browser configuration */
  browserConfig: {
    viewportWidth: number;
    viewportHeight: number;
    userAgent?: string;
  };
}

/** Props for the RecoveryDialog */
interface RecoveryDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** The checkpoint data */
  checkpoint: RecoveryCheckpoint;
  /** Called when user chooses to resume */
  onResume: () => void;
  /** Called when user chooses to start fresh */
  onStartFresh: () => void;
  /** Called when user dismisses the dialog */
  onDismiss?: () => void;
  /** Whether an action is in progress */
  isLoading?: boolean;
}

/**
 * RecoveryDialog component
 *
 * Shows a modal dialog with checkpoint information and options to resume or start fresh.
 */
export const RecoveryDialog: React.FC<RecoveryDialogProps> = ({
  isOpen,
  checkpoint,
  onResume,
  onStartFresh,
  onDismiss,
  isLoading = false,
}) => {
  const [selectedOption, setSelectedOption] = useState<'resume' | 'fresh' | null>(null);

  if (!isOpen) return null;

  const timeSinceCheckpoint = Date.now() - new Date(checkpoint.updatedAt).getTime();
  const formattedTime = formatDuration(timeSinceCheckpoint);

  const handleResume = () => {
    setSelectedOption('resume');
    onResume();
  };

  const handleStartFresh = () => {
    setSelectedOption('fresh');
    onStartFresh();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onDismiss}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        className="relative w-full max-w-md bg-gray-900 border border-gray-700 rounded-xl shadow-2xl overflow-hidden"
        role="dialog"
        aria-modal="true"
        aria-labelledby="recovery-dialog-title"
      >
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-700 bg-gradient-to-r from-yellow-900/20 to-transparent">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-full bg-yellow-500/10">
              <svg
                className="w-5 h-5 text-yellow-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                />
              </svg>
            </div>
            <div>
              <h2 id="recovery-dialog-title" className="text-lg font-semibold text-white">
                Recording Interrupted
              </h2>
              <p className="text-sm text-gray-400">Your recording session was saved</p>
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 py-5 space-y-4">
          {/* Summary */}
          <div className="text-sm text-gray-300">
            Your recording was interrupted{' '}
            <span className="text-white font-medium">{formattedTime} ago</span>.{' '}
            <span className="text-white font-medium">{checkpoint.actionCount} actions</span> were
            saved automatically.
          </div>

          {/* Checkpoint details */}
          <div className="p-3 bg-gray-800/50 border border-gray-700 rounded-lg space-y-2">
            <div className="flex items-start gap-2">
              <svg
                className="w-4 h-4 text-gray-500 mt-0.5 flex-shrink-0"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
                />
              </svg>
              <div className="min-w-0 flex-1">
                <div className="text-xs text-gray-500 uppercase tracking-wide">Last URL</div>
                <div className="text-sm text-gray-300 truncate" title={checkpoint.currentUrl}>
                  {checkpoint.currentUrl || 'about:blank'}
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2 text-xs text-gray-500">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                />
              </svg>
              <span>
                {checkpoint.browserConfig.viewportWidth} x {checkpoint.browserConfig.viewportHeight}
              </span>
            </div>
          </div>

          {/* Options */}
          <div className="space-y-2">
            {/* Resume option */}
            <button
              onClick={handleResume}
              disabled={isLoading}
              className={`w-full flex items-center gap-3 p-4 rounded-lg border-2 transition-all ${
                selectedOption === 'resume' && isLoading
                  ? 'bg-blue-900/30 border-blue-500'
                  : 'bg-gray-800/50 border-gray-700 hover:border-blue-500 hover:bg-blue-900/20'
              } ${isLoading && selectedOption !== 'resume' ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <div className="flex items-center justify-center w-10 h-10 rounded-full bg-blue-500/10">
                {selectedOption === 'resume' && isLoading ? (
                  <div className="w-5 h-5 border-2 border-blue-400 border-t-transparent rounded-full animate-spin" />
                ) : (
                  <svg
                    className="w-5 h-5 text-blue-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                    />
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                )}
              </div>
              <div className="text-left">
                <div className="font-medium text-white">Resume Recording</div>
                <div className="text-sm text-gray-400">
                  Continue from where you left off with {checkpoint.actionCount} saved actions
                </div>
              </div>
            </button>

            {/* Start fresh option */}
            <button
              onClick={handleStartFresh}
              disabled={isLoading}
              className={`w-full flex items-center gap-3 p-4 rounded-lg border-2 transition-all ${
                selectedOption === 'fresh' && isLoading
                  ? 'bg-gray-700/50 border-gray-500'
                  : 'bg-gray-800/50 border-gray-700 hover:border-gray-500'
              } ${isLoading && selectedOption !== 'fresh' ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <div className="flex items-center justify-center w-10 h-10 rounded-full bg-gray-700/50">
                {selectedOption === 'fresh' && isLoading ? (
                  <div className="w-5 h-5 border-2 border-gray-400 border-t-transparent rounded-full animate-spin" />
                ) : (
                  <svg
                    className="w-5 h-5 text-gray-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 6v6m0 0v6m0-6h6m-6 0H6"
                    />
                  </svg>
                )}
              </div>
              <div className="text-left">
                <div className="font-medium text-white">Start Fresh</div>
                <div className="text-sm text-gray-400">
                  Discard saved actions and start a new recording
                </div>
              </div>
            </button>
          </div>
        </div>

        {/* Footer */}
        {onDismiss && (
          <div className="px-6 py-3 border-t border-gray-700 bg-gray-800/30">
            <button
              onClick={onDismiss}
              disabled={isLoading}
              className="text-sm text-gray-400 hover:text-gray-300 transition-colors"
            >
              Decide later
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

/**
 * Hook to check for available recovery checkpoints.
 *
 * This would typically call an API endpoint to get recoverable sessions.
 */
export function useRecoveryCheck(): {
  checkpoint: RecoveryCheckpoint | null;
  isLoading: boolean;
  checkForRecovery: () => Promise<void>;
  resumeRecording: () => Promise<void>;
  startFresh: () => Promise<void>;
} {
  const [checkpoint, setCheckpoint] = React.useState<RecoveryCheckpoint | null>(null);
  const [isLoading, setIsLoading] = React.useState(false);

  const checkForRecovery = React.useCallback(async () => {
    try {
      const response = await fetch('/api/recording/recovery/check');
      if (response.ok) {
        const data = await response.json();
        if (data.checkpoint) {
          setCheckpoint(data.checkpoint);
        } else {
          setCheckpoint(null);
        }
      }
    } catch (error) {
      console.error('Failed to check for recovery:', error);
      setCheckpoint(null);
    }
  }, []);

  const resumeRecording = React.useCallback(async () => {
    if (!checkpoint) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/recording/recovery/${checkpoint.sessionId}/resume`, {
        method: 'POST',
      });
      if (response.ok) {
        setCheckpoint(null);
        // Navigation to recording session would happen here
      }
    } catch (error) {
      console.error('Failed to resume recording:', error);
    } finally {
      setIsLoading(false);
    }
  }, [checkpoint]);

  const startFresh = React.useCallback(async () => {
    if (!checkpoint) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/recording/recovery/${checkpoint.sessionId}`, {
        method: 'DELETE',
      });
      if (response.ok) {
        setCheckpoint(null);
      }
    } catch (error) {
      console.error('Failed to delete checkpoint:', error);
    } finally {
      setIsLoading(false);
    }
  }, [checkpoint]);

  return {
    checkpoint,
    isLoading,
    checkForRecovery,
    resumeRecording,
    startFresh,
  };
}
