import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { GENERATION_PRESETS } from "../../lib/constants";

interface PresetSelectorProps {
  selectedPreset: string | null;
  onSelectPreset: (preset: string) => void;
}

export function PresetSelector({ selectedPreset, onSelectPreset }: PresetSelectorProps) {
  return (
    <div data-testid={selectors.generate.presetSelector}>
      <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Quick start</p>
      <h3 className="mt-2 text-lg font-semibold">Preset templates</h3>
      <p className="mt-2 text-sm text-slate-300">
        Start with a common pattern or build your own prompt.
      </p>

      <div className="mt-4 grid gap-3 md:grid-cols-3">
        {GENERATION_PRESETS.map((preset) => (
          <button
            key={preset.key}
            type="button"
            onClick={() => onSelectPreset(preset.key)}
            className={cn(
              "rounded-xl border p-4 text-left transition",
              selectedPreset === preset.key
                ? "border-purple-400 bg-purple-400/10"
                : "border-white/10 bg-black/30 hover:border-white/30"
            )}
          >
            <span className="font-semibold">{preset.label}</span>
            <p className="mt-2 text-xs text-slate-400">{preset.description}</p>
          </button>
        ))}
      </div>
    </div>
  );
}
