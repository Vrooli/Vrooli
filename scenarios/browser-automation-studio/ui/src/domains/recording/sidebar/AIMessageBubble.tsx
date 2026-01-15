/**
 * AIMessageBubble Component
 *
 * Renders messages in the AI conversation interface as a timeline/activity feed.
 * Supports:
 * - User messages (prompts)
 * - Assistant messages (AI navigation with streaming steps)
 * - System messages (errors, info)
 * - Human intervention state
 */

import { useEffect, useRef, useState } from 'react';
import type { AIMessage, AIMessageStatus, AINavigationStep } from './types';
import type { BrowserAction } from '../ai-navigation/types';
import { EntitlementErrorCard } from './EntitlementErrorCard';

// ============================================================================
// Icons
// ============================================================================

function UserIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 20 20">
      <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
    </svg>
  );
}

function SparklesIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
    </svg>
  );
}

function GlobeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
    </svg>
  );
}

function CheckCircleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 20 20">
      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
    </svg>
  );
}

function XCircleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 20 20">
      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
    </svg>
  );
}

function HandIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 11.5V14m0-2.5v-6a1.5 1.5 0 113 0m-3 6a1.5 1.5 0 00-3 0v2a7.5 7.5 0 0015 0v-5a1.5 1.5 0 00-3 0m-6-3V11m0-5.5v-1a1.5 1.5 0 013 0v1m0 0V11m0-5.5a1.5 1.5 0 013 0v3m0 0V11" />
    </svg>
  );
}

function LoaderIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
    </svg>
  );
}

function MousePointerIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
    </svg>
  );
}

function KeyboardIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 18h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
    </svg>
  );
}

function ArrowsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
    </svg>
  );
}

function EyeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
    </svg>
  );
}

function ClockIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function ListIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
    </svg>
  );
}

function CommandIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18 3a3 3 0 00-3 3v12a3 3 0 003 3 3 3 0 003-3 3 3 0 00-3-3H6a3 3 0 00-3 3 3 3 0 003 3 3 3 0 003-3V6a3 3 0 00-3-3 3 3 0 00-3 3 3 3 0 003 3h12a3 3 0 003-3 3 3 0 00-3-3z" />
    </svg>
  );
}

function ChevronDownIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
    </svg>
  );
}

function StopIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 20 20">
      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clipRule="evenodd" />
    </svg>
  );
}

// ============================================================================
// Helper Functions
// ============================================================================

function formatTime(date: Date): string {
  return date.toLocaleTimeString(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  });
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function truncateUrl(url: string, maxLength = 40): string {
  if (!url || url.length <= maxLength) return url;
  try {
    const parsed = new URL(url);
    const hostAndPath = parsed.host + parsed.pathname;
    if (hostAndPath.length <= maxLength) return hostAndPath;
    return `${parsed.host}/...${parsed.pathname.slice(-15)}`;
  } catch {
    return url.slice(0, maxLength) + '...';
  }
}

function getActionIcon(actionType: BrowserAction['type']): JSX.Element {
  const iconClass = "w-3.5 h-3.5";
  switch (actionType) {
    case 'click':
      return <MousePointerIcon className={iconClass} />;
    case 'type':
      return <KeyboardIcon className={iconClass} />;
    case 'scroll':
      return <ArrowsIcon className={iconClass} />;
    case 'navigate':
      return <GlobeIcon className={iconClass} />;
    case 'hover':
      return <EyeIcon className={iconClass} />;
    case 'select':
      return <ListIcon className={iconClass} />;
    case 'wait':
      return <ClockIcon className={iconClass} />;
    case 'keypress':
      return <CommandIcon className={iconClass} />;
    case 'done':
      return <CheckCircleIcon className={iconClass} />;
    case 'request_human':
      return <HandIcon className={iconClass} />;
    default:
      return <MousePointerIcon className={iconClass} />;
  }
}

function getActionLabel(action: BrowserAction): string {
  switch (action.type) {
    case 'click':
      return 'Click';
    case 'type':
      return action.text ? `Type "${action.text.slice(0, 20)}${action.text.length > 20 ? '...' : ''}"` : 'Type';
    case 'scroll':
      return `Scroll ${action.direction || 'down'}`;
    case 'navigate':
      return 'Navigate';
    case 'hover':
      return 'Hover';
    case 'select':
      return action.text ? `Select "${action.text}"` : 'Select';
    case 'wait':
      return 'Wait';
    case 'keypress':
      return action.key ? `Press ${action.key}` : 'Keypress';
    case 'done':
      return 'Done';
    case 'request_human':
      return 'Request Help';
    default:
      return action.type;
  }
}

