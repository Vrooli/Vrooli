/**
 * Settings Page - User preferences and system configuration
 *
 * PURPOSE:
 * Allows users to configure their UI preferences and system behavior.
 * Controls the recommendation engine mode, insights engine, and visual theme.
 *
 * CURRENT STATUS: Static UI without persistence
 * The UI elements are rendered but have no state management or persistence.
 * All settings reset on page refresh. This is a known limitation documented
 * in ERROR-SEMANTICS.md under "Settings Flow".
 *
 * FUTURE BEHAVIOR (when implemented):
 * - Theme: Persist to localStorage, apply CSS variables on change
 * - Recommendation Mode: Persist and sync with backend recommendation engine
 *   - "Off": No recommendations generated
 *   - "Suggestions": Generate recommendations for user review
 *   - "YOLO": Auto-approve low-risk recommendations after delay
 * - Custom Focus: Free-text input to guide recommendation priorities
 * - Insights: Toggle pattern detection and auto-analysis features
 * - Save button: Validate and persist all settings atomically
 *
 * DEPENDENCIES (not yet connected):
 * - localStorage for theme persistence
 * - API endpoint for backend settings sync
 * - Settings context for cross-component state
 *
 * Experience Architecture (Phase 29):
 * - Added contextual explanations for each recommendation mode
 * - Reduces cognitive load by explaining implications of each choice
 * - Help users make informed decisions without external documentation
 *
 * Related PRD targets: OT-P1-010
 */

import { HelpCircle, Info } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";

/** Contextual hints for recommendation modes */
const REC_MODE_HINTS = {
  off: "No recommendations will be generated. Use this if you want full manual control.",
  suggestions: "System analyzes scenarios and suggests improvements. You review and approve each one.",
  yolo: "Low-risk recommendations are auto-approved after a brief delay. High-risk changes still require approval.",
} as const;

export function SettingsPage() {
  return (
    <div className="space-y-6" data-testid={selectors.settings.page}>
      <h2 className="text-xl font-semibold">Settings</h2>

      {/* Settings persistence notice - navigation integrity (Phase 30) */}
      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3">
        <Info className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-sm text-amber-200">Settings preview mode</p>
          <p className="text-xs text-slate-400 mt-1">
            Settings persistence is not yet implemented. Changes shown here are for preview only and will reset on page refresh.
          </p>
        </div>
      </div>

      {/* Theme Settings */}
      <Card data-testid={selectors.settings.themeSettings}>
        <h3 className="text-lg font-medium text-slate-200">Theme</h3>
        <p className="mt-1 text-sm text-slate-400">Choose your preferred color theme</p>
        <div className="mt-4 flex gap-2">
          <button
            className="flex-1 rounded-lg border border-cyan-500 bg-slate-900 py-3 text-sm font-medium text-cyan-400"
            data-testid={selectors.settings.themeDark}
          >
            Dark
          </button>
          <button
            className="flex-1 rounded-lg border border-white/10 bg-slate-800/50 py-3 text-sm font-medium text-slate-400 hover:border-white/20"
            data-testid={selectors.settings.themeLight}
          >
            Light
          </button>
          <button
            className="flex-1 rounded-lg border border-white/10 bg-slate-800/50 py-3 text-sm font-medium text-slate-400 hover:border-white/20"
            data-testid={selectors.settings.themeSystem}
          >
            System
          </button>
        </div>
      </Card>

      {/* Recommendation Engine Settings */}
      <Card data-testid={selectors.settings.recommendationSettings}>
        <h3 className="text-lg font-medium text-slate-200">Recommendation Engine</h3>
        <p className="mt-1 text-sm text-slate-400">Control how the system suggests improvements</p>
        <div className="mt-4 space-y-3">
          <div className="flex gap-2">
            <button
              className="flex-1 rounded-lg border border-cyan-500 bg-slate-900 py-3 text-sm font-medium text-cyan-400"
              data-testid={selectors.settings.recModeOff}
            >
              Off
            </button>
            <button
              className="flex-1 rounded-lg border border-white/10 bg-slate-800/50 py-3 text-sm font-medium text-slate-400 hover:border-white/20"
              data-testid={selectors.settings.recModeSuggestions}
            >
              Suggestions
            </button>
            <button
              className="flex-1 rounded-lg border border-white/10 bg-slate-800/50 py-3 text-sm font-medium text-slate-400 hover:border-white/20"
              data-testid={selectors.settings.recModeYolo}
            >
              YOLO
            </button>
          </div>
          {/* Contextual hint explaining the selected mode (Phase 29) */}
          <div className="flex items-start gap-2 rounded-lg bg-slate-800/50 px-3 py-2" data-testid={selectors.settings.recModeHint}>
            <HelpCircle className="h-4 w-4 text-slate-500 mt-0.5 flex-shrink-0" />
            <p className="text-xs text-slate-400">{REC_MODE_HINTS.off}</p>
          </div>
        </div>
        <div className="mt-4">
          <label className="block text-sm font-medium text-slate-300">Custom Focus</label>
          <p className="mt-1 text-xs text-slate-500">Guide the recommendation engine toward specific areas</p>
          <Input
            type="text"
            placeholder="e.g., Focus on test coverage..."
            className="mt-2"
            data-testid={selectors.settings.customFocus}
          />
        </div>
      </Card>

      {/* Insights Settings */}
      <Card data-testid={selectors.settings.insightsSettings}>
        <h3 className="text-lg font-medium text-slate-200">Insights Engine</h3>
        <p className="mt-1 text-sm text-slate-400">Self-improvement suggestions based on patterns</p>
        <div className="mt-4 space-y-3">
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-white/20 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
              data-testid={selectors.settings.insightsEnabled}
            />
            <span className="text-sm text-slate-300">Enable insights</span>
          </label>
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-white/20 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
              data-testid={selectors.settings.insightsAutoAnalyze}
            />
            <span className="text-sm text-slate-300">Auto-analyze on scenario changes</span>
          </label>
        </div>
      </Card>

      {/* Save Button */}
      <div className="flex justify-end">
        <Button
          disabled
          title="Settings persistence coming soon"
          data-testid={selectors.settings.saveButton}
        >
          Save Settings
        </Button>
      </div>
    </div>
  );
}
