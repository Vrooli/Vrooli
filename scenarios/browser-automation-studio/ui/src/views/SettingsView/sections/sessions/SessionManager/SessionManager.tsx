import { X, Settings } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { BrowserProfile } from '@/domains/recording/types/types';
import { SessionSidebar, isSettingsSection, type SectionId } from './SessionSidebar';
import { useSessionNavigation } from './hooks/navigation';
import { useProfileSettings } from './hooks/settings';
import { useSessionPersistence } from './hooks/persistence';
import { useProfileBoundResources } from './hooks/resources';
import { PresetsSection, FingerprintSection, BehaviorSection, AntiDetectionSection, ProxySection, ExtraHeadersSection } from '../settings';
import { CookiesTable } from '../storage/CookiesTable';
import { LocalStorageTable } from '../storage/LocalStorageTable';
import { ServiceWorkersTable } from '../storage/ServiceWorkersTable';
import { HistoryTable } from '../storage/HistoryTable';
import { HistorySettings } from '../storage/HistorySettings';
import { TabsTable } from '../storage/TabsTable';
import { ClearAllConfirmationModal, type ClearAllTarget } from './ClearAllConfirmationModal';
import { StorageSectionWrapper } from './StorageSectionWrapper';
import { SessionFooterStats } from './SessionFooterStats';

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
  // Domain hooks - each manages a specific concern
  const navigation = useSessionNavigation({ initialSection });
  const settings = useProfileSettings({ initialProfile });
  const persistence = useSessionPersistence({ settings, onSave });
  const resources = useProfileBoundResources({ profileId });

  // Fetch data for initial section on mount.
  // NOTE: We intentionally use an empty dependency array here. Section changes
  // after mount are handled by handleSectionChange, not this effect.
  useEffect(() => {
    const section = navigation.activeSection;
    if (section === 'cookies' || section === 'local-storage') {
      void resources.storage.fetch();
    } else if (section === 'service-workers') {
      void resources.serviceWorkers.fetch();
    } else if (section === 'history') {
      void resources.history.fetch();
    } else if (section === 'tabs') {
      void resources.tabs.fetch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Handle section changes with fetch/clear logic.
  // This approach avoids the infinite loop that occurred when resource objects
  // (which change on every state update) were in the useEffect dependency array.
  const handleSectionChange = useCallback(
    (newSection: SectionId) => {
      const oldSection = navigation.activeSection;

      // Skip if navigating to same section
      if (oldSection === newSection) return;

      // Clear old section's data
      if (oldSection === 'cookies' || oldSection === 'local-storage') {
        resources.storage.clear();
      } else if (oldSection === 'service-workers') {
        resources.serviceWorkers.clear();
      } else if (oldSection === 'history') {
        resources.history.clear();
      } else if (oldSection === 'tabs') {
        resources.tabs.clear();
      }

      // Change section
      navigation.setActiveSection(newSection);

      // Fetch new section's data
      if (newSection === 'cookies' || newSection === 'local-storage') {
        void resources.storage.fetch();
      } else if (newSection === 'service-workers') {
        void resources.serviceWorkers.fetch();
      } else if (newSection === 'history') {
        void resources.history.fetch();
      } else if (newSection === 'tabs') {
        void resources.tabs.fetch();
      }
    },
    [navigation, resources]
  );

  // Determine if there's an active session (service workers data has a non-empty session_id)
  const hasActiveSession = !!resources.serviceWorkers.data?.session_id;

  // Confirmation modal state for clear all operations
  const [confirmClearAll, setConfirmClearAll] = useState<ClearAllTarget | null>(null);

  // History settings modal state
  const [showHistorySettings, setShowHistorySettings] = useState(false);

  // Get the deleting state for the current clear all target
  const clearAllDeleting = useMemo(() => {
    if (!confirmClearAll) return false;
    switch (confirmClearAll) {
      case 'history': return resources.history.deleting;
      case 'tabs': return resources.tabs.deleting;
      default: return resources.storage.deleting;
    }
  }, [confirmClearAll, resources.history.deleting, resources.tabs.deleting, resources.storage.deleting]);

  const handleConfirmClearAll = async () => {
    if (confirmClearAll === 'cookies') {
      await resources.storage.clearAllCookies();
    } else if (confirmClearAll === 'localStorage') {
      await resources.storage.clearAllLocalStorage();
    } else if (confirmClearAll === 'history') {
      await resources.history.clearAll();
    } else if (confirmClearAll === 'tabs') {
      await resources.tabs.clearAll();
    }
    setConfirmClearAll(null);
  };

  const handleClose = () => {
    if (persistence.isDirty) {
      if (!window.confirm('You have unsaved changes. Discard them?')) {
        return;
      }
    }
    onClose();
  };

  const handleSaveAndClose = async () => {
    const success = await persistence.handleSave();
    if (success) {
      onClose();
    }
  };

  const isSettingsView = isSettingsSection(navigation.activeSection);

  // Compute localStorage count label with origin info
  const localStorageCountLabel = useMemo(() => {
    if (!resources.storage.state) return undefined;
    const { localStorageCount, originCount } = resources.storage.state.stats;
    const itemLabel = localStorageCount === 1 ? 'item' : 'items';
    const originLabel = originCount === 1 ? 'origin' : 'origins';
    return `${localStorageCount} ${itemLabel} across ${originCount} ${originLabel}`;
  }, [resources.storage.state]);

  // History settings button for extra header content
  const historySettingsButton = (
    <button
      onClick={() => setShowHistorySettings(true)}
      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700"
    >
      <Settings size={14} />
      Settings
    </button>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={handleClose}>
      <div
        className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-4xl h-[85vh] overflow-hidden flex flex-col"
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
            activeSection={navigation.activeSection}
            onSectionChange={handleSectionChange}
            hasStorageState={hasStorageState}
            cookieCount={resources.storage.state?.stats.cookieCount}
            localStorageCount={resources.storage.state?.stats.localStorageCount}
            serviceWorkerCount={resources.serviceWorkers.data?.workers?.length}
            historyCount={resources.history.data?.stats.totalEntries}
            tabsCount={resources.tabs.data.length}
            hasActiveSession={hasActiveSession}
          />

          <div className="flex-1 overflow-y-auto p-6">
            {/* Error display */}
            {persistence.error && (
              <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/40 dark:text-red-200">
                {persistence.error}
              </div>
            )}

            {/* Settings sections */}
            {navigation.activeSection === 'presets' && (
              <PresetsSection preset={settings.preset} onPresetChange={settings.applyPreset} />
            )}

            {navigation.activeSection === 'fingerprint' && (
              <FingerprintSection fingerprint={settings.fingerprint} onChange={settings.updateFingerprint} />
            )}

            {navigation.activeSection === 'behavior' && (
              <BehaviorSection behavior={settings.behavior} onChange={settings.updateBehavior} />
            )}

            {navigation.activeSection === 'anti-detection' && (
              <AntiDetectionSection antiDetection={settings.antiDetection} onChange={settings.updateAntiDetection} />
            )}

            {navigation.activeSection === 'proxy' && (
              <ProxySection proxy={settings.proxy} onChange={settings.updateProxy} />
            )}

            {navigation.activeSection === 'extra-headers' && (
              <ExtraHeadersSection
                headers={settings.extraHeaders}
                onAdd={settings.addExtraHeader}
                onUpdate={settings.updateExtraHeader}
                onRemove={settings.removeExtraHeader}
              />
            )}

            {/* Storage sections */}
            {navigation.activeSection === 'cookies' && (
              <StorageSectionWrapper
                count={resources.storage.state?.stats.cookieCount}
                itemLabel="cookie"
                loading={resources.storage.loading}
                error={resources.storage.error}
                emptyMessage="No cookies saved in this session."
                deleting={resources.storage.deleting}
                onClearAll={() => setConfirmClearAll('cookies')}
              >
                {resources.storage.state && (
                  <CookiesTable
                    cookies={resources.storage.state.cookies}
                    deleting={resources.storage.deleting}
                    onDeleteDomain={resources.storage.deleteCookiesByDomain}
                    onDeleteCookie={resources.storage.deleteCookie}
                  />
                )}
              </StorageSectionWrapper>
            )}

            {navigation.activeSection === 'local-storage' && (
              <StorageSectionWrapper
                count={resources.storage.state?.stats.localStorageCount}
                itemLabel="item"
                countLabel={localStorageCountLabel}
                loading={resources.storage.loading}
                error={resources.storage.error}
                emptyMessage="No localStorage data saved in this session."
                deleting={resources.storage.deleting}
                onClearAll={() => setConfirmClearAll('localStorage')}
              >
                {resources.storage.state && (
                  <LocalStorageTable
                    origins={resources.storage.state.origins}
                    deleting={resources.storage.deleting}
                    onDeleteOrigin={resources.storage.deleteLocalStorageByOrigin}
                    onDeleteItem={resources.storage.deleteLocalStorageItem}
                  />
                )}
              </StorageSectionWrapper>
            )}

            {navigation.activeSection === 'service-workers' && (
              <StorageSectionWrapper
                count={hasActiveSession ? resources.serviceWorkers.data?.workers?.length : 0}
                itemLabel="service worker"
                loading={resources.serviceWorkers.loading}
                error={resources.serviceWorkers.error}
                emptyMessage="No service workers registered."
                deleting={resources.serviceWorkers.deleting}
                onClearAll={resources.serviceWorkers.unregisterAll}
                clearAllLabel="Unregister All"
              >
                <ServiceWorkersTable
                  workers={resources.serviceWorkers.data?.workers ?? []}
                  controlMode={resources.serviceWorkers.data?.control?.mode}
                  hasActiveSession={hasActiveSession}
                  deleting={resources.serviceWorkers.deleting}
                  onUnregister={resources.serviceWorkers.unregister}
                />
              </StorageSectionWrapper>
            )}

            {navigation.activeSection === 'history' && (
              <StorageSectionWrapper
                count={resources.history.data?.stats.totalEntries}
                itemLabel="entry"
                countLabel={resources.history.data ? `${resources.history.data.stats.totalEntries} entr${resources.history.data.stats.totalEntries !== 1 ? 'ies' : 'y'}` : undefined}
                loading={resources.history.loading}
                error={resources.history.error}
                emptyMessage="No navigation history yet."
                deleting={resources.history.deleting}
                onClearAll={() => setConfirmClearAll('history')}
                extraHeaderContent={historySettingsButton}
              >
                {resources.history.data && (
                  <HistoryTable
                    entries={resources.history.data.entries ?? []}
                    deleting={resources.history.deleting}
                    navigating={resources.history.navigating}
                    onDelete={resources.history.deleteEntry}
                    onNavigate={hasActiveSession ? resources.history.navigateTo : undefined}
                  />
                )}
              </StorageSectionWrapper>
            )}

            {navigation.activeSection === 'tabs' && (
              <StorageSectionWrapper
                count={resources.tabs.data.length}
                itemLabel="saved tab"
                loading={resources.tabs.loading}
                error={resources.tabs.error}
                emptyMessage="No saved tabs."
                deleting={resources.tabs.deleting}
                onClearAll={() => setConfirmClearAll('tabs')}
              >
                <TabsTable
                  tabs={resources.tabs.data}
                  deleting={resources.tabs.deleting}
                  onDelete={resources.tabs.delete}
                />
              </StorageSectionWrapper>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between bg-gray-50 dark:bg-gray-800/50">
          {/* Storage stats when in storage view */}
          {!isSettingsView ? (
            <SessionFooterStats
              activeSection={navigation.activeSection}
              storageState={resources.storage.state}
              serviceWorkers={resources.serviceWorkers.data}
              history={resources.history.data}
              tabsCount={resources.tabs.data.length}
              hasActiveSession={hasActiveSession}
            />
          ) : (
            <div />
          )}

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
                disabled={persistence.saving || !persistence.isDirty}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {persistence.saving ? 'Saving...' : 'Save Changes'}
              </button>
            )}
          </div>
        </div>

        {/* Clear All Confirmation Modal */}
        {confirmClearAll && (
          <ClearAllConfirmationModal
            target={confirmClearAll}
            deleting={clearAllDeleting}
            onConfirm={handleConfirmClearAll}
            onCancel={() => setConfirmClearAll(null)}
          />
        )}

        {/* History Settings Modal */}
        {showHistorySettings && resources.history.data && (
          <HistorySettings
            settings={resources.history.data.settings}
            loading={resources.history.loading}
            onUpdate={resources.history.updateSettings}
            onClose={() => setShowHistorySettings(false)}
          />
        )}
      </div>
    </div>
  );
}
