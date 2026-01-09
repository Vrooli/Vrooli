import { Loader2, AlertCircle, Trash2 } from 'lucide-react';
import type { ReactNode } from 'react';

interface StorageSectionWrapperProps {
  /** Total count of items (used to determine if header should show) */
  count: number | undefined;
  /** Singular label for the item type (e.g., "cookie", "item", "entry") */
  itemLabel: string;
  /** Optional custom count label to override default "{count} {itemLabel}s" */
  countLabel?: string;
  /** Whether data is currently loading */
  loading: boolean;
  /** Error message if any */
  error: string | null;
  /** Message to show when there's no data */
  emptyMessage: string;
  /** Whether a delete operation is in progress */
  deleting: boolean;
  /** Called when "Clear All" is clicked */
  onClearAll: () => void;
  /** Custom label for the clear all button (default: "Clear All") */
  clearAllLabel?: string;
  /** Optional extra header content (e.g., Settings button for history) */
  extraHeaderContent?: ReactNode;
  /** The table/content to render when data is available */
  children: ReactNode;
}

export function StorageSectionWrapper({
  count,
  itemLabel,
  countLabel,
  loading,
  error,
  emptyMessage,
  deleting,
  onClearAll,
  clearAllLabel = 'Clear All',
  extraHeaderContent,
  children,
}: StorageSectionWrapperProps) {
  const hasData = count !== undefined && count > 0;
  const pluralLabel = count === 1 ? itemLabel : `${itemLabel}s`;
  const displayLabel = countLabel ?? `${count} ${pluralLabel}`;

  return (
    <div className="space-y-4">
      {/* Header with count and Clear All button */}
      {hasData && (
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
            {displayLabel}
          </h3>
          <div className="flex items-center gap-2">
            {extraHeaderContent}
            <button
              onClick={onClearAll}
              disabled={deleting}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-md hover:bg-red-100 dark:hover:bg-red-900/50 disabled:opacity-50"
            >
              <Trash2 size={14} />
              {clearAllLabel}
            </button>
          </div>
        </div>
      )}

      {/* Content area with loading/error/empty states */}
      <StorageContent loading={loading} error={error} emptyMessage={emptyMessage}>
        {children}
      </StorageContent>
    </div>
  );
}

/** Internal wrapper for storage content with loading/error states */
function StorageContent({
  loading,
  error,
  emptyMessage,
  children,
}: {
  loading: boolean;
  error: string | null;
  emptyMessage: string;
  children: ReactNode;
}) {
  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 size={24} className="animate-spin text-blue-500" />
        <span className="ml-2 text-gray-500 dark:text-gray-400">Loading storage data...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 p-3 rounded-lg bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800">
        <AlertCircle size={18} className="text-red-600 dark:text-red-400 flex-shrink-0" />
        <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
      </div>
    );
  }

  if (!children) {
    return <p className="text-sm text-gray-500 dark:text-gray-400 py-8 text-center">{emptyMessage}</p>;
  }

  return <>{children}</>;
}
