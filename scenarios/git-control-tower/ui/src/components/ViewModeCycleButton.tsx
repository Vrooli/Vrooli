import { memo } from "react";
import { List, Layers, FolderTree } from "lucide-react";
import type { FileViewMode } from "../lib/api";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";

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
    color: "text-slate-300",
    label: "Flat view",
    nextLabel: "grouped",
  },
  grouped: {
    Icon: Layers,
    color: "text-blue-300",
    label: "Grouped view",
    nextLabel: "tree",
  },
  tree: {
    Icon: FolderTree,
    color: "text-emerald-300",
    label: "Tree view",
    nextLabel: "flat",
  },
};

export const ViewModeCycleButton = memo(function ViewModeCycleButton({
  mode,
  onCycle,
  groupingAvailable,
  compact: _compact,
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
    <IconButton
      onClick={onCycle}
      aria-label={tooltip}
      title={tooltip}
      size="xs"
      surface="ghost"
      swapIdentity="gct-view-mode"
      // These are different Lucide component trees, not compatible morph
      // paths. Automatic morphing can briefly hide the glyph while the RCL
      // transition layer is waiting for a path match.
      morph="none"
      denseTapTarget
      className={`!h-8 !w-8 !min-h-0 !min-w-0 !border-0 !shadow-none ${color.split(" ").filter((token) => token.startsWith("text-")).join(" ")}`}
      data-testid="view-mode-cycle-button"
    >
      <Icon className="h-4 w-4" />
    </IconButton>
  );
});
