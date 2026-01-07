import { X, Loader2, AlertCircle, Cookie, Database, Trash2, Cog, History, Settings, Layers } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import type { BrowserProfile } from '@/domains/recording/types/types';
import { SessionSidebar, isSettingsSection } from './SessionSidebar';
import { useSessionManager } from './useSessionManager';
import { useHistory } from '@/domains/recording/hooks/useHistory';
import { useTabs } from '@/domains/recording/hooks/useTabs';
import { PresetsSection, FingerprintSection, BehaviorSection, AntiDetectionSection, ProxySection, ExtraHeadersSection } from '../settings';
import { CookiesTable } from '../storage/CookiesTable';
import { LocalStorageTable } from '../storage/LocalStorageTable';
import { ServiceWorkersTable } from '../storage/ServiceWorkersTable';
import { HistoryTable } from '../storage/HistoryTable';
import { HistorySettings } from '../storage/HistorySettings';
import { TabsTable } from '../storage/TabsTable';

interface SessionManagerProps {
  profileId: string;
  profileName: string;
  initialProfile?: BrowserProfile;
  hasStorageState?: boolean;
  /** Section to open when the dialog first opens */
  initialSection?: 'presets' | 'fingerprint' | 'behavior' | 'anti-detection' | 'proxy' | 'extra-headers' | 'cookies' | 'local-storage' | 'service-workers' | 'history' | 'tabs';
  onSave: (profile: BrowserProfile) => Promise<void>;
  onClose: () => void;
}

