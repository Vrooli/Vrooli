/**
 * AINavigationStepCard Component
 *
 * Renders a single AI navigation step in the timeline.
 * Shows the action, reasoning, tokens used, and any errors.
 */

import { useState } from 'react';
import type { AINavigationStep, BrowserAction } from './types';

interface AINavigationStepCardProps {
  step: AINavigationStep;
  index: number;
}

/** Get icon for action type */
function getActionIcon(actionType: BrowserAction['type']) {
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
    case 'scroll':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
        </svg>
      );
    case 'navigate':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
      );
    case 'hover':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
        </svg>
      );
    case 'wait':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    case 'keypress':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707" />
        </svg>
      );
    case 'done':
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
}

/** Format action for display */
function formatAction(action: BrowserAction): string {
  switch (action.type) {
    case 'click':
      if (action.elementId !== undefined) {
        return `Click element [${action.elementId}]`;
      }
      if (action.coordinates) {
        return `Click at (${action.coordinates.x}, ${action.coordinates.y})`;
      }
      return 'Click';
    case 'type':
      if (action.text) {
        const preview = action.text.length > 30 ? `${action.text.slice(0, 30)}...` : action.text;
        return `Type "${preview}"`;
      }
      return 'Type text';
    case 'scroll':
      return `Scroll ${action.direction ?? 'down'}`;
    case 'navigate':
      if (action.url) {
        const url = action.url.length > 40 ? `${action.url.slice(0, 40)}...` : action.url;
        return `Navigate to ${url}`;
      }
      return 'Navigate';
    case 'hover':
      if (action.elementId !== undefined) {
        return `Hover element [${action.elementId}]`;
      }
      return 'Hover';
    case 'wait':
      return 'Wait';
    case 'keypress':
      return `Press ${action.key ?? 'key'}`;
    case 'done':
      return action.success ? 'Goal achieved' : 'Goal not achieved';
    default:
      return action.type;
  }
}

export function AINavigationStepCard({ step }: AINavigationStepCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleToggle = () => {
    setIsExpanded(!isExpanded);
  };

  // Defensive defaults for potentially incomplete step data
  const action = step.action ?? { type: 'wait' as const };
  const tokensUsed = step.tokensUsed ?? { promptTokens: 0, completionTokens: 0, totalTokens: 0 };
  const durationMs = step.durationMs ?? 0;
  const currentUrl = step.currentUrl ?? '';
  const reasoning = step.reasoning ?? '';
  const timestamp = step.timestamp instanceof Date && !isNaN(step.timestamp.getTime())
    ? step.timestamp
    : new Date();

  return (
    <div
      className={`py-2 px-3 border-b border-gray-200 dark:border-gray-700 ${
        isExpanded ? 'bg-purple-50 dark:bg-purple-900/10' : 'hover:bg-gray-50 dark:hover:bg-gray-800'
      } ${step.error ? 'border-l-2 border-l-red-400' : ''} ${
        step.goalAchieved ? 'border-l-2 border-l-green-400' : ''
      }`}
    >
      {/* Header */}
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={handleToggle}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && handleToggle()}
      >
        {/* Step number */}
        <span className="flex-shrink-0 w-6 h-6 flex items-center justify-center text-xs font-mono text-white bg-purple-500 rounded">
          {step.stepNumber}
        </span>

        {/* Action icon */}
        <span className="flex-shrink-0 text-purple-600 dark:text-purple-400">
          {getActionIcon(action.type)}
        </span>

        {/* Action label */}
        <span className="flex-1 text-sm leading-snug break-words text-gray-900 dark:text-white">
          {formatAction(action)}
        </span>

        {/* AI badge */}
        <span className="px-1.5 py-0.5 text-[10px] font-medium bg-purple-100 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300 rounded">
          AI
        </span>

        {/* Status indicator */}
        {step.goalAchieved && (
          <span className="flex-shrink-0 text-green-500" title="Goal achieved">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
          </span>
        )}
        {step.error && (
          <span className="flex-shrink-0 text-red-500" title={step.error}>
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
          </span>
        )}

        {/* Duration */}
        <span className="flex-shrink-0 text-xs text-gray-400 font-mono">{durationMs}ms</span>

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
        <div className="mt-3 ml-9 space-y-3 text-sm">
          {/* Reasoning */}
          <div className="p-2 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-700">
            <span className="text-gray-500 text-xs uppercase tracking-wide block mb-1">AI Reasoning</span>
            <p className="text-gray-700 dark:text-gray-300 text-sm whitespace-pre-wrap">
              {reasoning || 'No reasoning provided'}
            </p>
          </div>

          {/* Error */}
          {step.error && (
            <div className="p-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <span className="text-red-600 dark:text-red-400 text-xs font-medium">Error:</span>
              <p className="text-red-700 dark:text-red-300 text-sm mt-1">{step.error}</p>
            </div>
          )}

          {/* Action Details */}
          <div>
            <span className="text-gray-500 text-xs uppercase tracking-wide">Action Details</span>
            <code className="block px-2 py-1 mt-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 rounded overflow-x-auto">
              {JSON.stringify(action, null, 2)}
            </code>
          </div>

          {/* Token Usage */}
          <div className="flex gap-4">
            <div>
              <span className="text-gray-500 text-xs uppercase tracking-wide">Tokens</span>
              <p className="text-xs text-gray-600 dark:text-gray-400 font-mono">
                {tokensUsed.totalTokens.toLocaleString()} total
                ({tokensUsed.promptTokens.toLocaleString()} in, {tokensUsed.completionTokens.toLocaleString()} out)
              </p>
            </div>
          </div>

          {/* URL */}
          <div>
            <span className="text-gray-500 text-xs uppercase tracking-wide">URL</span>
            <p className="text-xs text-gray-600 dark:text-gray-400 truncate">{currentUrl || 'Unknown'}</p>
          </div>

          {/* Timestamp */}
          <div className="text-xs text-gray-400">
            {timestamp.toLocaleTimeString()}
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * AINavigationTimeline - Renders a list of AI navigation steps.
 */
interface AINavigationTimelineProps {
  steps: AINavigationStep[];
  isNavigating?: boolean;
}

export function AINavigationTimeline({ steps, isNavigating = false }: AINavigationTimelineProps) {
  if (steps.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-32 text-gray-500">
        {isNavigating ? (
          <>
            <div className="animate-pulse flex space-x-1 mb-2">
              <div className="w-2 h-2 bg-purple-500 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
              <div className="w-2 h-2 bg-purple-500 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
              <div className="w-2 h-2 bg-purple-500 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
            </div>
            <p className="text-sm">AI is navigating...</p>
          </>
        ) : (
          <p className="text-sm">No AI navigation steps yet</p>
        )}
      </div>
    );
  }

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700">
      {steps.map((step, index) => (
        <AINavigationStepCard key={step.id} step={step} index={index} />
      ))}
    </div>
  );
}
