import { useState, useEffect, useCallback } from 'react';
import { X, Cookie, Database, AlertCircle, Loader2 } from 'lucide-react';
import { ResponsiveDialog } from '@shared/layout';
import { useStorageState } from '@/domains/recording';
import { CookiesTable } from './CookiesTable';
import { LocalStorageTable } from './LocalStorageTable';

interface StorageStateModalProps {
  profileId: string;
  profileName: string;
  onClose: () => void;
}

type TabId = 'cookies' | 'storage';

export function StorageStateModal({ profileId, profileName, onClose }: StorageStateModalProps) {
  const [activeTab, setActiveTab] = useState<TabId>('cookies');
  const { storageState, loading, error, fetchStorageState, clear } = useStorageState();

  useEffect(() => {
    void fetchStorageState(profileId);
    return () => {
      clear();
    };
  }, [profileId, fetchStorageState, clear]);

  const handleTabChange = useCallback((tab: TabId) => {
    setActiveTab(tab);
  }, []);

  return (
    <ResponsiveDialog isOpen={true} onDismiss={onClose} ariaLabel="View storage state" size="wide" className="!p-0">
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-xl max-h-[90vh] overflow-hidden flex flex-col min-w-[600px]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Storage State</h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">Session: {profileName}</p>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800"
          >
            <X size={20} />
          </button>
        </div>

        {/* Stats bar */}
        {storageState && !loading && (
          <div className="px-6 py-2 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700 flex items-center gap-4 text-xs text-gray-600 dark:text-gray-400">
            <span className="flex items-center gap-1">
              <Cookie size={12} />
              {storageState.stats.cookieCount} cookie{storageState.stats.cookieCount !== 1 ? 's' : ''}
            </span>
            <span className="flex items-center gap-1">
              <Database size={12} />
              {storageState.stats.localStorageCount} localStorage item
              {storageState.stats.localStorageCount !== 1 ? 's' : ''}
            </span>
            <span className="text-gray-400 dark:text-gray-500">
              across {storageState.stats.originCount} origin{storageState.stats.originCount !== 1 ? 's' : ''}
            </span>
          </div>
        )}

        {/* Tabs */}
        <div className="flex border-b border-gray-200 dark:border-gray-700 px-6">
          <button
            onClick={() => handleTabChange('cookies')}
            className={`px-4 py-3 text-sm font-medium border-b-2 -mb-px flex items-center gap-2 ${
              activeTab === 'cookies'
                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
            }`}
          >
            <Cookie size={16} />
            Cookies
            {storageState && (
              <span
                className={`px-1.5 py-0.5 text-[10px] rounded-full ${
                  activeTab === 'cookies'
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
                    : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                }`}
              >
                {storageState.stats.cookieCount}
              </span>
            )}
          </button>
          <button
            onClick={() => handleTabChange('storage')}
            className={`px-4 py-3 text-sm font-medium border-b-2 -mb-px flex items-center gap-2 ${
              activeTab === 'storage'
                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
            }`}
          >
            <Database size={16} />
            LocalStorage
            {storageState && (
              <span
                className={`px-1.5 py-0.5 text-[10px] rounded-full ${
                  activeTab === 'storage'
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300'
                    : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                }`}
              >
                {storageState.stats.localStorageCount}
              </span>
            )}
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-4 min-h-[300px]">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={24} className="animate-spin text-blue-500" />
              <span className="ml-2 text-gray-500 dark:text-gray-400">Loading storage state...</span>
            </div>
          )}

          {error && !loading && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800">
              <AlertCircle size={18} className="text-red-600 dark:text-red-400 flex-shrink-0" />
              <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            </div>
          )}

          {storageState && !loading && !error && (
            <>
              {activeTab === 'cookies' && <CookiesTable cookies={storageState.cookies} />}
              {activeTab === 'storage' && <LocalStorageTable origins={storageState.origins} />}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Close
          </button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
