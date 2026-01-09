import type { ProfilePreset } from '@/domains/recording/types/types';
import { PRESET_METADATA } from './presets';

interface PresetsSectionProps {
  preset: ProfilePreset;
  onPresetChange: (preset: ProfilePreset) => void;
}

export function PresetsSection({ preset, onPresetChange }: PresetsSectionProps) {
  return (
    <div className="space-y-4">
      <p className="text-sm text-gray-600 dark:text-gray-400">
        Choose a preset to quickly configure anti-detection and behavior settings. You can customize individual
        settings in other tabs.
      </p>
      <div className="grid grid-cols-2 gap-4">
        {PRESET_METADATA.map((p) => (
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
