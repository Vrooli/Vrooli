/**
 * ActionTimeline Component
 *
 * Displays a timeline of recorded actions with:
 * - Automatic merging of consecutive actions (scroll, type) to match workflow output
 * - Expandable action details
 * - Action-specific payload information (scroll distance, click type, key combos, etc.)
 * - Confidence indicators and warnings
 * - Selector editing with the SelectorEditor component
 * - Action payload editing
 * - Delete functionality
 * - Selection mode with checkboxes and range selection
 */

import { useState, useCallback, useMemo } from 'react';
import { SelectorEditor } from './SelectorEditor';
import type { RecordedAction, SelectorValidation } from './types';
import { mergeConsecutiveActions, getMergeDescription, type MergedAction } from './mergeActions';

interface ActionTimelineProps {
  /** List of recorded actions */
  actions: RecordedAction[];
  /** Whether recording is active */
  isRecording: boolean;
  /** Callback when an action is deleted */
  onDeleteAction?: (index: number) => void;
  /** Callback to validate a selector */
  onValidateSelector?: (selector: string) => Promise<SelectorValidation>;
  /** Callback when selector is edited */
  onEditSelector?: (index: number, newSelector: string) => void;
  /** Callback when action payload is edited */
  onEditPayload?: (index: number, payload: Record<string, unknown>) => void;
  /** Whether selection mode is active */
  isSelectionMode?: boolean;
  /** Set of selected indices */
  selectedIndices?: Set<number>;
  /** Callback when an action is clicked (for selection) */
  onActionClick?: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
}

/** Confidence thresholds */
const CONFIDENCE = {
  HIGH: 0.8,
  MEDIUM: 0.5,
};

