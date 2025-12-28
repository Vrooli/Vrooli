import type { FingerprintSettings, UserAgentPreset } from '@/domains/recording/types/types';
import { USER_AGENT_LABELS } from './presets';

interface FingerprintSectionProps {
  fingerprint: FingerprintSettings;
  onChange: <K extends keyof FingerprintSettings>(key: K, value: FingerprintSettings[K]) => void;
}

export function FingerprintSection({ fingerprint, onChange }: FingerprintSectionProps) {
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
              value={fingerprint.viewport_width ?? ''}
              onChange={(e) => onChange('viewport_width', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="1920"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Height</label>
            <input
              type="number"
              value={fingerprint.viewport_height ?? ''}
              onChange={(e) => onChange('viewport_height', e.target.value ? parseInt(e.target.value) : undefined)}
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
              value={fingerprint.user_agent_preset ?? 'chrome_windows'}
              onChange={(e) => onChange('user_agent_preset', e.target.value as UserAgentPreset)}
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            >
              {Object.entries(USER_AGENT_LABELS).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>
          {fingerprint.user_agent_preset === 'custom' && (
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Custom User Agent</label>
              <input
                type="text"
                value={fingerprint.user_agent ?? ''}
                onChange={(e) => onChange('user_agent', e.target.value || undefined)}
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
              value={fingerprint.timezone_id ?? ''}
              onChange={(e) => onChange('timezone_id', e.target.value || undefined)}
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
            checked={fingerprint.geolocation_enabled ?? false}
            onChange={(e) => onChange('geolocation_enabled', e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600"
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">Enable geolocation spoofing</span>
        </label>
        {fingerprint.geolocation_enabled && (
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
              value={fingerprint.device_scale_factor ?? ''}
              onChange={(e) => onChange('device_scale_factor', e.target.value ? parseFloat(e.target.value) : undefined)}
              placeholder="2"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">CPU Cores</label>
            <input
              type="number"
              value={fingerprint.hardware_concurrency ?? ''}
              onChange={(e) => onChange('hardware_concurrency', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="8"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Memory (GB)</label>
            <input
              type="number"
              value={fingerprint.device_memory ?? ''}
              onChange={(e) => onChange('device_memory', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="16"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
