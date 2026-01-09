import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { TimelineMode } from '../types/timeline-unified';
import type { RecordingSessionProfile } from '../types/types';
import type { StreamConnectionStatus } from './PlaywrightView';

/** Workflow type being created (from AI modal or template) */
type WorkflowTypeParam = 'action' | 'flow' | 'case';

interface RecordingHeaderProps {
  isRecording: boolean;
  onClose?: () => void;
  /** Current mode: 'recording' or 'execution' */
  mode?: TimelineMode;
  /** Callback when mode changes */
  onModeChange?: (mode: TimelineMode) => void;
  /** Whether mode toggle should be shown */
  showModeToggle?: boolean;
  /** Callback when Execute button is clicked (opens workflow picker) */
  onExecuteClick?: () => void;
  /** Selected workflow name for display */
  selectedWorkflowName?: string | null;
  /** Whether to show the Run button */
  showRunButton?: boolean;
  /** Callback when Run button is clicked */
  onRun?: () => void;
  /** Whether an execution is currently running */
  isExecuting?: boolean;
  /** Callback when Stop button is clicked */
  onStop?: () => void;
  /** Whether session selector should be read-only (for historical executions) */
  sessionReadOnly?: boolean;
  /** Session profile data */
  sessionProfiles?: RecordingSessionProfile[];
  sessionProfilesLoading?: boolean;
  selectedSessionProfileId?: string | null;
  onSelectSessionProfile?: (profileId: string | null) => void;
  onCreateSessionProfile?: () => void;
  /** Stream connection status (for combined live indicator) */
  connectionStatus?: StreamConnectionStatus | null;
  /** Callback to configure a specific session's browser profile */
  onConfigureSession?: (profileId: string) => void;
  /** Callback to navigate to the Sessions settings page */
  onNavigateToSessionSettings?: () => void;
  /** Type of workflow being created (from AI modal) */
  workflowType?: WorkflowTypeParam;
}

// Workflow type display config
const WORKFLOW_TYPE_CONFIG: Record<WorkflowTypeParam, { label: string; color: string }> = {
  action: { label: 'Action', color: 'bg-green-500/20 text-green-400 border-green-500/30' },
  flow: { label: 'Flow', color: 'bg-blue-500/20 text-blue-400 border-blue-500/30' },
  case: { label: 'Test Case', color: 'bg-purple-500/20 text-purple-400 border-purple-500/30' },
};

