/**
 * DisplayTab Component
 * Settings for UI display preferences
 */

import { Sun, Moon, Laptop, Eye } from 'lucide-react';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { useTheme } from '@/contexts/ThemeContext';
import type { DisplaySettings } from '@/types/api';

interface DisplayTabProps {
  settings: DisplaySettings;
  onChange: (updates: Partial<DisplaySettings>) => void;
}

export function DisplayTab({ settings, onChange }: DisplayTabProps) {
  const { setTheme } = useTheme();

  const handleThemeChange = (theme: 'light' | 'dark' | 'auto') => {
    onChange({ theme });
    setTheme(theme); // Live preview
  };

  const themeOptions = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'auto', label: 'Auto (System)', icon: Laptop },
  ];

  return (
    <div className="space-y-6">
      {/* Theme Selector */}
      <div className="space-y-3">
        <Label htmlFor="theme">Color Theme</Label>
        <Select
          value={settings.theme}
          onValueChange={(value: string) => handleThemeChange(value as 'light' | 'dark' | 'auto')}
        >
          <SelectTrigger id="theme">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {themeOptions.map(({ value, label, icon: Icon }) => (
              <SelectItem key={value} value={value}>
                <div className="flex items-center gap-2">
                  <Icon className="h-4 w-4" />
                  {label}
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs text-slate-500">
          Choose your preferred color scheme. Changes apply immediately.
        </p>
      </div>

      {/* Theme Preview Cards */}
      <div className="grid grid-cols-3 gap-3">
        {themeOptions.map(({ value, label, icon: Icon }) => (
          <button
            key={value}
            onClick={() => handleThemeChange(value as 'light' | 'dark' | 'auto')}
            className={`
              relative rounded-lg border-2 p-4 transition-all
              ${settings.theme === value
                ? 'border-blue-500 bg-blue-500/10'
                : 'border-white/10 hover:border-white/20'
              }
            `}
          >
            <Icon className="h-8 w-8 mx-auto mb-2 text-slate-400" />
            <p className="text-xs font-medium text-center">{label}</p>
            {settings.theme === value && (
              <div className="absolute top-2 right-2">
                <Eye className="h-4 w-4 text-blue-400" />
              </div>
            )}
          </button>
        ))}
      </div>

      {/* Condensed Mode */}
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <Checkbox
            id="condensed-mode"
            checked={settings.condensed_mode}
            onCheckedChange={(checked) => onChange({ condensed_mode: !!checked })}
          />
          <Label htmlFor="condensed-mode" className="cursor-pointer font-medium">
            Condensed Mode
          </Label>
        </div>
        <p className="text-xs text-slate-500 ml-8">
          Reduces spacing and padding throughout the UI to show more content on screen. Useful for smaller displays.
        </p>
      </div>

      {/* Info Box */}
      <div className="rounded-lg border border-purple-500/20 bg-purple-500/10 p-4">
        <h4 className="text-sm font-medium text-purple-400 mb-2">Display Preferences</h4>
        <ul className="text-xs text-slate-400 space-y-1">
          <li>• <strong>Light:</strong> Bright theme optimized for well-lit environments</li>
          <li>• <strong>Dark:</strong> Low-contrast theme for reduced eye strain</li>
          <li>• <strong>Auto:</strong> Follows your system's color scheme preference</li>
          <li>• Theme changes are previewed live and apply immediately</li>
        </ul>
      </div>

      {/* Theme Preview Demo */}
      <div className="rounded-lg border border-white/10 p-4 space-y-3">
        <h4 className="text-sm font-medium">Theme Preview</h4>
        <div className="grid grid-cols-2 gap-2">
          <div className="bg-background border rounded p-2 text-xs">
            <div className="font-medium mb-1">Background</div>
            <div className="text-muted-foreground">Muted text</div>
          </div>
          <div className="bg-primary text-primary-foreground rounded p-2 text-xs">
            <div className="font-medium mb-1">Primary</div>
            <div>Primary text</div>
          </div>
          <div className="bg-secondary text-secondary-foreground rounded p-2 text-xs">
            <div className="font-medium mb-1">Secondary</div>
            <div>Secondary text</div>
          </div>
          <div className="bg-accent text-accent-foreground rounded p-2 text-xs">
            <div className="font-medium mb-1">Accent</div>
            <div>Accent text</div>
          </div>
        </div>
      </div>
    </div>
  );
}