function getStatusConfig(status: AIMessageStatus): {
  icon: JSX.Element;
  label: string;
  className: string;
} {
  switch (status) {
    case 'pending':
      return {
        icon: <LoaderIcon className="w-3 h-3 animate-spin" />,
        label: 'Starting',
        className: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400',
      };
    case 'running':
      return {
        icon: <LoaderIcon className="w-3 h-3 animate-spin" />,
        label: 'Navigating',
        className: 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300',
      };
    case 'aborting':
      return {
        icon: <StopIcon className="w-3 h-3 animate-pulse" />,
        label: 'Stopping',
        className: 'bg-orange-100 dark:bg-orange-900/40 text-orange-700 dark:text-orange-300',
      };
    case 'completed':
      return {
        icon: <CheckCircleIcon className="w-3 h-3" />,
        label: 'Completed',
        className: 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300',
      };
    case 'failed':
      return {
        icon: <XCircleIcon className="w-3 h-3" />,
        label: 'Failed',
        className: 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300',
      };
    case 'aborted':
      return {
        icon: <StopIcon className="w-3 h-3" />,
        label: 'Stopped',
        className: 'bg-yellow-100 dark:bg-yellow-900/40 text-yellow-700 dark:text-yellow-300',
      };
    case 'awaiting_human':
      return {
        icon: <HandIcon className="w-3 h-3" />,
        label: 'Waiting for you',
        className: 'bg-purple-100 dark:bg-purple-900/40 text-purple-700 dark:text-purple-300',
      };
    default:
      return {
        icon: <LoaderIcon className="w-3 h-3" />,
        label: 'Unknown',
        className: 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400',
      };
  }
}

// ============================================================================
// Sub-Components
// ============================================================================

function UserMessage({ message }: { message: AIMessage }) {
  return (
    <div className="flex items-start gap-3 mb-4">
      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-purple-600 flex items-center justify-center">
        <UserIcon className="w-4 h-4 text-white" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-sm font-medium text-gray-900 dark:text-white">You</span>
          <span className="text-xs text-gray-500 dark:text-gray-400">{formatTime(message.timestamp)}</span>
        </div>
        <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
          {message.content}
        </p>
      </div>
    </div>
  );
}

function SystemMessage({ message }: { message: AIMessage }) {
  // Check if this is an entitlement error that needs special rendering
  if (message.errorCode === 'AI_NOT_AVAILABLE' || message.errorCode === 'INSUFFICIENT_CREDITS') {
    return (
      <EntitlementErrorCard
        errorCode={message.errorCode}
        details={message.errorDetails}
      />
    );
  }

  // Regular system message
  return (
    <div className="flex justify-center mb-4">
      <div className="bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg px-3 py-1.5 text-xs max-w-[90%]">
        {message.content}
      </div>
    </div>
  );
}

interface TimelineStepProps {
  step: AINavigationStep;
  isRunning: boolean;
  isAborting: boolean;
  isLatest: boolean;
}

