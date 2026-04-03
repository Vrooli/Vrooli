/**
 * General settings tab - Theme and UI preferences.
 */

import { Card } from "../ui/card";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import type { Settings } from "../../types";
import { ToggleButtons } from "./ToggleButtons";

export interface GeneralTabProps {
  form: Settings;
  patch: (updates: Partial<Settings>) => void;
  onThemeChange: (theme: Settings["theme"]) => void;
}

export function GeneralTab({ form, patch, onThemeChange }: GeneralTabProps) {
  return (
    <div className="space-y-6">
      {/* Theme */}
      <Card data-testid={selectors.settings.themeSettings}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Theme</h3>
            <p className="mt-1 text-sm text-slate-400">Choose your preferred color theme</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({ theme: DEFAULT_SETTINGS.theme })}>Reset</button>
        </div>
        <div className="mt-4 flex gap-2">
          <button
            className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.theme === "dark" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
            data-testid={selectors.settings.themeDark}
            onClick={() => onThemeChange("dark")}
          >
            Dark
          </button>
          <button
            className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.theme === "light" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
            data-testid={selectors.settings.themeLight}
            onClick={() => onThemeChange("light")}
          >
            Light
          </button>
          <button
            className={`flex-1 rounded-lg border py-3 text-sm font-medium ${form.theme === "system" ? "border-cyan-500 bg-slate-900 text-cyan-400" : "border-white/10 bg-slate-800/50 text-slate-400 hover:border-white/20"}`}
            data-testid={selectors.settings.themeSystem}
            onClick={() => onThemeChange("system")}
          >
            System
          </button>
        </div>
      </Card>

      {/* UI Preferences */}
      <Card data-testid={selectors.settings.uiPreferences}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">UI Preferences</h3>
            <p className="mt-1 text-sm text-slate-400">Customize the interface behavior.</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            searchDebounceMs: DEFAULT_SETTINGS.searchDebounceMs,
            toastDurationMs: DEFAULT_SETTINGS.toastDurationMs,
            confirmDestructiveActions: DEFAULT_SETTINGS.confirmDestructiveActions,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Search Debounce (ms)</label>
            <p className="mt-1 text-xs text-slate-400">Delay before search requests fire while typing (100-2000).</p>
            <Input
              type="number"
              min={100}
              max={2000}
              className="mt-1"
              value={form.searchDebounceMs}
              onChange={(e) => patch({ searchDebounceMs: Math.max(100, Math.min(2000, Number(e.target.value || 100))) })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300">Toast Duration (seconds)</label>
            <p className="mt-1 text-xs text-slate-400">How long notification messages remain visible (1-30).</p>
            <Input
              type="number"
              min={1}
              max={30}
              className="mt-1"
              value={Math.round(form.toastDurationMs / 1000)}
              onChange={(e) => patch({ toastDurationMs: Math.max(1000, Math.min(30000, Number(e.target.value || 1) * 1000)) })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Confirm Destructive Actions</label>
            <p className="mt-1 text-xs text-slate-400">Show confirmation dialogs before irreversible operations.</p>
            <ToggleButtons
              value={form.confirmDestructiveActions}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ confirmDestructiveActions: v })}
            />
          </div>
        </div>
      </Card>
    </div>
  );
}
