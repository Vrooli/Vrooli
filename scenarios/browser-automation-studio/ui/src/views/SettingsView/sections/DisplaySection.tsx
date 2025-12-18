import { Monitor, Sun, Moon, Laptop, Type, Accessibility, Eye, Minimize2, Check } from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import type { ThemeMode, FontSize, FontFamily } from '@stores/settingsStore';
import { SettingSection, ToggleSwitch } from './shared';

const THEME_OPTIONS: Array<{ id: ThemeMode; label: string; icon: React.ReactNode; description: string }> = [
  { id: 'light', label: 'Light', icon: <Sun size={20} />, description: 'Bright theme for well-lit environments' },
  { id: 'dark', label: 'Dark', icon: <Moon size={20} />, description: 'Dark theme that reduces eye strain' },
  { id: 'system', label: 'System', icon: <Laptop size={20} />, description: 'Follow your system preferences' },
];

const FONT_SIZE_OPTIONS: Array<{ id: FontSize; label: string; description: string }> = [
  { id: 'small', label: 'Small', description: 'Compact interface, more content visible' },
  { id: 'medium', label: 'Medium', description: 'Default balanced size' },
  { id: 'large', label: 'Large', description: 'Larger text for better readability' },
];

const FONT_FAMILY_OPTIONS: Array<{ id: FontFamily; label: string; preview: string; description: string }> = [
  { id: 'sans', label: 'Sans Serif', preview: 'Aa Bb Cc', description: 'Clean and modern appearance' },
  { id: 'mono', label: 'Monospace', preview: 'Aa Bb Cc', description: 'Technical feel, fixed-width characters' },
  { id: 'system', label: 'System', preview: 'Aa Bb Cc', description: 'Match your OS default font' },
];

export function DisplaySection() {
  const { display, setDisplaySetting, getEffectiveTheme } = useSettingsStore();

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3 mb-6">
        <Monitor size={24} className="text-purple-400" />
        <div>
          <h2 className="text-lg font-semibold text-surface">Display Settings</h2>
          <p className="text-sm text-gray-400">Customize appearance and accessibility</p>
        </div>
      </div>

      <SettingSection title="Theme" tooltip="Choose your preferred color scheme.">
        <div className="grid grid-cols-3 gap-3">
          {THEME_OPTIONS.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setDisplaySetting('themeMode', option.id)}
              className={`flex flex-col items-center p-4 rounded-lg border transition-all ${
                display.themeMode === option.id
                  ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
                  : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
              }`}
            >
              <div className={`mb-2 ${display.themeMode === option.id ? 'text-flow-accent' : 'text-gray-400'}`}>
                {option.icon}
              </div>
              <span className="text-sm font-medium text-surface">{option.label}</span>
              <span className="text-xs text-gray-500 mt-1 text-center">{option.description}</span>
            </button>
          ))}
        </div>
        {display.themeMode === 'system' && (
          <p className="mt-3 text-xs text-gray-500 flex items-center gap-1">
            <Laptop size={12} />
            Currently using: {getEffectiveTheme()} theme based on system preference
          </p>
        )}
      </SettingSection>

      <SettingSection title="Font Size" tooltip="Adjust the text size throughout the application.">
        <div className="space-y-3">
          {FONT_SIZE_OPTIONS.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setDisplaySetting('fontSize', option.id)}
              className={`w-full flex items-center justify-between p-3 rounded-lg border transition-all ${
                display.fontSize === option.id
                  ? 'border-flow-accent bg-flow-accent/10'
                  : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
              }`}
            >
              <div className="flex items-center gap-3">
                <Type size={18} className={display.fontSize === option.id ? 'text-flow-accent' : 'text-gray-400'} />
                <div className="text-left">
                  <span className="text-sm font-medium text-surface block">{option.label}</span>
                  <span className="text-xs text-gray-500">{option.description}</span>
                </div>
              </div>
              {display.fontSize === option.id && (
                <Check size={16} className="text-flow-accent" />
              )}
            </button>
          ))}
        </div>
      </SettingSection>

      <SettingSection title="Font Family" tooltip="Choose your preferred typeface." defaultOpen={false}>
        <div className="space-y-3">
          {FONT_FAMILY_OPTIONS.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setDisplaySetting('fontFamily', option.id)}
              className={`w-full flex items-center justify-between p-3 rounded-lg border transition-all ${
                display.fontFamily === option.id
                  ? 'border-flow-accent bg-flow-accent/10'
                  : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
              }`}
            >
              <div className="flex items-center gap-3">
                <span
                  className={`text-lg ${option.id === 'mono' ? 'font-mono' : option.id === 'sans' ? 'font-sans' : 'font-system'} ${
                    display.fontFamily === option.id ? 'text-flow-accent' : 'text-gray-400'
                  }`}
                >
                  {option.preview}
                </span>
                <div className="text-left">
                  <span className="text-sm font-medium text-surface block">{option.label}</span>
                  <span className="text-xs text-gray-500">{option.description}</span>
                </div>
              </div>
              {display.fontFamily === option.id && (
                <Check size={16} className="text-flow-accent" />
              )}
            </button>
          ))}
        </div>
      </SettingSection>

      <SettingSection title="Accessibility" tooltip="Options to improve usability and reduce visual stress.">
        <div className="space-y-3">
          <ToggleSwitch
            checked={display.reducedMotion}
            onChange={(v) => setDisplaySetting('reducedMotion', v)}
            label="Reduced Motion"
            description="Minimize animations and transitions"
            icon={<Accessibility size={18} />}
          />
          <ToggleSwitch
            checked={display.highContrast}
            onChange={(v) => setDisplaySetting('highContrast', v)}
            label="High Contrast"
            description="Increase text and border contrast"
            icon={<Eye size={18} />}
            className="border-t border-gray-800"
          />
          <ToggleSwitch
            checked={display.compactMode}
            onChange={(v) => setDisplaySetting('compactMode', v)}
            label="Compact Mode"
            description="Reduce spacing for denser information"
            icon={<Minimize2 size={18} />}
            className="border-t border-gray-800"
          />
        </div>
      </SettingSection>
    </div>
  );
}
