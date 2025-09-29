import type { DragEvent, KeyboardEventHandler, MouseEvent } from 'react';
import { Archive, CalendarClock, Hash, Tag, Trash2 } from 'lucide-react';
import { Issue } from '../data/sampleData';
import { formatDistanceToNow } from '../utils/date';

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
  const className = [
    'issue-card',
    `priority-${issue.priority.toLowerCase()}`,
    isFocused ? 'issue-card--focused' : '',
  ]
    .filter(Boolean)
    .join(' ');

  const handleSelect = () => {
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
    onDragStart?.(issue, event);
  };

  const handleDragEndInternal = (event: DragEvent<HTMLDivElement>) => {
    onDragEnd?.(issue, event);
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
    >
      <header className="issue-card-header">
        <div className="issue-card-meta">
          <span className="issue-priority">{issue.priority}</span>
          <span className="issue-assignee">{issue.assignee}</span>
        </div>
        <div className="issue-card-actions">
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
