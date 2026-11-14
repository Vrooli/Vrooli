import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { AppliedOp, BaseImage } from "./useWorkspace";

export interface HistoryRailProps {
  base: BaseImage | null;
  entries: AppliedOp[];
}

/**
 * The unified non-destructive history: the source image followed by each
 * applied step in order. Read-only in Stage 0b (undo/redo drive it); per-step
 * reorder/toggle/remove arrive with the Edit-hero polish in Stage 1.
 */
export function HistoryRail({ base, entries }: HistoryRailProps) {
  const { t } = useTranslation();

  return (
    <aside
      data-testid={selectors.workspace.history.rail}
      aria-label={t(strings.workspace.history.title)}
      className="rounded-panel border border-app-border bg-app-surface p-3"
    >
      <h3 className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
        {t(strings.workspace.history.title)}
      </h3>
      {base === null ? (
        <p
          data-testid={selectors.workspace.history.empty}
          className="mt-2 text-xs text-app-muted-foreground"
        >
          {t(strings.workspace.history.empty)}
        </p>
      ) : (
        <ol
          data-testid={selectors.workspace.history.list}
          className="mt-2 space-y-1 text-xs text-app-foreground"
        >
          <li
            data-testid={selectors.workspace.history.source}
            className="rounded border border-app-border px-2 py-1 text-app-muted-foreground"
          >
            {t(strings.workspace.history.source)}
          </li>
          {entries.map((entry, index) => (
            <li
              key={`${entry.operation}-${index}`}
              data-testid={selectors.workspace.historyStep({ index: index + 1 })}
              className="rounded border border-app-border px-2 py-1"
            >
              {t(strings.workspace.history.step, { index: index + 1, operation: entry.operation })}
            </li>
          ))}
        </ol>
      )}
    </aside>
  );
}
