import { useEffect, useRef } from 'react';
import type { DragEvent, KeyboardEventHandler, MouseEvent, PointerEvent, TouchEvent } from 'react';
import { Archive, Brain, CalendarClock, GripVertical, Hash, Tag, Trash2, AlertCircle, StopCircle, Lock, Info } from 'lucide-react';
import type { Issue } from '../types/issue';
import { MANUAL_FAILURE_REASON_LABELS, type ManualFailureReason } from '../types/issue';
import { formatDistanceToNow } from '../utils/date';
import { toTitleCase } from '../utils/string';

const TOUCH_MOVE_THRESHOLD = 14;

// Extended HTMLDivElement interface for drag-and-drop data attributes
interface DraggableCardElement extends HTMLDivElement {
  dataset: DOMStringMap & {
    mouseDragState?: 'pending' | 'active';
    preventClick?: 'true' | 'false';
  };
}

// Extract a concise error summary from investigation report
function extractErrorSummary(report: string): string | null {
  if (!report) return null;

  // Check for specific error types with user-friendly messages
  if (/reached max turns/i.test(report)) {
    const match = report.match(/max turns \((\d+)\)/i);
    const turns = match ? match[1] : 'limit';
    return `Agent reached maximum turns (${turns}) - work may be incomplete`;
  }

  if (/timeout/i.test(report)) {
    return 'Agent execution timed out';
  }

  if (/parsing error/i.test(report)) {
    return 'Agent output could not be parsed - check investigation report for details';
  }

  // Look for common error patterns
  const errorPatterns = [
    /Error: (.+?)(?:\n|$)/i,
    /failed[^:]*: (.+?)(?:\n|$)/i,
    /Investigation failed[^:]*: (.+?)(?:\n|$)/i,
  ];

  for (const pattern of errorPatterns) {
    const match = report.match(pattern);
    if (match && match[1]) {
      // Return first 200 chars of the error message
      return match[1].trim().substring(0, 200);
    }
  }

  // If no specific pattern matched, return first line if it contains "error" or "fail"
  const firstLine = report.split('\n')[0];
  if (firstLine && /error|fail/i.test(firstLine)) {
    return firstLine.substring(0, 200);
  }

  return null;
}

interface IssueCardProps {
  issue: Issue;
  isFocused?: boolean;
  runningProcess?: { agent_id: string; start_time: string; duration?: string; status?: string };
  onSelect?: (issueId: string) => void;
  onDelete?: (issue: Issue) => void;
  onArchive?: (issue: Issue) => void;
  onStopAgent?: (issueId: string) => void;
  draggable?: boolean;
  onDragStart?: (issue: Issue, event: DragEvent<HTMLDivElement>) => void;
  onDragEnd?: (issue: Issue, event: DragEvent<HTMLDivElement>) => void;
  onPointerDown?: (issue: Issue, event: PointerEvent<HTMLDivElement>, card: HTMLElement) => void;
}

