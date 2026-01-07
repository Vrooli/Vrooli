import { useCallback, useState } from 'react';
import { Star, Trash2, Globe } from 'lucide-react';
import type { TabInfo } from '@/domains/recording/hooks/useTabs';

interface TabsTableProps {
  tabs: TabInfo[];
  deleting?: boolean;
  onDelete?: (order: number) => Promise<boolean>;
}

/**
 * Get the domain from a URL for display.
 */
function getDomain(url: string): string {
  try {
    const urlObj = new URL(url);
    return urlObj.hostname;
  } catch {
    return url;
  }
}

/**
 * Get a color for the favicon placeholder based on the domain.
 */
function getPlaceholderColor(domain: string): string {
  const colors = [
    'bg-blue-500',
    'bg-green-500',
    'bg-purple-500',
    'bg-orange-500',
    'bg-pink-500',
    'bg-teal-500',
    'bg-indigo-500',
    'bg-red-500',
  ];
  let hash = 0;
  for (let i = 0; i < domain.length; i++) {
    hash = domain.charCodeAt(i) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length];
}

/**
 * Get the first letter of the domain for the favicon placeholder.
 */
function getFirstLetter(domain: string): string {
  // Skip common prefixes like www
  const clean = domain.replace(/^www\./, '');
  return clean.charAt(0).toUpperCase();
}

interface TabRowProps {
  tab: TabInfo;
  deleting?: boolean;
  onDelete?: (order: number) => Promise<boolean>;
}

function TabRow({ tab, deleting, onDelete }: TabRowProps) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const domain = getDomain(tab.url);
  const placeholderColor = getPlaceholderColor(domain);
  const firstLetter = getFirstLetter(domain);

  const handleDelete = useCallback(async () => {
    if (onDelete) {
      await onDelete(tab.order);
    }
    setConfirmDelete(false);
  }, [tab.order, onDelete]);

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden hover:border-gray-300 dark:hover:border-gray-600 transition-colors">
      <div className="flex items-center p-3 gap-3">
        {/* Favicon placeholder */}
        <div
          className={`flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center text-white text-sm font-semibold ${placeholderColor}`}
        >
          {firstLetter}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Domain with active indicator */}
          <div className="flex items-center gap-2">
            {tab.isActive && (
              <span title="Active tab">
                <Star size={14} className="flex-shrink-0 text-yellow-500 fill-yellow-500" />
              </span>
            )}
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate" title={tab.url}>
              {domain}
            </p>
          </div>

          {/* Title */}
          {tab.title && (
            <p className="text-xs text-gray-500 dark:text-gray-400 truncate mt-0.5" title={tab.title}>
              {tab.title}
            </p>
          )}
        </div>

        {/* Actions */}
        <div className="flex-shrink-0">
          {onDelete && (
            <button
              onClick={() => setConfirmDelete(true)}
              disabled={deleting}
              className="p-1.5 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
              title="Delete this tab"
            >
              <Trash2 size={14} />
            </button>
          )}
        </div>
      </div>

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="px-3 py-2 bg-red-50 dark:bg-red-900/30 border-t border-red-200 dark:border-red-800">
          <div className="flex items-center justify-between">
            <p className="text-sm text-red-700 dark:text-red-300">Remove this tab?</p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setConfirmDelete(false)}
                className="px-2 py-1 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="px-2 py-1 text-xs font-medium text-white bg-red-600 rounded hover:bg-red-700 disabled:opacity-50"
              >
                {deleting ? 'Removing...' : 'Remove'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export function TabsTable({ tabs, deleting, onDelete }: TabsTableProps) {
  if (!tabs || tabs.length === 0) {
    return (
      <div className="py-8 text-center">
        <Globe size={32} className="mx-auto text-gray-300 dark:text-gray-600 mb-3" />
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-1">No saved tabs</p>
        <p className="text-xs text-gray-400 dark:text-gray-500">
          Start a recording session and close it to save your open tabs.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {tabs.map((tab) => (
        <TabRow key={tab.order} tab={tab} deleting={deleting} onDelete={onDelete} />
      ))}
    </div>
  );
}
