import { useState, useCallback } from 'react';
import { ChevronDown, ChevronRight, Globe, Database } from 'lucide-react';
import type { StorageStateOrigin } from '@/domains/recording';

interface LocalStorageTableProps {
  origins: StorageStateOrigin[];
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

function OriginSection({ origin }: { origin: StorageStateOrigin }) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [expandedItem, setExpandedItem] = useState<{ name: string; value: string } | null>(null);

  const toggleExpanded = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  const truncateValue = (value: string, maxLength = 100): string => {
    if (value.length <= maxLength) return value;
    return value.substring(0, maxLength) + '...';
  };

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
      <button
        onClick={toggleExpanded}
        className="w-full flex items-center gap-2 px-4 py-3 bg-gray-50 dark:bg-gray-800/50 hover:bg-gray-100 dark:hover:bg-gray-800 text-left"
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

      {isExpanded && (
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {origin.localStorage.map((item, index) => (
            <div
              key={`${item.name}-${index}`}
              className="px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/30"
            >
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 w-1/3">
                  <span className="font-medium text-sm text-gray-900 dark:text-gray-100 break-all">
                    {item.name}
                  </span>
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
              </div>
            </div>
          ))}
        </div>
      )}

      {expandedItem && (
        <ExpandedValueModal
          itemName={expandedItem.name}
          value={expandedItem.value}
          onClose={() => setExpandedItem(null)}
        />
      )}
    </div>
  );
}

export function LocalStorageTable({ origins }: LocalStorageTableProps) {
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
        <OriginSection key={`${origin.origin}-${index}`} origin={origin} />
      ))}
    </div>
  );
}
