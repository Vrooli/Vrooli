import { useMemo } from 'react';
import { useStorageState, useServiceWorkers, useHistory } from '@/domains/recording';
import { useTabs } from '@/domains/recording/hooks/useTabs';
import type {
  StorageStateResponse,
  HistoryResponse,
  HistorySettings,
} from '@/domains/recording/types/types';
import type { ServiceWorkersResponse, ServiceWorkerInfo } from '@/domains/recording/hooks/useServiceWorkers';
import type { TabInfo } from '@/domains/recording/hooks/useTabs';

export interface UseProfileBoundResourcesProps {
  /** Profile ID to bind all resource operations to */
  profileId: string;
}

/**
 * Storage resource with profileId-bound operations
 */
export interface BoundStorageResource {
  state: StorageStateResponse | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetch: () => Promise<void>;
  clear: () => void;
  clearAllCookies: () => Promise<boolean>;
  deleteCookiesByDomain: (domain: string) => Promise<boolean>;
  deleteCookie: (domain: string, name: string) => Promise<boolean>;
  clearAllLocalStorage: () => Promise<boolean>;
  deleteLocalStorageByOrigin: (origin: string) => Promise<boolean>;
  deleteLocalStorageItem: (origin: string, name: string) => Promise<boolean>;
}

/**
 * Service workers resource with profileId-bound operations
 */
export interface BoundServiceWorkersResource {
  data: ServiceWorkersResponse | null;
  workers: ServiceWorkerInfo[];
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetch: () => Promise<void>;
  clear: () => void;
  unregisterAll: () => Promise<boolean>;
  unregister: (scopeURL: string) => Promise<boolean>;
}

/**
 * History resource with profileId-bound operations
 */
export interface BoundHistoryResource {
  data: HistoryResponse | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
  navigating: boolean;
  fetch: () => Promise<void>;
  clear: () => void;
  clearAll: () => Promise<boolean>;
  deleteEntry: (entryId: string) => Promise<boolean>;
  updateSettings: (settings: Partial<HistorySettings>) => Promise<boolean>;
  navigateTo: (url: string) => Promise<boolean>;
}

/**
 * Tabs resource with profileId-bound operations
 */
export interface BoundTabsResource {
  data: TabInfo[];
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetch: () => Promise<void>;
  clear: () => void;
  clearAll: () => Promise<boolean>;
  delete: (order: number) => Promise<boolean>;
}

export interface UseProfileBoundResourcesReturn {
  storage: BoundStorageResource;
  serviceWorkers: BoundServiceWorkersResource;
  history: BoundHistoryResource;
  tabs: BoundTabsResource;
  /** Whether there's an active browser session (service workers has session_id) */
  hasActiveSession: boolean;
}

/**
 * Wraps existing resource hooks and binds profileId to all operations.
 *
 * Responsibilities:
 * - Compose useStorageState, useServiceWorkers, useHistory, useTabs
 * - Bind profileId to all callbacks (no more passing profileId at call sites)
 * - Provide namespaced access to each resource
 * - Derive cross-resource values (hasActiveSession)
 */