export function ActionTimeline({
  actions,
  isRecording,
  onDeleteAction,
  onValidateSelector,
  onEditSelector,
  onEditPayload,
  isSelectionMode = false,
  selectedIndices = new Set(),
  onActionClick,
}: ActionTimelineProps) {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editMode, setEditMode] = useState<'selector' | 'payload' | null>(null);
  const [editingPayload, setEditingPayload] = useState<Record<string, unknown>>({});

  // Merge consecutive actions to preview what the workflow will look like
  const mergedActions = useMemo(() => mergeConsecutiveActions(actions), [actions]);

  // Build a map from merged action index to original action indices for delete/edit callbacks
  const mergedToOriginalMap = useMemo(() => {
    const map = new Map<number, number[]>();
    let mergedIdx = 0;

    for (const action of mergedActions) {
      const merged = action as MergedAction;
      if (merged._merged && merged._merged.mergedIds.length > 1) {
        // Find indices of all original actions that were merged
        const originalIndices = merged._merged.mergedIds
          .map((id) => actions.findIndex((a) => a.id === id))
          .filter((idx) => idx !== -1);
        map.set(mergedIdx, originalIndices);
      } else {
        // Single action - find its original index
        const originalIdx = actions.findIndex((a) => a.id === action.id);
        map.set(mergedIdx, originalIdx !== -1 ? [originalIdx] : []);
      }
      mergedIdx++;
    }
    return map;
  }, [actions, mergedActions]);

  const handleToggleExpand = (index: number, event?: React.MouseEvent) => {
    if (editingIndex !== null) return; // Don't toggle while editing

    // In selection mode, clicking toggles selection instead of expanding
    if (isSelectionMode && onActionClick) {
      onActionClick(index, event?.shiftKey ?? false, event?.ctrlKey ?? event?.metaKey ?? false);
      return;
    }

    setExpandedIndex(expandedIndex === index ? null : index);
  };

  const handleStartEditSelector = (index: number) => {
    setEditingIndex(index);
    setEditMode('selector');
  };

  const handleStartEditPayload = (index: number, action: MergedAction) => {
    setEditingIndex(index);
    setEditMode('payload');
    setEditingPayload(action.payload || {});
  };

  // Handle delete for merged actions - deletes all original actions in the merge
  const handleDeleteMergedAction = useCallback(
    (mergedIndex: number) => {
      if (!onDeleteAction) return;
      const originalIndices = mergedToOriginalMap.get(mergedIndex) || [];
      // Delete in reverse order to preserve indices
      const sorted = [...originalIndices].sort((a, b) => b - a);
      for (const idx of sorted) {
        onDeleteAction(idx);
      }
    },
    [onDeleteAction, mergedToOriginalMap]
  );

  // Handle selector edit for merged actions - edits the first original action
  const handleEditMergedSelector = useCallback(
    (mergedIndex: number, newSelector: string) => {
      if (!onEditSelector) return;
      const originalIndices = mergedToOriginalMap.get(mergedIndex) || [];
      if (originalIndices.length > 0) {
        onEditSelector(originalIndices[0], newSelector);
      }
      setEditingIndex(null);
      setEditMode(null);
    },
    [onEditSelector, mergedToOriginalMap]
  );

  // Handle payload edit for merged actions - edits all original actions
  const handleEditMergedPayload = useCallback(
    (mergedIndex: number) => {
      if (!onEditPayload) return;
      const originalIndices = mergedToOriginalMap.get(mergedIndex) || [];
      for (const idx of originalIndices) {
        onEditPayload(idx, editingPayload);
      }
      setEditingIndex(null);
      setEditMode(null);
      setEditingPayload({});
    },
    [onEditPayload, mergedToOriginalMap, editingPayload]
  );


  const handleCancelEdit = useCallback(() => {
    setEditingIndex(null);
    setEditMode(null);
    setEditingPayload({});
  }, []);

  const getActionIcon = (actionType: string) => {
    const normalizedType =
      actionType === 'type' ? 'input' : actionType === 'keypress' ? 'keyboard' : actionType;
    switch (normalizedType) {
      case 'click':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
          </svg>
        );
      case 'input':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
        );
      case 'navigate':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
        );
      case 'scroll':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
          </svg>
        );
      case 'keyboard':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707" />
          </svg>
        );
      case 'select':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
          </svg>
        );
      case 'focus':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        );
      default:
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
          </svg>
        );
    }
  };

  const getActionLabel = (action: RecordedAction) => {
    const rawType: string = action.actionType;
    const normalizedType =
      rawType === 'type' ? 'input' : rawType === 'keypress' ? 'keyboard' : rawType;
    switch (normalizedType) {
      case 'click':
        if (action.elementMeta?.innerText) {
          return `Click: "${truncate(action.elementMeta.innerText, 30)}"`;
        }
        if (action.elementMeta?.ariaLabel) {
          return `Click: ${action.elementMeta.ariaLabel}`;
        }
        return `Click ${action.elementMeta?.tagName || 'element'}`;
      case 'input':
        if (action.payload?.text) {
          return `Type: "${truncate(String(action.payload.text), 30)}"`;
        }
        return 'Type text';
      case 'navigate':
        return `Navigate to ${truncate(action.url, 40)}`;
      case 'scroll':
        return `Scroll ${action.payload?.deltaY !== undefined && Number(action.payload.deltaY) > 0 ? 'down' : 'up'}`;
      case 'keyboard':
        return `Press ${action.payload?.key || 'key'}`;
      case 'select':
        return `Select: ${action.payload?.selectedText || action.payload?.value || 'option'}`;
      case 'focus':
        return `Focus ${action.elementMeta?.tagName || 'element'}`;
      default:
        return normalizedType;
    }
  };

  const truncate = (str: string, maxLen: number) => {
    if (str.length <= maxLen) return str;
    return str.slice(0, maxLen) + '...';
  };

  const getConfidenceLevel = (confidence: number): 'high' | 'medium' | 'low' => {
    if (confidence >= CONFIDENCE.HIGH) return 'high';
    if (confidence >= CONFIDENCE.MEDIUM) return 'medium';
    return 'low';
  };

  const getConfidenceIndicator = (confidence: number) => {
    const level = getConfidenceLevel(confidence);
    const percentage = Math.round(confidence * 100);

    if (level === 'high') {
      return (
        <span
          className="flex-shrink-0 text-green-500"
          title={`High confidence: ${percentage}%`}
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
          </svg>
        </span>
      );
    }

    if (level === 'medium') {
      return (
        <span
          className="flex-shrink-0 text-yellow-500"
          title={`Medium confidence: ${percentage}% - selector may be unstable`}
        >
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
          </svg>
        </span>
      );
    }

    return (
      <span
        className="flex-shrink-0 text-red-500"
        title={`Low confidence: ${percentage}% - selector likely unstable, please review`}
      >
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
        </svg>
      </span>
    );
  };

  const shouldWarnOnSelector = (action: RecordedAction) => {
    // Non-element actions like scroll/navigate/keyboard don't need selector warnings.
    return ['click', 'input', 'select', 'focus', 'hover', 'blur'].includes(action.actionType);
  };

  /** Format modifier keys for display */
  const formatModifiers = (modifiers?: Array<'ctrl' | 'shift' | 'alt' | 'meta'>) => {
    if (!modifiers || modifiers.length === 0) return null;
    const labels: Record<string, string> = {
      ctrl: 'Ctrl',
      shift: 'Shift',
      alt: 'Alt',
      meta: navigator.platform.includes('Mac') ? 'Cmd' : 'Win',
    };
    return modifiers.map(m => labels[m] || m).join(' + ');
  };

  /** Format click button for display */
  const formatClickButton = (button?: 'left' | 'right' | 'middle') => {
    switch (button) {
      case 'right': return 'Right-click';
      case 'middle': return 'Middle-click';
      default: return 'Left-click';
    }
  };

  /** Format scroll distance with direction */
  const formatScrollDistance = (deltaY?: number, deltaX?: number) => {
    const parts: string[] = [];
    if (deltaY !== undefined && deltaY !== 0) {
      const direction = deltaY > 0 ? 'down' : 'up';
      parts.push(`${Math.abs(Math.round(deltaY))}px ${direction}`);
    }
    if (deltaX !== undefined && deltaX !== 0) {
      const direction = deltaX > 0 ? 'right' : 'left';
      parts.push(`${Math.abs(Math.round(deltaX))}px ${direction}`);
    }
    return parts.length > 0 ? parts.join(', ') : null;
  };

  /** Format scroll position */
  const formatScrollPosition = (scrollX?: number, scrollY?: number) => {
    if (scrollX === undefined && scrollY === undefined) return null;
    return `(${Math.round(scrollX || 0)}, ${Math.round(scrollY || 0)})`;
  };

  if (actions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-gray-500">
        {isRecording ? (
          <>
            <div className="animate-pulse flex space-x-1 mb-2">
              <div className="w-2 h-2 bg-red-500 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
              <div className="w-2 h-2 bg-red-500 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
              <div className="w-2 h-2 bg-red-500 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
            </div>
            <p className="text-sm">Recording... Perform actions in the browser</p>
          </>
        ) : (
          <>
            <svg className="w-12 h-12 mb-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
            </svg>
            <p className="text-sm">No actions recorded yet</p>
            <p className="text-xs text-gray-400 mt-1">Click "Start Recording" to begin</p>
          </>
        )}
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700">
      {mergedActions.map((action, index) => {
        const isExpanded = expandedIndex === index;
        const isEditing = editingIndex === index;
        const confidenceLevel = getConfidenceLevel(action.confidence);
        const warnable = shouldWarnOnSelector(action);
        const mergedAction = action as MergedAction;
        const mergeInfo = getMergeDescription(mergedAction._merged);
        const isSelected = selectedIndices.has(index);

        return (
          <div
            key={action.id}
            className={`py-2 px-3 transition-colors ${
              isExpanded ? 'bg-gray-50 dark:bg-gray-800/50' : 'hover:bg-gray-50 dark:hover:bg-gray-800'
            } ${confidenceLevel === 'low' ? 'border-l-2 border-l-red-400' : ''} ${
              isSelected ? 'bg-blue-50 dark:bg-blue-900/30 border-l-2 border-l-blue-500' : ''
            }`}
          >
            {/* Action header */}
            <div
              className={`flex items-center gap-3 ${!isEditing ? 'cursor-pointer' : ''}`}
              onClick={(e) => handleToggleExpand(index, e)}
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
                    onChange={(e) => {
                      e.stopPropagation();
                      if (onActionClick) {
                        // Cast native event to InputEvent which has modifier key properties
                        const evt = e.nativeEvent as InputEvent & { shiftKey?: boolean; ctrlKey?: boolean; metaKey?: boolean };
                        onActionClick(index, evt.shiftKey ?? false, evt.ctrlKey ?? evt.metaKey ?? false);
                      }
                    }}
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
                {getActionIcon(action.actionType)}
              </span>

              {/* Label */}
              <span className="flex-1 text-sm leading-snug break-words">
                {getActionLabel(action)}
                {/* Merged badge */}
                {mergeInfo && (
                  <span
                    className="ml-1.5 inline-flex items-center px-1.5 py-0.5 text-[10px] font-medium bg-purple-100 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300 rounded"
                    title={mergeInfo}
                  >
                    <svg className="w-3 h-3 mr-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16m-7 6h7" />
                    </svg>
                    {mergedAction._merged?.mergedCount}
                  </span>
                )}
              </span>

              {/* Confidence indicator */}
              {warnable && action.selector && getConfidenceIndicator(action.confidence)}

              {/* Delete button */}
              {onDeleteAction && !isEditing && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDeleteMergedAction(index);
                  }}
                  className="flex-shrink-0 p-1 text-gray-400 hover:text-red-500 transition-colors"
                  title={mergeInfo ? `Delete ${mergedAction._merged?.mergedCount} merged actions` : 'Delete action'}
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              )}

              {/* Expand indicator */}
              {!isEditing && (
                <svg
                  className={`w-4 h-4 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              )}
            </div>

            {/* Expanded details */}
            {isExpanded && !isEditing && (
              <div className="mt-3 ml-9 space-y-3 text-sm">
                {/* Merged actions info */}
                {mergeInfo && mergedAction._merged && (
                  <div className="p-2 rounded-lg bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800">
                    <p className="text-xs font-medium text-purple-700 dark:text-purple-300">
                      {mergeInfo}
                    </p>
                    <p className="text-xs text-purple-600 dark:text-purple-400 mt-0.5">
                      This shows the combined result that will appear in your workflow.
                    </p>
                  </div>
                )}

                {/* Unstable selector warning */}
                {warnable && action.selector && confidenceLevel !== 'high' && (
                  <div className={`p-2 rounded-lg ${
                    confidenceLevel === 'low'
                      ? 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
                      : 'bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800'
                  }`}>
                    <p className={`text-xs font-medium ${
                      confidenceLevel === 'low'
                        ? 'text-red-700 dark:text-red-300'
                        : 'text-yellow-700 dark:text-yellow-300'
                    }`}>
                      {confidenceLevel === 'low'
                        ? 'This selector is likely to break. Click "Edit Selector" to improve it.'
                        : 'This selector may be unstable. Consider reviewing it.'}
                    </p>
                  </div>
                )}

                {/* ============================================================ */}
                {/* ACTION-SPECIFIC DETAILS */}
                {/* ============================================================ */}

                {/* Click action details */}
                {action.actionType === 'click' && action.payload && (
                  <div className="space-y-1">
                    <span className="text-gray-500 text-xs uppercase tracking-wide">Click Details</span>
                    <div className="flex flex-wrap gap-1.5 mt-1">
                      <span className="px-2 py-0.5 text-xs bg-indigo-100 dark:bg-indigo-900/50 text-indigo-700 dark:text-indigo-300 rounded">
                        {formatClickButton(action.payload.button as 'left' | 'right' | 'middle' | undefined)}
                      </span>
                      {(action.payload.clickCount as number) > 1 && (
                        <span className="px-2 py-0.5 text-xs bg-orange-100 dark:bg-orange-900/50 text-orange-700 dark:text-orange-300 rounded">
                          {action.payload.clickCount}x click
                        </span>
                      )}
                      {formatModifiers(action.payload.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'> | undefined) && (
                        <span className="px-2 py-0.5 text-xs bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300 rounded font-mono">
                          {formatModifiers(action.payload.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'>)}
                        </span>
                      )}
                    </div>
                    {action.cursorPos && (
                      <p className="text-xs text-gray-500 mt-1">
                        Position: ({Math.round(action.cursorPos.x)}, {Math.round(action.cursorPos.y)})
                      </p>
                    )}
                  </div>
                )}

                {/* Scroll action details */}
                {action.actionType === 'scroll' && action.payload && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Scroll Details</span>
                      {onEditPayload && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStartEditPayload(index, action);
                          }}
                          className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                        >
                          Edit
                        </button>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-2 mt-1">
                      {formatScrollDistance(
                        action.payload.deltaY as number | undefined,
                        action.payload.deltaX as number | undefined
                      ) && (
                        <div className="px-2 py-1.5 bg-cyan-50 dark:bg-cyan-900/30 rounded border border-cyan-200 dark:border-cyan-800">
                          <span className="text-[10px] text-cyan-600 dark:text-cyan-400 uppercase tracking-wide block">Distance</span>
                          <span className="text-xs font-medium text-cyan-800 dark:text-cyan-200">
                            {formatScrollDistance(
                              action.payload.deltaY as number | undefined,
                              action.payload.deltaX as number | undefined
                            )}
                          </span>
                        </div>
                      )}
                      {formatScrollPosition(
                        action.payload.scrollX as number | undefined,
                        action.payload.scrollY as number | undefined
                      ) && (
                        <div className="px-2 py-1.5 bg-teal-50 dark:bg-teal-900/30 rounded border border-teal-200 dark:border-teal-800">
                          <span className="text-[10px] text-teal-600 dark:text-teal-400 uppercase tracking-wide block">Final Position</span>
                          <span className="text-xs font-medium font-mono text-teal-800 dark:text-teal-200">
                            {formatScrollPosition(
                              action.payload.scrollX as number | undefined,
                              action.payload.scrollY as number | undefined
                            )}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Keyboard action details */}
                {action.actionType === 'keyboard' && action.payload && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Key Details</span>
                      {onEditPayload && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStartEditPayload(index, action);
                          }}
                          className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                        >
                          Edit
                        </button>
                      )}
                    </div>
                    <div className="flex flex-wrap gap-1.5 mt-1">
                      {formatModifiers(action.payload.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'> | undefined) && (
                        <span className="px-2 py-1 text-xs bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300 rounded font-mono">
                          {formatModifiers(action.payload.modifiers as Array<'ctrl' | 'shift' | 'alt' | 'meta'>)}
                        </span>
                      )}
                      <span className="text-gray-400">+</span>
                      <span className="px-2 py-1 text-xs bg-violet-100 dark:bg-violet-900/50 text-violet-700 dark:text-violet-300 rounded font-mono font-medium">
                        {String(action.payload.key || 'Unknown')}
                      </span>
                    </div>
                    {action.payload.code && action.payload.code !== action.payload.key && (
                      <p className="text-xs text-gray-500 mt-1">
                        Code: <code className="bg-gray-100 dark:bg-gray-800 px-1 rounded">{String(action.payload.code)}</code>
                      </p>
                    )}
                  </div>
                )}

                {/* Select action details */}
                {action.actionType === 'select' && action.payload && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Selection Details</span>
                      {onEditPayload && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStartEditPayload(index, action);
                          }}
                          className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                        >
                          Edit
                        </button>
                      )}
                    </div>
                    <div className="space-y-1.5 mt-1">
                      {action.payload.selectedText && (
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-gray-500">Text:</span>
                          <code className="px-2 py-0.5 text-xs font-mono bg-emerald-100 dark:bg-emerald-900/50 text-emerald-700 dark:text-emerald-300 rounded">
                            "{String(action.payload.selectedText)}"
                          </code>
                        </div>
                      )}
                      {action.payload.value && action.payload.value !== action.payload.selectedText && (
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-gray-500">Value:</span>
                          <code className="px-2 py-0.5 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded">
                            {String(action.payload.value)}
                          </code>
                        </div>
                      )}
                      {action.payload.selectedIndex !== undefined && (
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-gray-500">Index:</span>
                          <span className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded font-mono">
                            {String(action.payload.selectedIndex)}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Navigate action details */}
                {action.actionType === 'navigate' && (
                  <div className="space-y-1">
                    <span className="text-gray-500 text-xs uppercase tracking-wide">Navigation Details</span>
                    <div className="space-y-1.5 mt-1">
                      {action.payload?.targetUrl && (
                        <div>
                          <span className="text-xs text-gray-500 block">Target URL:</span>
                          <code className="block px-2 py-1 text-xs font-mono bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded break-all">
                            {String(action.payload.targetUrl)}
                          </code>
                        </div>
                      )}
                      {action.payload?.waitForSelector !== undefined && action.payload.waitForSelector !== null && (
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-gray-500">Wait for:</span>
                          <code className="px-2 py-0.5 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded">
                            {String(action.payload.waitForSelector)}
                          </code>
                        </div>
                      )}
                      {action.payload?.timeoutMs !== undefined && action.payload.timeoutMs !== null && (
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-gray-500">Timeout:</span>
                          <span className="text-xs text-gray-700 dark:text-gray-300">
                            {String(action.payload.timeoutMs)}ms
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Input action details (text input) */}
                {action.actionType === 'input' && action.payload?.text && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Text Input</span>
                      {onEditPayload && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStartEditPayload(index, action);
                          }}
                          className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                        >
                          Edit
                        </button>
                      )}
                    </div>
                    <code className="block px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded">
                      "{String(action.payload.text)}"
                    </code>
                    {action.payload.delay && (
                      <p className="text-xs text-gray-500">
                        Typing delay: {String(action.payload.delay)}ms
                      </p>
                    )}
                    {action.payload.clearFirst && (
                      <span className="inline-block px-1.5 py-0.5 text-[10px] bg-yellow-100 dark:bg-yellow-900/50 text-yellow-700 dark:text-yellow-300 rounded">
                        Clears existing text first
                      </span>
                    )}
                  </div>
                )}

                {/* ============================================================ */}
                {/* COMMON DETAILS (Selector, Element, URL, Timestamp) */}
                {/* ============================================================ */}

                {/* Selector */}
                {action.selector && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Selector</span>
                      {onValidateSelector && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStartEditSelector(index);
                          }}
                          className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                        >
                          Edit Selector
                        </button>
                      )}
                    </div>
                    <code className="block px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded overflow-x-auto">
                      {action.selector.primary}
                    </code>
                    {action.selector.candidates.length > 1 && (
                      <p className="text-xs text-gray-500">
                        +{action.selector.candidates.length - 1} alternative selectors available
                      </p>
                    )}
                  </div>
                )}

                {/* Element info */}
                {action.elementMeta && (
                  <div>
                    <span className="text-gray-500 text-xs uppercase tracking-wide">Element</span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      <span className="px-1.5 py-0.5 text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded">
                        {action.elementMeta.tagName}
                      </span>
                      {action.elementMeta.id && (
                        <span className="px-1.5 py-0.5 text-xs bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 rounded">
                          #{action.elementMeta.id}
                        </span>
                      )}
                      {action.elementMeta.role && (
                        <span className="px-1.5 py-0.5 text-xs bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 rounded">
                          role={action.elementMeta.role}
                        </span>
                      )}
                    </div>
                  </div>
                )}

                {/* URL */}
                <div>
                  <span className="text-gray-500 text-xs uppercase tracking-wide">URL</span>
                  <p className="text-xs text-gray-600 dark:text-gray-400 truncate">{action.url}</p>
                </div>

                {/* Timestamp */}
                <div className="text-xs text-gray-400">
                  {new Date(action.timestamp).toLocaleTimeString()}
                </div>
              </div>
            )}

            {/* Selector Editor */}
            {isEditing && editMode === 'selector' && action.selector && onValidateSelector && (
              <div className="mt-3 ml-9">
                <SelectorEditor
                  selectorSet={action.selector}
                  confidence={action.confidence}
                  onValidate={onValidateSelector}
                  onSave={(newSelector) => handleEditMergedSelector(index, newSelector)}
                  onCancel={handleCancelEdit}
                />
              </div>
            )}

            {/* Payload Editor */}
            {isEditing && editMode === 'payload' && (
              <div className="mt-3 ml-9 space-y-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
                {/* Input action editor */}
                {action.actionType === 'input' && (
                  <div className="space-y-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                        Text to Type
                      </label>
                      <input
                        type="text"
                        value={String(editingPayload.text || '')}
                        onChange={(e) => setEditingPayload({ ...editingPayload, text: e.target.value })}
                        className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="Enter text to type"
                      />
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="flex-1">
                        <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                          Typing Delay (ms)
                        </label>
                        <input
                          type="number"
                          value={editingPayload.delay !== undefined ? String(editingPayload.delay) : ''}
                          onChange={(e) => setEditingPayload({ ...editingPayload, delay: e.target.value ? Number(e.target.value) : undefined })}
                          className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          placeholder="0"
                          min="0"
                        />
                      </div>
                      <div className="flex items-center gap-2 pt-5">
                        <input
                          type="checkbox"
                          id="clearFirst"
                          checked={Boolean(editingPayload.clearFirst)}
                          onChange={(e) => setEditingPayload({ ...editingPayload, clearFirst: e.target.checked })}
                          className="w-4 h-4 text-blue-500 border-gray-300 rounded focus:ring-blue-500"
                        />
                        <label htmlFor="clearFirst" className="text-xs text-gray-700 dark:text-gray-300">
                          Clear existing text first
                        </label>
                      </div>
                    </div>
                  </div>
                )}

                {/* Keyboard action editor */}
                {action.actionType === 'keyboard' && (
                  <div className="space-y-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                        Key
                      </label>
                      <input
                        type="text"
                        value={String(editingPayload.key || '')}
                        onChange={(e) => setEditingPayload({ ...editingPayload, key: e.target.value })}
                        className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="e.g., Enter, Escape, Tab"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                        Modifiers
                      </label>
                      <div className="flex flex-wrap gap-2">
                        {(['ctrl', 'shift', 'alt', 'meta'] as const).map((mod) => {
                          const modifiers = (editingPayload.modifiers as string[] | undefined) || [];
                          const isChecked = modifiers.includes(mod);
                          return (
                            <label key={mod} className="flex items-center gap-1.5 cursor-pointer">
                              <input
                                type="checkbox"
                                checked={isChecked}
                                onChange={(e) => {
                                  const newModifiers = e.target.checked
                                    ? [...modifiers, mod]
                                    : modifiers.filter((m) => m !== mod);
                                  setEditingPayload({ ...editingPayload, modifiers: newModifiers.length > 0 ? newModifiers : undefined });
                                }}
                                className="w-4 h-4 text-blue-500 border-gray-300 rounded focus:ring-blue-500"
                              />
                              <span className="text-xs font-mono text-gray-700 dark:text-gray-300">
                                {mod === 'meta' ? (navigator.platform.includes('Mac') ? 'Cmd' : 'Win') : mod.charAt(0).toUpperCase() + mod.slice(1)}
                              </span>
                            </label>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                )}

                {/* Scroll action editor */}
                {action.actionType === 'scroll' && (
                  <div className="space-y-3">
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                          Scroll X (final position)
                        </label>
                        <input
                          type="number"
                          value={editingPayload.scrollX !== undefined ? String(editingPayload.scrollX) : ''}
                          onChange={(e) => setEditingPayload({ ...editingPayload, scrollX: e.target.value ? Number(e.target.value) : undefined })}
                          className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          placeholder="0"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                          Scroll Y (final position)
                        </label>
                        <input
                          type="number"
                          value={editingPayload.scrollY !== undefined ? String(editingPayload.scrollY) : ''}
                          onChange={(e) => setEditingPayload({ ...editingPayload, scrollY: e.target.value ? Number(e.target.value) : undefined })}
                          className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          placeholder="0"
                        />
                      </div>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                          Delta X (scroll distance)
                        </label>
                        <input
                          type="number"
                          value={editingPayload.deltaX !== undefined ? String(editingPayload.deltaX) : ''}
                          onChange={(e) => setEditingPayload({ ...editingPayload, deltaX: e.target.value ? Number(e.target.value) : undefined })}
                          className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          placeholder="0"
                        />
                        <p className="text-[10px] text-gray-500 mt-0.5">Positive = right, Negative = left</p>
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                          Delta Y (scroll distance)
                        </label>
                        <input
                          type="number"
                          value={editingPayload.deltaY !== undefined ? String(editingPayload.deltaY) : ''}
                          onChange={(e) => setEditingPayload({ ...editingPayload, deltaY: e.target.value ? Number(e.target.value) : undefined })}
                          className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          placeholder="0"
                        />
                        <p className="text-[10px] text-gray-500 mt-0.5">Positive = down, Negative = up</p>
                      </div>
                    </div>
                  </div>
                )}

                {/* Select action editor */}
                {action.actionType === 'select' && (
                  <div className="space-y-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                        Selected Value
                      </label>
                      <input
                        type="text"
                        value={String(editingPayload.value || '')}
                        onChange={(e) => setEditingPayload({ ...editingPayload, value: e.target.value })}
                        className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="Option value attribute"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                        Selected Text (display label)
                      </label>
                      <input
                        type="text"
                        value={String(editingPayload.selectedText || '')}
                        onChange={(e) => setEditingPayload({ ...editingPayload, selectedText: e.target.value })}
                        className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="Visible option text"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1.5">
                        Option Index
                      </label>
                      <input
                        type="number"
                        value={editingPayload.selectedIndex !== undefined ? String(editingPayload.selectedIndex) : ''}
                        onChange={(e) => setEditingPayload({ ...editingPayload, selectedIndex: e.target.value ? Number(e.target.value) : undefined })}
                        className="w-full px-3 py-2 text-sm font-mono bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        placeholder="0"
                        min="0"
                      />
                    </div>
                  </div>
                )}

                <div className="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
                  <button
                    onClick={handleCancelEdit}
                    className="px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => handleEditMergedPayload(index)}
                    className="px-4 py-1.5 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 transition-colors"
                  >
                    Save
                  </button>
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
