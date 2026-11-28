import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";

export interface CanvasActionBarProps {
  canUndo: boolean;
  canRedo: boolean;
  hasSteps: boolean;
  /** Object URL of the current result; download is shown only when present. */
  downloadUrl: string | null;
  downloadName: string;
  onUndo: () => void;
  onRedo: () => void;
  onReset: () => void;
}

/**
 * The Workspace canvas action bar: undo/redo/reset over the non-destructive
 * history plus a download of the current result. Lives above the canvas so the
 * primary editing controls sit next to the work surface.
 */
export function CanvasActionBar({
  canUndo,
  canRedo,
  hasSteps,
  downloadUrl,
  downloadName,
  onUndo,
  onRedo,
  onReset,
}: CanvasActionBarProps) {
  const { t } = useTranslation();

  return (
    <div
      data-testid={selectors.workspace.actions.bar}
      className="flex flex-wrap items-center gap-2"
    >
      <Button
        variant="ghost"
        size="sm"
        type="button"
        data-testid={selectors.workspace.actions.undo}
        onClick={onUndo}
        disabled={!canUndo}
        shape="square"
        className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
      >
        {t(strings.workspace.actions.undo)}
      </Button>
      <Button
        variant="ghost"
        size="sm"
        type="button"
        data-testid={selectors.workspace.actions.redo}
        onClick={onRedo}
        disabled={!canRedo}
        shape="square"
        className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
      >
        {t(strings.workspace.actions.redo)}
      </Button>
      <Button
        variant="ghost"
        size="sm"
        type="button"
        data-testid={selectors.workspace.actions.reset}
        onClick={onReset}
        disabled={!hasSteps}
        shape="square"
        className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
      >
        {t(strings.workspace.actions.reset)}
      </Button>
      {hasSteps && downloadUrl && (
        <a
          data-testid={selectors.workspace.actions.download}
          href={downloadUrl}
          download={downloadName}
          className="inline-flex h-9 items-center rounded-control px-4 text-sm font-medium text-app-primary hover:bg-app-surface-muted"
        >
          {t(strings.workspace.actions.download)}
        </a>
      )}
    </div>
  );
}
