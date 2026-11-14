import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { clearHistory, useSearchHistory, type SearchMode } from "../../lib/searchHistory";
import { useTranslation } from "../../i18n";

const MODE_LABEL = {
  live: strings.history.modeLive,
  learnings: strings.history.modeLearnings,
} as const satisfies Record<SearchMode, string>;

/**
 * HistoryPanel lists the recent searches (persisted in localStorage) and lets
 * the user replay one — which re-runs it through the SearchPanel via the
 * `onReplay` callback the page wires up.
 */
export function HistoryPanel({
  onReplay,
}: {
  onReplay: (query: string, mode: SearchMode) => void;
}) {
  const { t } = useTranslation();
  const entries = useSearchHistory();

  return (
    <aside
      data-testid={selectors.history.panel}
      aria-labelledby="history-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex items-center justify-between">
        <h3 id="history-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.history.title)}
        </h3>
        {entries.length > 0 && (
          <Button
            data-testid={selectors.history.clear}
            variant="outline"
            size="sm"
            onClick={() => clearHistory()}
            className="text-xs"
          >
            {t(strings.history.clear)}
          </Button>
        )}
      </div>

      {entries.length === 0 ? (
        <p data-testid={selectors.history.empty} className="mt-2 text-sm text-app-muted-foreground">
          {t(strings.history.empty)}
        </p>
      ) : (
        <ul data-testid={selectors.history.list} className="mt-2 flex flex-col gap-1">
          {entries.map((entry) => (
            <li key={`${entry.mode}:${entry.query}`}>
              <button
                type="button"
                data-testid={selectors.history.item}
                aria-label={t(strings.history.replay)}
                onClick={() => onReplay(entry.query, entry.mode)}
                className="flex w-full items-center justify-between gap-2 rounded-control px-2 py-1.5 text-start text-sm hover:bg-app-surface-muted"
              >
                <span className="truncate text-app-foreground">{entry.query}</span>
                <span className="shrink-0 rounded-pill border border-app-border px-1.5 py-0.5 text-xs text-app-muted-foreground">
                  {t(MODE_LABEL[entry.mode])}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
