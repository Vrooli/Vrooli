import { useCallback, useState } from 'react';
import { Settings, Camera, Clock, Database } from 'lucide-react';
import type { HistorySettings as HistorySettingsType } from '@/domains/recording';

interface HistorySettingsProps {
  settings: HistorySettingsType;
  loading?: boolean;
  onUpdate: (settings: Partial<HistorySettingsType>) => Promise<boolean>;
  onClose: () => void;
}

export function HistorySettings({ settings, loading, onUpdate, onClose }: HistorySettingsProps) {
  const [localSettings, setLocalSettings] = useState<HistorySettingsType>({
    maxEntries: settings.maxEntries,
    retentionDays: settings.retentionDays,
    captureThumbnails: settings.captureThumbnails,
  });
  const [hasChanges, setHasChanges] = useState(false);

  const handleChange = useCallback(
    (field: keyof HistorySettingsType, value: number | boolean) => {
      setLocalSettings((prev) => {
        const updated = { ...prev, [field]: value };
        setHasChanges(
          updated.maxEntries !== settings.maxEntries ||
            updated.retentionDays !== settings.retentionDays ||
            updated.captureThumbnails !== settings.captureThumbnails
        );
        return updated;
      });
    },
    [settings]
  );

  const handleSave = useCallback(async () => {
    const success = await onUpdate(localSettings);
    if (success) {
      setHasChanges(false);
      onClose();
    }
  }, [localSettings, onUpdate, onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center gap-2 px-4 py-3 border-b border-gray-200 dark:border-gray-700">
          <Settings size={18} className="text-gray-500" />
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">History Settings</h3>
        </div>

        {/* Content */}
        <div className="p-4 space-y-4">
          {/* Max Entries */}
          <div className="space-y-2">
            <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <Database size={14} />
              Maximum Entries
            </label>
            <div className="flex items-center gap-3">
              <input
                type="range"
                min="10"
                max="1000"
                step="10"
                value={localSettings.maxEntries}
                onChange={(e) => handleChange('maxEntries', parseInt(e.target.value, 10))}
                className="flex-1"
              />
              <input
                type="number"
                min="10"
                max="1000"
                value={localSettings.maxEntries}
                onChange={(e) => {
                  const value = parseInt(e.target.value, 10);
                  if (!isNaN(value) && value >= 10 && value <= 1000) {
                    handleChange('maxEntries', value);
                  }
                }}
                className="w-20 px-2 py-1 text-sm text-center border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
              />
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Maximum number of history entries to store. Oldest entries are removed first.
            </p>
          </div>

          {/* Retention Days */}
          <div className="space-y-2">
            <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <Clock size={14} />
              Retention Period (Days)
            </label>
            <div className="flex items-center gap-3">
              <input
                type="range"
                min="0"
                max="365"
                step="1"
                value={localSettings.retentionDays}
                onChange={(e) => handleChange('retentionDays', parseInt(e.target.value, 10))}
                className="flex-1"
              />
              <input
                type="number"
                min="0"
                max="365"
                value={localSettings.retentionDays}
                onChange={(e) => {
                  const value = parseInt(e.target.value, 10);
                  if (!isNaN(value) && value >= 0 && value <= 365) {
                    handleChange('retentionDays', value);
                  }
                }}
                className="w-20 px-2 py-1 text-sm text-center border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
              />
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Entries older than this are automatically removed. Set to 0 for unlimited retention.
            </p>
          </div>

          {/* Capture Thumbnails */}
          <div className="space-y-2">
            <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <Camera size={14} />
              Capture Thumbnails
            </label>
            <div className="flex items-center gap-3">
              <button
                onClick={() => handleChange('captureThumbnails', !localSettings.captureThumbnails)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  localSettings.captureThumbnails
                    ? 'bg-blue-600'
                    : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    localSettings.captureThumbnails ? 'translate-x-6' : 'translate-x-1'
                  }`}
                />
              </button>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {localSettings.captureThumbnails ? 'Enabled' : 'Disabled'}
              </span>
            </div>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              Store thumbnail screenshots for visual history browsing. Disabling reduces storage usage.
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={loading || !hasChanges}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </div>
    </div>
  );
}