function TimelineStep({ step, isRunning, isAborting, isLatest }: TimelineStepProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isCurrentlyRunning = isRunning && isLatest && !isAborting;
  const isCurrentlyAborting = isAborting && isLatest;
  const hasError = !!step.error;
  const hasReasoning = !!step.reasoning;

  // Determine node color and ring
  // - Error: red border
  // - Currently aborting: orange border with ring
  // - Currently running: purple border with pulsing glow
  // - Completed (not running): purple ring (no glow)
  // - Goal achieved: green border
  // - Default: gray border
  const nodeColorClass = hasError
    ? 'border-red-500 dark:border-red-400 bg-red-50 dark:bg-red-900/20'
    : isCurrentlyAborting
      ? 'border-orange-500 dark:border-orange-400 ring-4 ring-orange-500/30 bg-orange-50 dark:bg-orange-900/20'
    : isCurrentlyRunning
      ? 'border-purple-500 dark:border-purple-400 ring-4 ring-purple-500/30 animate-pulse'
      : step.goalAchieved
        ? 'border-green-500 dark:border-green-400 bg-green-50 dark:bg-green-900/20 ring-2 ring-purple-500/40'
        : 'border-purple-400 dark:border-purple-500 bg-white dark:bg-gray-900 ring-2 ring-purple-500/30';

  return (
    <div className={`relative ml-8 mb-3 ${isCurrentlyRunning ? 'ring-2 ring-purple-500/20 rounded-lg' : ''}`}>
      {/* Timeline node */}
      <div className={`absolute -left-10 top-3 w-6 h-6 rounded-full border-2 flex items-center justify-center text-gray-500 dark:text-gray-400 ${nodeColorClass}`}>
        {getActionIcon(step.action.type)}
      </div>

      {/* Step card */}
      <div className="bg-gray-50 dark:bg-gray-800/60 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm hover:shadow-md transition-shadow overflow-hidden">
        {/* Header - clickable to expand/collapse */}
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          className="w-full flex items-center gap-2 px-3 py-2 bg-white/50 dark:bg-gray-800/50 hover:bg-gray-100/50 dark:hover:bg-gray-700/50 transition-colors text-left"
        >
          <span className="flex-shrink-0 w-5 h-5 rounded-full bg-purple-100 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300 text-xs font-bold flex items-center justify-center">
            {step.stepNumber}
          </span>
          <span className="text-sm font-medium text-gray-900 dark:text-white flex-1 min-w-0 truncate">
            {getActionLabel(step.action)}
          </span>
          {step.durationMs > 0 && (
            <span className="text-xs font-mono text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded flex-shrink-0">
              {formatDuration(step.durationMs)}
            </span>
          )}
          {hasReasoning && (
            <ChevronDownIcon className={`w-4 h-4 text-gray-400 flex-shrink-0 transition-transform ${isExpanded ? 'rotate-180' : ''}`} />
          )}
        </button>

        {/* URL - always visible */}
        {step.currentUrl && (
          <div className="flex items-center gap-2 px-3 py-1.5 text-xs border-t border-gray-200 dark:border-gray-700">
            <GlobeIcon className="w-3 h-3 flex-shrink-0 text-gray-400" />
            <span className="font-mono text-gray-600 dark:text-gray-400 truncate" title={step.currentUrl}>
              {truncateUrl(step.currentUrl)}
            </span>
          </div>
        )}

        {/* Expandable content */}
        {isExpanded && (
          <>
            {/* Reasoning */}
            {hasReasoning && (
              <div className="px-3 py-2 bg-purple-50/50 dark:bg-purple-900/10 border-l-2 border-purple-400 border-t border-gray-200 dark:border-gray-700">
                <p className="text-xs font-semibold text-purple-600 dark:text-purple-400 uppercase tracking-wider mb-0.5">
                  Reasoning
                </p>
                <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
                  {step.reasoning}
                </p>
              </div>
            )}

            {/* Action-specific details */}
            {(step.action.coordinates || (step.action.url && step.action.type === 'navigate')) && (
              <div className="px-3 py-2 space-y-1.5 border-t border-gray-200 dark:border-gray-700">
                {step.action.coordinates && (
                  <div className="text-xs bg-gray-100 dark:bg-gray-700/50 rounded px-2 py-1 font-mono text-gray-600 dark:text-gray-400 inline-block">
                    ({step.action.coordinates.x}, {step.action.coordinates.y})
                  </div>
                )}

                {step.action.url && step.action.type === 'navigate' && (
                  <div className="text-xs text-gray-600 dark:text-gray-400">
                    <span className="font-medium">To:</span>{' '}
                    <span className="font-mono">{truncateUrl(step.action.url)}</span>
                  </div>
                )}
              </div>
            )}
          </>
        )}

        {/* Error display - always visible if present */}
        {hasError && (
          <div className="px-3 py-2 bg-red-50 dark:bg-red-900/20 border-t border-red-200 dark:border-red-800">
            <p className="text-xs text-red-700 dark:text-red-300">{step.error}</p>
          </div>
        )}
      </div>
    </div>
  );
}

interface NavigationTimelineProps {
  message: AIMessage;
  onAbort?: () => void;
  onHumanDone?: () => void;
}

