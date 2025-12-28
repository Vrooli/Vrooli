import { useState, useCallback } from 'react';
import { ChevronDown, ChevronRight, Globe, Database, Trash2 } from 'lucide-react';
import type { StorageStateOrigin } from '@/domains/recording';

interface LocalStorageTableProps {
  origins: StorageStateOrigin[];
  deleting?: boolean;
  onDeleteOrigin?: (origin: string) => Promise<boolean>;
  onDeleteItem?: (origin: string, name: string) => Promise<boolean>;
}

interface ExpandedValueModalProps {
  itemName: string;
  value: string;
  onClose: () => void;
}

function ExpandedValueModal({ itemName, value, onClose }: ExpandedValueModalProps) {
  // Try to format as JSON if possible
  let formattedValue = value;
  try {
    const parsed = JSON.parse(value);
    formattedValue = JSON.stringify(parsed, null, 2);
  } catch {
    // Not JSON, keep as is
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
          <h3 className="font-medium text-gray-900 dark:text-gray-100">{itemName}</h3>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            Close
          </button>
        </div>
        <div className="flex-1 overflow-auto p-4">
          <pre className="text-sm font-mono text-gray-700 dark:text-gray-300 whitespace-pre-wrap break-all">
            {formattedValue}
          </pre>
        </div>
      </div>
    </div>
  );
}

interface OriginSectionProps {
  origin: StorageStateOrigin;
  deleting?: boolean;
  onDeleteOrigin?: (origin: string) => Promise<boolean>;
  onDeleteItem?: (origin: string, name: string) => Promise<boolean>;
}

function OriginSection({ origin, deleting, onDeleteOrigin, onDeleteItem }: OriginSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [expandedItem, setExpandedItem] = useState<{ name: string; value: string } | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<'origin' | string | null>(null);

  const toggleExpanded = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  const truncateValue = (value: string, maxLength = 100): string => {
    if (value.length <= maxLength) return value;
    return value.substring(0, maxLength) + '...';
  };

  const handleDeleteOrigin = useCallback(async () => {
    if (onDeleteOrigin) {
      await onDeleteOrigin(origin.origin);
    }
    setConfirmDelete(null);
  }, [origin.origin, onDeleteOrigin]);

  const handleDeleteItem = useCallback(
    async (itemName: string) => {
      if (onDeleteItem) {
        await onDeleteItem(origin.origin, itemName);
      }
      setConfirmDelete(null);
    },
    [origin.origin, onDeleteItem]
  );

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
      {/* Origin header */}
      <div className="flex items-center bg-gray-50 dark:bg-gray-800/50">
        <button
          onClick={toggleExpanded}
          className="flex-1 flex items-center gap-2 px-4 py-3 hover:bg-gray-100 dark:hover:bg-gray-800 text-left"
        >
          {isExpanded ? (
            <ChevronDown size={16} className="text-gray-500" />
          ) : (
            <ChevronRight size={16} className="text-gray-500" />
          )}
          <Globe size={14} className="text-gray-500" />
          <span className="font-medium text-gray-900 dark:text-gray-100 font-mono text-sm">{origin.origin}</span>
          <span className="ml-auto text-xs text-gray-500 dark:text-gray-400">
            {origin.localStorage.length} item{origin.localStorage.length !== 1 ? 's' : ''}
          </span>
        </button>
        {onDeleteOrigin && (
          <button
            onClick={() => setConfirmDelete('origin')}
            disabled={deleting}
            className="p-2 mr-2 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
            title={`Delete all items for ${origin.origin}`}
          >
            <Trash2 size={14} />
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {origin.localStorage.map((item, index) => (
            <div key={`${item.name}-${index}`} className="px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/30">
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 w-1/3">
                  <span className="font-medium text-sm text-gray-900 dark:text-gray-100 break-all">{item.name}</span>
                </div>
                <div className="flex-1 min-w-0">
                  <button
                    onClick={() => setExpandedItem({ name: item.name, value: item.value })}
                    className="text-left w-full"
                  >
                    <span
                      className="text-sm font-mono text-gray-600 dark:text-gray-400 break-all hover:text-blue-600 dark:hover:text-blue-400"
                      title="Click to expand"
                    >
                      {truncateValue(item.value)}
                    </span>
                  </button>
                </div>
                {onDeleteItem && (
                  <button
                    onClick={() => setConfirmDelete(item.name)}
                    disabled={deleting}
                    className="flex-shrink-0 p-1.5 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
                    title={`Delete ${item.name}`}
                  >
                    <Trash2 size={12} />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Delete confirmation inline */}
      {confirmDelete && (
        <div className="px-4 py-3 bg-red-50 dark:bg-red-900/30 border-t border-red-200 dark:border-red-800">
          <div className="flex items-center justify-between">
            <p className="text-sm text-red-700 dark:text-red-300">
              {confirmDelete === 'origin'
                ? `Delete all ${origin.localStorage.length} items for ${origin.origin}?`
                : `Delete item "${confirmDelete}"?`}
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setConfirmDelete(null)}
                className="px-3 py-1 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={() => (confirmDelete === 'origin' ? handleDeleteOrigin() : handleDeleteItem(confirmDelete))}
                disabled={deleting}
                className="px-3 py-1 text-xs font-medium text-white bg-red-600 rounded hover:bg-red-700 disabled:opacity-50"
              >
                {deleting ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}

      {expandedItem && (
        <ExpandedValueModal itemName={expandedItem.name} value={expandedItem.value} onClose={() => setExpandedItem(null)} />
      )}
    </div>
  );
}

export function LocalStorageTable({ origins, deleting, onDeleteOrigin, onDeleteItem }: LocalStorageTableProps) {
  const nonEmptyOrigins = origins.filter((o) => o.localStorage.length > 0);

  if (nonEmptyOrigins.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        <Database size={24} className="mx-auto mb-2 opacity-50" />
        No localStorage data stored in this session.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {nonEmptyOrigins.map((origin, index) => (
        <OriginSection
          key={`${origin.origin}-${index}`}
          origin={origin}
          deleting={deleting}
          onDeleteOrigin={onDeleteOrigin}
          onDeleteItem={onDeleteItem}
        />
      ))}
    </div>
  );
}
