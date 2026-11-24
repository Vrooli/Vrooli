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
    description: 'Requests a Chromium automation-feature adjustment; site compatibility is not guaranteed.',
  },
  {
    key: 'patch_navigator_webdriver',
    label: 'Patch navigator.webdriver',
    description: 'Requests an adjustment to navigator.webdriver; detection outcomes vary by site and browser mode.',
  },
  {
    key: 'patch_navigator_plugins',
    label: 'Patch navigator.plugins',
    description: 'Requests an adjustment to the reported browser plugin list.',
  },
  {
    key: 'patch_navigator_languages',
    label: 'Patch navigator.languages',
    description: 'Requests a language-value adjustment to match the configured locale.',
  },
  {
    key: 'patch_webgl',
    label: 'Patch WebGL',
    description: 'Requests an adjustment to reported WebGL renderer details.',
  },
  {
    key: 'patch_canvas',
    label: 'Patch Canvas Fingerprint',
    description: 'Requests an adjustment to canvas output.',
  },
  {
    key: 'patch_audio_context',
    label: 'Patch AudioContext',
    description: 'Requests an adjustment to AudioContext output.',
  },
  {
    key: 'headless_detection_bypass',
    label: 'Headless Signal Adjustment',
    description: 'Legacy option name. It does not guarantee that headless mode is accepted.',
  },
  {
    key: 'disable_webrtc',
    label: 'Disable WebRTC',
    description: 'Requests WebRTC disablement; may affect sites and does not replace network-policy checks.',
  },
  {
    key: 'patch_fonts',
    label: 'Patch Font Fingerprinting',
    description: 'Requests an adjustment to the fonts exposed to page scripts.',
  },
  {
    key: 'patch_screen_properties',
    label: 'Patch Screen Properties',
    description: 'Requests screen properties matching the configured viewport.',
  },
  {
    key: 'patch_battery_api',
    label: 'Patch Battery API',
    description: 'Requests a battery-status adjustment.',
  },
  {
    key: 'patch_connection_api',
    label: 'Patch Connection API',
    description: 'Requests a connection-type adjustment; it does not change the actual network path.',
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

      {/* Browser signal adjustment controls */}
      <FormFieldGroup
        title="Browser Signal Options"
        description="Requests adjustments to selected browser signals. Support varies by browser mode and version; these controls do not guarantee site access or a detection outcome."
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
