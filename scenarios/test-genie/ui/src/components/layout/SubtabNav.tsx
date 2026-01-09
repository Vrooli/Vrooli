import { cn } from "../../lib/utils";
import type { RunsSubtabKey } from "../../types";
import { selectors } from "../../consts/selectors";

interface SubtabNavProps {
  activeSubtab: RunsSubtabKey;
  onSubtabChange: (subtab: RunsSubtabKey) => void;
}

const SUBTABS = [
  { key: "scenarios" as const, label: "Scenarios" },
  { key: "history" as const, label: "History" }
];

export function SubtabNav({ activeSubtab, onSubtabChange }: SubtabNavProps) {
  return (
    <div className="flex gap-1 rounded-full border border-white/10 bg-black/30 p-1">
      {SUBTABS.map((subtab) => (
        <button
          key={subtab.key}
          type="button"
          onClick={() => onSubtabChange(subtab.key)}
          className={cn(
            "rounded-full px-4 py-1.5 text-sm transition",
            activeSubtab === subtab.key
              ? "bg-white/10 text-white"
              : "text-slate-400 hover:text-white"
          )}
          data-testid={subtab.key === "scenarios" ? selectors.runs.subtabScenarios : selectors.runs.subtabHistory}
        >
          {subtab.label}
        </button>
      ))}
    </div>
  );
}
