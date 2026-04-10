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

  const {
    storageState,
    loading: storageLoading,
    error: storageError,
    deleting: storageDeleting,
    fetchStorageState,
    clear: clearStorage,
    clearAllCookies,
    deleteCookiesByDomain,
    deleteCookie,
    clearAllLocalStorage,
    deleteLocalStorageByOrigin,
    deleteLocalStorageItem,
  } = storageHook;
  const {
    serviceWorkers: serviceWorkersData,
    loading: serviceWorkersLoading,
    error: serviceWorkersError,
    deleting: serviceWorkersDeleting,
    fetchServiceWorkers,
    clear: clearServiceWorkers,
    unregisterAll,
    unregisterWorker,
  } = serviceWorkersHook;
  const {
    history: historyData,
    loading: historyLoading,
    error: historyError,
    deleting: historyDeleting,
    navigating,
    fetchHistory,
    clear: clearHistory,
    clearAllHistory,
    deleteHistoryEntry,
    updateSettings,
    navigateToUrl,
  } = historyHook;
  const {
    tabs: tabsData,
    loading: tabsLoading,
    error: tabsError,
    deleting: tabsDeleting,
    fetchTabs,
    clear: clearTabs,
    clearAllTabs,
    deleteTab,
  } = tabsHook;

  // Derive hasActiveSession from service workers data
  const hasActiveSession = !!serviceWorkersData?.session_id;

  // Create bound storage resource
  const storage = useMemo<BoundStorageResource>(
    () => ({
      state: storageState,
      loading: storageLoading,
      error: storageError,
      deleting: storageDeleting,
      fetch: () => fetchStorageState(profileId),
      clear: clearStorage,
      clearAllCookies: () => clearAllCookies(profileId),
      deleteCookiesByDomain: (domain: string) => deleteCookiesByDomain(profileId, domain),
      deleteCookie: (domain: string, name: string) => deleteCookie(profileId, domain, name),
      clearAllLocalStorage: () => clearAllLocalStorage(profileId),
      deleteLocalStorageByOrigin: (origin: string) => deleteLocalStorageByOrigin(profileId, origin),
      deleteLocalStorageItem: (origin: string, name: string) =>
        deleteLocalStorageItem(profileId, origin, name),
    }),
    [
      profileId,
      storageState,
      storageLoading,
      storageError,
      storageDeleting,
      fetchStorageState,
      clearStorage,
      clearAllCookies,
      deleteCookiesByDomain,
      deleteCookie,
      clearAllLocalStorage,
      deleteLocalStorageByOrigin,
      deleteLocalStorageItem,
    ]
  );

  // Create bound service workers resource
  const serviceWorkers = useMemo<BoundServiceWorkersResource>(
    () => ({
      data: serviceWorkersData,
      workers: serviceWorkersData?.workers ?? [],
      loading: serviceWorkersLoading,
      error: serviceWorkersError,
      deleting: serviceWorkersDeleting,
      fetch: () => fetchServiceWorkers(profileId),
      clear: clearServiceWorkers,
      unregisterAll: () => unregisterAll(profileId),
      unregister: (scopeURL: string) => unregisterWorker(profileId, scopeURL),
    }),
    [
      profileId,
      serviceWorkersData,
      serviceWorkersLoading,
      serviceWorkersError,
      serviceWorkersDeleting,
      fetchServiceWorkers,
      clearServiceWorkers,
      unregisterAll,
      unregisterWorker,
    ]
  );

  // Create bound history resource
  const history = useMemo<BoundHistoryResource>(
    () => ({
      data: historyData,
      loading: historyLoading,
      error: historyError,
      deleting: historyDeleting,
      navigating,
      fetch: () => fetchHistory(profileId),
      clear: clearHistory,
      clearAll: () => clearAllHistory(profileId),
      deleteEntry: (entryId: string) => deleteHistoryEntry(profileId, entryId),
      updateSettings: (settings: Partial<HistorySettings>) => updateSettings(profileId, settings),
      navigateTo: (url: string) => navigateToUrl(profileId, url),
    }),
    [
      profileId,
      historyData,
      historyLoading,
      historyError,
      historyDeleting,
      navigating,
      fetchHistory,
      clearHistory,
      clearAllHistory,
      deleteHistoryEntry,
      updateSettings,
      navigateToUrl,
    ]
  );

  // Create bound tabs resource
  const tabs = useMemo<BoundTabsResource>(
    () => ({
      data: tabsData,
      loading: tabsLoading,
      error: tabsError,
      deleting: tabsDeleting,
      fetch: () => fetchTabs(profileId),
      clear: clearTabs,
      clearAll: () => clearAllTabs(profileId),
      delete: (order: number) => deleteTab(profileId, order),
    }),
    [
      profileId,
      tabsData,
      tabsLoading,
      tabsError,
      tabsDeleting,
      fetchTabs,
      clearTabs,
      clearAllTabs,
      deleteTab,
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
