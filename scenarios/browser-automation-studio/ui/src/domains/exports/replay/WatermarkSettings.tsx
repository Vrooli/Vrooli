/**
 * WatermarkSettings - Configuration panel for watermark settings
 *
 * Integrates with entitlements:
 * - Free/Solo tiers: Watermark always on, locked to Vrooli Ascension logo
 * - Pro+ tiers: Can disable watermark or use custom logos
 */

import { useState, useCallback, useEffect, useMemo } from 'react';
import { Image as ImageIcon, X, Lock, Crown, ArrowRight, Sparkles } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import type { WatermarkSettings as WatermarkSettingsType, WatermarkPosition } from '@stores/settingsStore';
import { PositionPicker } from './PositionPicker';
import { AssetPicker } from './AssetPicker';
import {
  BUILTIN_ASSET_IDS,
  isBuiltInAssetId,
  getBuiltInAsset,
} from '@lib/builtInAssets';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

export function WatermarkSettings() {
  const { replay, setReplaySetting } = useSettingsStore();
  const { watermark } = replay;
  const { assets, isInitialized, initialize } = useAssetStore();
  const { status, userEmail } = useEntitlementStore();

  const [showAssetPicker, setShowAssetPicker] = useState(false);

  // Initialize asset store
  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  // Entitlement checks
  const entitlementsEnabled = status?.entitlements_enabled ?? false;
  const requiresWatermark = entitlementsEnabled && (status?.requires_watermark ?? true);
  const currentTier = status?.tier ?? 'free';

  // Determine if user can customize watermark
  const canDisableWatermark = !requiresWatermark;
  const canUseCustomLogo = !requiresWatermark;

  // The minimum tier needed to remove watermark (Pro)
  const minTierForCustomization = 'pro' as const;
  const tierConfig = TIER_CONFIG[minTierForCustomization];

  // Find selected asset (could be built-in or user asset)
  const selectedAsset = useMemo(() => {
    if (!watermark.assetId) return null;

    // Check if it's a built-in asset
    if (isBuiltInAssetId(watermark.assetId)) {
      const builtIn = getBuiltInAsset(watermark.assetId);
      if (builtIn) {
        return {
          id: builtIn.id,
          name: builtIn.name,
          thumbnail: builtIn.url,
          width: builtIn.width,
          height: builtIn.height,
          isBuiltIn: true,
        };
      }
    }

    // Check user assets
    const userAsset = assets.find((a) => a.id === watermark.assetId);
    if (userAsset) {
      return {
        id: userAsset.id,
        name: userAsset.name,
        thumbnail: userAsset.thumbnail,
        width: userAsset.width,
        height: userAsset.height,
        isBuiltIn: false,
      };
    }

    return null;
  }, [watermark.assetId, assets]);

  // Get the Vrooli Ascension asset for display
  const vrooliAsset = useMemo(() => {
    const asset = getBuiltInAsset(BUILTIN_ASSET_IDS.VROOLI_ASCENSION);
    return asset
      ? {
          id: asset.id,
          name: asset.name,
          thumbnail: asset.url,
          width: asset.width,
          height: asset.height,
        }
      : null;
  }, []);

  // Update a watermark setting
  const updateWatermark = useCallback(
    (updates: Partial<WatermarkSettingsType>) => {
      setReplaySetting('watermark', { ...watermark, ...updates });
    },
    [watermark, setReplaySetting],
  );

  // Handle toggling watermark (respecting entitlements)
  const handleToggleWatermark = useCallback(() => {
    if (!canDisableWatermark && !watermark.enabled) {
      // Can't enable if they want to disable and don't have permission
      return;
    }
    if (!canDisableWatermark && watermark.enabled) {
      // Can't disable - show upgrade prompt instead
      return;
    }
    updateWatermark({ enabled: !watermark.enabled });
  }, [canDisableWatermark, watermark.enabled, updateWatermark]);

  // Handle asset selection
  const handleAssetSelect = useCallback(
    (assetId: string | null) => {
      // If restricted and trying to select non-built-in, force Vrooli logo
      if (!canUseCustomLogo && assetId && !isBuiltInAssetId(assetId)) {
        updateWatermark({ assetId: BUILTIN_ASSET_IDS.VROOLI_ASCENSION });
        return;
      }
      updateWatermark({ assetId });
    },
    [canUseCustomLogo, updateWatermark],
  );

  // Enforce watermark constraints for restricted tiers
  useEffect(() => {
    if (requiresWatermark) {
      // Ensure watermark is enabled
      if (!watermark.enabled) {
        updateWatermark({ enabled: true });
      }
      // Ensure using Vrooli logo (or allow built-in assets)
      if (watermark.assetId && !isBuiltInAssetId(watermark.assetId)) {
        updateWatermark({ assetId: BUILTIN_ASSET_IDS.VROOLI_ASCENSION });
      }
      // If no asset selected, set Vrooli logo
      if (!watermark.assetId) {
        updateWatermark({ assetId: BUILTIN_ASSET_IDS.VROOLI_ASCENSION });
      }
    }
  }, [requiresWatermark, watermark.enabled, watermark.assetId, updateWatermark]);

  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  // If entitlements are disabled, show full controls
  if (!entitlementsEnabled) {
    return <UnrestrictedWatermarkSettings />;
  }

  return (
    <div className="space-y-4">
      {/* Enable Toggle - locked for free/solo tiers */}
      <div className="flex items-center justify-between py-2">
        <div className="flex-1">
          <label className="text-sm text-gray-300 block">Enable Watermark</label>
          <span className="text-xs text-gray-500">Show logo overlay on replays</span>
        </div>
        <div className="flex items-center gap-3">
          {!canDisableWatermark && (
            <span className="flex items-center gap-1 text-xs text-amber-400/80">
              <Lock size={10} />
              Required
            </span>
          )}
          <button
            type="button"
            role="switch"
            aria-checked={watermark.enabled}
            disabled={!canDisableWatermark}
            onClick={handleToggleWatermark}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
              watermark.enabled ? 'bg-flow-accent' : 'bg-gray-700'
            } ${!canDisableWatermark ? 'opacity-60 cursor-not-allowed' : ''}`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
                watermark.enabled ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
        </div>
      </div>

      {/* Upgrade banner for restricted tiers */}
      {requiresWatermark && (
        <div className="rounded-xl border border-amber-500/30 bg-gradient-to-r from-amber-950/30 to-amber-900/20 p-4">
          <div className="flex items-start gap-3">
            <div className="p-2 rounded-lg bg-amber-500/10">
              <Crown size={18} className="text-amber-400" />
            </div>
            <div className="flex-1">
              <h4 className="text-sm font-medium text-amber-200">
                Watermark required on {TIER_CONFIG[currentTier]?.label ?? currentTier} plan
              </h4>
              <p className="text-xs text-amber-200/60 mt-1">
                Upgrade to {tierConfig.label} to remove watermarks or use your own logo.
              </p>
              <a
                href={upgradeUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 mt-3 px-3 py-1.5 text-xs font-medium text-amber-900 bg-gradient-to-r from-amber-400 to-amber-500 hover:from-amber-300 hover:to-amber-400 rounded-lg transition-all"
              >
                Upgrade to {tierConfig.label}
                <ArrowRight size={12} />
              </a>
            </div>
          </div>
        </div>
      )}

      {/* Settings (always show when watermark is enabled) */}
      {watermark.enabled && (
        <div className="space-y-4 pt-2 border-t border-gray-800">
          {/* Asset Selection */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm text-gray-400">Logo Image</label>
              {!canUseCustomLogo && (
                <span className="flex items-center gap-1 text-xs text-amber-400/80">
                  <Lock size={10} />
                  {tierConfig.label}+ for custom
                </span>
              )}
            </div>

            {/* Show selected asset or selection prompt */}
            {selectedAsset ? (
              <div
                className={`flex items-center gap-3 p-3 rounded-lg border ${
                  selectedAsset.isBuiltIn
                    ? 'bg-amber-950/20 border-amber-500/30'
                    : 'bg-gray-800 border-gray-700'
                }`}
              >
                <div className="w-12 h-12 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0 flex items-center justify-center">
                  {selectedAsset.thumbnail ? (
                    <img
                      src={selectedAsset.thumbnail}
                      alt={selectedAsset.name}
                      className="w-full h-full object-contain p-1"
                    />
                  ) : (
                    <ImageIcon size={20} className="text-gray-600" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-white truncate">{selectedAsset.name}</span>
                    {selectedAsset.isBuiltIn && (
                      <span className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                        <Sparkles size={8} />
                        Built-in
                      </span>
                    )}
                  </div>
                  <span className="text-xs text-gray-500">
                    {selectedAsset.width}x{selectedAsset.height}
                  </span>
                </div>
                <div className="flex gap-2">
                  {canUseCustomLogo ? (
                    <>
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
                    </>
                  ) : (
                    <span className="px-3 py-1.5 text-xs text-gray-500">
                      Default logo
                    </span>
                  )}
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

            {/* Vrooli logo quick-select for all tiers */}
            {!selectedAsset?.isBuiltIn && vrooliAsset && (
              <button
                type="button"
                onClick={() => updateWatermark({ assetId: BUILTIN_ASSET_IDS.VROOLI_ASCENSION })}
                className="mt-2 w-full flex items-center justify-center gap-2 p-2 text-xs text-amber-400/80 hover:text-amber-400 hover:bg-amber-950/20 rounded-lg transition-colors"
              >
                <Sparkles size={12} />
                Use Vrooli Ascension logo (always available)
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
        onSelect={handleAssetSelect}
        selectedId={watermark.assetId}
        filterType="logo"
        title={canUseCustomLogo ? 'Select Watermark Logo' : 'Select Logo (Built-in Only)'}
        builtInOnly={!canUseCustomLogo}
        recommendedAssetId={BUILTIN_ASSET_IDS.VROOLI_ASCENSION}
      />
    </div>
  );
}

/**
 * Unrestricted watermark settings (when entitlements are disabled)
 * This is the original behavior - full control over watermark settings
 */
function UnrestrictedWatermarkSettings() {
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
  const selectedAsset = useMemo(() => {
    if (!watermark.assetId) return null;

    // Check if it's a built-in asset
    if (isBuiltInAssetId(watermark.assetId)) {
      const builtIn = getBuiltInAsset(watermark.assetId);
      if (builtIn) {
        return {
          id: builtIn.id,
          name: builtIn.name,
          thumbnail: builtIn.url,
          width: builtIn.width,
          height: builtIn.height,
          isBuiltIn: true,
        };
      }
    }

    // Check user assets
    const userAsset = assets.find((a) => a.id === watermark.assetId);
    if (userAsset) {
      return {
        id: userAsset.id,
        name: userAsset.name,
        thumbnail: userAsset.thumbnail,
        width: userAsset.width,
        height: userAsset.height,
        isBuiltIn: false,
      };
    }

    return null;
  }, [watermark.assetId, assets]);

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
              <div
                className={`flex items-center gap-3 p-3 rounded-lg border ${
                  selectedAsset.isBuiltIn
                    ? 'bg-amber-950/20 border-amber-500/30'
                    : 'bg-gray-800 border-gray-700'
                }`}
              >
                <div className="w-12 h-12 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0 flex items-center justify-center">
                  {selectedAsset.thumbnail ? (
                    <img
                      src={selectedAsset.thumbnail}
                      alt={selectedAsset.name}
                      className="w-full h-full object-contain p-1"
                    />
                  ) : (
                    <ImageIcon size={20} className="text-gray-600" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-white truncate">{selectedAsset.name}</span>
                    {selectedAsset.isBuiltIn && (
                      <span className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                        <Sparkles size={8} />
                        Built-in
                      </span>
                    )}
                  </div>
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

            {/* Vrooli logo quick-select */}
            {!selectedAsset?.isBuiltIn && (
              <button
                type="button"
                onClick={() => updateWatermark({ assetId: BUILTIN_ASSET_IDS.VROOLI_ASCENSION })}
                className="mt-2 w-full flex items-center justify-center gap-2 p-2 text-xs text-amber-400/80 hover:text-amber-400 hover:bg-amber-950/20 rounded-lg transition-colors"
              >
                <Sparkles size={12} />
                Use Vrooli Ascension logo
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
        recommendedAssetId={BUILTIN_ASSET_IDS.VROOLI_ASCENSION}
      />
    </div>
  );
}

export default WatermarkSettings;
