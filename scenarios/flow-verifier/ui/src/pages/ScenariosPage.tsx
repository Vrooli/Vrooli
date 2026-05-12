/**
 * ScenariosPage — primary inventory landing.
 *
 * Renders one card per scenario discovered under the Vrooli root.
 * Clicking a card drills into the scenario-detail page, which is
 * where flows are listed and verified. The flat "all flows" view at
 * `/flows` is still reachable; this page is the *first* surface a
 * user sees in the inventory side of the app.
 *
 * The diagnostic empty state surfaces the resolved Vrooli root and a
 * count of directories scanned, so a misconfigured deploy is
 * self-debugging instead of producing a silent zero-state.
 */
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Layers, AlertTriangle } from "lucide-react";

import { fetchScenarios, type ScenarioSummary } from "../api/scenarios";
import { errorMessage } from "../lib/errorMessage";
import { useTranslation } from "../i18n";

const SCENARIOS_KEY = ["scenarios"] as const;

export function ScenariosPage() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: SCENARIOS_KEY,
    queryFn: fetchScenarios,
  });

  return (
    <div data-testid="scenarios-page" className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("scenarios.heading", { defaultValue: "Scenarios" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("scenarios.subtitle", {
            defaultValue: "Every scenario discovered under the Vrooli root. Click one to inspect its flows.",
          })}
        </p>
      </header>

      {query.isLoading && (
        <p data-testid="scenarios-loading" className="text-sm text-app-muted-foreground">
          {t("scenarios.loading", { defaultValue: "Discovering scenarios…" })}
        </p>
      )}

      {query.error && (
        <p data-testid="scenarios-error" className="text-sm text-app-danger">
          {errorMessage(query.error, t)}
        </p>
      )}

      {!query.isLoading && !query.error && query.data && (
        <ScenariosBody data={query.data} />
      )}
    </div>
  );
}

function ScenariosBody({
  data,
}: {
  data: { vrooliRoot: string; scenarios: ScenarioSummary[] };
}) {
  const { t } = useTranslation();
  if (data.scenarios.length === 0) {
    return (
      <div
        data-testid="scenarios-empty"
        className="rounded-panel border border-app-border bg-app-surface p-6 text-sm"
      >
        <h2 className="text-base font-semibold text-app-foreground">
          {t("scenarios.empty.title", { defaultValue: "No scenarios found" })}
        </h2>
        <p className="mt-2 text-app-muted-foreground">
          {t("scenarios.empty.body", {
            defaultValue:
              "Looked under {{root}}/scenarios. Generate a scenario from a template, then it will appear here.",
            root: data.vrooliRoot,
          })}
        </p>
      </div>
    );
  }

  return (
    <ul
      data-testid="scenarios-grid"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
    >
      {data.scenarios.map((s) => (
        <li key={s.id}>
          <ScenarioCard scenario={s} />
        </li>
      ))}
    </ul>
  );
}

function ScenarioCard({ scenario }: { scenario: ScenarioSummary }) {
  const { t } = useTranslation();
  const flowCountLabel = t("scenarios.card.flowCount", {
    defaultValue: "{{count}} flow",
    defaultValue_other: "{{count}} flows",
    count: scenario.flowCount,
  });
  return (
    <Link
      to={`/scenarios/${encodeURIComponent(scenario.id)}`}
      data-testid={`scenario-card-${scenario.id}`}
      className="flex h-full flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4 transition-colors hover:border-app-primary hover:bg-app-surface-muted"
    >
      <header className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-base font-semibold text-app-foreground">
            {scenario.displayName}
          </h2>
          <p className="truncate font-mono text-xs text-app-muted-foreground">
            {scenario.id}
          </p>
        </div>
        <span
          aria-hidden
          className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-control bg-app-primary/10 text-app-primary"
        >
          <Layers className="h-4 w-4" />
        </span>
      </header>
      {scenario.description && (
        <p className="line-clamp-2 text-sm text-app-muted-foreground">
          {scenario.description}
        </p>
      )}
      <footer className="mt-auto flex items-center gap-2 pt-2 text-xs">
        {scenario.discoveryError ? (
          <span
            data-testid={`scenario-card-error-${scenario.id}`}
            className="inline-flex items-center gap-1 text-app-danger"
          >
            <AlertTriangle className="h-3 w-3" />
            {scenario.discoveryError}
          </span>
        ) : (
          <span
            data-testid={`scenario-card-flowcount-${scenario.id}`}
            className="text-app-muted-foreground"
          >
            {flowCountLabel}
          </span>
        )}
      </footer>
    </Link>
  );
}
