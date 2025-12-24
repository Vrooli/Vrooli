/**
 * OutroCardSettings - Configuration panel for outro card
 *
 * Entitlement gating:
 * - Free tier: Feature locked, shows upgrade prompt
 * - Solo+ tiers: Full access to outro cards
 */

import { useState, useCallback, useEffect, useMemo } from 'react';
import { Image as ImageIcon, X, Lock, Crown, ArrowRight, Sparkles } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import { useAssetStore } from '@stores/assetStore';
import { useEntitlementStore, TIER_CONFIG } from '@stores/entitlementStore';
import type { OutroCardSettings as OutroCardSettingsType } from '@stores/settingsStore';
import { AssetPicker } from './AssetPicker';
import { isBuiltInAssetId, getBuiltInAsset, BUILTIN_ASSET_IDS } from '@lib/builtInAssets';

// Get landing page URL from environment or use default
const LANDING_PAGE_URL = import.meta.env.VITE_LANDING_PAGE_URL || 'https://browser-automation-studio.com';

// Minimum tier required for outro cards
const REQUIRED_TIER = 'solo' as const;

export function OutroCardSettings() {
  const { replay, setReplaySetting } = useSettingsStore();
  const { outroCard } = replay;
  const { assets, isInitialized, initialize } = useAssetStore();
  const { status, userEmail } = useEntitlementStore();

  const [showLogoPicker, setShowLogoPicker] = useState(false);
  const [showBackgroundPicker, setShowBackgroundPicker] = useState(false);

  // Initialize asset store
  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  // Entitlement checks
  const entitlementsEnabled = status?.entitlements_enabled ?? false;
  const currentTier = status?.tier ?? 'free';

  // Check if user has access to outro cards (solo+ tiers)
  const hasAccess = useMemo(() => {
    if (!entitlementsEnabled) return true; // No restrictions when entitlements disabled
    const tierOrder = ['free', 'solo', 'pro', 'studio', 'business'];
    const currentIndex = tierOrder.indexOf(currentTier);
    const requiredIndex = tierOrder.indexOf(REQUIRED_TIER);
    return currentIndex >= requiredIndex;
  }, [entitlementsEnabled, currentTier]);

  const tierConfig = TIER_CONFIG[REQUIRED_TIER];

  // Find selected assets (handle both user and built-in assets)
  const logoAsset = useMemo(() => {
    if (!outroCard.logoAssetId) return null;
    if (isBuiltInAssetId(outroCard.logoAssetId)) {
      const builtIn = getBuiltInAsset(outroCard.logoAssetId);
      return builtIn ? { ...builtIn, thumbnail: builtIn.url, isBuiltIn: true } : null;
    }
    const asset = assets.find((a) => a.id === outroCard.logoAssetId);
    return asset ? { ...asset, isBuiltIn: false } : null;
  }, [outroCard.logoAssetId, assets]);

  const backgroundAsset = useMemo(() => {
    if (!outroCard.backgroundAssetId) return null;
    if (isBuiltInAssetId(outroCard.backgroundAssetId)) {
      const builtIn = getBuiltInAsset(outroCard.backgroundAssetId);
      return builtIn ? { ...builtIn, thumbnail: builtIn.url, isBuiltIn: true } : null;
    }
    const asset = assets.find((a) => a.id === outroCard.backgroundAssetId);
    return asset ? { ...asset, isBuiltIn: false } : null;
  }, [outroCard.backgroundAssetId, assets]);

  // Update outro card setting
  const updateOutroCard = useCallback(
    (updates: Partial<OutroCardSettingsType>) => {
      setReplaySetting('outroCard', { ...outroCard, ...updates });
    },
    [outroCard, setReplaySetting],
  );

  // Enforce disabled state for restricted tiers
  useEffect(() => {
    if (!hasAccess && outroCard.enabled) {
      updateOutroCard({ enabled: false });
    }
  }, [hasAccess, outroCard.enabled, updateOutroCard]);

  const upgradeUrl = userEmail
    ? `${LANDING_PAGE_URL}/pricing?email=${encodeURIComponent(userEmail)}`
    : `${LANDING_PAGE_URL}/pricing`;

  // If entitlements disabled, show unrestricted UI
  if (!entitlementsEnabled) {
    return <UnrestrictedOutroCardSettings />;
  }

  return (
    <div className="space-y-4">
      {/* Enable Toggle */}
      <div className="flex items-center justify-between py-2">
        <div className="flex-1">
          <label className="text-sm text-gray-300 block">Enable Outro Card</label>
          <span className="text-xs text-gray-500">Show a closing slide after the replay ends</span>
        </div>
        <div className="flex items-center gap-3">
          {!hasAccess && (
            <span className="flex items-center gap-1 text-xs text-amber-400/80">
              <Lock size={10} />
              {tierConfig.label}+
            </span>
          )}
          <button
            type="button"
            role="switch"
            aria-checked={outroCard.enabled}
            disabled={!hasAccess}
            onClick={() => hasAccess && updateOutroCard({ enabled: !outroCard.enabled })}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-flow-accent/50 ${
              outroCard.enabled ? 'bg-flow-accent' : 'bg-gray-700'
            } ${!hasAccess ? 'opacity-60 cursor-not-allowed' : ''}`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
                outroCard.enabled ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
        </div>
      </div>

      {/* Upgrade banner for restricted tiers */}
      {!hasAccess && (
        <div className="rounded-xl border border-amber-500/30 bg-gradient-to-r from-amber-950/30 to-amber-900/20 p-4">
          <div className="flex items-start gap-3">
            <div className="p-2 rounded-lg bg-amber-500/10">
              <Crown size={18} className="text-amber-400" />
            </div>
            <div className="flex-1">
              <h4 className="text-sm font-medium text-amber-200">
                Outro Cards require {tierConfig.label}
              </h4>
              <p className="text-xs text-amber-200/60 mt-1">
                Add professional closing slides with CTAs to your replays.
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

      {/* Settings (only show when enabled and has access) */}
      {outroCard.enabled && hasAccess && (
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
              <div
                className={`flex items-center gap-3 p-3 rounded-lg border ${
                  logoAsset.isBuiltIn ? 'bg-amber-950/20 border-amber-500/30' : 'bg-gray-800 border-gray-700'
                }`}
              >
                <div className="w-10 h-10 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0 flex items-center justify-center">
                  {logoAsset.thumbnail ? (
                    <img
                      src={logoAsset.thumbnail}
                      alt={logoAsset.name}
                      className="w-full h-full object-contain p-1"
                    />
                  ) : (
                    <ImageIcon size={16} className="text-gray-600" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-white truncate">{logoAsset.name}</span>
                    {logoAsset.isBuiltIn && (
                      <span className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                        <Sparkles size={8} />
                        Built-in
                      </span>
                    )}
                  </div>
                </div>
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
              <div
                className={`flex items-center gap-3 p-3 rounded-lg border ${
                  backgroundAsset.isBuiltIn ? 'bg-amber-950/20 border-amber-500/30' : 'bg-gray-800 border-gray-700'
                }`}
              >
                <div className="w-10 h-10 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0">
                  {backgroundAsset.thumbnail ? (
                    <img
                      src={backgroundAsset.thumbnail}
                      alt={backgroundAsset.name}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <ImageIcon size={16} className="text-gray-600" />
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
        recommendedAssetId={BUILTIN_ASSET_IDS.VROOLI_ASCENSION}
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

/**
 * Unrestricted outro card settings (when entitlements are disabled)
 */
function UnrestrictedOutroCardSettings() {
  const { replay, setReplaySetting } = useSettingsStore();
  const { outroCard } = replay;
  const { assets, isInitialized, initialize } = useAssetStore();

  const [showLogoPicker, setShowLogoPicker] = useState(false);
  const [showBackgroundPicker, setShowBackgroundPicker] = useState(false);

  useEffect(() => {
    if (!isInitialized) {
      initialize();
    }
  }, [isInitialized, initialize]);

  const logoAsset = useMemo(() => {
    if (!outroCard.logoAssetId) return null;
    if (isBuiltInAssetId(outroCard.logoAssetId)) {
      const builtIn = getBuiltInAsset(outroCard.logoAssetId);
      return builtIn ? { ...builtIn, thumbnail: builtIn.url, isBuiltIn: true } : null;
    }
    const asset = assets.find((a) => a.id === outroCard.logoAssetId);
    return asset ? { ...asset, isBuiltIn: false } : null;
  }, [outroCard.logoAssetId, assets]);

  const backgroundAsset = useMemo(() => {
    if (!outroCard.backgroundAssetId) return null;
    if (isBuiltInAssetId(outroCard.backgroundAssetId)) {
      const builtIn = getBuiltInAsset(outroCard.backgroundAssetId);
      return builtIn ? { ...builtIn, thumbnail: builtIn.url, isBuiltIn: true } : null;
    }
    const asset = assets.find((a) => a.id === outroCard.backgroundAssetId);
    return asset ? { ...asset, isBuiltIn: false } : null;
  }, [outroCard.backgroundAssetId, assets]);

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
              <div
                className={`flex items-center gap-3 p-3 rounded-lg border ${
                  logoAsset.isBuiltIn ? 'bg-amber-950/20 border-amber-500/30' : 'bg-gray-800 border-gray-700'
                }`}
              >
                <div className="w-10 h-10 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0 flex items-center justify-center">
                  {logoAsset.thumbnail ? (
                    <img
                      src={logoAsset.thumbnail}
                      alt={logoAsset.name}
                      className="w-full h-full object-contain p-1"
                    />
                  ) : (
                    <ImageIcon size={16} className="text-gray-600" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-white truncate">{logoAsset.name}</span>
                    {logoAsset.isBuiltIn && (
                      <span className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium bg-amber-500/20 text-amber-400 rounded">
                        <Sparkles size={8} />
                        Built-in
                      </span>
                    )}
                  </div>
                </div>
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
              <div
                className={`flex items-center gap-3 p-3 rounded-lg border ${
                  backgroundAsset.isBuiltIn ? 'bg-amber-950/20 border-amber-500/30' : 'bg-gray-800 border-gray-700'
                }`}
              >
                <div className="w-10 h-10 rounded-lg overflow-hidden bg-gray-900 flex-shrink-0">
                  {backgroundAsset.thumbnail ? (
                    <img
                      src={backgroundAsset.thumbnail}
                      alt={backgroundAsset.name}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <ImageIcon size={16} className="text-gray-600" />
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

          {/* Background Color */}
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
        recommendedAssetId={BUILTIN_ASSET_IDS.VROOLI_ASCENSION}
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
