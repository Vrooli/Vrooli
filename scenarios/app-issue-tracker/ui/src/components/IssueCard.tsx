import { useRef } from 'react';
import type { DragEvent, KeyboardEventHandler, MouseEvent, PointerEvent, TouchEvent } from 'react';
import { Archive, CalendarClock, GripVertical, Hash, Tag, Trash2 } from 'lucide-react';
import { Issue } from '../data/sampleData';
import { formatDistanceToNow } from '../utils/date';

const TOUCH_MOVE_THRESHOLD = 14;

interface IssueCardProps {
  issue: Issue;
  isFocused?: boolean;
  onSelect?: (issueId: string) => void;
  onDelete?: (issue: Issue) => void;
  onArchive?: (issue: Issue) => void;
  draggable?: boolean;
  onDragStart?: (issue: Issue, event: DragEvent<HTMLDivElement>) => void;
  onDragEnd?: (issue: Issue, event: DragEvent<HTMLDivElement>) => void;
}

export function IssueCard({
  issue,
  isFocused = false,
  onSelect,
  onDelete,
  onArchive,
  draggable = true,
  onDragStart,
  onDragEnd,
}: IssueCardProps) {
  const touchStateRef = useRef<{ x: number; y: number; moved: boolean }>({
    x: 0,
    y: 0,
    moved: false,
  });
  const ignoreClickRef = useRef(false);
  const dragHandleActiveRef = useRef(false);

  const className = [
    'issue-card',
    `priority-${issue.priority.toLowerCase()}`,
    isFocused ? 'issue-card--focused' : '',
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

  const handlePointerUp = () => {
    dragHandleActiveRef.current = false;
  };

  const handlePointerCancel = () => {
    dragHandleActiveRef.current = false;
  };

  const handleDragHandlePointerDown = (event: PointerEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    dragHandleActiveRef.current = true;
  };

  const handleDragHandlePointerUp = (event: PointerEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    dragHandleActiveRef.current = false;
  };

  const handleDragHandlePointerLeave = (event: PointerEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    dragHandleActiveRef.current = false;
  };

  const handleTouchStart = (event: TouchEvent<HTMLDivElement>) => {
    const touch = event.touches[0];
    if (!touch) {
      return;
    }
    touchStateRef.current = { x: touch.clientX, y: touch.clientY, moved: false };
  };

  const handleTouchMove = (event: TouchEvent<HTMLDivElement>) => {
    const touch = event.touches[0];
    if (!touch) {
      return;
    }
    const dx = touch.clientX - touchStateRef.current.x;
    const dy = touch.clientY - touchStateRef.current.y;
    if (!touchStateRef.current.moved) {
      const distance = Math.sqrt(dx * dx + dy * dy);
      if (distance > TOUCH_MOVE_THRESHOLD) {
        touchStateRef.current.moved = true;
      }
    }
  };

  const handleTouchEnd = (event: TouchEvent<HTMLDivElement>) => {
    const target = event.target as HTMLElement | null;
    if (target?.closest('.issue-card-action')) {
      ignoreClickRef.current = true;
      window.setTimeout(() => {
        ignoreClickRef.current = false;
      }, 0);
      touchStateRef.current = { x: 0, y: 0, moved: false };
      return;
    }

    const moved = touchStateRef.current.moved;
    ignoreClickRef.current = true;
    window.setTimeout(() => {
      ignoreClickRef.current = false;
    }, 0);

    if (!moved && !target?.closest('.issue-card-drag-handle')) {
      onSelect?.(issue.id);
    }

    touchStateRef.current = { x: 0, y: 0, moved: false };
  };

  const summaryText = issue.summary?.trim();
  const appLabel = issue.app?.trim();
  const isArchived = issue.status === 'archived';

  return (
    <div
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
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerCancel}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onTouchCancel={handleTouchEnd}
    >
      <header className="issue-card-header">
        <div className="issue-card-meta">
          <span className="issue-priority">{issue.priority}</span>
          <span className="issue-assignee">{issue.assignee}</span>
        </div>
        <div className="issue-card-actions">
          <button
            type="button"
            className="issue-card-action issue-card-drag-handle"
            aria-label={`Drag ${issue.id}`}
            onPointerDown={handleDragHandlePointerDown}
            onPointerUp={handleDragHandlePointerUp}
            onPointerLeave={handleDragHandlePointerLeave}
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
      {summaryText && <p className="issue-summary">{summaryText}</p>}
      {appLabel && (
        <div className="issue-flags">
          <span className="issue-app-tag">{appLabel}</span>
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
