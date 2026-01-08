import { useCallback, useState } from 'react';
import { ExternalLink, Trash2, Clock, Image as ImageIcon } from 'lucide-react';
import type { HistoryEntry } from '@/domains/recording';
import { InlineDeleteConfirmation } from './InlineDeleteConfirmation';

interface HistoryTableProps {
  entries: HistoryEntry[];
  deleting?: boolean;
  navigating?: boolean;
  onDelete?: (entryId: string) => Promise<boolean>;
  onNavigate?: (url: string) => Promise<boolean>;
}

/**
 * Format a timestamp as a relative time string (e.g., "2 hours ago").
 */
function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffDays > 0) {
    return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
  }
  if (diffHours > 0) {
    return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
  }
  if (diffMinutes > 0) {
    return `${diffMinutes} min${diffMinutes !== 1 ? 's' : ''} ago`;
  }
  return 'Just now';
}

/**
 * Format a timestamp as a full date string.
 */
function formatFullDate(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Truncate a URL for display.
 */
function truncateUrl(url: string, maxLength = 60): string {
  if (url.length <= maxLength) return url;
  return url.slice(0, maxLength - 3) + '...';
}

interface HistoryEntryRowProps {
  entry: HistoryEntry;
  deleting?: boolean;
  navigating?: boolean;
  onDelete?: (entryId: string) => Promise<boolean>;
  onNavigate?: (url: string) => Promise<boolean>;
}

function HistoryEntryRow({ entry, deleting, navigating, onDelete, onNavigate }: HistoryEntryRowProps) {
  const [confirmDelete, setConfirmDelete] = useState(false);

  const handleDelete = useCallback(async () => {
    if (onDelete) {
      await onDelete(entry.id);
    }
    setConfirmDelete(false);
  }, [entry.id, onDelete]);

  const handleNavigate = useCallback(async () => {
    if (onNavigate) {
      await onNavigate(entry.url);
    }
  }, [entry.url, onNavigate]);

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden hover:border-gray-300 dark:hover:border-gray-600 transition-colors">
      <div className="flex items-start p-3 gap-3">
        {/* Thumbnail */}
        <div className="flex-shrink-0 w-24 h-16 bg-gray-100 dark:bg-gray-800 rounded overflow-hidden">
          {entry.thumbnail ? (
            <img
              src={`data:image/jpeg;base64,${entry.thumbnail}`}
              alt=""
              className="w-full h-full object-cover"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-gray-400">
              <ImageIcon size={24} />
            </div>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Title */}
          <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate" title={entry.title}>
            {entry.title || '(No title)'}
          </h4>

          {/* URL */}
          <p
            className="text-xs text-gray-500 dark:text-gray-400 font-mono truncate mt-0.5"
            title={entry.url}
          >
            {truncateUrl(entry.url)}
          </p>

          {/* Timestamp */}
          <p className="text-xs text-gray-400 dark:text-gray-500 mt-1 flex items-center gap-1">
            <Clock size={12} />
            <span title={formatFullDate(entry.timestamp)}>{formatRelativeTime(entry.timestamp)}</span>
          </p>
        </div>

        {/* Actions */}
        <div className="flex-shrink-0 flex items-start gap-1">
          {onNavigate && (
            <button
              onClick={handleNavigate}
              disabled={navigating}
              className="p-1.5 text-gray-400 hover:text-blue-500 dark:hover:text-blue-400 disabled:opacity-50"
              title="Navigate to this URL"
            >
              <ExternalLink size={14} />
            </button>
          )}
          {onDelete && (
            <button
              onClick={() => setConfirmDelete(true)}
              disabled={deleting}
              className="p-1.5 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
              title="Delete this entry"
            >
              <Trash2 size={14} />
            </button>
          )}
        </div>
      </div>

      {/* Delete confirmation */}
      {confirmDelete && (
        <InlineDeleteConfirmation
          message="Delete this history entry?"
          deleting={deleting}
          onConfirm={handleDelete}
          onCancel={() => setConfirmDelete(false)}
        />
      )}
    </div>
  );
}

export function HistoryTable({ entries, deleting, navigating, onDelete, onNavigate }: HistoryTableProps) {
  if (!entries || entries.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        No navigation history yet.
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {entries.map((entry) => (
        <HistoryEntryRow
          key={entry.id}
          entry={entry}
          deleting={deleting}
          navigating={navigating}
          onDelete={onDelete}
          onNavigate={onNavigate}
        />
      ))}
    </div>
  );
}
