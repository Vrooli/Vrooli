import { useCallback, useEffect, useState } from 'react';
import type {
  BrowserProfile,
  ProfilePreset,
  FingerprintSettings,
  BehaviorSettings,
  AntiDetectionSettings,
  UserAgentPreset,
  MouseMovementStyle,
  ScrollStyle,
} from '@/domains/recording/types/types';

interface BrowserProfileEditorProps {
  profileName: string;
  initialProfile?: BrowserProfile;
  onSave: (profile: BrowserProfile) => Promise<void>;
  onClose: () => void;
}

// Preset configurations
const PRESET_CONFIGS: Record<ProfilePreset, { behavior: Partial<BehaviorSettings>; antiDetection: Partial<AntiDetectionSettings> }> = {
  stealth: {
    behavior: {
      typingDelayMin: 50,
      typingDelayMax: 150,
      mouseMovementStyle: 'natural',
      mouseJitterAmount: 2,
      clickDelayMin: 100,
      clickDelayMax: 300,
      scrollStyle: 'natural',
      microPauseEnabled: true,
      microPauseMinMs: 500,
      microPauseMaxMs: 2000,
      microPauseFrequency: 0.15,
    },
    antiDetection: {
      disableAutomationControlled: true,
      patchNavigatorWebdriver: true,
      patchNavigatorPlugins: true,
      patchNavigatorLanguages: true,
      patchWebGL: true,
      patchCanvas: true,
      headlessDetectionBypass: true,
      disableWebRTC: true,
    },
  },
  balanced: {
    behavior: {
      typingDelayMin: 30,
      typingDelayMax: 80,
      mouseMovementStyle: 'bezier',
      mouseJitterAmount: 1,
      clickDelayMin: 50,
      clickDelayMax: 150,
      scrollStyle: 'smooth',
      microPauseEnabled: true,
      microPauseMinMs: 200,
      microPauseMaxMs: 800,
      microPauseFrequency: 0.08,
    },
    antiDetection: {
      disableAutomationControlled: true,
      patchNavigatorWebdriver: true,
      patchNavigatorPlugins: true,
      patchNavigatorLanguages: false,
      patchWebGL: false,
      patchCanvas: false,
      headlessDetectionBypass: true,
      disableWebRTC: false,
    },
  },
  fast: {
    behavior: {
      typingDelayMin: 10,
      typingDelayMax: 30,
      mouseMovementStyle: 'linear',
      mouseJitterAmount: 0,
      clickDelayMin: 20,
      clickDelayMax: 50,
      scrollStyle: 'smooth',
      microPauseEnabled: false,
      microPauseMinMs: 0,
      microPauseMaxMs: 0,
      microPauseFrequency: 0,
    },
    antiDetection: {
      disableAutomationControlled: true,
      patchNavigatorWebdriver: true,
      patchNavigatorPlugins: false,
      patchNavigatorLanguages: false,
      patchWebGL: false,
      patchCanvas: false,
      headlessDetectionBypass: false,
      disableWebRTC: false,
    },
  },
  none: {
    behavior: {
      typingDelayMin: 0,
      typingDelayMax: 0,
      mouseMovementStyle: 'linear',
      mouseJitterAmount: 0,
      clickDelayMin: 0,
      clickDelayMax: 0,
      scrollStyle: 'instant',
      microPauseEnabled: false,
      microPauseMinMs: 0,
      microPauseMaxMs: 0,
      microPauseFrequency: 0,
    },
    antiDetection: {
      disableAutomationControlled: false,
      patchNavigatorWebdriver: false,
      patchNavigatorPlugins: false,
      patchNavigatorLanguages: false,
      patchWebGL: false,
      patchCanvas: false,
      headlessDetectionBypass: false,
      disableWebRTC: false,
    },
  },
};

const USER_AGENT_LABELS: Record<UserAgentPreset, string> = {
  chrome_windows: 'Chrome on Windows',
  chrome_mac: 'Chrome on macOS',
  chrome_linux: 'Chrome on Linux',
  firefox_windows: 'Firefox on Windows',
  firefox_mac: 'Firefox on macOS',
  safari_mac: 'Safari on macOS',
  edge_windows: 'Edge on Windows',
  custom: 'Custom User Agent',
};

type TabId = 'presets' | 'fingerprint' | 'behavior' | 'anti-detection';

