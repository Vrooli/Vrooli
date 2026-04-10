import { memo } from "react";
import { List, Layers, FolderTree } from "lucide-react";
import type { FileViewMode } from "../lib/api";

interface ViewModeCycleButtonProps {
  mode: FileViewMode;
  onCycle: () => void;
  groupingAvailable: boolean;
  compact?: boolean;
}

const modeConfig: Record<
  FileViewMode,
  { Icon: typeof List; color: string; label: string; nextLabel: string }
> = {
  flat: {
    Icon: List,
    color: "text-slate-300 border-slate-600 hover:bg-slate-800/50",
    label: "Flat view",
    nextLabel: "grouped",
  },
  grouped: {
    Icon: Layers,
    color: "text-blue-300 border-blue-500/40 bg-blue-500/10 hover:bg-blue-500/20",
    label: "Grouped view",
    nextLabel: "tree",
  },
  tree: {
    Icon: FolderTree,
    color: "text-emerald-300 border-emerald-500/40 bg-emerald-500/10 hover:bg-emerald-500/20",
    label: "Tree view",
    nextLabel: "flat",
  },
};

export const ViewModeCycleButton = memo(function ViewModeCycleButton({
  mode,
  onCycle,
  groupingAvailable,
  compact,
}: ViewModeCycleButtonProps) {
  const { Icon, color, label, nextLabel } = modeConfig[mode];

  // Determine actual next mode based on groupingAvailable
  const actualNextLabel =
    mode === "flat" && !groupingAvailable
      ? "tree"
      : mode === "grouped"
        ? "tree"
        : nextLabel;

  const tooltip = `${label} (click for ${actualNextLabel})`;

  return (
    <button
      type="button"
      onClick={onCycle}
      className={`${compact ? "h-7 w-7" : "h-9 w-9"} inline-flex items-center justify-center rounded-full border transition-colors ${color}`}
      title={tooltip}
      aria-label={tooltip}
      data-testid="view-mode-cycle-button"
    >
      <Icon className={compact ? "h-3 w-3" : "h-4 w-4"} />
    </button>
  );
});
