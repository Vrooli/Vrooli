import { useRef } from 'react';
import type { DragEvent, KeyboardEventHandler, MouseEvent, PointerEvent, TouchEvent } from 'react';
import { Archive, Brain, CalendarClock, GripVertical, Hash, Tag, Trash2, AlertCircle } from 'lucide-react';
import { Issue } from '../data/sampleData';
import { formatDistanceToNow } from '../utils/date';

const TOUCH_MOVE_THRESHOLD = 14;

interface IssueCardProps {
  issue: Issue;
  isFocused?: boolean;
  runningProcess?: { agent_id: string; start_time: string; duration?: string };
  onSelect?: (issueId: string) => void;
  onDelete?: (issue: Issue) => void;
  onArchive?: (issue: Issue) => void;
  draggable?: boolean;
  onDragStart?: (issue: Issue, event: DragEvent<HTMLDivElement>) => void;
  onDragEnd?: (issue: Issue, event: DragEvent<HTMLDivElement>) => void;
  onPointerDown?: (issue: Issue, event: PointerEvent<HTMLDivElement>, card: HTMLElement) => void;
  onPointerMove?: (event: PointerEvent<HTMLDivElement>, card: HTMLElement) => void;
  onPointerUp?: (event: PointerEvent<HTMLDivElement>, card: HTMLElement) => void;
}

export function IssueCard({
  issue,
  isFocused = false,
  runningProcess,
  onSelect,
  onDelete,
  onArchive,
  draggable = true,
  onDragStart,
  onDragEnd,
  onPointerDown,
  onPointerMove,
  onPointerUp,
}: IssueCardProps) {
  const ignoreClickRef = useRef(false);
  const dragHandleActiveRef = useRef(false);
  const cardRef = useRef<HTMLDivElement>(null);

  const isRunning = !!runningProcess;
  const isFailed = issue.status === 'failed';
  const errorMessage = isFailed && issue.metadata?.extra?.agent_last_error;

  const className = [
    'issue-card',
    `priority-${issue.priority.toLowerCase()}`,
    isFocused ? 'issue-card--focused' : '',
    isRunning ? 'issue-card--running' : '',
  ]
    .filter(Boolean)
    .join(' ');

  const handleSelect = () => {
    if (ignoreClickRef.current) {
      ignoreClickRef.current = false;
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

  const handleDragStartInternal = (event: DragEvent<HTMLDivElement>) => {
    if (!dragHandleActiveRef.current) {
      event.preventDefault();
      return;
    }
    onDragStart?.(issue, event);
  };

  const handleDragEndInternal = (event: DragEvent<HTMLDivElement>) => {
    dragHandleActiveRef.current = false;
    onDragEnd?.(issue, event);
  };

  const handleCardPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement | null;
    const handle = target?.closest('.issue-card-drag-handle');

    if (!handle || !cardRef.current) {
      return;
    }

    event.stopPropagation();
    onPointerDown?.(issue, event, cardRef.current);
  };

  const handleCardPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (!cardRef.current) {
      return;
    }
    onPointerMove?.(event, cardRef.current);
  };

  const handleCardPointerUp = (event: PointerEvent<HTMLDivElement>) => {
    if (!cardRef.current) {
      return;
    }
    onPointerUp?.(event, cardRef.current);
  };

  const isArchived = issue.status === 'archived';

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
      draggable={draggable}
      onDragStart={handleDragStartInternal}
      onDragEnd={handleDragEndInternal}
      onPointerDown={handleCardPointerDown}
      onPointerMove={handleCardPointerMove}
      onPointerUp={handleCardPointerUp}
      onPointerCancel={handleCardPointerUp}
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
          >
            <GripVertical size={16} />
          </button>
          <button
            type="button"
            className="issue-card-action issue-card-action--archive"
            aria-label={`Archive ${issue.id}`}
            onClick={handleArchiveClick}
            disabled={isArchived}
          >
            <Archive size={16} />
          </button>
          <button
            type="button"
            className="issue-card-action issue-card-action--delete"
            aria-label={`Delete ${issue.id}`}
            onClick={handleDeleteClick}
          >
            <Trash2 size={16} />
          </button>
        </div>
      </header>
      <h3 className="issue-title">{issue.title}</h3>
      {isRunning && (
        <div className="issue-running-indicator">
          <Brain size={14} className="issue-running-icon" />
          <div className="issue-running-details">
            <span className="issue-running-text">Executing with {runningProcess.agent_id}</span>
            {runningProcess.duration && (
              <span className="issue-running-duration">{runningProcess.duration}</span>
            )}
          </div>
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
