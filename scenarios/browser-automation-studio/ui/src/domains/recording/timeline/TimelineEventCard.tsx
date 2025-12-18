/**
 * TimelineEventCard Component
 *
 * Unified timeline card that renders both recording and execution events.
 * Uses the TimelineItem interface from timeline-unified.ts.
 *
 * Features:
 * - Mode indicator (recording vs execution)
 * - Success/failure status for execution
 * - Expandable details
 * - Action-specific icons and labels
 * - Confidence indicators (recording mode)
 */

import { useState } from 'react';
import type { TimelineItem, TimelineMode } from '../types/timeline-unified';

interface TimelineEventCardProps {
  /** The timeline item to render */
  item: TimelineItem;
  /** Index in the timeline (for display) */
  index: number;
  /** Whether this card is selected */
  isSelected?: boolean;
  /** Whether selection mode is active */
  isSelectionMode?: boolean;
  /** Callback when card is clicked */
  onClick?: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  /** Callback to delete this item */
  onDelete?: (index: number) => void;
  /** Callback to edit selector (recording mode only) */
  onEditSelector?: (index: number) => void;
}

/** Get icon for action type */
function getActionIcon(actionType: string) {
  switch (actionType) {
    case 'click':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"
          />
        </svg>
      );
    case 'type':
    case 'input':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
          />
        </svg>
      );
    case 'navigate':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
          />
        </svg>
      );
    case 'scroll':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"
          />
        </svg>
      );
    case 'wait':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
      );
    case 'assert':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
      );
    case 'keyboard':
    case 'keypress':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707"
          />
        </svg>
      );
    case 'hover':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
          />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
          />
        </svg>
      );
    case 'screenshot':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"
          />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 13a3 3 0 11-6 0 3 3 0 016 0z"
          />
        </svg>
      );
    default:
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z"
          />
        </svg>
      );
  }
}

/** Format action type for display */
function formatActionType(actionType: string): string {
  // Capitalize first letter
  return actionType.charAt(0).toUpperCase() + actionType.slice(1);
}

/** Get status indicator */
function getStatusIndicator(item: TimelineItem, mode: TimelineMode) {
  if (mode === 'recording') {
    // Recording mode: show recording indicator
    return (
      <span className="flex-shrink-0 w-2 h-2 rounded-full bg-red-500 animate-pulse" title="Recording" />
    );
  }

  // Execution mode: show success/failure
  if (item.success === true) {
    return (
      <span className="flex-shrink-0 text-green-500" title="Success">
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
      </span>
    );
  }

  if (item.success === false) {
    return (
      <span className="flex-shrink-0 text-red-500" title={item.error ?? 'Failed'}>
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
            clipRule="evenodd"
          />
        </svg>
      </span>
    );
  }

  // Pending/in-progress
  return (
    <span className="flex-shrink-0 text-blue-500 animate-spin" title="In Progress">
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
        />
      </svg>
    </span>
  );
}

/** Mode badge */
function ModeBadge({ mode }: { mode: TimelineMode }) {
  if (mode === 'recording') {
    return (
      <span className="px-1.5 py-0.5 text-[10px] font-medium bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300 rounded">
        REC
      </span>
    );
  }

  return (
    <span className="px-1.5 py-0.5 text-[10px] font-medium bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded">
      EXEC
    </span>
  );
}

