import { Cookie, Database, Cog, History, Layers } from 'lucide-react';
import type { SectionId } from './SessionSidebar';
import type { StorageStateResponse, HistoryResponse } from '@/domains/recording/types/types';

interface ServiceWorkersData {
  workers: unknown[];
}

interface SessionFooterStatsProps {
  activeSection: SectionId;
  storageState: StorageStateResponse | null;
  serviceWorkers: ServiceWorkersData | null;
  history: HistoryResponse | null;
  tabsCount: number;
  hasActiveSession: boolean;
}

export function SessionFooterStats({
  activeSection,
  storageState,
  serviceWorkers,
  history,
  tabsCount,
  hasActiveSession,
}: SessionFooterStatsProps) {
  // Cookies & LocalStorage sections share the same stats
  if ((activeSection === 'cookies' || activeSection === 'local-storage') && storageState) {
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <span className="flex items-center gap-1">
          <Cookie size={12} />
          {storageState.stats.cookieCount} cookie{storageState.stats.cookieCount !== 1 ? 's' : ''}
        </span>
        <span className="flex items-center gap-1">
          <Database size={12} />
          {storageState.stats.localStorageCount} item
          {storageState.stats.localStorageCount !== 1 ? 's' : ''}
        </span>
        <span className="text-gray-400 dark:text-gray-500">
          across {storageState.stats.originCount} origin
          {storageState.stats.originCount !== 1 ? 's' : ''}
        </span>
      </div>
    );
  }

  // Service Workers section
  if (activeSection === 'service-workers') {
    const workerCount = serviceWorkers?.workers?.length ?? 0;
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <span className="flex items-center gap-1">
          <Cog size={12} />
          {workerCount} service worker{workerCount !== 1 ? 's' : ''}
        </span>
        {hasActiveSession ? (
          <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
            Active session
          </span>
        ) : (
          <span className="text-gray-400 dark:text-gray-500">No active session</span>
        )}
      </div>
    );
  }

  // History section
  if (activeSection === 'history' && history) {
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <span className="flex items-center gap-1">
          <History size={12} />
          {history.stats.totalEntries} entr{history.stats.totalEntries !== 1 ? 'ies' : 'y'}
        </span>
        <span className="text-gray-400 dark:text-gray-500">
          Max: {history.settings.maxEntries} | TTL:{' '}
          {history.settings.retentionDays > 0 ? `${history.settings.retentionDays}d` : 'unlimited'}
        </span>
      </div>
    );
  }

  // Tabs section
  if (activeSection === 'tabs') {
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <span className="flex items-center gap-1">
          <Layers size={12} />
          {tabsCount} tab{tabsCount !== 1 ? 's' : ''} saved
        </span>
        <span className="text-gray-400 dark:text-gray-500">Will restore on session start</span>
      </div>
    );
  }

  // Settings sections or unknown - return empty placeholder
  return <div />;
}