export function RecordingHeader({
  isRecording,
  onClose,
  mode = 'recording',
  onModeChange,
  showModeToggle = false,
  onExecuteClick,
  selectedWorkflowName,
  showRunButton = false,
  onRun,
  isExecuting = false,
  onStop,
  sessionReadOnly = false,
  sessionProfiles = [],
  sessionProfilesLoading = false,
  selectedSessionProfileId,
  onSelectSessionProfile,
  onCreateSessionProfile,
  connectionStatus,
  onConfigureSession,
  onNavigateToSessionSettings,
  workflowType,
}: RecordingHeaderProps) {
  const [sessionMenuOpen, setSessionMenuOpen] = useState(false);

  const selectedSession = useMemo(
    () => sessionProfiles.find((profile) => profile.id === selectedSessionProfileId),
    [sessionProfiles, selectedSessionProfileId]
  );

  // Ref for click-outside detection
  const sessionMenuRef = useRef<HTMLDivElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    if (!sessionMenuOpen) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (sessionMenuRef.current && !sessionMenuRef.current.contains(event.target as Node)) {
        setSessionMenuOpen(false);
      }
    };

    // Use mousedown to catch clicks before they bubble
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [sessionMenuOpen]);

  // Close dropdown on Escape key
  useEffect(() => {
    if (!sessionMenuOpen) return;

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setSessionMenuOpen(false);
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [sessionMenuOpen]);

  const handleSessionSelect = useCallback((profileId: string | null) => {
    onSelectSessionProfile?.(profileId);
    setSessionMenuOpen(false);
  }, [onSelectSessionProfile]);

  const formatLastUsed = (value?: string) => {
    if (!value) return 'never';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return 'unknown';
    }
    return date.toLocaleString();
  };

  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
      <div className="flex items-center gap-3">
        {/* Mode toggle buttons */}
        {showModeToggle && (
          <div className="flex items-center bg-gray-100 dark:bg-gray-800 rounded-lg p-0.5">
            <button
              onClick={() => onModeChange?.('recording')}
              className={`flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                mode === 'recording'
                  ? 'bg-white dark:bg-gray-700 text-red-600 dark:text-red-400 shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
              }`}
              title="Switch to recording mode"
            >
              <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                <circle cx="10" cy="10" r="6" />
              </svg>
              Record
            </button>
            <button
              onClick={() => onExecuteClick ? onExecuteClick() : onModeChange?.('execution')}
              className={`flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                mode === 'execution'
                  ? 'bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
              }`}
              title="Select a workflow to execute"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Execute
            </button>
          </div>
        )}

        {/* Workflow type badge (from AI modal) */}
        {workflowType && mode === 'recording' && (
          <div
            className={`px-2.5 py-1 text-xs font-medium rounded-md border ${WORKFLOW_TYPE_CONFIG[workflowType].color}`}
            title={`Creating ${WORKFLOW_TYPE_CONFIG[workflowType].label}`}
          >
            {WORKFLOW_TYPE_CONFIG[workflowType].label}
          </div>
        )}

        {/* Selected workflow name (in execution mode) */}
        {mode === 'execution' && selectedWorkflowName && (
          <span className="text-sm text-gray-600 dark:text-gray-400 truncate max-w-[200px]" title={selectedWorkflowName}>
            {selectedWorkflowName}
          </span>
        )}

        {/* Combined Recording + Connection Status indicator */}
        {mode === 'recording' && isRecording && (
          <span
            className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-full cursor-help border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-200"
            title={connectionStatus?.isWebSocket
              ? "Recording via WebSocket (real-time)"
              : "Recording via polling (fallback)"
            }
          >
            {/* Recording dot (red, pulsing) */}
            <span className="w-1.5 h-1.5 bg-red-500 rounded-full animate-pulse" />
            <span className="text-red-600 dark:text-red-400">Recording</span>

            {/* Separator */}
            <span className="w-px h-3 bg-gray-300 dark:bg-gray-600" />

            {/* Connection status dot (green for WS, yellow for polling) */}
            <span className={`w-1.5 h-1.5 rounded-full ${
              connectionStatus?.isConnected
                ? connectionStatus?.isWebSocket
                  ? 'bg-green-500'
                  : 'bg-yellow-500'
                : 'bg-gray-400'
            }`} />
            <span className="text-gray-600 dark:text-gray-300">
              {connectionStatus?.isConnected
                ? connectionStatus?.isWebSocket
                  ? 'Live'
                  : 'Polling'
                : 'Connecting…'
              }
            </span>
          </span>
        )}

        {/* Execution indicator (only when actively running) */}
        {isExecuting && (
          <span className="flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 rounded-full">
            <span className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-pulse" />
            Running
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        {/* Session selector */}
        {onSelectSessionProfile && (
          <div className="relative" ref={sessionMenuRef}>
            <button
              type="button"
              className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-700 dark:text-gray-200 ${
                sessionReadOnly
                  ? 'opacity-50 cursor-not-allowed'
                  : 'hover:bg-gray-100 dark:hover:bg-gray-700'
              }`}
              onClick={() => !sessionReadOnly && setSessionMenuOpen((open) => !open)}
              disabled={sessionReadOnly}
              title={sessionReadOnly ? "Session is fixed for this execution" : "Select recording session"}
            >
              <svg className="w-3.5 h-3.5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              <span className="font-medium max-w-[100px] truncate">{selectedSession?.name ?? 'Default'}</span>
              {selectedSession?.has_storage_state && (
                <span className="w-1.5 h-1.5 rounded-full bg-green-500" title="Auth saved" />
              )}
              <svg className={`w-3.5 h-3.5 text-gray-400 transition-transform ${sessionMenuOpen ? 'rotate-180' : ''}`} viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.25a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z" clipRule="evenodd" />
              </svg>
            </button>
            {sessionMenuOpen && (
              <div className="absolute top-full right-0 z-20 mt-1 w-72 rounded-md border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 shadow-lg">
                <div className="px-3 py-2 border-b border-gray-100 dark:border-gray-700">
                  <p className="text-xs font-semibold text-gray-700 dark:text-gray-200">Recording sessions</p>
                  <p className="text-[11px] text-gray-500 dark:text-gray-400">
                    Reuse a session to stay signed in or start fresh.
                  </p>
                </div>
                <div className="max-h-64 overflow-y-auto">
                  {sessionProfilesLoading ? (
                    <div className="px-3 py-2 text-xs text-gray-500">Loading sessions…</div>
                  ) : sessionProfiles.length === 0 ? (
                    <button
                      className="w-full text-left px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
                      onClick={() => {
                        onCreateSessionProfile?.();
                        setSessionMenuOpen(false);
                      }}
                    >
                      Use new session
                      <div className="text-xs text-gray-500">No saved sessions yet</div>
                    </button>
                  ) : (
                    sessionProfiles.map((profile) => (
                      <div
                        key={profile.id}
                        className={`flex items-center hover:bg-gray-50 dark:hover:bg-gray-700 ${
                          profile.id === selectedSessionProfileId
                            ? 'bg-blue-50 dark:bg-blue-900/20'
                            : ''
                        }`}
                      >
                        <button
                          className={`flex-1 text-left px-3 py-2 text-sm ${
                            profile.id === selectedSessionProfileId
                              ? 'text-blue-700 dark:text-blue-200'
                              : 'text-gray-800 dark:text-gray-100'
                          }`}
                          onClick={() => handleSessionSelect(profile.id)}
                        >
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{profile.name}</span>
                            {profile.has_storage_state && (
                              <span className="text-[10px] px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-200">
                                Auth saved
                              </span>
                            )}
                          </div>
                          <div className="text-[11px] text-gray-500 dark:text-gray-400">
                            Last used {formatLastUsed(profile.last_used_at)}
                          </div>
                        </button>
                        {onConfigureSession && (
                          <button
                            className="p-2 mr-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-600"
                            onClick={(e) => {
                              e.stopPropagation();
                              onConfigureSession(profile.id);
                              setSessionMenuOpen(false);
                            }}
                            title={`Configure ${profile.name}`}
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            </svg>
                          </button>
                        )}
                      </div>
                    ))
                  )}
                </div>
                <div className="flex border-t border-gray-100 dark:border-gray-700">
                  <button
                    className="flex-1 text-left px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
                    onClick={() => {
                      onCreateSessionProfile?.();
                      setSessionMenuOpen(false);
                    }}
                  >
                    + New session
                  </button>
                  {onNavigateToSessionSettings && (
                    <button
                      className="px-3 py-2 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 border-l border-gray-100 dark:border-gray-700"
                      onClick={() => {
                        onNavigateToSessionSettings();
                        setSessionMenuOpen(false);
                      }}
                      title="Open session settings page"
                    >
                      Session settings
                    </button>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Run button (when workflow selected but not running) */}
        {showRunButton && onRun && (
          <button
            onClick={onRun}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md bg-green-500 text-white hover:bg-green-600 transition-colors"
            title="Run the selected workflow"
          >
            <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
            </svg>
            Run
          </button>
        )}

        {/* Stop button (when execution is running) */}
        {isExecuting && onStop && (
          <button
            onClick={onStop}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md bg-red-500 text-white hover:bg-red-600 transition-colors"
            title="Stop execution"
          >
            <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clipRule="evenodd" />
            </svg>
            Stop
          </button>
        )}

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
  );
}
