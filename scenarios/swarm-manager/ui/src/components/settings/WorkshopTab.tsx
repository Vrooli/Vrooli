/**
 * Workshop settings tab - Auto-initialize, auto-advance, and auto-cascade controls.
 */

import { Card } from "../ui/card";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import type { Settings } from "../../types";
import { ToggleButtons } from "./ToggleButtons";

export interface WorkshopTabProps {
  form: Settings;
  patch: (updates: Partial<Settings>) => void;
}

export function WorkshopTab({ form, patch }: WorkshopTabProps) {
  return (
    <div className="space-y-6">
      <Card data-testid={selectors.settings.workshopSettings}>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-slate-200">Workshop</h3>
            <p className="mt-1 text-sm text-slate-400">Controls for the workshop refinement system.</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              className="rounded border border-white/10 px-2 py-1 text-xs text-slate-400 hover:border-white/20 hover:text-slate-300"
              onClick={() => patch({
                autoInitializeWorkshop: false,
                autoAdvanceWorkshop: false,
                autoCascadeWorkshop: false,
              })}
            >
              Disable All
            </button>
            <button
              className="rounded border border-white/10 px-2 py-1 text-xs text-slate-400 hover:border-white/20 hover:text-slate-300"
              onClick={() => patch({
                autoInitializeWorkshop: true,
                autoAdvanceWorkshop: true,
                autoCascadeWorkshop: true,
              })}
            >
              Enable All
            </button>
            <button className="text-xs text-slate-500 hover:text-slate-300" onClick={() => patch({
              autoInitializeWorkshop: DEFAULT_SETTINGS.autoInitializeWorkshop,
              autoAdvanceWorkshop: DEFAULT_SETTINGS.autoAdvanceWorkshop,
              autoCascadeWorkshop: DEFAULT_SETTINGS.autoCascadeWorkshop,
              maxAutoRounds: DEFAULT_SETTINGS.maxAutoRounds,
              autoAdvanceDelaySeconds: DEFAULT_SETTINGS.autoAdvanceDelaySeconds,
            })}>Reset</button>
          </div>
        </div>
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300">Auto-Initialize Workshop</label>
            <p className="mt-1 text-xs text-slate-400">Automatically spawn the first workshop round when a backlog item is created.</p>
            <ToggleButtons
              value={form.autoInitializeWorkshop}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ autoInitializeWorkshop: v })}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Auto-Advance Workshop</label>
            <p className="mt-1 text-xs text-slate-400">Automatically continue workshop saves into either the next round or a final synthesis pass.</p>
            <ToggleButtons
              value={form.autoAdvanceWorkshop}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ autoAdvanceWorkshop: v })}
            />
          </div>
          <div
            className="border-t border-white/5 pt-4"
            style={{
              opacity: form.autoAdvanceWorkshop ? 1 : 0.5,
              pointerEvents: form.autoAdvanceWorkshop ? "auto" : "none",
            }}
          >
            <label className="block text-sm font-medium text-slate-300">Max Auto Rounds</label>
            <p className="mt-1 text-xs text-slate-400">Maximum workshop rounds before auto-advancement stops (0-50).</p>
            <Input
              type="number"
              min={0}
              max={50}
              className="mt-1"
              value={form.maxAutoRounds}
              onChange={(e) => patch({ maxAutoRounds: Math.max(0, Math.min(50, Number(e.target.value || 0))) })}
              disabled={!form.autoAdvanceWorkshop}
            />
          </div>
          <div
            className="border-t border-white/5 pt-4"
            style={{
              opacity: form.autoAdvanceWorkshop ? 1 : 0.5,
              pointerEvents: form.autoAdvanceWorkshop ? "auto" : "none",
            }}
          >
            <label className="block text-sm font-medium text-slate-300">Auto-Advance Delay</label>
            <p className="mt-1 text-xs text-slate-400">Grace period (seconds) before the next round spawns after all decisions are answered. Set to 0 for immediate.</p>
            <Input
              type="number"
              min={0}
              max={120}
              className="mt-1"
              value={form.autoAdvanceDelaySeconds}
              onChange={(e) => patch({ autoAdvanceDelaySeconds: Math.max(0, Math.min(120, Number(e.target.value || 0))) })}
              disabled={!form.autoAdvanceWorkshop}
            />
          </div>
          <div className="border-t border-white/5 pt-4">
            <label className="block text-sm font-medium text-slate-300">Auto-Cascade Workshop</label>
            <p className="mt-1 text-xs text-slate-400">Automatically trigger dependent item workshops when a dependency becomes ready.</p>
            <ToggleButtons
              value={form.autoCascadeWorkshop}
              options={[
                { value: false as const, label: "Disabled" },
                { value: true as const, label: "Enabled" },
              ]}
              onChange={(v) => patch({ autoCascadeWorkshop: v })}
            />
          </div>
        </div>
      </Card>
    </div>
  );
}
