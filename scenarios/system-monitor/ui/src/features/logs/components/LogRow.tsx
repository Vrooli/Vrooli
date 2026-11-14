import type { LogEntry } from '../types';
import { PriorityBadge } from './PriorityBadge';

interface LogRowProps {
  entry: LogEntry;
  style?: React.CSSProperties;
  expanded?: boolean;
  onToggleExpanded?: () => void;
  onSelect?: () => void;
  selected?: boolean;
}

const LONG_MESSAGE_LENGTH = 180;

export const LogRow = ({ entry, style, expanded = false, onToggleExpanded, onSelect, selected }: LogRowProps) => {
  const date = entry.timestamp ? new Date(entry.timestamp) : null;
  const ts = date && Number.isFinite(date.getTime()) ? date.toISOString() : '';
  const timeLabel = ts ? `${ts.slice(11, 19)} UTC` : entry.timestamp ? 'Time unrecognized' : 'Time unknown';
  const ident = entry.unit || entry.userUnit || entry.identifier || '';
  const message = entry.message || entry.raw || '';
  const canExpand = message.length > LONG_MESSAGE_LENGTH;
  return (
    <div
      className={`log-row${selected ? ' is-selected' : ''}`}
      style={{
        ...style,
      }}
    >
      <time className="log-row__timestamp" dateTime={ts || undefined} title={entry.timestamp || undefined}>{timeLabel}</time>
      <span className="log-row__priority"><PriorityBadge priority={entry.priority} /></span>
      <span className="log-row__source" title={ident}>{ident || 'system'}</span>
      <span
        className={`log-row__message${expanded ? ' is-expanded' : ''}`}
        title={expanded ? undefined : message}
        role={expanded ? 'region' : undefined}
        aria-label={expanded ? 'Expanded log message' : undefined}
        tabIndex={expanded ? 0 : undefined}
      >
        {message}
      </span>
      <span className="log-row__actions">
      {canExpand && onToggleExpanded && (
        <button type="button" className="log-row__toggle" onClick={onToggleExpanded} aria-expanded={expanded} title={expanded ? 'Collapse message' : 'Expand message'}>
          {expanded ? 'Collapse message' : 'Expand message'}
        </button>
      )}
      {onSelect && <button type="button" className="log-row__select" onClick={onSelect} aria-pressed={selected} aria-label={`Read entry ${ts || (entry.timestamp ? 'time unrecognized' : 'time unknown')} ${ident || 'system'}`}>Read entry</button>}
      </span>
    </div>
  );
};
