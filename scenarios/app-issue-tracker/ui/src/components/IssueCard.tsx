import type { KeyboardEventHandler } from 'react';
import { CalendarClock, Hash, Tag } from 'lucide-react';
import { Issue } from '../data/sampleData';
import { formatDistanceToNow } from '../utils/date';

interface IssueCardProps {
  issue: Issue;
  isFocused?: boolean;
  onSelect?: (issueId: string) => void;
}

export function IssueCard({ issue, isFocused = false, onSelect }: IssueCardProps) {
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

  return (
    <div
      className={className}
      data-issue-id={issue.id}
      role="button"
      tabIndex={0}
      aria-pressed={isFocused}
      onClick={handleSelect}
      onKeyDown={handleKeyDown}
    >
      <header className="issue-card-header">
        <span className="issue-priority">{issue.priority}</span>
        <span className="issue-assignee">{issue.assignee}</span>
      </header>
      <h3 className="issue-title">{issue.title}</h3>
      <p className="issue-summary">{issue.summary}</p>
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
