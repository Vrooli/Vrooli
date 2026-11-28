import { useMemo } from "react";
import { FileDiff, FileText, Eye } from "lucide-react";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import type { ViewMode } from "../lib/api";
import { getFileTypeInfo } from "../lib/fileTypes";

interface ViewModeSelectorProps {
  mode: ViewMode;
  onChange: (mode: ViewMode) => void;
  disabled?: boolean;
  className?: string;
  compact?: boolean;
  /** File path to determine if preview is available */
  filePath?: string;
  /** Whether the file has git changes (shows Diff/Full+Diff buttons) */
  hasDiff?: boolean;
}

interface ModeOption {
  value: ViewMode;
  label: string;
  shortLabel: string;
  description: string;
  icon: typeof FileDiff;
}

const baseModes: ModeOption[] = [
  {
    value: "diff",
    label: "Diff",
    shortLabel: "Diff",
    description: "Show only changed lines"
    , icon: FileDiff
  },
  {
    value: "full_diff",
    label: "Full + Diff",
    shortLabel: "Full",
    description: "Show full file with changes highlighted"
    , icon: FileText
  },
  {
    value: "source",
    label: "Source",
    shortLabel: "Src",
    description: "Show file content only"
    , icon: FileText
  }
];

const previewMode: ModeOption = {
  value: "preview",
  label: "Preview",
  shortLabel: "View",
  description: "Preview rendered content"
  , icon: Eye
};

export function ViewModeSelector({
  mode,
  onChange,
  disabled = false,
  className,
  compact = false,
  filePath,
  hasDiff = true
}: ViewModeSelectorProps) {
  const modes = useMemo(() => {
    // Start with base modes, filtering out diff modes if no changes
    let availableModes = hasDiff
      ? baseModes
      : baseModes.filter((m) => m.value !== "diff" && m.value !== "full_diff");

    // Add preview mode if file type supports it
    if (filePath) {
      const fileType = getFileTypeInfo(filePath);
      if (fileType.canPreview) {
        availableModes = [...availableModes, previewMode];
      }
    }

    return availableModes;
  }, [filePath, hasDiff]);

  return (
    <div className={`${disabled ? "pointer-events-none opacity-50" : ""} ${className ?? ""}`}>
      <Tabs
        density="compact"
        items={modes.map((option) => ({
          id: option.value,
          label: compact ? option.shortLabel : option.label,
          icon: <option.icon aria-hidden="true" />,
        }))}
        active={mode}
        onChange={(next) => onChange(next as ViewMode)}
        ariaLabel="View mode"
        itemTestId={(item) => `view-mode-${item}`}
      />
    </div>
  );
}
