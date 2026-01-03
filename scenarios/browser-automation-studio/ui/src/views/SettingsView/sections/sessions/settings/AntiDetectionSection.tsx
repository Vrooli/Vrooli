import type { AdBlockingMode, AntiDetectionSettings } from '@/domains/recording/types/types';

interface AntiDetectionSectionProps {
  antiDetection: AntiDetectionSettings;
  onChange: <K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => void;
}

const AD_BLOCKING_OPTIONS: { value: AdBlockingMode; label: string; description: string }[] = [
  { value: 'none', label: 'Disabled', description: 'No ad blocking' },
  { value: 'ads_only', label: 'Ads Only', description: 'Block advertisements only' },
  { value: 'ads_and_tracking', label: 'Ads & Tracking', description: 'Block ads and tracking scripts (recommended)' },
];

// Boolean-only keys from AntiDetectionSettings (excludes ad_blocking_mode which is a string)
type BooleanAntiDetectionKey = Exclude<keyof AntiDetectionSettings, 'ad_blocking_mode'>;

const TOGGLES: { key: BooleanAntiDetectionKey; label: string; description: string }[] = [
  {
    key: 'disable_automation_controlled',
    label: 'Disable Automation Controlled Flag',
    description: 'Removes browser automation indicators from window properties',
  },
  {
    key: 'patch_navigator_webdriver',
    label: 'Patch navigator.webdriver',
    description: 'Hides the webdriver property that indicates automation',
  },
  {
    key: 'patch_navigator_plugins',
    label: 'Patch navigator.plugins',
    description: 'Spoofs browser plugins to appear as a real browser',
  },
  {
    key: 'patch_navigator_languages',
    label: 'Patch navigator.languages',
    description: 'Sets language preferences to match locale settings',
  },
  {
    key: 'patch_webgl',
    label: 'Patch WebGL',
    description: 'Modifies WebGL renderer and vendor strings',
  },
  {
    key: 'patch_canvas',
    label: 'Patch Canvas Fingerprint',
    description: 'Adds noise to canvas fingerprinting attempts',
  },
  {
    key: 'patch_audio_context',
    label: 'Patch AudioContext',
    description: 'Adds noise to audio processing output to prevent audio fingerprinting',
  },
  {
    key: 'headless_detection_bypass',
    label: 'Headless Detection Bypass',
    description: 'Bypasses checks for headless browser mode',
  },
  {
    key: 'disable_webrtc',
    label: 'Disable WebRTC',
    description: 'Prevents WebRTC from leaking real IP address',
  },
];

export function AntiDetectionSection({ antiDetection, onChange }: AntiDetectionSectionProps) {
  return (
    <div className="space-y-6">
      {/* Ad Blocking Section */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Ad Blocking</h4>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
          Block advertisements and tracking scripts to improve page load performance and privacy.
        </p>
        <div className="space-y-2">
          {AD_BLOCKING_OPTIONS.map((option) => (
            <label
              key={option.value}
              className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer"
            >
              <input
                type="radio"
                name="ad_blocking_mode"
                value={option.value}
                checked={(antiDetection.ad_blocking_mode ?? 'ads_and_tracking') === option.value}
                onChange={() => onChange('ad_blocking_mode', option.value)}
                className="mt-0.5"
              />
              <div>
                <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{option.label}</span>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{option.description}</p>
              </div>
            </label>
          ))}
        </div>
      </div>

      {/* Bot Detection Bypass Section */}
      <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">Bot Detection Bypass</h4>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
          These settings help bypass bot detection systems by making the browser appear more like a real user's browser.
        </p>
        <div className="space-y-3">
          {TOGGLES.map((toggle) => (
            <label
              key={toggle.key}
              className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800/50 cursor-pointer"
            >
              <input
                type="checkbox"
                checked={Boolean(antiDetection[toggle.key])}
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
    </div>
  );
}
