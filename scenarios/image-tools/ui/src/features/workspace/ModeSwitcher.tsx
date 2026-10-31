import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { MODE_LABEL } from "./modeLabels";
import { WORKSPACE_MODES, type WorkspaceMode } from "./useWorkspace";

export interface ModeSwitcherProps {
  mode: WorkspaceMode;
  onModeChange: (mode: WorkspaceMode) => void;
}

/**
 * Segmented control for the four Workspace modes the loaded image flows
 * across. Edit is wired in Stage 0b; the AI modes render a roadmap placeholder
 * in the Inspector until their stage lands.
 */
export function ModeSwitcher({ mode, onModeChange }: ModeSwitcherProps) {
  const { t } = useTranslation();

  return (
    <div
      role="group"
      aria-label={t(strings.workspace.mode.switcherLabel)}
      data-testid={selectors.workspace.modeSwitcher}
      className="inline-flex flex-wrap gap-1 rounded-control border border-app-border bg-app-surface-muted p-1"
    >
      {WORKSPACE_MODES.map((m) => {
        const active = mode === m;
        return (
          <button
            key={m}
            type="button"
            aria-pressed={active}
            data-testid={selectors.workspace.modeOption({ mode: m })}
            onClick={() => onModeChange(m)}
            className={
              active
                ? "rounded-control bg-app-primary px-3 py-1.5 text-sm font-medium text-app-primary-foreground"
                : "rounded-control px-3 py-1.5 text-sm font-medium text-app-muted-foreground hover:text-app-foreground"
            }
          >
            {t(MODE_LABEL[m])}
          </button>
        );
      })}
    </div>
  );
}
