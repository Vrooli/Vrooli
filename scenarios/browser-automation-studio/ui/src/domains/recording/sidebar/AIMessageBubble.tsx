/**
 * AIMessageBubble Component
 *
 * Renders a single message in the AI conversation interface.
 * Supports:
 * - User messages (prompts)
 * - Assistant messages (AI responses with status and steps)
 * - System messages (errors, info)
 * - Human intervention state
 */

import { useState, useMemo } from 'react';
import type { AIMessage, AIMessageStatus } from './types';

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Format timestamp for display.
 */
function formatTime(date: Date): string {
  return date.toLocaleTimeString(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  });
}

/**
 * Get status badge styling.
 */
function getStatusBadge(status: AIMessageStatus): {
  bgClass: string;
  textClass: string;
  label: string;
} {
  switch (status) {
    case 'pending':
      return {
        bgClass: 'bg-gray-100 dark:bg-gray-700',
        textClass: 'text-gray-600 dark:text-gray-400',
        label: 'Starting...',
      };
    case 'running':
      return {
        bgClass: 'bg-blue-100 dark:bg-blue-900/30',
        textClass: 'text-blue-700 dark:text-blue-300',
        label: 'Navigating',
      };
    case 'completed':
      return {
        bgClass: 'bg-green-100 dark:bg-green-900/30',
        textClass: 'text-green-700 dark:text-green-300',
        label: 'Completed',
      };
    case 'failed':
      return {
        bgClass: 'bg-red-100 dark:bg-red-900/30',
        textClass: 'text-red-700 dark:text-red-300',
        label: 'Failed',
      };
    case 'aborted':
      return {
        bgClass: 'bg-yellow-100 dark:bg-yellow-900/30',
        textClass: 'text-yellow-700 dark:text-yellow-300',
        label: 'Aborted',
      };
    case 'awaiting_human':
      return {
        bgClass: 'bg-purple-100 dark:bg-purple-900/30',
        textClass: 'text-purple-700 dark:text-purple-300',
        label: 'Waiting for you',
      };
    default:
      return {
        bgClass: 'bg-gray-100 dark:bg-gray-700',
        textClass: 'text-gray-600 dark:text-gray-400',
        label: 'Unknown',
      };
  }
}

// ============================================================================
// Component Props
// ============================================================================

export interface AIMessageBubbleProps {
  /** The message to display */
  message: AIMessage;
  /** Callback to abort navigation (for running messages) */
  onAbort?: () => void;
  /** Callback when human intervention is complete */
  onHumanDone?: () => void;
}

// ============================================================================
// Component
// ============================================================================

export function AIMessageBubble({ message, onAbort, onHumanDone }: AIMessageBubbleProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const isUser = message.role === 'user';
  const isSystem = message.role === 'system';
  const isAssistant = message.role === 'assistant';

  // Get summary for assistant messages
  const assistantSummary = useMemo(() => {
    if (!isAssistant) return null;

    const stepCount = message.steps?.length ?? 0;
    const lastStep = message.steps?.[stepCount - 1];
    const goalAchieved = lastStep?.goalAchieved ?? false;

    if (message.status === 'pending') {
      return 'Starting navigation...';
    }

    if (message.status === 'running') {
      return `Step ${stepCount}${lastStep ? `: ${lastStep.action.type}` : ''}...`;
    }

    if (message.status === 'awaiting_human') {
      return message.humanIntervention?.reason ?? 'Waiting for human intervention...';
    }

    if (message.status === 'completed' && goalAchieved) {
      return `Completed in ${stepCount} step${stepCount !== 1 ? 's' : ''}`;
    }

    if (message.status === 'completed') {
      return `Finished after ${stepCount} step${stepCount !== 1 ? 's' : ''} (goal not confirmed)`;
    }

    if (message.error) {
      return message.error;
    }

    return `${stepCount} step${stepCount !== 1 ? 's' : ''}`;
  }, [isAssistant, message]);

  // User message
  if (isUser) {
    return (
      <div className="flex justify-end mb-3">
        <div className="max-w-[30rem] bg-purple-600 text-white rounded-2xl rounded-tr-sm px-4 py-2">
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          <p className="text-xs text-purple-200 mt-1">{formatTime(message.timestamp)}</p>
        </div>
      </div>
    );
  }

  // System message
  if (isSystem) {
    return (
      <div className="flex justify-center mb-3">
        <div className="bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg px-3 py-1.5 text-xs">
          {message.content}
        </div>
      </div>
    );
  }

  // Assistant message
  const statusBadge = message.status ? getStatusBadge(message.status) : null;
  const isRunning = message.status === 'running' || message.status === 'pending';
  const isAwaitingHuman = message.status === 'awaiting_human';
  const stepCount = message.steps?.length ?? 0;

  return (
    <div className="flex justify-start mb-3">
      <div className="max-w-[30rem] bg-gray-100 dark:bg-gray-800 rounded-2xl rounded-tl-sm px-4 py-3">
        {/* Status badge */}
        {statusBadge && (
          <div className="flex items-center gap-2 mb-2">
            <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full ${statusBadge.bgClass} ${statusBadge.textClass}`}>
              {isRunning && (
                <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              )}
              {message.status === 'completed' && (
                <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                </svg>
              )}
              {statusBadge.label}
              {isRunning && stepCount > 0 && (
                <span className="ml-1 text-[10px] opacity-75">({stepCount})</span>
              )}
            </span>
          </div>
        )}

        {/* Summary/Content */}
        <p className="text-sm text-gray-900 dark:text-white">
          {assistantSummary}
        </p>

        {/* Human intervention details */}
        {isAwaitingHuman && message.humanIntervention && (
          <div className="mt-2 p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
            {message.humanIntervention.instructions && (
              <p className="text-xs text-purple-700 dark:text-purple-300 mb-2">
                {message.humanIntervention.instructions}
              </p>
            )}
            {onHumanDone && (
              <button
                onClick={onHumanDone}
                className="w-full px-3 py-1.5 text-xs font-medium text-white bg-purple-600 rounded-lg hover:bg-purple-700 transition-colors"
              >
                I'm Done
              </button>
            )}
          </div>
        )}

        {/* Token usage (when not running) */}
        {!isRunning && message.totalTokens && message.totalTokens > 0 && (
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {message.totalTokens.toLocaleString()} tokens
          </p>
        )}

        {/* Expandable steps */}
        {stepCount > 0 && !isRunning && (
          <button
            onClick={() => setIsExpanded(!isExpanded)}
            className="mt-2 text-xs text-purple-600 dark:text-purple-400 hover:underline flex items-center gap-1"
          >
            <svg
              className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            {isExpanded ? 'Hide' : 'Show'} {stepCount} step{stepCount !== 1 ? 's' : ''}
          </button>
        )}

        {isExpanded && message.steps && (
          <div className="mt-2 space-y-1 pl-3 border-l-2 border-gray-200 dark:border-gray-700">
            {message.steps.map((step) => (
              <div key={step.id} className="text-xs">
                <span className="font-medium text-gray-700 dark:text-gray-300">
                  {step.stepNumber}. {step.action.type}
                </span>
                {step.reasoning && (
                  <p className="text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-2">
                    {step.reasoning}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Abort button */}
        {message.canAbort && isRunning && onAbort && (
          <button
            onClick={onAbort}
            className="mt-3 w-full px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-100 dark:bg-red-900/30 rounded-lg hover:bg-red-200 dark:hover:bg-red-900/50 transition-colors"
          >
            Abort Navigation
          </button>
        )}

        {/* Timestamp */}
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-2">{formatTime(message.timestamp)}</p>
      </div>
    </div>
  );
}
