/**
 * PreviewSettingsHeader Component
 *
 * Minimal header for the Preview Settings panel. Matches BrowserChrome height.
 * Controls are on the left for visual continuity with the main toolbar.
 *
 * Includes:
 * - Close button
 * - Replay presets dropdown
 */

import { useState, useCallback, useMemo } from 'react';
import { X, ChevronDown, Check, Bookmark, Shuffle, Save, Trash2 } from 'lucide-react';
import { useSettingsStore, BUILT_IN_PRESETS } from '@stores/settingsStore';

interface PreviewSettingsHeaderProps {
  /** Callback to close the panel */
  onClose: () => void;
  /** Callback to randomize replay settings */
  onRandomize: () => void;
  /** Callback to save current settings as preset */
  onSavePreset: () => void;
}

export function PreviewSettingsHeader({
  onClose,
  onRandomize,
  onSavePreset,
}: PreviewSettingsHeaderProps) {
  const {
    userPresets,
    activePresetId,
    loadPreset,
    deletePreset,
    getAllPresets,
  } = useSettingsStore();

  const [showPresetDropdown, setShowPresetDropdown] = useState(false);
  const [presetToDelete, setPresetToDelete] = useState<string | null>(null);

  const allPresets = useMemo(() => getAllPresets(), [getAllPresets, userPresets]);

  const activePreset = useMemo(() => {
    if (!activePresetId) return null;
    return allPresets.find((p) => p.id === activePresetId) || null;
  }, [activePresetId, allPresets]);

  const handlePresetSelect = useCallback((presetId: string) => {
    loadPreset(presetId);
    setShowPresetDropdown(false);
  }, [loadPreset]);

  const handleDeletePreset = useCallback((presetId: string) => {
    deletePreset(presetId);
    setPresetToDelete(null);
  }, [deletePreset]);

  return (
    <>
      {/* Header - matches BrowserChrome height */}
      <div className="px-4 py-2.5 border-b border-gray-200 dark:border-gray-700 flex items-center gap-2 min-h-[52px]">
        {/* Close button - on the left to align with settings button position in BrowserChrome */}
        <button
          onClick={onClose}
          className="flex-shrink-0 p-1.5 text-gray-300 hover:text-white bg-gray-800/70 border border-gray-700 rounded-lg transition-colors"
          title="Close settings"
          aria-label="Close settings"
        >
          <X size={16} />
        </button>

        {/* Replay Presets Dropdown */}
        <div className="relative">
          <button
            type="button"
            onClick={() => setShowPresetDropdown(!showPresetDropdown)}
            className="flex items-center gap-2 px-2.5 py-1.5 text-sm bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg text-gray-700 dark:text-gray-200 hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
          >
            <Bookmark size={14} className={activePreset ? 'text-flow-accent' : 'text-gray-500'} />
            <span className="max-w-[100px] truncate">{activePreset ? activePreset.name : 'Presets'}</span>
            <ChevronDown size={14} className={`text-gray-400 transition-transform ${showPresetDropdown ? 'rotate-180' : ''}`} />
          </button>

          {showPresetDropdown && (
            <>
              <div className="fixed inset-0 z-20" onClick={() => setShowPresetDropdown(false)} />
              <div className="absolute left-0 top-full mt-2 z-30 w-64 bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg shadow-xl overflow-hidden max-h-80 overflow-y-auto">
                <div className="px-3 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide bg-gray-50 dark:bg-gray-800/50">
                  Built-in Presets
                </div>
                {BUILT_IN_PRESETS.map((preset) => (
                  <button
                    key={preset.id}
                    type="button"
                    onClick={() => handlePresetSelect(preset.id)}
                    className={`w-full flex items-center gap-3 px-4 py-2.5 text-left text-sm hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors ${
                      activePresetId === preset.id ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300' : 'text-gray-700 dark:text-gray-200'
                    }`}
                  >
                    {activePresetId === preset.id ? <Check size={14} className="text-blue-500" /> : <Bookmark size={14} className="text-gray-400" />}
                    <span>{preset.name}</span>
                  </button>
                ))}

                {userPresets.length > 0 && (
                  <>
                    <div className="px-3 py-2 text-xs font-medium text-gray-500 uppercase tracking-wide bg-gray-50 dark:bg-gray-800/50 border-t border-gray-200 dark:border-gray-700">
                      Your Presets
                    </div>
                    {userPresets.map((preset) => (
                      <div
                        key={preset.id}
                        className={`flex items-center hover:bg-gray-50 dark:hover:bg-gray-800 ${
                          activePresetId === preset.id ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                        }`}
                      >
                        <button
                          type="button"
                          onClick={() => handlePresetSelect(preset.id)}
                          className={`flex-1 flex items-center gap-3 px-4 py-2.5 text-left text-sm ${
                            activePresetId === preset.id ? 'text-blue-700 dark:text-blue-300' : 'text-gray-700 dark:text-gray-200'
                          }`}
                        >
                          {activePresetId === preset.id ? <Check size={14} className="text-blue-500" /> : <Bookmark size={14} className="text-gray-400" />}
                          <span>{preset.name}</span>
                        </button>
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowPresetDropdown(false);
                            setPresetToDelete(preset.id);
                          }}
                          className="p-2 mr-2 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                          title="Delete preset"
                        >
                          <Trash2 size={14} />
                        </button>
                      </div>
                    ))}
                  </>
                )}

                <div className="border-t border-gray-200 dark:border-gray-700 p-2 bg-gray-50 dark:bg-gray-800/30">
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => { setShowPresetDropdown(false); onRandomize(); }}
                      className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-900/20 rounded-lg transition-colors"
                    >
                      <Shuffle size={12} />
                      Random
                    </button>
                    <button
                      type="button"
                      onClick={() => { setShowPresetDropdown(false); onSavePreset(); }}
                      className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors"
                    >
                      <Save size={12} />
                      Save
                    </button>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Delete Preset Confirmation Modal */}
      {presetToDelete && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
              <Trash2 size={20} className="text-red-500" />
              Delete Preset
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              Are you sure you want to delete this preset? This action cannot be undone.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setPresetToDelete(null)}
                className="flex-1 px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleDeletePreset(presetToDelete)}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors flex items-center justify-center gap-2"
              >
                <Trash2 size={16} />
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