export function BrowserProfileEditor({ profileName, initialProfile, onSave, onClose }: BrowserProfileEditorProps) {
  const [activeTab, setActiveTab] = useState<TabId>('presets');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);

  // Profile state
  const [preset, setPreset] = useState<ProfilePreset>(initialProfile?.preset ?? 'none');
  const [fingerprint, setFingerprint] = useState<FingerprintSettings>(initialProfile?.fingerprint ?? {});
  const [behavior, setBehavior] = useState<BehaviorSettings>(initialProfile?.behavior ?? {});
  const [antiDetection, setAntiDetection] = useState<AntiDetectionSettings>(initialProfile?.antiDetection ?? {});

  // Apply preset defaults when preset changes
  const applyPreset = useCallback((newPreset: ProfilePreset) => {
    setPreset(newPreset);
    const config = PRESET_CONFIGS[newPreset];
    setBehavior((prev) => ({ ...prev, ...config.behavior }));
    setAntiDetection((prev) => ({ ...prev, ...config.antiDetection }));
    setIsDirty(true);
  }, []);

  // Initialize from initialProfile
  useEffect(() => {
    if (initialProfile) {
      setPreset(initialProfile.preset ?? 'none');
      setFingerprint(initialProfile.fingerprint ?? {});
      setBehavior(initialProfile.behavior ?? {});
      setAntiDetection(initialProfile.antiDetection ?? {});
    }
  }, [initialProfile]);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const profile: BrowserProfile = {
        preset,
        fingerprint: Object.keys(fingerprint).length > 0 ? fingerprint : undefined,
        behavior: Object.keys(behavior).length > 0 ? behavior : undefined,
        antiDetection: Object.keys(antiDetection).length > 0 ? antiDetection : undefined,
      };
      await onSave(profile);
      setIsDirty(false);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save profile');
    } finally {
      setSaving(false);
    }
  };

  const handleClose = () => {
    if (isDirty) {
      if (!window.confirm('You have unsaved changes. Discard them?')) {
        return;
      }
    }
    onClose();
  };

  const updateFingerprint = <K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => {
    setFingerprint((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  };

  const updateBehavior = <K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => {
    setBehavior((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  };

  const updateAntiDetection = <K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => {
    setAntiDetection((prev) => ({ ...prev, [key]: value }));
    setIsDirty(true);
  };

  const tabs: { id: TabId; label: string }[] = [
    { id: 'presets', label: 'Presets' },
    { id: 'fingerprint', label: 'Fingerprint' },
    { id: 'behavior', label: 'Behavior' },
    { id: 'anti-detection', label: 'Anti-Detection' },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={handleClose}>
      <div
        className="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Configure Browser Profile</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400">{profileName}</p>
            </div>
            <button
              onClick={handleClose}
              className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-200 dark:border-gray-700 px-6">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {error && (
            <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/40 dark:text-red-200">
              {error}
            </div>
          )}

          {activeTab === 'presets' && (
            <PresetsTab preset={preset} onPresetChange={applyPreset} />
          )}

          {activeTab === 'fingerprint' && (
            <FingerprintTab fingerprint={fingerprint} onChange={updateFingerprint} />
          )}

          {activeTab === 'behavior' && (
            <BehaviorTab behavior={behavior} onChange={updateBehavior} />
          )}

          {activeTab === 'anti-detection' && (
            <AntiDetectionTab antiDetection={antiDetection} onChange={updateAntiDetection} />
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-end gap-3">
          <button
            onClick={handleClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !isDirty}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Tab Components
// ============================================================================

function PresetsTab({ preset, onPresetChange }: { preset: ProfilePreset; onPresetChange: (p: ProfilePreset) => void }) {
  const presets: { id: ProfilePreset; name: string; description: string; recommended?: boolean }[] = [
    {
      id: 'stealth',
      name: 'Stealth',
      description: 'Maximum anti-detection with natural human-like behavior. Best for sites with strict bot detection.',
      recommended: true,
    },
    {
      id: 'balanced',
      name: 'Balanced',
      description: 'Good anti-detection with reasonable speed. Suitable for most websites.',
    },
    {
      id: 'fast',
      name: 'Fast',
      description: 'Minimal delays with basic anti-detection. Good for trusted internal tools.',
    },
    {
      id: 'none',
      name: 'None',
      description: 'No modifications. Raw automation speed for debugging or testing.',
    },
  ];

  return (
    <div className="space-y-4">
      <p className="text-sm text-gray-600 dark:text-gray-400">
        Choose a preset to quickly configure anti-detection and behavior settings. You can customize individual settings in other tabs.
      </p>
      <div className="grid grid-cols-2 gap-4">
        {presets.map((p) => (
          <button
            key={p.id}
            onClick={() => onPresetChange(p.id)}
            className={`p-4 rounded-lg border-2 text-left transition-colors ${
              preset === p.id
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
            }`}
          >
            <div className="flex items-center gap-2">
              <span className="font-medium text-gray-900 dark:text-gray-100">{p.name}</span>
              {p.recommended && (
                <span className="text-xs px-2 py-0.5 rounded-full bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300">
                  Recommended
                </span>
              )}
            </div>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{p.description}</p>
          </button>
        ))}
      </div>
    </div>
  );
}

function FingerprintTab({
  fingerprint,
  onChange,
}: {
  fingerprint: FingerprintSettings;
  onChange: <K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => void;
}) {
  return (
    <div className="space-y-6">
      {/* Viewport */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Viewport</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Width</label>
            <input
              type="number"
              value={fingerprint.viewportWidth ?? ''}
              onChange={(e) => onChange('viewportWidth', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="1920"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Height</label>
            <input
              type="number"
              value={fingerprint.viewportHeight ?? ''}
              onChange={(e) => onChange('viewportHeight', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="1080"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* User Agent */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Browser Identity</h4>
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">User Agent Preset</label>
            <select
              value={fingerprint.userAgentPreset ?? 'chrome_windows'}
              onChange={(e) => onChange('userAgentPreset', e.target.value as UserAgentPreset)}
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            >
              {Object.entries(USER_AGENT_LABELS).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>
          {fingerprint.userAgentPreset === 'custom' && (
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Custom User Agent</label>
              <input
                type="text"
                value={fingerprint.userAgent ?? ''}
                onChange={(e) => onChange('userAgent', e.target.value || undefined)}
                placeholder="Mozilla/5.0..."
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
          )}
        </div>
      </div>

      {/* Locale & Timezone */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Locale & Timezone</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Locale</label>
            <input
              type="text"
              value={fingerprint.locale ?? ''}
              onChange={(e) => onChange('locale', e.target.value || undefined)}
              placeholder="en-US"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Timezone</label>
            <input
              type="text"
              value={fingerprint.timezoneId ?? ''}
              onChange={(e) => onChange('timezoneId', e.target.value || undefined)}
              placeholder="America/New_York"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Geolocation */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Geolocation</h4>
        <label className="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            checked={fingerprint.geolocationEnabled ?? false}
            onChange={(e) => onChange('geolocationEnabled', e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600"
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">Enable geolocation spoofing</span>
        </label>
        {fingerprint.geolocationEnabled && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Latitude</label>
              <input
                type="number"
                step="0.000001"
                value={fingerprint.latitude ?? ''}
                onChange={(e) => onChange('latitude', e.target.value ? parseFloat(e.target.value) : undefined)}
                placeholder="40.7128"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Longitude</label>
              <input
                type="number"
                step="0.000001"
                value={fingerprint.longitude ?? ''}
                onChange={(e) => onChange('longitude', e.target.value ? parseFloat(e.target.value) : undefined)}
                placeholder="-74.0060"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
          </div>
        )}
      </div>

      {/* Device Properties */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Device Properties</h4>
        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Scale Factor</label>
            <input
              type="number"
              step="0.5"
              value={fingerprint.deviceScaleFactor ?? ''}
              onChange={(e) => onChange('deviceScaleFactor', e.target.value ? parseFloat(e.target.value) : undefined)}
              placeholder="2"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">CPU Cores</label>
            <input
              type="number"
              value={fingerprint.hardwareConcurrency ?? ''}
              onChange={(e) => onChange('hardwareConcurrency', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="8"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Memory (GB)</label>
            <input
              type="number"
              value={fingerprint.deviceMemory ?? ''}
              onChange={(e) => onChange('deviceMemory', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="16"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function BehaviorTab({
  behavior,
  onChange,
}: {
  behavior: BehaviorSettings;
  onChange: <K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => void;
}) {
  return (
    <div className="space-y-6">
      {/* Typing Delays */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Typing Behavior</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min Delay (ms)</label>
            <input
              type="number"
              value={behavior.typingDelayMin ?? ''}
              onChange={(e) => onChange('typingDelayMin', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="50"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max Delay (ms)</label>
            <input
              type="number"
              value={behavior.typingDelayMax ?? ''}
              onChange={(e) => onChange('typingDelayMax', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="150"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Mouse Movement */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Mouse Movement</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Movement Style</label>
            <select
              value={behavior.mouseMovementStyle ?? 'linear'}
              onChange={(e) => onChange('mouseMovementStyle', e.target.value as MouseMovementStyle)}
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            >
              <option value="linear">Linear (Fast)</option>
              <option value="bezier">Bezier (Smooth)</option>
              <option value="natural">Natural (Human-like)</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Jitter Amount (px)</label>
            <input
              type="number"
              step="0.5"
              value={behavior.mouseJitterAmount ?? ''}
              onChange={(e) => onChange('mouseJitterAmount', e.target.value ? parseFloat(e.target.value) : undefined)}
              placeholder="2"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Click Delays */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Click Behavior</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min Delay (ms)</label>
            <input
              type="number"
              value={behavior.clickDelayMin ?? ''}
              onChange={(e) => onChange('clickDelayMin', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="100"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max Delay (ms)</label>
            <input
              type="number"
              value={behavior.clickDelayMax ?? ''}
              onChange={(e) => onChange('clickDelayMax', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="300"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Scroll Style */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Scroll Behavior</h4>
        <div>
          <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Scroll Style</label>
          <select
            value={behavior.scrollStyle ?? 'smooth'}
            onChange={(e) => onChange('scrollStyle', e.target.value as ScrollStyle)}
            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
          >
            <option value="instant">Instant</option>
            <option value="smooth">Smooth</option>
            <option value="natural">Natural (Variable speed)</option>
          </select>
        </div>
      </div>

      {/* Micro-pauses */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Micro-Pauses</h4>
        <label className="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            checked={behavior.microPauseEnabled ?? false}
            onChange={(e) => onChange('microPauseEnabled', e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600"
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">Enable random micro-pauses</span>
        </label>
        {behavior.microPauseEnabled && (
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min (ms)</label>
              <input
                type="number"
                value={behavior.microPauseMinMs ?? ''}
                onChange={(e) => onChange('microPauseMinMs', e.target.value ? parseInt(e.target.value) : undefined)}
                placeholder="500"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max (ms)</label>
              <input
                type="number"
                value={behavior.microPauseMaxMs ?? ''}
                onChange={(e) => onChange('microPauseMaxMs', e.target.value ? parseInt(e.target.value) : undefined)}
                placeholder="2000"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Frequency</label>
              <input
                type="number"
                step="0.01"
                value={behavior.microPauseFrequency ?? ''}
                onChange={(e) => onChange('microPauseFrequency', e.target.value ? parseFloat(e.target.value) : undefined)}
                placeholder="0.15"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function AntiDetectionTab({
  antiDetection,
  onChange,
}: {
  antiDetection: AntiDetectionSettings;
  onChange: <K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => void;
}) {
  const toggles: { key: keyof AntiDetectionSettings; label: string; description: string }[] = [
    {
      key: 'disableAutomationControlled',
      label: 'Disable Automation Controlled Flag',
      description: 'Removes browser automation indicators from window properties',
    },
    {
      key: 'patchNavigatorWebdriver',
      label: 'Patch navigator.webdriver',
      description: 'Hides the webdriver property that indicates automation',
    },
    {
      key: 'patchNavigatorPlugins',
      label: 'Patch navigator.plugins',
      description: 'Spoofs browser plugins to appear as a real browser',
    },
    {
      key: 'patchNavigatorLanguages',
      label: 'Patch navigator.languages',
      description: 'Sets language preferences to match locale settings',
    },
    {
      key: 'patchWebGL',
      label: 'Patch WebGL',
      description: 'Modifies WebGL renderer and vendor strings',
    },
    {
      key: 'patchCanvas',
      label: 'Patch Canvas Fingerprint',
      description: 'Adds noise to canvas fingerprinting attempts',
    },
    {
      key: 'headlessDetectionBypass',
      label: 'Headless Detection Bypass',
      description: 'Bypasses checks for headless browser mode',
    },
    {
      key: 'disableWebRTC',
      label: 'Disable WebRTC',
      description: 'Prevents WebRTC from leaking real IP address',
    },
  ];

  return (
    <div className="space-y-4">
      <p className="text-sm text-gray-600 dark:text-gray-400">
        These settings help bypass bot detection systems by making the browser appear more like a real user's browser.
      </p>
      <div className="space-y-3">
        {toggles.map((toggle) => (
          <label key={toggle.key} className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer">
            <input
              type="checkbox"
              checked={antiDetection[toggle.key] ?? false}
              onChange={(e) => onChange(toggle.key, e.target.checked)}
              className="mt-0.5 rounded border-gray-300 dark:border-gray-600"
            />
            <div>
              <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{toggle.label}</span>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{toggle.description}</p>
            </div>
          </label>
        ))}
      </div>
    </div>
  );
}
