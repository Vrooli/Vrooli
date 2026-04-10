import { Code2, Eye, SplitSquareVertical } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import type { DocViewMode } from "../../../shared/hooks/viewerHooks";
import { selectors } from "../../../consts/selectors";

export type ViewModeToggleProps = {
  mode: DocViewMode;
  onChange: (mode: DocViewMode) => void;
};

const options: Array<{ mode: DocViewMode; label: string; icon: JSX.Element }> = [
  { mode: "code", label: "Code", icon: <Code2 className="h-4 w-4" /> },
  { mode: "preview", label: "Preview", icon: <Eye className="h-4 w-4" /> },
  { mode: "split", label: "Split", icon: <SplitSquareVertical className="h-4 w-4" /> },
];

export function ViewModeToggle({ mode, onChange }: ViewModeToggleProps) {
  return (
    <div className="ko-viewer-mode-toggle" data-testid={selectors.viewer.modeToggle}>
      {options.map((option) => {
        const isActive = mode === option.mode;
        return (
          <Button
            key={option.mode}
            type="button"
            variant={isActive ? "primary" : "outline"}
            size="compact"
            onClick={() => onChange(option.mode)}
            className="ko-viewer-mode-button"
          >
            {option.icon}
            {option.label}
          </Button>
        );
      })}
    </div>
  );
}