export function TimelineEventCard({
  item,
  index,
  isSelected = false,
  isSelectionMode = false,
  onClick,
  onDelete,
  onEditSelector,
}: TimelineEventCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleClick = (e: React.MouseEvent) => {
    if (isSelectionMode && onClick) {
      onClick(index, e.shiftKey, e.ctrlKey || e.metaKey);
      return;
    }
    setIsExpanded(!isExpanded);
  };

  const handleCheckboxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.stopPropagation();
    if (onClick) {
      const nativeEvent = e.nativeEvent as InputEvent & {
        shiftKey?: boolean;
        ctrlKey?: boolean;
        metaKey?: boolean;
      };
      onClick(index, nativeEvent.shiftKey ?? false, nativeEvent.ctrlKey ?? nativeEvent.metaKey ?? false);
    }
  };

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDelete) {
      onDelete(index);
    }
  };

  const handleEditSelectorClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onEditSelector) {
      onEditSelector(index);
    }
  };

  return (
    <div
      className={`py-2 px-3 transition-colors border-b border-gray-200 dark:border-gray-700 ${
        isExpanded ? 'bg-gray-50 dark:bg-gray-800/50' : 'hover:bg-gray-50 dark:hover:bg-gray-800'
      } ${isSelected ? 'bg-blue-50 dark:bg-blue-900/30 border-l-2 border-l-blue-500' : ''} ${
        item.success === false ? 'border-l-2 border-l-red-400' : ''
      }`}
    >
      {/* Card header */}
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={handleClick}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && handleClick(e as unknown as React.MouseEvent)}
      >
        {/* Checkbox (selection mode) or Index */}
        {isSelectionMode ? (
          <label
            className="flex-shrink-0 flex items-center justify-center w-6 h-6"
            onClick={(e) => e.stopPropagation()}
          >
            <input
              type="checkbox"
              checked={isSelected}
              onChange={handleCheckboxChange}
              className="w-4 h-4 text-blue-500 border-gray-300 rounded focus:ring-blue-500 cursor-pointer"
            />
          </label>
        ) : (
          <span className="flex-shrink-0 w-6 h-6 flex items-center justify-center text-xs font-mono text-gray-400 bg-gray-100 dark:bg-gray-800 rounded">
            {index + 1}
          </span>
        )}

        {/* Icon */}
        <span className="flex-shrink-0 text-gray-600 dark:text-gray-400">
          {getActionIcon(item.actionType)}
        </span>

        {/* Label */}
        <span className="flex-1 text-sm leading-snug break-words">
          {formatActionType(item.actionType)}
          {item.selector && (
            <span className="ml-1 text-gray-500 dark:text-gray-400 font-mono text-xs truncate max-w-[200px] inline-block align-middle">
              {item.selector.length > 30 ? `${item.selector.slice(0, 30)}...` : item.selector}
            </span>
          )}
        </span>

        {/* Mode badge */}
        <ModeBadge mode={item.mode} />

        {/* Status indicator */}
        {getStatusIndicator(item, item.mode)}

        {/* Duration */}
        {item.durationMs !== undefined && (
          <span className="flex-shrink-0 text-xs text-gray-400 font-mono">{item.durationMs}ms</span>
        )}

        {/* Delete button */}
        {onDelete && (
          <button
            onClick={handleDeleteClick}
            className="flex-shrink-0 p-1 text-gray-400 hover:text-red-500 transition-colors"
            title="Delete"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </button>
        )}

        {/* Expand indicator */}
        <svg
          className={`w-4 h-4 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </div>

      {/* Expanded details */}
      {isExpanded && (
        <div className="mt-3 ml-9 space-y-2 text-sm">
          {/* Error message */}
          {item.error && (
            <div className="p-2 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <p className="text-xs font-medium text-red-700 dark:text-red-300">{item.error}</p>
            </div>
          )}

          {/* Selector */}
          {item.selector && (
            <div className="space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-gray-500 text-xs uppercase tracking-wide">Selector</span>
                {item.mode === 'recording' && onEditSelector && (
                  <button
                    onClick={handleEditSelectorClick}
                    className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Edit Selector
                  </button>
                )}
              </div>
              <code className="block px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded overflow-x-auto">
                {item.selector}
              </code>
            </div>
          )}

          {/* URL */}
          {item.url && (
            <div>
              <span className="text-gray-500 text-xs uppercase tracking-wide">URL</span>
              <p className="text-xs text-gray-600 dark:text-gray-400 truncate">{item.url}</p>
            </div>
          )}

          {/* Timestamp */}
          <div className="text-xs text-gray-400">{item.timestamp.toLocaleTimeString()}</div>
        </div>
      )}
    </div>
  );
}

/**
 * Timeline container that renders a list of TimelineEventCards.
 */
interface UnifiedTimelineProps {
  /** Timeline items to render */
  items: TimelineItem[];
  /** Whether items are currently loading */
  isLoading?: boolean;
  /** Whether the timeline is receiving live updates */
  isLive?: boolean;
  /** Current mode */
  mode: TimelineMode;
  /** Selection mode */
  isSelectionMode?: boolean;
  /** Selected indices */
  selectedIndices?: Set<number>;
  /** Callback when item is clicked */
  onItemClick?: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  /** Callback to delete item */
  onDeleteItem?: (index: number) => void;
  /** Callback to edit selector */
  onEditSelector?: (index: number) => void;
}

export function UnifiedTimeline({
  items,
  isLoading = false,
  isLive = false,
  mode,
  isSelectionMode = false,
  selectedIndices = new Set(),
  onItemClick,
  onDeleteItem,
  onEditSelector,
}: UnifiedTimelineProps) {
  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-gray-500">
        {isLive ? (
          <>
            <div className="animate-pulse flex space-x-1 mb-2">
              <div
                className="w-2 h-2 bg-red-500 rounded-full animate-bounce"
                style={{ animationDelay: '0ms' }}
              />
              <div
                className="w-2 h-2 bg-red-500 rounded-full animate-bounce"
                style={{ animationDelay: '150ms' }}
              />
              <div
                className="w-2 h-2 bg-red-500 rounded-full animate-bounce"
                style={{ animationDelay: '300ms' }}
              />
            </div>
            <p className="text-sm">
              {mode === 'recording' ? 'Recording... Perform actions in the browser' : 'Executing workflow...'}
            </p>
          </>
        ) : isLoading ? (
          <p className="text-sm">Loading timeline...</p>
        ) : (
          <>
            <svg
              className="w-12 h-12 mb-2 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"
              />
            </svg>
            <p className="text-sm">No events yet</p>
            <p className="text-xs text-gray-400 mt-1">
              {mode === 'recording' ? 'Start recording to capture actions' : 'Start execution to see progress'}
            </p>
          </>
        )}
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700">
      {items.map((item, index) => (
        <TimelineEventCard
          key={item.id}
          item={item}
          index={index}
          isSelected={selectedIndices.has(index)}
          isSelectionMode={isSelectionMode}
          onClick={onItemClick}
          onDelete={onDeleteItem}
          onEditSelector={mode === 'recording' ? onEditSelector : undefined}
        />
      ))}
    </div>
  );
}