export function SessionManager({
  profileId,
  profileName,
  initialProfile,
  hasStorageState,
  initialSection,
  onSave,
  onClose,
}: SessionManagerProps) {
  const {
    activeSection,
    setActiveSection,
    preset,
    fingerprint,
    behavior,
    antiDetection,
    proxy,
    extraHeaders,
    applyPreset,
    updateFingerprint,
    updateBehavior,
    updateAntiDetection,
    updateProxy,
    addExtraHeader,
    updateExtraHeader,
    removeExtraHeader,
    storageState,
    storageLoading,
    storageError,
    storageDeleting,
    clearAllCookies,
    deleteCookiesByDomain,
    deleteCookie,
    clearAllLocalStorage,
    deleteLocalStorageByOrigin,
    deleteLocalStorageItem,
    serviceWorkers,
    swLoading,
    swError,
    swDeleting,
    unregisterAllServiceWorkers,
    unregisterServiceWorker,
    saving,
    error,
    isDirty,
    handleSave,
  } = useSessionManager({ profileId, initialProfile, onSave });

  // History hook
  const {
    history,
    loading: historyLoading,
    error: historyError,
    deleting: historyDeleting,
    navigating: historyNavigating,
    fetchHistory,
    clear: clearHistoryState,
    clearAllHistory,
    deleteHistoryEntry,
    updateSettings: updateHistorySettings,
    navigateToUrl: navigateToHistoryUrl,
  } = useHistory();

  // Tabs hook
  const {
    tabs,
    loading: tabsLoading,
    error: tabsError,
    deleting: tabsDeleting,
    fetchTabs,
    clear: clearTabsState,
    clearAllTabs,
    deleteTab,
  } = useTabs();

  // Set initial section on mount if provided
  useEffect(() => {
    if (initialSection) {
      setActiveSection(initialSection);
    }
  }, [initialSection, setActiveSection]);

  // Fetch history when the history section is activated (refetch every time, like other storage tabs)
  useEffect(() => {
    const isHistorySection = activeSection === 'history' || initialSection === 'history';
    if (isHistorySection) {
      fetchHistory(profileId);
    }
    return () => {
      if (isHistorySection) {
        clearHistoryState();
      }
    };
  }, [activeSection, initialSection, profileId, fetchHistory, clearHistoryState]);

  // Fetch tabs when the tabs section is activated
  useEffect(() => {
    const isTabsSection = activeSection === 'tabs' || initialSection === 'tabs';
    if (isTabsSection) {
      fetchTabs(profileId);
    }
    return () => {
      if (isTabsSection) {
        clearTabsState();
      }
    };
  }, [activeSection, initialSection, profileId, fetchTabs, clearTabsState]);

  // Determine if there's an active session (service workers data has a non-empty session_id)
  const hasActiveSession = !!serviceWorkers?.session_id;

  // Confirmation modal state for clear all operations
  const [confirmClearAll, setConfirmClearAll] = useState<'cookies' | 'localStorage' | 'history' | 'tabs' | null>(null);

  // History settings modal state
  const [showHistorySettings, setShowHistorySettings] = useState(false);

  const handleConfirmClearAll = async () => {
    if (confirmClearAll === 'cookies') {
      await clearAllCookies();
    } else if (confirmClearAll === 'localStorage') {
      await clearAllLocalStorage();
    } else if (confirmClearAll === 'history') {
      await clearAllHistory(profileId);
    } else if (confirmClearAll === 'tabs') {
      await clearAllTabs(profileId);
    }
    setConfirmClearAll(null);
  };

  // Wrapper callbacks for history operations
  const handleDeleteHistoryEntry = useCallback(
    async (entryId: string) => {
      return deleteHistoryEntry(profileId, entryId);
    },
    [deleteHistoryEntry, profileId]
  );

  const handleNavigateToHistoryUrl = useCallback(
    async (url: string) => {
      return navigateToHistoryUrl(profileId, url);
    },
    [navigateToHistoryUrl, profileId]
  );

  const handleUpdateHistorySettings = useCallback(
    async (settings: Parameters<typeof updateHistorySettings>[1]) => {
      return updateHistorySettings(profileId, settings);
    },
    [updateHistorySettings, profileId]
  );

  // Wrapper callback for tab operations
  const handleDeleteTab = useCallback(
    async (order: number) => {
      return deleteTab(profileId, order);
    },
    [deleteTab, profileId]
  );

  const handleClose = () => {
    if (isDirty) {
      if (!window.confirm('You have unsaved changes. Discard them?')) {
        return;
      }
    }
    onClose();
  };

  const handleSaveAndClose = async () => {
    const success = await handleSave();
    if (success) {
      onClose();
    }
  };

  const isSettingsView = isSettingsSection(activeSection);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={handleClose}>
      <div
        className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Session Settings</h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">{profileName}</p>
          </div>
          <button
            onClick={handleClose}
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body - Sidebar + Content */}
        <div className="flex-1 flex overflow-hidden">
          <SessionSidebar
            activeSection={activeSection}
            onSectionChange={setActiveSection}
            hasStorageState={hasStorageState}
            cookieCount={storageState?.stats.cookieCount}
            localStorageCount={storageState?.stats.localStorageCount}
            serviceWorkerCount={serviceWorkers?.workers?.length}
            historyCount={history?.stats.totalEntries}
            tabsCount={tabs.length}
            hasActiveSession={hasActiveSession}
          />

          <div className="flex-1 overflow-y-auto p-6">
            {/* Error display */}
            {error && (
              <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/40 dark:text-red-200">
                {error}
              </div>
            )}

            {/* Settings sections */}
            {activeSection === 'presets' && <PresetsSection preset={preset} onPresetChange={applyPreset} />}

            {activeSection === 'fingerprint' && (
              <FingerprintSection fingerprint={fingerprint} onChange={updateFingerprint} />
            )}

            {activeSection === 'behavior' && <BehaviorSection behavior={behavior} onChange={updateBehavior} />}

            {activeSection === 'anti-detection' && (
              <AntiDetectionSection antiDetection={antiDetection} onChange={updateAntiDetection} />
            )}

            {activeSection === 'proxy' && <ProxySection proxy={proxy} onChange={updateProxy} />}

            {activeSection === 'extra-headers' && (
              <ExtraHeadersSection
                headers={extraHeaders}
                onAdd={addExtraHeader}
                onUpdate={updateExtraHeader}
                onRemove={removeExtraHeader}
              />
            )}

            {/* Storage sections */}
            {activeSection === 'cookies' && (
              <div className="space-y-4">
                {/* Header with Clear All button */}
                {storageState && storageState.cookies.length > 0 && (
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {storageState.stats.cookieCount} cookie{storageState.stats.cookieCount !== 1 ? 's' : ''}
                    </h3>
                    <button
                      onClick={() => setConfirmClearAll('cookies')}
                      disabled={storageDeleting}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-md hover:bg-red-100 dark:hover:bg-red-900/50 disabled:opacity-50"
                    >
                      <Trash2 size={14} />
                      Clear All
                    </button>
                  </div>
                )}
                <StorageContent
                  loading={storageLoading}
                  error={storageError}
                  emptyMessage="No cookies saved in this session."
                >
                  {storageState && (
                    <CookiesTable
                      cookies={storageState.cookies}
                      deleting={storageDeleting}
                      onDeleteDomain={deleteCookiesByDomain}
                      onDeleteCookie={deleteCookie}
                    />
                  )}
                </StorageContent>
              </div>
            )}

            {activeSection === 'local-storage' && (
              <div className="space-y-4">
                {/* Header with Clear All button */}
                {storageState && storageState.origins.some((o) => o.localStorage.length > 0) && (
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {storageState.stats.localStorageCount} item{storageState.stats.localStorageCount !== 1 ? 's' : ''}{' '}
                      across {storageState.stats.originCount} origin{storageState.stats.originCount !== 1 ? 's' : ''}
                    </h3>
                    <button
                      onClick={() => setConfirmClearAll('localStorage')}
                      disabled={storageDeleting}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-md hover:bg-red-100 dark:hover:bg-red-900/50 disabled:opacity-50"
                    >
                      <Trash2 size={14} />
                      Clear All
                    </button>
                  </div>
                )}
                <StorageContent
                  loading={storageLoading}
                  error={storageError}
                  emptyMessage="No localStorage data saved in this session."
                >
                  {storageState && (
                    <LocalStorageTable
                      origins={storageState.origins}
                      deleting={storageDeleting}
                      onDeleteOrigin={deleteLocalStorageByOrigin}
                      onDeleteItem={deleteLocalStorageItem}
                    />
                  )}
                </StorageContent>
              </div>
            )}

            {activeSection === 'service-workers' && (
              <div className="space-y-4">
                {/* Header with Clear All button */}
                {hasActiveSession && serviceWorkers && serviceWorkers.workers.length > 0 && (
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {serviceWorkers.workers.length} service worker{serviceWorkers.workers.length !== 1 ? 's' : ''}
                    </h3>
                    <button
                      onClick={unregisterAllServiceWorkers}
                      disabled={swDeleting}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-md hover:bg-red-100 dark:hover:bg-red-900/50 disabled:opacity-50"
                    >
                      <Trash2 size={14} />
                      Unregister All
                    </button>
                  </div>
                )}
                <StorageContent
                  loading={swLoading}
                  error={swError}
                  emptyMessage="No service workers registered."
                >
                  <ServiceWorkersTable
                    workers={serviceWorkers?.workers ?? []}
                    controlMode={serviceWorkers?.control?.mode}
                    hasActiveSession={hasActiveSession}
                    deleting={swDeleting}
                    onUnregister={unregisterServiceWorker}
                  />
                </StorageContent>
              </div>
            )}

            {activeSection === 'history' && (
              <div className="space-y-4">
                {/* Header with Settings and Clear All buttons */}
                {history && history.entries && history.entries.length > 0 && (
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {history.stats.totalEntries} entr{history.stats.totalEntries !== 1 ? 'ies' : 'y'}
                    </h3>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setShowHistorySettings(true)}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        <Settings size={14} />
                        Settings
                      </button>
                      <button
                        onClick={() => setConfirmClearAll('history')}
                        disabled={historyDeleting}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-md hover:bg-red-100 dark:hover:bg-red-900/50 disabled:opacity-50"
                      >
                        <Trash2 size={14} />
                        Clear All
                      </button>
                    </div>
                  </div>
                )}
                <StorageContent
                  loading={historyLoading}
                  error={historyError}
                  emptyMessage="No navigation history yet."
                >
                  {history && (
                    <HistoryTable
                      entries={history.entries ?? []}
                      deleting={historyDeleting}
                      navigating={historyNavigating}
                      onDelete={handleDeleteHistoryEntry}
                      onNavigate={hasActiveSession ? handleNavigateToHistoryUrl : undefined}
                    />
                  )}
                </StorageContent>
              </div>
            )}

            {activeSection === 'tabs' && (
              <div className="space-y-4">
                {/* Header with Clear All button */}
                {tabs.length > 0 && (
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {tabs.length} saved tab{tabs.length !== 1 ? 's' : ''}
                    </h3>
                    <button
                      onClick={() => setConfirmClearAll('tabs')}
                      disabled={tabsDeleting}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-md hover:bg-red-100 dark:hover:bg-red-900/50 disabled:opacity-50"
                    >
                      <Trash2 size={14} />
                      Clear All
                    </button>
                  </div>
                )}
                <StorageContent loading={tabsLoading} error={tabsError} emptyMessage="No saved tabs.">
                  <TabsTable tabs={tabs} deleting={tabsDeleting} onDelete={handleDeleteTab} />
                </StorageContent>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between bg-gray-50 dark:bg-gray-800/50">
          {/* Storage stats when in storage view */}
          {!isSettingsView && (activeSection === 'cookies' || activeSection === 'local-storage') && storageState && (
            <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <span className="flex items-center gap-1">
                <Cookie size={12} />
                {storageState.stats.cookieCount} cookie{storageState.stats.cookieCount !== 1 ? 's' : ''}
              </span>
              <span className="flex items-center gap-1">
                <Database size={12} />
                {storageState.stats.localStorageCount} item{storageState.stats.localStorageCount !== 1 ? 's' : ''}
              </span>
              <span className="text-gray-400 dark:text-gray-500">
                across {storageState.stats.originCount} origin{storageState.stats.originCount !== 1 ? 's' : ''}
              </span>
            </div>
          )}
          {!isSettingsView && activeSection === 'service-workers' && (
            <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <span className="flex items-center gap-1">
                <Cog size={12} />
                {serviceWorkers?.workers?.length ?? 0} service worker{(serviceWorkers?.workers?.length ?? 0) !== 1 ? 's' : ''}
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
          )}
          {!isSettingsView && activeSection === 'history' && history && (
            <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <span className="flex items-center gap-1">
                <History size={12} />
                {history.stats.totalEntries} entr{history.stats.totalEntries !== 1 ? 'ies' : 'y'}
              </span>
              <span className="text-gray-400 dark:text-gray-500">
                Max: {history.settings.maxEntries} | TTL: {history.settings.retentionDays > 0 ? `${history.settings.retentionDays}d` : 'unlimited'}
              </span>
            </div>
          )}
          {!isSettingsView && activeSection === 'tabs' && (
            <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
              <span className="flex items-center gap-1">
                <Layers size={12} />
                {tabs.length} tab{tabs.length !== 1 ? 's' : ''} saved
              </span>
              <span className="text-gray-400 dark:text-gray-500">Will restore on session start</span>
            </div>
          )}
          {isSettingsView && <div />}

          {/* Action buttons */}
          <div className="flex items-center gap-3">
            <button
              onClick={handleClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              {isSettingsView ? 'Cancel' : 'Close'}
            </button>
            {isSettingsView && (
              <button
                onClick={handleSaveAndClose}
                disabled={saving || !isDirty}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </button>
            )}
          </div>
        </div>

        {/* Clear All Confirmation Modal */}
        {confirmClearAll && (
          <div
            className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50"
            onClick={() => setConfirmClearAll(null)}
          >
            <div
              className="bg-white dark:bg-gray-900 rounded-lg shadow-xl p-6 max-w-sm mx-4"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                Clear All {confirmClearAll === 'cookies' ? 'Cookies' : confirmClearAll === 'localStorage' ? 'LocalStorage' : confirmClearAll === 'tabs' ? 'Tabs' : 'History'}?
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                This will permanently delete all{' '}
                {confirmClearAll === 'cookies' ? 'cookies' : confirmClearAll === 'localStorage' ? 'localStorage data' : confirmClearAll === 'tabs' ? 'saved tabs' : 'navigation history'} from this session.
                This action cannot be undone.
              </p>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setConfirmClearAll(null)}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  Cancel
                </button>
                <button
                  onClick={handleConfirmClearAll}
                  disabled={confirmClearAll === 'history' ? historyDeleting : confirmClearAll === 'tabs' ? tabsDeleting : storageDeleting}
                  className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
                >
                  {(confirmClearAll === 'history' ? historyDeleting : confirmClearAll === 'tabs' ? tabsDeleting : storageDeleting) ? 'Deleting...' : 'Clear All'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* History Settings Modal */}
        {showHistorySettings && history && (
          <HistorySettings
            settings={history.settings}
            loading={historyLoading}
            onUpdate={handleUpdateHistorySettings}
            onClose={() => setShowHistorySettings(false)}
          />
        )}
      </div>
    </div>
  );
}

/** Wrapper for storage content with loading/error states */
function StorageContent({
  loading,
  error,
  emptyMessage,
  children,
}: {
  loading: boolean;
  error: string | null;
  emptyMessage: string;
  children: React.ReactNode;
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