function NavigationTimeline({ message, onAbort, onHumanDone }: NavigationTimelineProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const latestStepRef = useRef<HTMLDivElement>(null);
  const steps = message.steps ?? [];
  const status = message.status ?? 'pending';
  const lastStep = steps[steps.length - 1];
  const currentUrl = lastStep?.currentUrl ?? '';
  const goalAchieved = lastStep?.goalAchieved ?? false;

  // Check if the last step is a "done" action - this means navigation is effectively complete
  // even if the status hasn't been updated yet
  const isDoneAction = lastStep?.action.type === 'done';
  const effectiveStatus = isDoneAction && (status === 'running' || status === 'pending') ? 'completed' : status;
  // isRunning means we're actively navigating (show skeleton, progress bar, etc.)
  const isRunning = (effectiveStatus === 'running' || effectiveStatus === 'pending') && !isDoneAction;
  // isAborting means we're in the process of stopping (show different UI state)
  const isAborting = effectiveStatus === 'aborting';
  // isInProgress means we're either running or aborting (for skeleton and progress display)
  const isInProgress = isRunning || isAborting;
  const isAwaitingHuman = effectiveStatus === 'awaiting_human';
  const statusConfig = getStatusConfig(effectiveStatus);

  // Auto-scroll to latest step (only when not collapsed)
  useEffect(() => {
    if (isRunning && !isCollapsed && latestStepRef.current) {
      latestStepRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  }, [steps.length, isRunning, isCollapsed]);

  return (
    <div className="mb-4">
      {/* Header card - clickable to collapse/expand */}
      <div className={`bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 px-3 py-2 ${isCollapsed ? 'rounded-lg' : 'rounded-t-lg'}`}>
        {/* Goal display */}
        <button
          type="button"
          onClick={() => setIsCollapsed(!isCollapsed)}
          className="w-full text-left hover:opacity-80 transition-opacity"
        >
          <div className="flex items-start gap-2">
            <SparklesIcon className="w-4 h-4 text-purple-500 flex-shrink-0 mt-0.5" />
            <p className="text-sm text-purple-900 dark:text-purple-100 font-medium flex-1 min-w-0">
              {message.content || 'AI Navigation'}
            </p>
            <ChevronDownIcon className={`w-4 h-4 text-purple-500 flex-shrink-0 transition-transform ${isCollapsed ? '' : 'rotate-180'}`} />
          </div>
        </button>

        {/* Status bar */}
        <div className="mt-2 flex items-center gap-3 flex-wrap">
          <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-medium rounded-full ${statusConfig.className}`}>
            {statusConfig.icon}
            {statusConfig.label}
            {steps.length > 0 && (
              <span className="opacity-75">({steps.length})</span>
            )}
          </span>

          {currentUrl && (
            <div className="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-400 min-w-0">
              <GlobeIcon className="w-3 h-3 flex-shrink-0" />
              <span className="font-mono truncate" title={currentUrl}>
                {truncateUrl(currentUrl, 30)}
              </span>
            </div>
          )}

          {/* Stop button in header (visible when running, hidden when already aborting) */}
          {message.canAbort && isRunning && !isAborting && onAbort && (
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); onAbort(); }}
              className="ml-auto p-1 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 rounded transition-colors"
              title="Stop Navigation"
            >
              <StopIcon className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Progress bar (when in progress) */}
        {isInProgress && steps.length > 0 && (
          <div className="mt-2 flex items-center gap-2">
            <div className="flex-1 h-1.5 bg-purple-200 dark:bg-purple-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-purple-500 transition-all duration-300 ease-out"
                style={{ width: `${Math.min((steps.length / 20) * 100, 95)}%` }}
              />
            </div>
            <span className="text-xs text-purple-600 dark:text-purple-400 font-mono">
              {steps.length}/20
            </span>
          </div>
        )}

        {/* Timestamp */}
        <p className="text-xs text-purple-600/70 dark:text-purple-400/70 mt-2">
          {formatTime(message.timestamp)}
        </p>
      </div>

      {/* Collapsible content */}
      {!isCollapsed && (
        <>
          {/* Timeline container */}
          {steps.length > 0 && (
            <div className="relative border-l border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900/50 px-3 py-3">
              {/* Vertical timeline line - positioned to go through center of step nodes
                   Node position: px-3 (12px) + ml-8 (32px) - left-10 (40px) = 4px, center at 4 + 12 = 16px */}
              <div className="absolute left-[15px] top-0 bottom-0 w-0.5 bg-gradient-to-b from-purple-500 via-purple-400 to-gray-300 dark:from-purple-600 dark:via-purple-500 dark:to-gray-700" />

              {/* Steps */}
              {steps.map((step, index) => (
                <div key={step.id} ref={index === steps.length - 1 ? latestStepRef : undefined}>
                  <TimelineStep
                    step={step}
                    isRunning={isRunning}
                    isAborting={isAborting}
                    isLatest={index === steps.length - 1}
                  />
                </div>
              ))}

              {/* Skeleton for next step (when running, not when aborting) */}
              {isRunning && !isAborting && (
                <div className="relative ml-8 mb-3 animate-pulse">
                  <div className="absolute -left-10 top-3 w-6 h-6 rounded-full bg-gray-200 dark:bg-gray-700" />
                  <div className="bg-gray-100 dark:bg-gray-800 rounded-lg p-3 space-y-2">
                    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
                    <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-2/3" />
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Pending state (no steps yet) */}
          {steps.length === 0 && isInProgress && (
            <div className="border-l border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900/50 px-3 py-4">
              <div className="flex items-center gap-3 text-gray-500 dark:text-gray-400">
                <LoaderIcon className="w-5 h-5 animate-spin" />
                <span className="text-sm">Starting navigation...</span>
              </div>
            </div>
          )}

          {/* Footer card */}
          <div className="rounded-b-lg border border-t-0 border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 px-3 py-2">
            {/* Human intervention panel */}
            {isAwaitingHuman && message.humanIntervention && (
              <div className="mb-3 p-2 bg-purple-50 dark:bg-purple-900/30 rounded-lg border border-purple-200 dark:border-purple-800">
                <div className="flex items-center gap-2 mb-1.5">
                  <HandIcon className="w-4 h-4 text-purple-600 dark:text-purple-400" />
                  <span className="text-sm font-medium text-purple-800 dark:text-purple-200">
                    Human Intervention Required
                  </span>
                </div>
                <p className="text-xs text-purple-700 dark:text-purple-300 mb-2">
                  {message.humanIntervention.reason}
                </p>
                {message.humanIntervention.instructions && (
                  <p className="text-xs text-purple-600 dark:text-purple-400 italic mb-2">
                    {message.humanIntervention.instructions}
                  </p>
                )}
                {onHumanDone && (
                  <button
                    onClick={(e) => { e.stopPropagation(); onHumanDone(); }}
                    className="w-full px-3 py-1.5 text-xs font-medium text-white bg-purple-600 rounded-lg hover:bg-purple-700 transition-colors"
                  >
                    I'm Done
                  </button>
                )}
              </div>
            )}

            {/* Completion summary */}
            {effectiveStatus === 'completed' && (goalAchieved || isDoneAction) && (
              <div className="flex items-center gap-2 text-green-700 dark:text-green-300">
                <CheckCircleIcon className="w-4 h-4" />
                <span className="text-sm font-medium">Goal achieved in {steps.length} step{steps.length !== 1 ? 's' : ''}</span>
              </div>
            )}

            {effectiveStatus === 'completed' && !goalAchieved && !isDoneAction && (
              <div className="flex items-center gap-2 text-yellow-700 dark:text-yellow-300">
                <CheckCircleIcon className="w-4 h-4" />
                <span className="text-sm">Finished after {steps.length} step{steps.length !== 1 ? 's' : ''} (goal not confirmed)</span>
              </div>
            )}

            {effectiveStatus === 'failed' && message.error && (
              <div className="flex items-start gap-2 text-red-700 dark:text-red-300">
                <XCircleIcon className="w-4 h-4 flex-shrink-0 mt-0.5" />
                <span className="text-sm">{message.error}</span>
              </div>
            )}

            {effectiveStatus === 'aborted' && (
              <div className="flex items-center gap-2 text-yellow-700 dark:text-yellow-300">
                <XCircleIcon className="w-4 h-4" />
                <span className="text-sm">Navigation aborted after {steps.length} step{steps.length !== 1 ? 's' : ''}</span>
              </div>
            )}

            {/* Aborting message */}
            {isAborting && (
              <div className="flex items-center gap-2 text-orange-700 dark:text-orange-300">
                <StopIcon className="w-4 h-4 animate-pulse" />
                <span className="text-sm">Stopping after current step...</span>
              </div>
            )}

            {/* Token usage (when not in progress) */}
            {!isInProgress && message.totalTokens && message.totalTokens > 0 && (
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                {message.totalTokens.toLocaleString()} tokens used
              </p>
            )}

            {/* Stop button (hidden when already aborting) */}
            {message.canAbort && isRunning && !isAborting && onAbort && (
              <button
                onClick={(e) => { e.stopPropagation(); onAbort(); }}
                className="mt-2 w-full px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-100 dark:bg-red-900/30 rounded-lg hover:bg-red-200 dark:hover:bg-red-900/50 transition-colors border border-red-200 dark:border-red-800 flex items-center justify-center gap-1.5"
              >
                <StopIcon className="w-3.5 h-3.5" />
                Stop Navigation
              </button>
            )}
          </div>
        </>
      )}
    </div>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export interface AIMessageBubbleProps {
  /** The message to display */
  message: AIMessage;
  /** Callback to abort navigation (for running messages) */
  onAbort?: () => void;
  /** Callback when human intervention is complete */
  onHumanDone?: () => void;
}

export function AIMessageBubble({ message, onAbort, onHumanDone }: AIMessageBubbleProps) {
  if (message.role === 'user') {
    return <UserMessage message={message} />;
  }

  if (message.role === 'system') {
    return <SystemMessage message={message} />;
  }

  // Assistant message - render as timeline
  return (
    <NavigationTimeline
      message={message}
      onAbort={onAbort}
      onHumanDone={onHumanDone}
    />
  );
}
