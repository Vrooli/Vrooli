import type { LayoutPreset, LayoutSection } from "./LayoutSettingsModal";
import { Button } from "./ui/button";

interface SettingsTabLayoutProps {
  preset: LayoutPreset;
  primaryPanel: LayoutSection;
  onChangePreset: (preset: LayoutPreset) => void;
  onChangePrimary: (panel: LayoutSection) => void;
  onReset: () => void;
  isMobile: boolean;
}

const presetLabels: Record<LayoutPreset, string> = {
  classic: "Classic sidebar",
  split: "Split right",
  bottom: "Bottom stack"
};

const panelLabels: Record<LayoutSection, string> = {
  changes: "Changes",
  history: "History",
  commit: "Commit",
  diff: "Diff viewer"
};

export function SettingsTabLayout({
  preset,
  primaryPanel,
  onChangePreset,
  onChangePrimary,
  onReset,
  isMobile
}: SettingsTabLayoutProps) {
  if (isMobile) {
    return (
      <div className="space-y-6">
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-slate-200">Layout Preset</h3>
          <div className="grid grid-cols-1 gap-3">
            {(Object.keys(presetLabels) as LayoutPreset[]).map((key) => (
              <button
                key={key}
                type="button"
                onClick={() => onChangePreset(key)}
                className={`rounded-xl border px-4 py-4 text-sm text-left transition touch-target ${
                  preset === key
                    ? "border-blue-400/60 bg-blue-500/10 text-blue-100"
                    : "border-slate-800 bg-slate-900/40 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700/60"
                }`}
              >
                {presetLabels[key]}
              </button>
            ))}
          </div>
          <p className="text-xs text-slate-500">
            Presets move the stack of non-primary panels.
          </p>
        </div>

        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-slate-200">Primary Panel</h3>
          <div className="grid grid-cols-2 gap-3">
            {(Object.keys(panelLabels) as LayoutSection[]).map((panel) => (
              <button
                key={panel}
                type="button"
                onClick={() => onChangePrimary(panel)}
                className={`rounded-xl border px-4 py-4 text-sm text-left transition touch-target ${
                  primaryPanel === panel
                    ? "border-emerald-400/60 bg-emerald-500/10 text-emerald-100"
                    : "border-slate-800 bg-slate-900/40 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700/60"
                }`}
              >
                {panelLabels[panel]}
              </button>
            ))}
          </div>
          <p className="text-xs text-slate-500">
            The primary panel takes the large area. Others remain stacked.
          </p>
        </div>

        <div className="pt-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onReset}
            className="w-full h-11 text-sm"
          >
            Reset to Default
          </Button>
        </div>
      </div>
    );
  }

  // Desktop layout
  return (
    <div className="space-y-5">
      <div className="space-y-2">
        <h3 className="text-xs font-semibold text-slate-200">Layout preset</h3>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
          {(Object.keys(presetLabels) as LayoutPreset[]).map((key) => (
            <button
              key={key}
              type="button"
              onClick={() => onChangePreset(key)}
              className={`rounded-lg border px-3 py-2 text-xs text-left transition ${
                preset === key
                  ? "border-blue-400/60 bg-blue-500/10 text-blue-100"
                  : "border-slate-800 bg-slate-950/60 text-slate-300 hover:bg-slate-900/60"
              }`}
            >
              {presetLabels[key]}
            </button>
          ))}
        </div>
        <p className="text-[11px] text-slate-500">
          Presets move the stack of non-primary panels.
        </p>
      </div>

      <div className="space-y-2">
        <h3 className="text-xs font-semibold text-slate-200">Primary panel</h3>
        <div className="grid grid-cols-2 gap-2">
          {(Object.keys(panelLabels) as LayoutSection[]).map((panel) => (
            <button
              key={panel}
              type="button"
              onClick={() => onChangePrimary(panel)}
              className={`rounded-lg border px-3 py-2 text-xs text-left transition ${
                primaryPanel === panel
                  ? "border-emerald-400/60 bg-emerald-500/10 text-emerald-100"
                  : "border-slate-800 bg-slate-950/60 text-slate-300 hover:bg-slate-900/60"
              }`}
            >
              {panelLabels[panel]}
            </button>
          ))}
        </div>
        <p className="text-[11px] text-slate-500">
          The primary panel takes the large area. Others remain stacked.
        </p>
      </div>

      <div className="pt-1">
        <Button variant="outline" size="sm" onClick={onReset} className="h-8 px-3">
          Reset to default
        </Button>
      </div>
    </div>
  );
}