export function IssueCard({
  issue,
  isFocused = false,
  runningProcess,
  onSelect,
  onDelete,
  onArchive,
  onStopAgent,
  draggable = true,
  onDragStart,
  onDragEnd,
  onPointerDown,
}: IssueCardProps) {
  const ignoreClickRef = useRef(false);
  const dragHandleActiveRef = useRef(false);
  const cardRef = useRef<HTMLDivElement>(null);
  const pointerStartRef = useRef<{ x: number; y: number } | null>(null);

  const isRunning = !!runningProcess;
  const isFailed = issue.status === 'failed';
  // Check multiple error sources: manual failure reason first (with label), then agent_last_error, then investigation.report
  const errorMessage = isFailed && (
    (issue.manual_review?.marked_as_failed && issue.manual_review?.failure_reason
      ? MANUAL_FAILURE_REASON_LABELS[issue.manual_review.failure_reason as ManualFailureReason] || issue.manual_review.failure_reason
      : null) ||
    issue.metadata?.extra?.agent_last_error ||
    (issue.investigation?.report && extractErrorSummary(issue.investigation.report))
  );

  // Check if issue is blocked by target conflicts
  const isBlocked = issue.status === 'open' && issue.metadata?.extra?.blocked_reason === 'target_conflict';
  const blockedByIssues = isBlocked && issue.metadata?.extra?.blocked_by_issues
    ? issue.metadata.extra.blocked_by_issues.split(',').map(id => id.trim())
    : [];

  // Debug logging for blocked detection
  if (issue.status === 'open' && issue.metadata?.extra?.blocked_reason) {
    console.log('[IssueCard] Blocked issue detected:', {
      id: issue.id,
      status: issue.status,
      blocked_reason: issue.metadata.extra.blocked_reason,
      blocked_by_issues: issue.metadata.extra.blocked_by_issues,
      isBlocked
    });
  }

  const className = [
    'issue-card',
    `priority-${issue.priority.toLowerCase()}`,
    isFocused ? 'issue-card--focused' : '',
    isRunning ? 'issue-card--running' : '',
    isBlocked ? 'issue-card--blocked' : '',
  ]
    .filter(Boolean)
    .join(' ');

  const runningStatusLabel = runningProcess?.status
    ? toTitleCase(runningProcess.status.replace(/[-_]+/g, ' '))
    : 'Executing';

  const handleSelect = () => {
    if (ignoreClickRef.current) {
      ignoreClickRef.current = false;
      return;
    }
    // Check if drag just happened
    if (cardRef.current && (cardRef.current as DraggableCardElement).dataset.preventClick === 'true') {
      return;
    }
    onSelect?.(issue.id);
  };

  const handleKeyDown: KeyboardEventHandler<HTMLDivElement> = (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onSelect?.(issue.id);
    }
  };

  const handleDeleteClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onDelete?.(issue);
  };

  const handleArchiveClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onArchive?.(issue);
  };

  const handleActionPointerDown = (event: PointerEvent<HTMLButtonElement>) => {
    event.stopPropagation();
  };

  const handleActionTouchStart = (event: TouchEvent<HTMLButtonElement>) => {
    event.stopPropagation();
  };

  const handleStopClick = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onStopAgent?.(issue.id);
  };

  const handleDragStartInternal = (event: DragEvent<HTMLDivElement>) => {
    const card = cardRef.current as DraggableCardElement | null;
    if (!card || card.dataset.mouseDragState !== 'pending') {
      event.preventDefault();
      return;
    }
    card.dataset.mouseDragState = 'active';
    onDragStart?.(issue, event);
  };

  const handleDragEndInternal = (event: DragEvent<HTMLDivElement>) => {
    const card = cardRef.current as DraggableCardElement | null;
    if (card) {
      delete card.dataset.mouseDragState;
      card.draggable = false;
    }
    onDragEnd?.(issue, event);
  };

  const isArchived = issue.status === 'archived';

  const handleCardPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement | null;
    if (target?.closest('.issue-card-action') && !target.closest('.issue-card-drag-handle')) {
      return;
    }

    const handle = target?.closest('.issue-card-drag-handle');
    if (!handle || !cardRef.current) {
      return;
    }

    event.stopPropagation();
    pointerStartRef.current = { x: event.clientX, y: event.clientY };

    // Mouse drag: enable native drag
    if (event.pointerType !== 'touch') {
      const card = cardRef.current as DraggableCardElement;
      card.dataset.mouseDragState = 'pending';
      card.draggable = true;

      const onPointerEnd = () => {
        if (card.dataset.mouseDragState === 'pending') {
          delete card.dataset.mouseDragState;
          card.draggable = false;
        }
        card.removeEventListener('pointerup', onPointerEnd);
        card.removeEventListener('pointercancel', onPointerEnd);
      };

      card.addEventListener('pointerup', onPointerEnd, { once: true });
      card.addEventListener('pointercancel', onPointerEnd, { once: true });
      return;
    }

    // Touch drag: use custom pointer handling
    onPointerDown?.(issue, event, cardRef.current);
  };


  return (
    <div
      ref={cardRef}
      className={className}
      data-issue-id={issue.id}
      role="button"
      tabIndex={0}
      aria-pressed={isFocused}
      onClick={handleSelect}
      onKeyDown={handleKeyDown}
      draggable={false}
      onDragStart={handleDragStartInternal}
      onDragEnd={handleDragEndInternal}
      onPointerDown={handleCardPointerDown}
    >
      <header className="issue-card-header">
        <div className="issue-card-meta">
          <span className="issue-priority">{issue.priority}</span>
        </div>
        <div className="issue-card-actions">
          <button
            type="button"
            className="issue-card-action issue-card-drag-handle"
            aria-label={`Drag ${issue.id}`}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
          >
            <GripVertical size={16} />
          </button>
          <button
            type="button"
            className="issue-card-action issue-card-action--archive"
            aria-label={`Archive ${issue.id}`}
            onClick={handleArchiveClick}
            onPointerDown={handleActionPointerDown}
            onTouchStart={handleActionTouchStart}
            disabled={isArchived}
          >
            <Archive size={16} />
          </button>
          <button
            type="button"
            className="issue-card-action issue-card-action--delete"
            aria-label={`Delete ${issue.id}`}
            onClick={handleDeleteClick}
            onPointerDown={handleActionPointerDown}
            onTouchStart={handleActionTouchStart}
          >
            <Trash2 size={16} />
          </button>
        </div>
      </header>
      <h3 className="issue-title" title={issue.title}>
        {issue.title}
      </h3>
      {isBlocked && (
        <div className="issue-blocked-indicator">
          <Lock size={14} className="issue-blocked-icon" />
          <div className="issue-blocked-details">
            <span className="issue-blocked-text">Waiting: targets in use</span>
            {blockedByIssues.length > 0 && (
              <span className="issue-blocked-by" title={`Blocked by: ${blockedByIssues.join(', ')}`}>
                {blockedByIssues.length} active {blockedByIssues.length === 1 ? 'issue' : 'issues'}
              </span>
            )}
          </div>
          <button
            type="button"
            className="issue-blocked-info"
            aria-label="Show blocking details"
            onClick={(e) => {
              e.stopPropagation();
              onSelect?.(issue.id);
            }}
            onPointerDown={handleActionPointerDown}
            onTouchStart={handleActionTouchStart}
          >
            <Info size={14} />
          </button>
        </div>
      )}
      {isRunning && (
        <div className="issue-running-indicator">
          <Brain size={14} className="issue-running-icon" />
          <div className="issue-running-details">
            <span className="issue-running-text">
              {runningStatusLabel}
              <span className="ellipsis-animated">...</span>
            </span>
            {runningProcess.duration && (
              <span className="issue-running-duration">{runningProcess.duration}</span>
            )}
          </div>
          <button
            type="button"
            className="issue-stop-button"
            aria-label="Stop agent"
            onClick={handleStopClick}
            onPointerDown={handleActionPointerDown}
            onTouchStart={handleActionTouchStart}
          >
            <StopCircle size={14} />
          </button>
        </div>
      )}
      {errorMessage && (
        <div className="issue-error-message">
          <AlertCircle size={14} />
          <span>{String(errorMessage)}</span>
        </div>
      )}
      <footer className="issue-footer">
        <span className="issue-meta">
          <Hash size={14} />
          {issue.id}
        </span>
        <span className="issue-meta">
          <CalendarClock size={14} />
          {formatDistanceToNow(issue.createdAt)}
        </span>
      </footer>
      {issue.tags.length > 0 && (
        <div className="issue-tags">
          <Tag size={14} />
          {issue.tags.map((tag) => (
            <span key={tag} className="issue-tag">
              {tag}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
