import { Cookie, Database, Cog, History, Layers, type LucideIcon } from 'lucide-react';
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

interface StatItem {
  icon: LucideIcon;
  text: string;
  className?: string;
}

function StatBadge({ icon: Icon, text, className }: StatItem) {
  return (
    <span className={`flex items-center gap-1 ${className ?? ''}`}>
      <Icon size={12} />
      {text}
    </span>
  );
}

function pluralize(count: number, singular: string, plural?: string): string {
  return count === 1 ? singular : (plural ?? `${singular}s`);
}

export function SessionFooterStats({
  activeSection,
  storageState,
  serviceWorkers,
  history,
  tabsCount,
  hasActiveSession,
}: SessionFooterStatsProps) {
  // Storage sections (cookies & localStorage) share the same stats
  if ((activeSection === 'cookies' || activeSection === 'local-storage') && storageState) {
    const { cookieCount, localStorageCount, originCount } = storageState.stats;
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <StatBadge icon={Cookie} text={`${cookieCount} ${pluralize(cookieCount, 'cookie')}`} />
        <StatBadge icon={Database} text={`${localStorageCount} ${pluralize(localStorageCount, 'item')}`} />
        <span className="text-gray-400 dark:text-gray-500">
          across {originCount} {pluralize(originCount, 'origin')}
        </span>
      </div>
    );
  }

  // Service Workers section
  if (activeSection === 'service-workers') {
    const workerCount = serviceWorkers?.workers?.length ?? 0;
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <StatBadge icon={Cog} text={`${workerCount} service ${pluralize(workerCount, 'worker')}`} />
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
    const { totalEntries } = history.stats;
    const { maxEntries, retentionDays } = history.settings;
    const ttlText = retentionDays > 0 ? `${retentionDays}d` : 'unlimited';
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <StatBadge icon={History} text={`${totalEntries} ${pluralize(totalEntries, 'entry', 'entries')}`} />
        <span className="text-gray-400 dark:text-gray-500">
          Max: {maxEntries} | TTL: {ttlText}
        </span>
      </div>
    );
  }

  // Tabs section
  if (activeSection === 'tabs') {
    return (
      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <StatBadge icon={Layers} text={`${tabsCount} ${pluralize(tabsCount, 'tab')} saved`} />
        <span className="text-gray-400 dark:text-gray-500">Will restore on session start</span>
      </div>
    );
  }

  // Settings sections or unknown - return empty placeholder
  return <div />;
}
