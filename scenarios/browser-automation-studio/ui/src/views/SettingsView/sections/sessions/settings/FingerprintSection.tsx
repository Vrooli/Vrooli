import type { FingerprintSettings, UserAgentPreset } from '@/domains/recording/types/types';
import {
  FormNumberInput,
  FormTextInput,
  FormSelect,
  FormCheckbox,
  FormFieldGroup,
  FormGrid,
} from '@/components/form';
import { USER_AGENT_LABELS } from './presets';

interface FingerprintSectionProps {
  fingerprint: FingerprintSettings;
  onChange: <K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => void;
}

// Convert USER_AGENT_LABELS to options array
const userAgentOptions = Object.entries(USER_AGENT_LABELS).map(([value, label]) => ({
  value: value as UserAgentPreset,
  label,
}));

const colorSchemeOptions = [
  { value: 'no-preference' as const, label: 'No Preference (System Default)' },
  { value: 'light' as const, label: 'Light' },
  { value: 'dark' as const, label: 'Dark' },
];

export function FingerprintSection({ fingerprint, onChange }: FingerprintSectionProps) {
  return (
    <div className="space-y-6">
      {/* Viewport */}
      <FormFieldGroup title="Viewport">
        <FormGrid cols={2}>
          <FormNumberInput
            value={fingerprint.viewport_width}
            onChange={(v) => onChange('viewport_width', v)}
            label="Width"
            placeholder="1920"
          />
          <FormNumberInput
            value={fingerprint.viewport_height}
            onChange={(v) => onChange('viewport_height', v)}
            label="Height"
            placeholder="1080"
          />
        </FormGrid>
      </FormFieldGroup>

      {/* User Agent */}
      <FormFieldGroup title="Browser Identity">
        <div className="space-y-3">
          <FormSelect
            value={fingerprint.user_agent_preset ?? 'chrome_windows'}
            onChange={(v) => onChange('user_agent_preset', v)}
            label="User Agent Preset"
            options={userAgentOptions}
          />
          {fingerprint.user_agent_preset === 'custom' && (
            <FormTextInput
              value={fingerprint.user_agent}
              onChange={(v) => onChange('user_agent', v)}
              label="Custom User Agent"
              placeholder="Mozilla/5.0..."
            />
          )}
        </div>
      </FormFieldGroup>

      {/* Locale & Timezone */}
      <FormFieldGroup title="Locale & Timezone">
        <FormGrid cols={2}>
          <FormTextInput
            value={fingerprint.locale}
            onChange={(v) => onChange('locale', v)}
            label="Locale"
            placeholder="en-US"
          />
          <FormTextInput
            value={fingerprint.timezone_id}
            onChange={(v) => onChange('timezone_id', v)}
            label="Timezone"
            placeholder="America/New_York"
          />
        </FormGrid>
      </FormFieldGroup>

      {/* Geolocation */}
      <FormFieldGroup title="Geolocation">
        <FormCheckbox
          checked={fingerprint.geolocation_enabled ?? false}
          onChange={(v) => onChange('geolocation_enabled', v)}
          label="Enable geolocation spoofing"
          className="mb-3"
        />
        {fingerprint.geolocation_enabled && (
          <>
            <FormGrid cols={2}>
              <FormNumberInput
                value={fingerprint.latitude}
                onChange={(v) => onChange('latitude', v)}
                label="Latitude"
                placeholder="40.7128"
                step={0.000001}
              />
              <FormNumberInput
                value={fingerprint.longitude}
                onChange={(v) => onChange('longitude', v)}
                label="Longitude"
                placeholder="-74.0060"
                step={0.000001}
              />
            </FormGrid>
            <FormNumberInput
              value={fingerprint.accuracy}
              onChange={(v) => onChange('accuracy', v)}
              label="Accuracy (meters)"
              placeholder="10"
              min={1}
              className="mt-3 w-32"
            />
          </>
        )}
      </FormFieldGroup>

      {/* Appearance */}
      <FormFieldGroup title="Appearance">
        <FormSelect
          value={fingerprint.color_scheme ?? 'no-preference'}
          onChange={(v) => onChange('color_scheme', v)}
          label="Color Scheme"
          options={colorSchemeOptions}
          description="Sets the prefers-color-scheme media query value"
        />
      </FormFieldGroup>

      {/* Device Properties */}
      <FormFieldGroup title="Device Properties">
        <FormGrid cols={3}>
          <FormNumberInput
            value={fingerprint.device_scale_factor}
            onChange={(v) => onChange('device_scale_factor', v)}
            label="Scale Factor"
            placeholder="2"
            step={0.5}
          />
          <FormNumberInput
            value={fingerprint.hardware_concurrency}
            onChange={(v) => onChange('hardware_concurrency', v)}
            label="CPU Cores"
            placeholder="8"
          />
          <FormNumberInput
            value={fingerprint.device_memory}
            onChange={(v) => onChange('device_memory', v)}
            label="Memory (GB)"
            placeholder="16"
          />
        </FormGrid>
      </FormFieldGroup>
    </div>
  );
}
