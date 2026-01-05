// Time window selector component

import { useTimeWindow, getPresetShortLabel } from "../../hooks/useTimeWindow";
import type { TimePreset } from "../../api/types";

export function TimeWindowSelector() {
  const { preset, setPreset, presetOptions } = useTimeWindow();

  return (
    <div className="flex items-center gap-1 rounded-lg border border-border/60 bg-card/50 p-1">
      {presetOptions.map((option) => (
        <button
          key={option}
          onClick={() => setPreset(option)}
          className={`
            px-3 py-1.5 text-sm font-medium rounded-md transition-colors
            ${
              preset === option
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
            }
          `}
          aria-pressed={preset === option}
        >
          {getPresetShortLabel(option)}
        </button>
      ))}
    </div>
  );
}
