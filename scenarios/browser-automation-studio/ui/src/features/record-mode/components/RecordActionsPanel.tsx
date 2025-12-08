import { ActionTimeline } from '../ActionTimeline';
import type { RecordedAction, SelectorValidation } from '../types';

interface RecordActionsPanelProps {
  actions: RecordedAction[];
  isRecording: boolean;
  isStartingRecording: boolean;
  isLoading: boolean;
  isReplaying: boolean;
  hasUnstableSelectors: boolean;
  timelineWidth: number;
  isResizingSidebar: boolean;
  onResizeStart: (event: React.MouseEvent<HTMLDivElement>) => void;
  onStartRecording: () => void;
  onStopRecording: () => void;
  onClearRequested: () => void;
  onTestRecording: () => void;
  onGenerateWorkflow: () => void;
  onDeleteAction?: (index: number) => void;
  onValidateSelector?: (selector: string) => Promise<SelectorValidation>;
  onEditSelector?: (index: number, newSelector: string) => void;
  onEditPayload?: (index: number, payload: Record<string, unknown>) => void;
}

export function RecordActionsPanel({
  actions,
  isRecording,
  isStartingRecording,
  isLoading,
  isReplaying,
  hasUnstableSelectors,
  timelineWidth,
  isResizingSidebar,
  onResizeStart,
  onStartRecording,
  onStopRecording,
  onClearRequested,
  onTestRecording,
  onGenerateWorkflow,
  onDeleteAction,
  onValidateSelector,
  onEditSelector,
  onEditPayload,
}: RecordActionsPanelProps) {
  return (
    <div
      className="relative h-full flex flex-col border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900"
      style={{ width: `${timelineWidth}px`, minWidth: '240px', maxWidth: '640px' }}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          {!isRecording ? (
            <button
              onClick={onStartRecording}
              disabled={isStartingRecording}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-red-500 rounded-md hover:bg-red-600 disabled:opacity-50"
            >
              {isStartingRecording ? (
                <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="8" />
                </svg>
              )}
              Record
            </button>
          ) : (
            <button
              onClick={onStopRecording}
              disabled={isLoading}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-gray-700 rounded-md hover:bg-gray-800 disabled:opacity-50"
            >
              {isLoading ? (
                <svg className="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              ) : (
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <rect x="6" y="6" width="12" height="12" rx="1" />
                </svg>
              )}
              Stop
            </button>
          )}

          {actions.length > 0 && !isRecording && (
            <button
              onClick={onClearRequested}
              className="text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 px-2 py-1"
            >
              Clear
            </button>
          )}
        </div>

        <div className="flex items-center gap-2 text-xs text-gray-500">
          {actions.length > 0 && hasUnstableSelectors && !isRecording && (
            <span className="px-2 py-0.5 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300 rounded-full">
              Review selectors
            </span>
          )}
          <span>
            {actions.length} step{actions.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <ActionTimeline
          actions={actions}
          isRecording={isRecording}
          onDeleteAction={onDeleteAction}
          onValidateSelector={onValidateSelector}
          onEditSelector={onEditSelector}
          onEditPayload={onEditPayload}
        />
      </div>

      {actions.length > 0 && !isRecording && (
        <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <div className="flex gap-2">
            <button
              onClick={onTestRecording}
              disabled={isReplaying || isLoading}
              className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
            >
              {isReplaying ? 'Testing…' : 'Test'}
            </button>
            <button
              onClick={onGenerateWorkflow}
              disabled={isReplaying || isLoading}
              className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-blue-500 rounded-md hover:bg-blue-600 disabled:opacity-50"
            >
              Generate
            </button>
          </div>
        </div>
      )}

      <div
        className={`absolute top-0 right-[-6px] h-full w-3 cursor-col-resize ${
          isResizingSidebar ? 'bg-blue-100 dark:bg-blue-900/40' : ''
        }`}
        onMouseDown={onResizeStart}
      />
    </div>
  );
}
