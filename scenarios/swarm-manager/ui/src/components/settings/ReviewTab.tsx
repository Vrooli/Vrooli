/**
 * Review settings tab - Review thresholds and quality gates.
 */

import { Card } from "../ui/card";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import type { Settings } from "../../types";
import { ToggleButtons } from "./ToggleButtons";

export interface ReviewTabProps {
  form: Settings;
  patch: (updates: Partial<Settings>) => void;
}

export function ReviewTab({ form, patch }: ReviewTabProps) {
  return (
    <div className="space-y-6">
      {/* Review Agent */}
      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Review Agent</h3>
            <p className="mt-1 text-sm text-slate-400">
              Automatically gather evidence after execution completes. You can always trigger reviews manually.
            </p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            reviewAgentEnabled: DEFAULT_SETTINGS.reviewAgentEnabled,
          })}>Reset</button>
        </div>
        <div className="mt-4">
          <ToggleButtons
            value={form.reviewAgentEnabled}
            options={[
              { value: false as const, label: "Disabled" },
              { value: true as const, label: "Enabled" },
            ]}
            onChange={(v) => patch({ reviewAgentEnabled: v })}
          />
        </div>
      </Card>

      <Card data-testid={selectors.settings.reviewSettings}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Review Thresholds</h3>
            <p className="mt-1 text-sm text-slate-400">Configure what Git Control Tower considers passing, warning, or failing.</p>
          </div>
          <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
            reviewCodeQualityMinScore: DEFAULT_SETTINGS.reviewCodeQualityMinScore,
            reviewTestMinPassRate: DEFAULT_SETTINGS.reviewTestMinPassRate,
            reviewMaxBlockingViolations: DEFAULT_SETTINGS.reviewMaxBlockingViolations,
            reviewMaxWarnings: DEFAULT_SETTINGS.reviewMaxWarnings,
            reviewRequireScreenshots: DEFAULT_SETTINGS.reviewRequireScreenshots,
            reviewRequireTests: DEFAULT_SETTINGS.reviewRequireTests,
          })}>Reset</button>
        </div>
        <div className="mt-4 space-y-4">
          {/* Code Quality */}
          <div>
            <label className="block text-sm font-medium text-slate-300">Minimum Code Quality Score</label>
            <p className="mt-1 text-xs text-slate-400">Minimum tidiness score required for green status (0-100).</p>
            <Input
              type="number"
              min={0}
              max={100}
              className="mt-1"
              value={form.reviewCodeQualityMinScore}
              onChange={(e) => patch({ reviewCodeQualityMinScore: Math.max(0, Math.min(100, Number(e.target.value || 0))) })}
            />
          </div>

          {/* Tests */}
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Require Tests</label>
            <p className="mt-1 text-xs text-slate-400">Whether tests must exist and pass for green status.</p>
            <ToggleButtons
              value={form.reviewRequireTests}
              options={[
                { value: false as const, label: "Not Required" },
                { value: true as const, label: "Required" },
              ]}
              onChange={(v) => patch({ reviewRequireTests: v })}
            />
          </div>
          {form.reviewRequireTests && (
            <div>
              <label className="block text-sm font-medium text-slate-300">Minimum Test Pass Rate (%)</label>
              <p className="mt-1 text-xs text-slate-400">Minimum percentage of tests that must pass for green status (0-100).</p>
              <Input
                type="number"
                min={0}
                max={100}
                className="mt-1"
                value={Math.round(form.reviewTestMinPassRate * 100)}
                onChange={(e) => patch({ reviewTestMinPassRate: Math.max(0, Math.min(100, Number(e.target.value || 0))) / 100 })}
              />
            </div>
          )}

          {/* Standards */}
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Max Blocking Violations</label>
            <p className="mt-1 text-xs text-slate-400">Maximum critical/error violations allowed for green status.</p>
            <Input
              type="number"
              min={0}
              className="mt-1"
              value={form.reviewMaxBlockingViolations}
              onChange={(e) => patch({ reviewMaxBlockingViolations: Math.max(0, Number(e.target.value || 0)) })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300">Max Warnings</label>
            <p className="mt-1 text-xs text-slate-400">Maximum warnings before yellow status. Use -1 for unlimited.</p>
            <Input
              type="number"
              min={-1}
              className="mt-1"
              value={form.reviewMaxWarnings}
              onChange={(e) => patch({ reviewMaxWarnings: Math.max(-1, Number(e.target.value || -1)) })}
            />
          </div>

          {/* Visual */}
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Require Screenshots</label>
            <p className="mt-1 text-xs text-slate-400">Whether screenshots are required for green status.</p>
            <ToggleButtons
              value={form.reviewRequireScreenshots}
              options={[
                { value: false as const, label: "Not Required" },
                { value: true as const, label: "Required" },
              ]}
              onChange={(v) => patch({ reviewRequireScreenshots: v })}
            />
          </div>
        </div>
      </Card>
    </div>
  );
}
