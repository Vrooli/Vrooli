/**
 * ActionTimeline Component
 *
 * Displays a timeline of recorded actions with the ability to edit/delete.
 */

import { useState } from 'react';
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
}

export function ActionTimeline({
  actions,
  isRecording,
  onDeleteAction,
  onValidateSelector,
  onEditSelector,
}: ActionTimelineProps) {
  const [expandedIndex, setExpandedIndex] = useState<number | null>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editingSelector, setEditingSelector] = useState('');
  const [validationResult, setValidationResult] = useState<SelectorValidation | null>(null);
  const [isValidating, setIsValidating] = useState(false);

  const handleToggleExpand = (index: number) => {
    setExpandedIndex(expandedIndex === index ? null : index);
    setEditingIndex(null);
    setValidationResult(null);
  };

  const handleStartEdit = (index: number) => {
    const action = actions[index];
    setEditingIndex(index);
    setEditingSelector(action.selector?.primary || '');
    setValidationResult(null);
  };

  const handleValidate = async () => {
    if (!onValidateSelector || !editingSelector) return;

    setIsValidating(true);
    try {
      const result = await onValidateSelector(editingSelector);
      setValidationResult(result);
    } catch (err) {
      setValidationResult({
        valid: false,
        match_count: 0,
        selector: editingSelector,
        error: err instanceof Error ? err.message : 'Validation failed',
      });
    } finally {
      setIsValidating(false);
    }
  };

  const handleSaveEdit = () => {
    if (editingIndex !== null && onEditSelector) {
      onEditSelector(editingIndex, editingSelector);
    }
    setEditingIndex(null);
    setValidationResult(null);
  };

  const handleCancelEdit = () => {
    setEditingIndex(null);
    setEditingSelector('');
    setValidationResult(null);
  };

  const getActionIcon = (actionType: string) => {
    switch (actionType) {
      case 'click':
        return (
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"
            />
          </svg>
        );
      case 'type':
        return (
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
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
          <svg
            className="w-4 h-4"
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
        );
      case 'scroll':
        return (
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4"
            />
          </svg>
        );
      case 'keypress':
        return (
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707"
            />
          </svg>
        );
      default:
        return (
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z"
            />
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
        return `Scroll ${action.payload?.deltaY !== undefined && action.payload.deltaY > 0 ? 'down' : 'up'}`;
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

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.8) return 'text-green-500';
    if (confidence >= 0.5) return 'text-yellow-500';
    return 'text-red-500';
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
            <p className="text-sm">No actions recorded yet</p>
            <p className="text-xs text-gray-400 mt-1">Click "Start Recording" to begin</p>
          </>
        )}
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700">
      {actions.map((action, index) => (
        <div
          key={action.id}
          className="py-2 px-3 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
        >
          {/* Action header */}
          <div
            className="flex items-center gap-3 cursor-pointer"
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
            <span
              className={`flex-shrink-0 text-xs ${getConfidenceColor(action.confidence)}`}
              title={`Confidence: ${Math.round(action.confidence * 100)}%`}
            >
              {action.confidence >= 0.8 ? '●' : action.confidence >= 0.5 ? '◐' : '○'}
            </span>

            {/* Delete button */}
            {onDeleteAction && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteAction(index);
                }}
                className="flex-shrink-0 p-1 text-gray-400 hover:text-red-500 transition-colors"
                title="Delete action"
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
              className={`w-4 h-4 text-gray-400 transition-transform ${
                expandedIndex === index ? 'rotate-180' : ''
              }`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 9l-7 7-7-7"
              />
            </svg>
          </div>

          {/* Expanded details */}
          {expandedIndex === index && (
            <div className="mt-3 ml-9 space-y-2 text-sm">
              {/* Selector */}
              {action.selector && (
                <div className="space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-500 text-xs uppercase tracking-wide">Selector</span>
                    {onValidateSelector && editingIndex !== index && (
                      <button
                        onClick={() => handleStartEdit(index)}
                        className="text-xs text-blue-500 hover:text-blue-600"
                      >
                        Edit
                      </button>
                    )}
                  </div>
                  {editingIndex === index ? (
                    <div className="space-y-2">
                      <input
                        type="text"
                        value={editingSelector}
                        onChange={(e) => setEditingSelector(e.target.value)}
                        className="w-full px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded"
                      />
                      {validationResult && (
                        <div
                          className={`text-xs ${validationResult.valid ? 'text-green-500' : 'text-red-500'}`}
                        >
                          {validationResult.valid
                            ? `Found ${validationResult.match_count} element(s)`
                            : validationResult.error || 'Invalid selector'}
                        </div>
                      )}
                      <div className="flex gap-2">
                        <button
                          onClick={handleValidate}
                          disabled={isValidating}
                          className="px-2 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
                        >
                          {isValidating ? 'Validating...' : 'Test'}
                        </button>
                        <button
                          onClick={handleSaveEdit}
                          className="px-2 py-1 text-xs bg-green-500 text-white rounded hover:bg-green-600"
                        >
                          Save
                        </button>
                        <button
                          onClick={handleCancelEdit}
                          className="px-2 py-1 text-xs bg-gray-300 dark:bg-gray-600 rounded hover:bg-gray-400"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : (
                    <code className="block px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded overflow-x-auto">
                      {action.selector.primary}
                    </code>
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
        </div>
      ))}
    </div>
  );
}
