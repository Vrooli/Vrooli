import type { TimelineMode } from '../types/timeline-unified';

interface RecordingHeaderProps {
  isRecording: boolean;
  actionCount: number;
  isSidebarOpen: boolean;
  onToggleTimeline: () => void;
  onClose?: () => void;
  /** Current mode: 'recording' or 'execution' */
  mode?: TimelineMode;
  /** Callback when mode changes */
  onModeChange?: (mode: TimelineMode) => void;
  /** Whether mode toggle should be shown */
  showModeToggle?: boolean;
  /** Whether execution mode can be selected (e.g., workflow available) */
  canExecute?: boolean;
}

export function RecordingHeader({
  isRecording,
  actionCount,
  isSidebarOpen,
  onToggleTimeline,
  onClose,
  mode = 'recording',
  onModeChange,
  showModeToggle = false,
  canExecute = false,
}: RecordingHeaderProps) {
  const title = mode === 'recording' ? 'Record Mode' : 'Execution Mode';

  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold text-surface">{title}</h1>

        {/* Mode toggle buttons */}
        {showModeToggle && onModeChange && (
          <div className="flex items-center bg-gray-100 dark:bg-gray-800 rounded-lg p-0.5">
            <button
              onClick={() => onModeChange('recording')}
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
              onClick={() => onModeChange('execution')}
              disabled={!canExecute}
              className={`flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                mode === 'execution'
                  ? 'bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm'
                  : canExecute
                    ? 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
                    : 'text-gray-400 dark:text-gray-600 cursor-not-allowed'
              }`}
              title={canExecute ? 'Switch to execution mode' : 'Select a workflow to execute'}
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Execute
            </button>
          </div>
        )}

        {/* Recording indicator */}
        {mode === 'recording' && isRecording && (
          <span className="flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400 rounded-full">
            <span className="w-1.5 h-1.5 bg-red-500 rounded-full animate-pulse" />
            Recording
          </span>
        )}

        {/* Execution indicator */}
        {mode === 'execution' && (
          <span className="flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 rounded-full">
            <span className="w-1.5 h-1.5 bg-blue-500 rounded-full animate-pulse" />
            Executing
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onToggleTimeline}
          className="relative p-2 text-subtle hover:text-surface"
          title={isSidebarOpen ? 'Hide timeline' : 'Show timeline'}
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h10M4 18h7" />
          </svg>
          <span className="absolute -top-1 -right-1 inline-flex items-center justify-center px-1.5 py-0.5 text-[10px] font-semibold text-white bg-blue-500 rounded-full">
            {actionCount}
          </span>
        </button>
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
