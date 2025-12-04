/**
 * OutroCardSettings - Configuration panel for outro card
 */

import { useState, useCallback, useEffect } from 'react';
import { Image as ImageIcon, X } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';
import type { OutroCardSettings as OutroCardSettingsType } from '@stores/settingsStore';
import { AssetPicker } from './AssetPicker';

export function OutroCardSettings() {
  const { replay, setReplaySetting } = useSettingsStore();
  const { outroCard } = replay;
  const { assets, isInitialized, initialize } = useAssetStore();

  const [showLogoPicker, setShowLogoPicker] = useState(false);
  const [showBackgroundPicker, setShowBackgroundPicker] = useState(false);

  // Initialize asset store
  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  // Find selected assets
  const logoAsset = outroCard.logoAssetId ? assets.find((a) => a.id === outroCard.logoAssetId) : null;
  const backgroundAsset = outroCard.backgroundAssetId
    ? assets.find((a) => a.id === outroCard.backgroundAssetId)
    : null;

  // Update outro card setting
  const updateOutroCard = useCallback(
    (updates: Partial<OutroCardSettingsType>) => {
      setReplaySetting('outroCard', { ...outroCard, ...updates });
    },
    [outroCard, setReplaySetting],
  );

  return (
    <div className="space-y-4">
      {/* Enable Toggle */}
      <div className="flex items-center justify-between py-2">
        <div>
          <label className="text-sm text-gray-300 block">Enable Outro Card</label>
          <span className="text-xs text-gray-500">Show a closing slide after the replay ends</span>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={outroCard.enabled}
          onClick={() => updateOutroCard({ enabled: !outroCard.enabled })}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
            outroCard.enabled ? 'bg-flow-accent' : 'bg-gray-700'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
              outroCard.enabled ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
      </div>

      {/* Settings (only show when enabled) */}
      {outroCard.enabled && (
        <div className="space-y-4 pt-2 border-t border-gray-800">
          {/* Title */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Title</label>
            <input
              type="text"
              value={outroCard.title}
              onChange={(e) => updateOutroCard({ title: e.target.value })}
              placeholder="Thanks for watching!"
              className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
          </div>

          {/* CTA Text */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">CTA Button Text</label>
            <input
              type="text"
              value={outroCard.ctaText}
              onChange={(e) => updateOutroCard({ ctaText: e.target.value })}
              placeholder="Learn More"
              className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
          </div>

          {/* CTA URL */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">CTA Button URL</label>
            <input
              type="url"
              value={outroCard.ctaUrl}
              onChange={(e) => updateOutroCard({ ctaUrl: e.target.value })}
              placeholder="https://example.com"
              className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
            <p className="text-xs text-gray-500 mt-1">Leave empty to show text without a link</p>
          </div>

          {/* Logo Selection */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Logo (optional)</label>
            {logoAsset ? (
              <div className="flex items-center gap-3 p-3 bg-gray-800 rounded-lg border border-gray-700">
                <div className="w-10 h-10 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0">
                  {logoAsset.thumbnail ? (
                    <img src={logoAsset.thumbnail} alt={logoAsset.name} className="w-full h-full object-cover" />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <ImageIcon size={16} className="text-gray-600" />
                    </div>
                  )}
                </div>
                <span className="flex-1 text-sm text-white truncate">{logoAsset.name}</span>
                <button
                  type="button"
                  onClick={() => setShowLogoPicker(true)}
                  className="px-2 py-1 text-xs text-flow-accent hover:bg-blue-900/30 rounded transition-colors"
                >
                  Change
                </button>
                <button
                  type="button"
                  onClick={() => updateOutroCard({ logoAssetId: null })}
                  className="p-1 text-gray-500 hover:text-red-400 rounded transition-colors"
                >
                  <X size={14} />
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => setShowLogoPicker(true)}
                className="w-full flex items-center justify-center gap-2 p-3 border border-dashed border-gray-700 rounded-lg text-gray-400 hover:border-gray-600 hover:text-white transition-colors"
              >
                <ImageIcon size={16} />
                <span className="text-sm">Select Logo</span>
              </button>
            )}
          </div>

          {/* Background Selection */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Background Image (optional)</label>
            {backgroundAsset ? (
              <div className="flex items-center gap-3 p-3 bg-gray-800 rounded-lg border border-gray-700">
                <div className="w-10 h-10 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0">
                  {backgroundAsset.thumbnail ? (
                    <img
                      src={backgroundAsset.thumbnail}
                      alt={backgroundAsset.name}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <ImageIcon size={16} className="text-gray-600" />
                    </div>
                  )}
                </div>
                <span className="flex-1 text-sm text-white truncate">{backgroundAsset.name}</span>
                <button
                  type="button"
                  onClick={() => setShowBackgroundPicker(true)}
                  className="px-2 py-1 text-xs text-flow-accent hover:bg-blue-900/30 rounded transition-colors"
                >
                  Change
                </button>
                <button
                  type="button"
                  onClick={() => updateOutroCard({ backgroundAssetId: null })}
                  className="p-1 text-gray-500 hover:text-red-400 rounded transition-colors"
                >
                  <X size={14} />
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => setShowBackgroundPicker(true)}
                className="w-full flex items-center justify-center gap-2 p-3 border border-dashed border-gray-700 rounded-lg text-gray-400 hover:border-gray-600 hover:text-white transition-colors"
              >
                <ImageIcon size={16} />
                <span className="text-sm">Select Background</span>
              </button>
            )}
          </div>

          {/* Background Color (fallback) */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Background Color</label>
            <div className="flex items-center gap-3">
              <input
                type="color"
                value={outroCard.backgroundColor}
                onChange={(e) => updateOutroCard({ backgroundColor: e.target.value })}
                className="w-10 h-10 rounded-lg border border-gray-700 cursor-pointer"
              />
              <input
                type="text"
                value={outroCard.backgroundColor}
                onChange={(e) => updateOutroCard({ backgroundColor: e.target.value })}
                className="flex-1 px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white font-mono text-sm"
              />
            </div>
          </div>

          {/* Text Color */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Text Color</label>
            <div className="flex items-center gap-3">
              <input
                type="color"
                value={outroCard.textColor}
                onChange={(e) => updateOutroCard({ textColor: e.target.value })}
                className="w-10 h-10 rounded-lg border border-gray-700 cursor-pointer"
              />
              <input
                type="text"
                value={outroCard.textColor}
                onChange={(e) => updateOutroCard({ textColor: e.target.value })}
                className="flex-1 px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white font-mono text-sm"
              />
            </div>
          </div>

          {/* Duration */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-400">Duration</label>
              <span className="text-sm font-medium text-flow-accent">{(outroCard.duration / 1000).toFixed(1)}s</span>
            </div>
            <input
              type="range"
              min={2000}
              max={8000}
              step={500}
              value={outroCard.duration}
              onChange={(e) => updateOutroCard({ duration: parseInt(e.target.value, 10) })}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>2s</span>
              <span>8s</span>
            </div>
          </div>
        </div>
      )}

      {/* Asset Pickers */}
      <AssetPicker
        isOpen={showLogoPicker}
        onClose={() => setShowLogoPicker(false)}
        onSelect={(assetId) => updateOutroCard({ logoAssetId: assetId })}
        selectedId={outroCard.logoAssetId}
        filterType="logo"
        title="Select Outro Logo"
      />

      <AssetPicker
        isOpen={showBackgroundPicker}
        onClose={() => setShowBackgroundPicker(false)}
        onSelect={(assetId) => updateOutroCard({ backgroundAssetId: assetId })}
        selectedId={outroCard.backgroundAssetId}
        filterType="background"
        title="Select Outro Background"
      />
    </div>
  );
}

export default OutroCardSettings;