export function useProfileBoundResources({
  profileId,
}: UseProfileBoundResourcesProps): UseProfileBoundResourcesReturn {
  // Compose underlying hooks
  const storageHook = useStorageState();
  const serviceWorkersHook = useServiceWorkers();
  const historyHook = useHistory();
  const tabsHook = useTabs();

  // Derive hasActiveSession from service workers data
  const hasActiveSession = !!serviceWorkersHook.serviceWorkers?.session_id;

  // Create bound storage resource
  const storage = useMemo<BoundStorageResource>(
    () => ({
      state: storageHook.storageState,
      loading: storageHook.loading,
      error: storageHook.error,
      deleting: storageHook.deleting,
      fetch: () => storageHook.fetchStorageState(profileId),
      clear: storageHook.clear,
      clearAllCookies: () => storageHook.clearAllCookies(profileId),
      deleteCookiesByDomain: (domain: string) => storageHook.deleteCookiesByDomain(profileId, domain),
      deleteCookie: (domain: string, name: string) => storageHook.deleteCookie(profileId, domain, name),
      clearAllLocalStorage: () => storageHook.clearAllLocalStorage(profileId),
      deleteLocalStorageByOrigin: (origin: string) => storageHook.deleteLocalStorageByOrigin(profileId, origin),
      deleteLocalStorageItem: (origin: string, name: string) =>
        storageHook.deleteLocalStorageItem(profileId, origin, name),
    }),
    [
      profileId,
      storageHook.storageState,
      storageHook.loading,
      storageHook.error,
      storageHook.deleting,
      storageHook.fetchStorageState,
      storageHook.clear,
      storageHook.clearAllCookies,
      storageHook.deleteCookiesByDomain,
      storageHook.deleteCookie,
      storageHook.clearAllLocalStorage,
      storageHook.deleteLocalStorageByOrigin,
      storageHook.deleteLocalStorageItem,
    ]
  );

  // Create bound service workers resource
  const serviceWorkers = useMemo<BoundServiceWorkersResource>(
    () => ({
      data: serviceWorkersHook.serviceWorkers,
      workers: serviceWorkersHook.serviceWorkers?.workers ?? [],
      loading: serviceWorkersHook.loading,
      error: serviceWorkersHook.error,
      deleting: serviceWorkersHook.deleting,
      fetch: () => serviceWorkersHook.fetchServiceWorkers(profileId),
      clear: serviceWorkersHook.clear,
      unregisterAll: () => serviceWorkersHook.unregisterAll(profileId),
      unregister: (scopeURL: string) => serviceWorkersHook.unregisterWorker(profileId, scopeURL),
    }),
    [
      profileId,
      serviceWorkersHook.serviceWorkers,
      serviceWorkersHook.loading,
      serviceWorkersHook.error,
      serviceWorkersHook.deleting,
      serviceWorkersHook.fetchServiceWorkers,
      serviceWorkersHook.clear,
      serviceWorkersHook.unregisterAll,
      serviceWorkersHook.unregisterWorker,
    ]
  );

  // Create bound history resource
  const history = useMemo<BoundHistoryResource>(
    () => ({
      data: historyHook.history,
      loading: historyHook.loading,
      error: historyHook.error,
      deleting: historyHook.deleting,
      navigating: historyHook.navigating,
      fetch: () => historyHook.fetchHistory(profileId),
      clear: historyHook.clear,
      clearAll: () => historyHook.clearAllHistory(profileId),
      deleteEntry: (entryId: string) => historyHook.deleteHistoryEntry(profileId, entryId),
      updateSettings: (settings: Partial<HistorySettings>) => historyHook.updateSettings(profileId, settings),
      navigateTo: (url: string) => historyHook.navigateToUrl(profileId, url),
    }),
    [
      profileId,
      historyHook.history,
      historyHook.loading,
      historyHook.error,
      historyHook.deleting,
      historyHook.navigating,
      historyHook.fetchHistory,
      historyHook.clear,
      historyHook.clearAllHistory,
      historyHook.deleteHistoryEntry,
      historyHook.updateSettings,
      historyHook.navigateToUrl,
    ]
  );

  // Create bound tabs resource
  const tabs = useMemo<BoundTabsResource>(
    () => ({
      data: tabsHook.tabs,
      loading: tabsHook.loading,
      error: tabsHook.error,
      deleting: tabsHook.deleting,
      fetch: () => tabsHook.fetchTabs(profileId),
      clear: tabsHook.clear,
      clearAll: () => tabsHook.clearAllTabs(profileId),
      delete: (order: number) => tabsHook.deleteTab(profileId, order),
    }),
    [
      profileId,
      tabsHook.tabs,
      tabsHook.loading,
      tabsHook.error,
      tabsHook.deleting,
      tabsHook.fetchTabs,
      tabsHook.clear,
      tabsHook.clearAllTabs,
      tabsHook.deleteTab,
    ]
  );

  return {
    storage,
    serviceWorkers,
    history,
    tabs,
    hasActiveSession,
  };
}
