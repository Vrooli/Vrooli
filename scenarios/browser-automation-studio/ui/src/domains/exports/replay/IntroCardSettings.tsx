/**
 * IntroCardSettings - Configuration panel for intro card
 */

import { useState, useCallback, useEffect } from 'react';
import { Image as ImageIcon, X } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';
import type { IntroCardSettings as IntroCardSettingsType } from '@stores/settingsStore';
import { AssetPicker } from './AssetPicker';

export function IntroCardSettings() {
  const { replay, setReplaySetting } = useSettingsStore();
  const { introCard } = replay;
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
  const logoAsset = introCard.logoAssetId ? assets.find((a) => a.id === introCard.logoAssetId) : null;
  const backgroundAsset = introCard.backgroundAssetId
    ? assets.find((a) => a.id === introCard.backgroundAssetId)
    : null;

  // Update intro card setting
  const updateIntroCard = useCallback(
    (updates: Partial<IntroCardSettingsType>) => {
      setReplaySetting('introCard', { ...introCard, ...updates });
    },
    [introCard, setReplaySetting],
  );

  return (
    <div className="space-y-4">
      {/* Enable Toggle */}
      <div className="flex items-center justify-between py-2">
        <div>
          <label className="text-sm text-gray-300 block">Enable Intro Card</label>
          <span className="text-xs text-gray-500">Show a title slide before the replay starts</span>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={introCard.enabled}
          onClick={() => updateIntroCard({ enabled: !introCard.enabled })}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
            introCard.enabled ? 'bg-flow-accent' : 'bg-gray-700'
          }`}
        >
          <span
            className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
              introCard.enabled ? 'translate-x-6' : 'translate-x-1'
            }`}
          />
        </button>
      </div>

      {/* Settings (only show when enabled) */}
      {introCard.enabled && (
        <div className="space-y-4 pt-2 border-t border-gray-800">
          {/* Title */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Title</label>
            <input
              type="text"
              value={introCard.title}
              onChange={(e) => updateIntroCard({ title: e.target.value })}
              placeholder="Welcome to our demo"
              className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
          </div>

          {/* Subtitle */}
          <div>
            <label className="text-sm text-gray-400 block mb-2">Subtitle</label>
            <input
              type="text"
              value={introCard.subtitle}
              onChange={(e) => updateIntroCard({ subtitle: e.target.value })}
              placeholder="A quick walkthrough of our features"
              className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
            />
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
                  onClick={() => updateIntroCard({ logoAssetId: null })}
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
                  onClick={() => updateIntroCard({ backgroundAssetId: null })}
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
                value={introCard.backgroundColor}
                onChange={(e) => updateIntroCard({ backgroundColor: e.target.value })}
                className="w-10 h-10 rounded-lg border border-gray-700 cursor-pointer"
              />
              <input
                type="text"
                value={introCard.backgroundColor}
                onChange={(e) => updateIntroCard({ backgroundColor: e.target.value })}
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
                value={introCard.textColor}
                onChange={(e) => updateIntroCard({ textColor: e.target.value })}
                className="w-10 h-10 rounded-lg border border-gray-700 cursor-pointer"
              />
              <input
                type="text"
                value={introCard.textColor}
                onChange={(e) => updateIntroCard({ textColor: e.target.value })}
                className="flex-1 px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white font-mono text-sm"
              />
            </div>
          </div>

          {/* Duration */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-400">Duration</label>
              <span className="text-sm font-medium text-flow-accent">{(introCard.duration / 1000).toFixed(1)}s</span>
            </div>
            <input
              type="range"
              min={1000}
              max={5000}
              step={500}
              value={introCard.duration}
              onChange={(e) => updateIntroCard({ duration: parseInt(e.target.value, 10) })}
              className="w-full accent-flow-accent"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>1s</span>
              <span>5s</span>
            </div>
          </div>
        </div>
      )}

      {/* Asset Pickers */}
      <AssetPicker
        isOpen={showLogoPicker}
        onClose={() => setShowLogoPicker(false)}
        onSelect={(assetId) => updateIntroCard({ logoAssetId: assetId })}
        selectedId={introCard.logoAssetId}
        filterType="logo"
        title="Select Intro Logo"
      />

      <AssetPicker
        isOpen={showBackgroundPicker}
        onClose={() => setShowBackgroundPicker(false)}
        onSelect={(assetId) => updateIntroCard({ backgroundAssetId: assetId })}
        selectedId={introCard.backgroundAssetId}
        filterType="background"
        title="Select Intro Background"
      />
    </div>
  );
}

export default IntroCardSettings;
