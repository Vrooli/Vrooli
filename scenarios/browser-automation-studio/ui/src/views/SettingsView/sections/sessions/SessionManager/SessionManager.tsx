import { X, Loader2, AlertCircle, Cookie, Database, Trash2 } from 'lucide-react';
import { useState } from 'react';
import type { BrowserProfile } from '@/domains/recording/types/types';
import { SessionSidebar, isSettingsSection } from './SessionSidebar';
import { useSessionManager } from './useSessionManager';
import { PresetsSection, FingerprintSection, BehaviorSection, AntiDetectionSection } from '../settings';
import { CookiesTable } from '../storage/CookiesTable';
import { LocalStorageTable } from '../storage/LocalStorageTable';

interface SessionManagerProps {
  profileId: string;
  profileName: string;
  initialProfile?: BrowserProfile;
  hasStorageState?: boolean;
  onSave: (profile: BrowserProfile) => Promise<void>;
  onClose: () => void;
}

export function SessionManager({
  profileId,
  profileName,
  initialProfile,
  hasStorageState,
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
    applyPreset,
    updateFingerprint,
    updateBehavior,
    updateAntiDetection,
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
    saving,
    error,
    isDirty,
    handleSave,
  } = useSessionManager({ profileId, initialProfile, onSave });

  // Confirmation modal state for clear all operations
  const [confirmClearAll, setConfirmClearAll] = useState<'cookies' | 'localStorage' | null>(null);

  const handleConfirmClearAll = async () => {
    if (confirmClearAll === 'cookies') {
      await clearAllCookies();
    } else if (confirmClearAll === 'localStorage') {
      await clearAllLocalStorage();
    }
    setConfirmClearAll(null);
  };

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
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between bg-gray-50 dark:bg-gray-800/50">
          {/* Storage stats when in storage view */}
          {!isSettingsView && storageState && (
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
                Clear All {confirmClearAll === 'cookies' ? 'Cookies' : 'LocalStorage'}?
              </h3>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                This will permanently delete all{' '}
                {confirmClearAll === 'cookies' ? 'cookies' : 'localStorage data'} from this session.
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
                  disabled={storageDeleting}
                  className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
                >
                  {storageDeleting ? 'Deleting...' : 'Clear All'}
                </button>
              </div>
            </div>
          </div>
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
