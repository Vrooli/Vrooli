import { UnifiedTimeline, type PageColorClass } from '../timeline/TimelineEventCard';
import type { RecordedAction, SelectorValidation } from '../types/types';
import type { TimelineItem, TimelineMode } from '../types/timeline-unified';
import type { Page } from '../hooks/usePages';

interface RecordActionsPanelProps {
  /** Legacy: RecordedActions for backward compatibility */
  actions: RecordedAction[];
  /** Unified timeline items (preferred) */
  timelineItems?: TimelineItem[];
  /** Override item count when actions are reconciled */
  itemCountOverride?: number;
  /** Current mode: recording or execution */
  mode?: TimelineMode;
  isRecording: boolean;
  isLoading: boolean;
  isReplaying: boolean;
  /** Whether timeline is receiving live updates */
  isLive?: boolean;
  hasUnstableSelectors: boolean;
  timelineWidth: number;
  isResizingSidebar: boolean;
  onResizeStart: (event: React.MouseEvent<HTMLDivElement>) => void;
  onClearRequested: () => void;
  onCreateWorkflow: () => void;
  onDeleteAction?: (index: number) => void;
  onValidateSelector?: (selector: string) => Promise<SelectorValidation>;
  onEditSelector?: (index: number, newSelector: string) => void;
  onEditPayload?: (index: number, payload: Record<string, unknown>) => void;
  /** Selection mode state */
  isSelectionMode: boolean;
  selectedIndices: Set<number>;
  onToggleSelectionMode: () => void;
  onActionClick: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  onSelectAll: () => void;
  onSelectNone: () => void;
  /** Callback to switch to AI navigation mode */
  onAINavigation?: () => void;
  /** Pages for multi-tab recording */
  pages?: Page[];
  /** Map of page ID to color class */
  pageColorMap?: Map<string, PageColorClass>;
}

export function RecordActionsPanel({
  actions,
  timelineItems,
  itemCountOverride,
  mode = 'recording',
  isRecording,
  isLoading,
  isReplaying,
  isLive,
  hasUnstableSelectors,
  timelineWidth,
  isResizingSidebar,
  onResizeStart,
  onClearRequested,
  onCreateWorkflow,
  onDeleteAction,
  onValidateSelector: _onValidateSelector,
  onEditSelector,
  onEditPayload: _onEditPayload,
  isSelectionMode,
  selectedIndices,
  onToggleSelectionMode,
  onActionClick,
  onSelectAll,
  onSelectNone,
  onAINavigation,
  pages,
  pageColorMap,
}: RecordActionsPanelProps) {
  // Use unified timeline items if provided, otherwise fall back to actions count
  const itemCount = itemCountOverride ?? timelineItems?.length ?? actions.length;
  const selectionCount = selectedIndices.size;

  return (
    <div
      className="relative h-full flex flex-col border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900"
      style={{ width: `${timelineWidth}px`, minWidth: '240px', maxWidth: '640px' }}
    >
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          {/* Select button - checkbox icon that toggles selection mode */}
          <button
            onClick={onToggleSelectionMode}
            disabled={itemCount === 0}
            className={`p-1.5 rounded-md transition-colors ${
              isSelectionMode
                ? 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/50'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'
            } disabled:opacity-50`}
            title={isSelectionMode ? 'Exit selection mode' : 'Select steps'}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              {isSelectionMode ? (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              )}
            </svg>
          </button>

          {/* Selection actions when in selection mode */}
          {isSelectionMode && (
            <div className="flex items-center gap-1">
              <button
                onClick={onSelectAll}
                className="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
              >
                All
              </button>
              <button
                onClick={onSelectNone}
                className="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
              >
                None
              </button>
            </div>
          )}

          {mode === 'recording' && itemCount > 0 && !isSelectionMode && (
            <button
              onClick={onClearRequested}
              className="text-xs text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 px-2 py-1"
            >
              Clear
            </button>
          )}
        </div>

        <div className="flex items-center gap-2 text-xs text-gray-500">
          {isSelectionMode && selectionCount > 0 && (
            <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full font-medium">
              {selectionCount} selected
            </span>
          )}
          {!isSelectionMode && itemCount > 0 && hasUnstableSelectors && (
            <span className="px-2 py-0.5 bg-yellow-50 dark:bg-yellow-900/20 text-yellow-700 dark:text-yellow-300 rounded-full">
              Review selectors
            </span>
          )}
          {/* Mode indicator for execution */}
          {mode === 'execution' && (
            <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full">
              Executing
            </span>
          )}
          <span>
            {itemCount} step{itemCount !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {timelineItems ? (
          <UnifiedTimeline
            items={timelineItems}
            isLoading={isLoading}
            isLive={isLive ?? isRecording}
            mode={mode}
            isSelectionMode={isSelectionMode}
            selectedIndices={selectedIndices}
            onItemClick={onActionClick}
            onDeleteItem={mode === 'recording' ? onDeleteAction : undefined}
            onEditSelector={mode === 'recording' && onEditSelector ? (index) => onEditSelector(index, '') : undefined}
            pageColorMap={pageColorMap}
            showPageColors={pages && pages.length > 1}
          />
        ) : (
          <UnifiedTimeline
            items={[]}
            isLoading={isLoading}
            isLive={isRecording}
            mode={mode}
            isSelectionMode={isSelectionMode}
            selectedIndices={selectedIndices}
            onItemClick={onActionClick}
            onDeleteItem={onDeleteAction}
            pageColorMap={pageColorMap}
            showPageColors={pages && pages.length > 1}
          />
        )}
      </div>

      {/* Footer: Action buttons (recording mode only) */}
      {mode === 'recording' && (
        <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 space-y-2">
          {/* AI Navigation button */}
          {onAINavigation && (
            <button
              onClick={onAINavigation}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium text-white bg-purple-500 rounded-lg hover:bg-purple-600 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              AI Navigation
            </button>
          )}

          {/* Create Workflow button when actions exist */}
          {itemCount > 0 && (
            <>
              <button
                onClick={onCreateWorkflow}
                disabled={isReplaying || isLoading || (isSelectionMode && selectionCount === 0)}
                className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isSelectionMode && selectionCount > 0 ? (
                  <>
                    Create Workflow ({selectionCount} steps)
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                    </svg>
                  </>
                ) : isSelectionMode ? (
                  'Select steps to create workflow'
                ) : (
                  <>
                    Create Workflow
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                    </svg>
                  </>
                )}
              </button>
              {!isSelectionMode && (
                <p className="text-xs text-gray-500 dark:text-gray-400 text-center">
                  Use <span className="font-medium">Select</span> to choose specific steps
                </p>
              )}
            </>
          )}
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
