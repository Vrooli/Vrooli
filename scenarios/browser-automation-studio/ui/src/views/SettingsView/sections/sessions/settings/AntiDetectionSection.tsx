import type { AdBlockingMode, AntiDetectionSettings } from '@/domains/recording/types/types';
import {
  FormFieldGroup,
  FormRadioGroup,
  FormTextInput,
  FormToggleCard,
  type FormRadioOption,
} from '@/components/form';

interface AntiDetectionSectionProps {
  antiDetection: AntiDetectionSettings;
  onChange: <K extends keyof AntiDetectionSettings>(key: K, value: AntiDetectionSettings[K]) => void;
}

const AD_BLOCKING_OPTIONS: FormRadioOption<AdBlockingMode>[] = [
  { value: 'none', label: 'Disabled', description: 'No ad blocking' },
  { value: 'ads_only', label: 'Ads Only', description: 'Block advertisements only' },
  { value: 'ads_and_tracking', label: 'Ads & Tracking', description: 'Block ads and tracking scripts (recommended)' },
];

// Boolean-only keys from AntiDetectionSettings (excludes non-boolean fields)
type BooleanAntiDetectionKey = Exclude<keyof AntiDetectionSettings, 'ad_blocking_mode' | 'ad_blocking_whitelist'>;

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
  {
    key: 'patch_fonts',
    label: 'Patch Font Fingerprinting',
    description: 'Returns only common system fonts to prevent font enumeration tracking',
  },
  {
    key: 'patch_screen_properties',
    label: 'Patch Screen Properties',
    description: 'Spoofs screen dimensions and color depth to match viewport settings',
  },
  {
    key: 'patch_battery_api',
    label: 'Patch Battery API',
    description: 'Returns consistent battery status to prevent power-based fingerprinting',
  },
  {
    key: 'patch_connection_api',
    label: 'Patch Connection API',
    description: 'Spoofs network type to appear as typical residential WiFi connection',
  },
];

export function AntiDetectionSection({ antiDetection, onChange }: AntiDetectionSectionProps) {
  const handleWhitelistChange = (value: string | undefined) => {
    if (!value) {
      onChange('ad_blocking_whitelist', []);
      return;
    }
    const domains = value
      .split(',')
      .map((d) => d.trim())
      .filter(Boolean);
    onChange('ad_blocking_whitelist', domains);
  };

  return (
    <div className="space-y-6">
      {/* Ad Blocking Section */}
      <FormFieldGroup
        title="Ad Blocking"
        description="Block advertisements and tracking scripts to improve page load performance and privacy."
      >
        <FormRadioGroup
          name="ad_blocking_mode"
          value={antiDetection.ad_blocking_mode ?? 'ads_and_tracking'}
          onChange={(v) => onChange('ad_blocking_mode', v)}
          options={AD_BLOCKING_OPTIONS}
        />

        {/* Ad Blocking Whitelist - only show when ad blocking is enabled */}
        {(antiDetection.ad_blocking_mode ?? 'ads_only') !== 'none' && (
          <FormTextInput
            value={(antiDetection.ad_blocking_whitelist ?? []).join(', ') || undefined}
            onChange={handleWhitelistChange}
            label="Domain Whitelist"
            description="Domains to exclude from ad blocking (comma-separated). Use *.domain.com for wildcards."
            placeholder="google.com, *.gstatic.com"
            className="mt-4"
          />
        )}
      </FormFieldGroup>

      {/* Bot Detection Bypass Section */}
      <FormFieldGroup
        title="Bot Detection Bypass"
        description="These settings help bypass bot detection systems by making the browser appear more like a real user's browser."
        className="pt-4 border-t border-gray-200 dark:border-gray-700"
      >
        <div className="space-y-3">
          {TOGGLES.map((toggle) => (
            <FormToggleCard
              key={toggle.key}
              checked={Boolean(antiDetection[toggle.key])}
              onChange={(checked) => onChange(toggle.key, checked)}
              label={toggle.label}
              description={toggle.description}
            />
          ))}
        </div>
      </FormFieldGroup>
    </div>
  );
}
