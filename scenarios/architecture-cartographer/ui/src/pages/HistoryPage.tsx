import * as React from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useRecentTargets } from "../features/targets/hooks/useRecentTargets";
import { BuildStatusDeltas } from "../features/analytics/BuildStatusDeltas";
import { EventsTimeline } from "../features/analytics/EventsTimeline";
import { StatsPanel } from "../features/analytics/StatsPanel";
import { EmptyState } from "../components/EmptyState";

export function HistoryPage() {
  const { t } = useTranslation();
  const { recent } = useRecentTargets();
  const [scenario, setScenario] = React.useState<string>(() => recent[0]?.scenario ?? "");

  // Keep the picker in sync if the recent-list becomes non-empty after first
  // render (e.g. user opens a target in another tab).
  React.useEffect(() => {
    if (scenario === "" && recent.length > 0) {
      setScenario(recent[0]?.scenario ?? "");
    }
  }, [recent, scenario]);

  return (
    <section
      data-testid={selectors.pages.history}
      aria-labelledby="history-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="history-heading" className="text-2xl font-semibold">
          {t(strings.pages.history.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.history.description)}
        </p>
      </header>

      {recent.length === 0 ? (
        <EmptyState title={t(strings.pages.history.noScenario)} />
      ) : (
        <>
          <label className="flex flex-col gap-1 text-sm">
            <span>{t(strings.pages.history.scenarioLabel)}</span>
            <select
              value={scenario}
              onChange={(e) => setScenario(e.target.value)}
              className="rounded-control border border-app-border bg-app-surface px-2 py-1"
            >
              {recent.map((r) => (
                <option key={r.scenario} value={r.scenario}>
                  {r.scenario}
                </option>
              ))}
            </select>
          </label>

          {scenario.length > 0 ? (
            <>
              <StatsPanel scenario={scenario} />
              <EventsTimeline scenario={scenario} />
              <BuildStatusDeltas scenario={scenario} />
            </>
          ) : null}
        </>
      )}
    </section>
  );
}
