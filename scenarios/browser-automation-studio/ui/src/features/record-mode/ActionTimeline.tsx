/**
 * ActionTimeline Component
 *
 * Displays a timeline of recorded actions with:
 * - Expandable action details
 * - Confidence indicators and warnings
 * - Selector editing with the SelectorEditor component
 * - Action payload editing
 * - Delete functionality
 */

import { useState, useCallback } from 'react';
import { SelectorEditor } from './SelectorEditor';
import type { RecordedAction, SelectorValidation } from './types';

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
}: ActionTimelineProps) {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editMode, setEditMode] = useState<'selector' | 'payload' | null>(null);
  const [editingPayload, setEditingPayload] = useState<Record<string, unknown>>({});

  const handleToggleExpand = (index: number) => {
    if (editingIndex !== null) return; // Don't toggle while editing
    setExpandedIndex(expandedIndex === index ? null : index);
  };

  const handleStartEditSelector = (index: number) => {
    setEditingIndex(index);
    setEditMode('selector');
  };

  const handleStartEditPayload = (index: number, action: RecordedAction) => {
    setEditingIndex(index);
    setEditMode('payload');
    setEditingPayload(action.payload || {});
  };

  const handleSaveSelector = useCallback(
    (index: number, newSelector: string) => {
      if (onEditSelector) {
        onEditSelector(index, newSelector);
      }
      setEditingIndex(null);
      setEditMode(null);
    },
    [onEditSelector]
  );

  const handleSavePayload = useCallback(
    (index: number) => {
      if (onEditPayload) {
        onEditPayload(index, editingPayload);
      }
      setEditingIndex(null);
      setEditMode(null);
      setEditingPayload({});
    },
    [onEditPayload, editingPayload]
  );

  const handleCancelEdit = useCallback(() => {
    setEditingIndex(null);
    setEditMode(null);
    setEditingPayload({});
  }, []);

  const getActionIcon = (actionType: string) => {
    switch (actionType) {
      case 'click':
        return (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
          </svg>
        );
      case 'type':
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
      case 'keypress':
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
    switch (action.actionType) {
      case 'click':
        if (action.elementMeta?.innerText) {
          return `Click: "${truncate(action.elementMeta.innerText, 30)}"`;
        }
        if (action.elementMeta?.ariaLabel) {
          return `Click: ${action.elementMeta.ariaLabel}`;
        }
        return `Click ${action.elementMeta?.tagName || 'element'}`;
      case 'type':
        if (action.payload?.text) {
          return `Type: "${truncate(String(action.payload.text), 30)}"`;
        }
        return 'Type text';
      case 'navigate':
        return `Navigate to ${truncate(action.url, 40)}`;
      case 'scroll':
        return `Scroll ${action.payload?.deltaY !== undefined && Number(action.payload.deltaY) > 0 ? 'down' : 'up'}`;
      case 'keypress':
        return `Press ${action.payload?.key || 'key'}`;
      case 'select':
        return `Select: ${action.payload?.selectedText || action.payload?.value || 'option'}`;
      case 'focus':
        return `Focus ${action.elementMeta?.tagName || 'element'}`;
      default:
        return action.actionType;
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
      {actions.map((action, index) => {
        const isExpanded = expandedIndex === index;
        const isEditing = editingIndex === index;
        const confidenceLevel = getConfidenceLevel(action.confidence);

        return (
          <div
            key={action.id}
            className={`py-2 px-3 transition-colors ${
              isExpanded ? 'bg-gray-50 dark:bg-gray-800/50' : 'hover:bg-gray-50 dark:hover:bg-gray-800'
            } ${confidenceLevel === 'low' ? 'border-l-2 border-l-red-400' : ''}`}
          >
            {/* Action header */}
            <div
              className={`flex items-center gap-3 ${!isEditing ? 'cursor-pointer' : ''}`}
              onClick={() => handleToggleExpand(index)}
            >
              {/* Index */}
              <span className="flex-shrink-0 w-6 h-6 flex items-center justify-center text-xs font-mono text-gray-400 bg-gray-100 dark:bg-gray-800 rounded">
                {index + 1}
              </span>

              {/* Icon */}
              <span className="flex-shrink-0 text-gray-600 dark:text-gray-400">
                {getActionIcon(action.actionType)}
              </span>

              {/* Label */}
              <span className="flex-1 text-sm truncate">{getActionLabel(action)}</span>

              {/* Confidence indicator */}
              {action.selector && getConfidenceIndicator(action.confidence)}

              {/* Delete button */}
              {onDeleteAction && !isEditing && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onDeleteAction(index);
                  }}
                  className="flex-shrink-0 p-1 text-gray-400 hover:text-red-500 transition-colors"
                  title="Delete action"
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
                {/* Unstable selector warning */}
                {action.selector && confidenceLevel !== 'high' && (
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

                {/* Payload (for type actions) */}
                {action.actionType === 'type' && action.payload?.text && (
                  <div className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Text</span>
                      {onEditPayload && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStartEditPayload(index, action);
                          }}
                          className="text-xs text-blue-500 hover:text-blue-600 font-medium"
                        >
                          Edit Text
                        </button>
                      )}
                    </div>
                    <code className="block px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded">
                      "{String(action.payload.text)}"
                    </code>
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
                  onSave={(newSelector) => handleSaveSelector(index, newSelector)}
                  onCancel={handleCancelEdit}
                />
              </div>
            )}

            {/* Payload Editor */}
            {isEditing && editMode === 'payload' && (
              <div className="mt-3 ml-9 space-y-3 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
                {action.actionType === 'type' && (
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
                )}

                {action.actionType === 'keypress' && (
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
                )}

                <div className="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
                  <button
                    onClick={handleCancelEdit}
                    className="px-3 py-1.5 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => handleSavePayload(index)}
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
