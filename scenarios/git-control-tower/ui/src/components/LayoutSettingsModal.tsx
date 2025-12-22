import { LayoutGrid, X } from "lucide-react";
import { Button } from "./ui/button";

export type LayoutPreset = "classic" | "split" | "bottom";
export type LayoutSection = "changes" | "history" | "commit" | "diff";

interface LayoutSettingsModalProps {
  isOpen: boolean;
  repoDir?: string;
  preset: LayoutPreset;
  primaryPanel: LayoutSection;
  onChangePreset: (preset: LayoutPreset) => void;
  onChangePrimary: (panel: LayoutSection) => void;
  onReset: () => void;
  onClose: () => void;
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

export function LayoutSettingsModal({
  isOpen,
  repoDir,
  preset,
  primaryPanel,
  onChangePreset,
  onChangePrimary,
  onReset,
  onClose
}: LayoutSettingsModalProps) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Layout settings"
    >
      <div className="w-full max-w-xl rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div className="flex items-center gap-2">
            <LayoutGrid className="h-4 w-4 text-slate-400" />
            <div>
              <h2 className="text-sm font-semibold text-slate-100">Layout settings</h2>
              {repoDir && (
                <p className="text-[11px] text-slate-500 mt-1">Repo: {repoDir}</p>
              )}
            </div>
          </div>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onClose}
            aria-label="Close layout settings"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-4 py-4 space-y-5">
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
        </div>

        <div className="flex items-center justify-between border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onReset} className="h-8 px-3">
            Reset to default
          </Button>
          <Button variant="outline" size="sm" onClick={onClose} className="h-8 px-3">
            Done
          </Button>
        </div>
      </div>
    </div>
  );
}
