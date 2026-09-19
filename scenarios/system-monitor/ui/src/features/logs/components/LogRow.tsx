import type { LogEntry } from '../types';
import { PriorityBadge } from './PriorityBadge';

interface LogRowProps {
  entry: LogEntry;
  style?: React.CSSProperties;
  expanded?: boolean;
  onToggleExpanded?: () => void;
}

const LONG_MESSAGE_LENGTH = 180;

export const LogRow = ({ entry, style, expanded = false, onToggleExpanded }: LogRowProps) => {
  const ts = entry.timestamp ? new Date(entry.timestamp).toISOString() : '';
  const ident = entry.unit || entry.userUnit || entry.identifier || '';
  const message = entry.message || entry.raw || '';
  const canExpand = message.length > LONG_MESSAGE_LENGTH;
  return (
    <div
      className="log-row"
      style={{
        ...style,
      }}
    >
      <time className="log-row__timestamp" dateTime={entry.timestamp || undefined}>{ts}</time>
      <span className="log-row__priority"><PriorityBadge priority={entry.priority} /></span>
      <span className="log-row__source" title={ident}>{ident || 'system'}</span>
      <span className={`log-row__message${expanded ? ' is-expanded' : ''}`} title={expanded ? undefined : message}>{message}</span>
      {canExpand && onToggleExpanded && (
        <button type="button" className="log-row__toggle" onClick={onToggleExpanded} aria-expanded={expanded}>
          {expanded ? 'Collapse message' : 'Expand message'}
        </button>
      )}
    </div>
  );
};
