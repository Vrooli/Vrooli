/**
 * WatermarkSettings - Configuration panel for watermark settings
 */

import { useState, useCallback, useEffect } from 'react';
import { Image as ImageIcon, X } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';
import type { WatermarkSettings as WatermarkSettingsType, WatermarkPosition } from '@stores/settingsStore';
import { PositionPicker } from './PositionPicker';
import { AssetPicker } from './AssetPicker';

export function WatermarkSettings() {
  const { replay, setReplaySetting } = useSettingsStore();
  const { watermark } = replay;
  const { assets, isInitialized, initialize } = useAssetStore();

  const [showAssetPicker, setShowAssetPicker] = useState(false);

  // Initialize asset store
  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  // Find selected asset
  const selectedAsset = watermark.assetId ? assets.find((a) => a.id === watermark.assetId) : null;

  // Update a watermark setting
  const updateWatermark = useCallback(
    (updates: Partial<WatermarkSettingsType>) => {
      setReplaySetting('watermark', { ...watermark, ...updates });
    },
    [watermark, setReplaySetting],
  );

  return (
    <div className="space-y-4">
      {/* Enable Toggle */}
      <div className="flex items-center justify-between py-2">
        <div>
          <label className="text-sm text-gray-300 block">Enable Watermark</label>
          <span className="text-xs text-gray-500">Show logo overlay on replays</span>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={watermark.enabled}
          onClick={() => updateWatermark({ enabled: !watermark.enabled })}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
            watermark.enabled ? 'bg-flow-accent' : 'bg-gray-700'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
              watermark.enabled ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
      </div>

      {/* Settings (only show when enabled) */}
      {watermark.enabled && (
        <div className="space-y-4 pt-2 border-t border-gray-800">
          {/* Asset Selection */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Logo Image</label>
            {selectedAsset ? (
              <div className="flex items-center gap-3 p-3 bg-gray-800 rounded-lg border border-gray-700">
                <div className="w-12 h-12 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0">
                  {selectedAsset.thumbnail ? (
                    <img
                      src={selectedAsset.thumbnail}
                      alt={selectedAsset.name}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <ImageIcon size={20} className="text-gray-600" />
                    </div>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <span className="text-sm text-white truncate block">{selectedAsset.name}</span>
                  <span className="text-xs text-gray-500">
                    {selectedAsset.width}x{selectedAsset.height}
                  </span>
                </div>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setShowAssetPicker(true)}
                    className="px-3 py-1.5 text-xs text-flow-accent hover:bg-blue-900/30 rounded-lg transition-colors"
                  >
                    Change
                  </button>
                  <button
                    type="button"
                    onClick={() => updateWatermark({ assetId: null })}
                    className="p-1.5 text-gray-500 hover:text-red-400 hover:bg-red-900/20 rounded-lg transition-colors"
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => setShowAssetPicker(true)}
                className="w-full flex items-center justify-center gap-2 p-4 border-2 border-dashed border-gray-700 rounded-lg text-gray-400 hover:border-gray-600 hover:text-white transition-colors"
              >
                <ImageIcon size={18} />
                <span className="text-sm">Select Logo</span>
              </button>
            )}
          </div>

          {/* Position */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Position</label>
            <div className="flex items-center gap-4">
              <PositionPicker
                value={watermark.position}
                onChange={(position: WatermarkPosition) => updateWatermark({ position })}
              />
              <span className="text-xs text-gray-500 capitalize">
                {watermark.position.replace('-', ' ')}
              </span>
            </div>
          </div>

          {/* Size */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-400">Size</label>
              <span className="text-sm font-medium text-flow-accent">{watermark.size}%</span>
            </div>
            <input
              type="range"
              min={5}
              max={30}
              step={1}
              value={watermark.size}
              onChange={(e) => updateWatermark({ size: parseInt(e.target.value, 10) })}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Small</span>
              <span>Large</span>
            </div>
          </div>

          {/* Opacity */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-400">Opacity</label>
              <span className="text-sm font-medium text-flow-accent">{watermark.opacity}%</span>
            </div>
            <input
              type="range"
              min={10}
              max={100}
              step={5}
              value={watermark.opacity}
              onChange={(e) => updateWatermark({ opacity: parseInt(e.target.value, 10) })}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Subtle</span>
              <span>Full</span>
            </div>
          </div>

          {/* Margin */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-400">Margin</label>
              <span className="text-sm font-medium text-flow-accent">{watermark.margin}px</span>
            </div>
            <input
              type="range"
              min={8}
              max={48}
              step={4}
              value={watermark.margin}
              onChange={(e) => updateWatermark({ margin: parseInt(e.target.value, 10) })}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>Edge</span>
              <span>Inset</span>
            </div>
          </div>
        </div>
      )}

      {/* Asset Picker Modal */}
      <AssetPicker
        isOpen={showAssetPicker}
        onClose={() => setShowAssetPicker(false)}
        onSelect={(assetId) => updateWatermark({ assetId })}
        selectedId={watermark.assetId}
        filterType="logo"
        title="Select Watermark Logo"
      />
    </div>
  );
}

export default WatermarkSettings;
