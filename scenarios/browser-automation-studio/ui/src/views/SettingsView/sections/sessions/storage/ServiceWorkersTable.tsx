import { useState, useCallback } from 'react';
import { ChevronDown, ChevronRight, Globe, Trash2, Cog } from 'lucide-react';
import type { ServiceWorkerInfo } from '@/domains/recording';

interface ServiceWorkersTableProps {
  workers: ServiceWorkerInfo[];
  controlMode?: string;
  hasActiveSession: boolean;
  deleting?: boolean;
  onUnregister?: (scopeURL: string) => Promise<boolean>;
}

interface ScopeGroupProps {
  scopeURL: string;
  worker: ServiceWorkerInfo;
  hasActiveSession: boolean;
  deleting?: boolean;
  onUnregister?: (scopeURL: string) => Promise<boolean>;
}

function getStatusBadgeClasses(status: string): string {
  switch (status) {
    case 'running':
      return 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300';
    case 'activating':
      return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300';
    case 'installed':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300';
    case 'stopped':
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400';
  }
}

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return url;
  }
}

function ScopeGroup({ scopeURL, worker, hasActiveSession, deleting, onUnregister }: ScopeGroupProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const toggleExpanded = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  const handleUnregister = useCallback(async () => {
    if (onUnregister) {
      await onUnregister(scopeURL);
    }
    setConfirmDelete(false);
  }, [scopeURL, onUnregister]);

  const domain = extractDomain(scopeURL);

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
      {/* Scope header */}
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
          <span className="font-medium text-gray-900 dark:text-gray-100 font-mono text-sm">{domain}</span>
          <span
            className={`ml-2 px-1.5 py-0.5 text-[10px] font-medium rounded ${getStatusBadgeClasses(worker.status)}`}
          >
            {worker.status}
          </span>
        </button>
        {onUnregister && hasActiveSession && (
          <button
            onClick={() => setConfirmDelete(true)}
            disabled={deleting}
            className="p-2 mr-2 text-gray-400 hover:text-red-500 dark:hover:text-red-400 disabled:opacity-50"
            title={`Unregister service worker for ${domain}`}
          >
            <Trash2 size={14} />
          </button>
        )}
      </div>

      {/* Service worker details */}
      {isExpanded && (
        <div className="px-4 py-3 text-sm border-t border-gray-100 dark:border-gray-800">
          <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
            <span className="text-gray-500 dark:text-gray-400">Scope URL:</span>
            <span className="font-mono text-xs text-gray-700 dark:text-gray-300 break-all">{scopeURL}</span>

            <span className="text-gray-500 dark:text-gray-400">Script URL:</span>
            <span className="font-mono text-xs text-gray-700 dark:text-gray-300 break-all">{worker.scriptURL}</span>

            <span className="text-gray-500 dark:text-gray-400">Status:</span>
            <span className={`inline-flex w-fit px-1.5 py-0.5 text-[10px] font-medium rounded ${getStatusBadgeClasses(worker.status)}`}>
              {worker.status}
            </span>

            {worker.versionId && (
              <>
                <span className="text-gray-500 dark:text-gray-400">Version:</span>
                <span className="font-mono text-xs text-gray-600 dark:text-gray-400">{worker.versionId}</span>
              </>
            )}
          </div>
        </div>
      )}

      {/* Delete confirmation inline */}
      {confirmDelete && (
        <div className="px-4 py-3 bg-red-50 dark:bg-red-900/30 border-t border-red-200 dark:border-red-800">
          <div className="flex items-center justify-between">
            <p className="text-sm text-red-700 dark:text-red-300">
              Unregister service worker for {domain}?
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setConfirmDelete(false)}
                className="px-3 py-1 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleUnregister}
                disabled={deleting}
                className="px-3 py-1 text-xs font-medium text-white bg-red-600 rounded hover:bg-red-700 disabled:opacity-50"
              >
                {deleting ? 'Unregistering...' : 'Unregister'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export function ServiceWorkersTable({
  workers,
  controlMode = 'allow',
  hasActiveSession,
  deleting,
  onUnregister,
}: ServiceWorkersTableProps) {
  if (!hasActiveSession) {
    return (
      <div className="py-8 text-center">
        <Cog size={24} className="mx-auto mb-2 text-gray-400 opacity-50" />
        <p className="text-sm text-gray-500 dark:text-gray-400">
          No active session for this profile.
        </p>
        <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
          Start a recording session to view and manage service workers.
        </p>
      </div>
    );
  }

  if (workers.length === 0) {
    return (
      <div className="py-8 text-center">
        <Cog size={24} className="mx-auto mb-2 text-gray-400 opacity-50" />
        <p className="text-sm text-gray-500 dark:text-gray-400">
          No service workers registered.
        </p>
        {controlMode === 'block' && (
          <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">
            Service workers are currently blocked for this session.
          </p>
        )}
      </div>
    );
  }

  // Sort workers by scope URL
  const sortedWorkers = [...workers].sort((a, b) => a.scopeURL.localeCompare(b.scopeURL));

  return (
    <div className="space-y-3">
      {sortedWorkers.map((worker) => (
        <ScopeGroup
          key={worker.registrationId}
          scopeURL={worker.scopeURL}
          worker={worker}
          hasActiveSession={hasActiveSession}
          deleting={deleting}
          onUnregister={onUnregister}
        />
      ))}
    </div>
  );
}
