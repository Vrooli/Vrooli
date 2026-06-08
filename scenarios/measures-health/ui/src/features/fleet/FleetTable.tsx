import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, XCircle, RefreshCw } from "lucide-react";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { fleetClient } from "../../api/fleet";
import { tierMeta } from "./coverage";

const FLEET_QUERY_KEY = ["fleet-coverage"] as const;

/**
 * FleetTable is the headline measures surface: a cross-scenario coverage
 * rollup over `ValidationService.ListFleetCoverage`. Each row is a button that
 * selects the scenario for the drill-down panel; the verdict mirrors the
 * gating contract (`passed=false` iff any stateful domain is UNCOVERED). The
 * worst tier across a scenario's covered domains is shown as a chip so a
 * fleet operator can spot weak-extraction scenarios at a glance.
 */
export function FleetTable({
  selected,
  onSelect,
}: {
  selected?: string;
  onSelect: (scenario: string) => void;
}) {
  const { t } = useTranslation();

  const query = useQuery({
    queryKey: FLEET_QUERY_KEY,
    queryFn: () => fleetClient.listFleetCoverage({}),
  });

  const entries = query.data?.entries ?? [];

  return (
    <section
      data-testid={selectors.fleet.table}
      aria-label={t(strings.fleet.title)}
      className="rounded-xl border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.fleet.title)}</h2>
        <Button
          data-testid={selectors.fleet.refreshButton}
          variant="outline"
          size="sm"
          className="ms-auto"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
          aria-label={t(strings.fleet.refresh)}
        >
          <RefreshCw aria-hidden="true" className={["h-4 w-4", query.isFetching ? "animate-spin" : ""].join(" ")} />
        </Button>
      </div>

      {query.isLoading && (
        <p data-testid={selectors.fleet.loading} className="mt-4 text-app-muted-foreground">
          {t(strings.fleet.loading)}
        </p>
      )}

      {query.error && (
        <p data-testid={selectors.fleet.error} className="mt-4 text-red-400">
          {errorMessage(query.error, t)}
        </p>
      )}

      {query.data && entries.length === 0 && (
        <p data-testid={selectors.fleet.empty} className="mt-4 text-app-muted-foreground">
          {t(strings.fleet.empty)}
        </p>
      )}

      {entries.length > 0 && (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="text-start text-xs uppercase tracking-wide text-app-muted-foreground">
                <th scope="col" className="px-2 py-1 text-start font-medium">{t(strings.fleet.col.scenario)}</th>
                <th scope="col" className="px-2 py-1 text-start font-medium">{t(strings.fleet.col.verdict)}</th>
                <th scope="col" className="px-2 py-1 text-end font-medium">{t(strings.fleet.col.expected)}</th>
                <th scope="col" className="px-2 py-1 text-end font-medium">{t(strings.fleet.col.covered)}</th>
                <th scope="col" className="px-2 py-1 text-end font-medium">{t(strings.fleet.col.waived)}</th>
                <th scope="col" className="px-2 py-1 text-end font-medium">{t(strings.fleet.col.uncovered)}</th>
                <th scope="col" className="px-2 py-1 text-start font-medium">{t(strings.fleet.col.tier)}</th>
                <th scope="col" className="px-2 py-1 text-end font-medium">{t(strings.fleet.col.measures)}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => {
                const tier = tierMeta(entry.worstTier);
                const isSelected = entry.scenario === selected;
                return (
                  <tr
                    key={entry.scenario}
                    data-testid={selectors.fleet.row({ scenario: entry.scenario })}
                    data-selected={isSelected}
                    className={[
                      "cursor-pointer border-t border-app-border transition-colors",
                      isSelected ? "bg-app-surface-muted" : "hover:bg-app-surface-muted/60",
                    ].join(" ")}
                    onClick={() => onSelect(entry.scenario)}
                  >
                    <td className="px-2 py-1.5">
                      <button
                        type="button"
                        className="font-medium text-app-foreground hover:underline focus:underline focus:outline-none"
                        aria-pressed={isSelected}
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelect(entry.scenario);
                        }}
                      >
                        {entry.scenario}
                      </button>
                    </td>
                    <td className="px-2 py-1.5">
                      <span
                        data-testid={selectors.fleet.rowVerdict({ scenario: entry.scenario })}
                        data-passed={entry.passed}
                        className={[
                          "inline-flex items-center gap-1 rounded-control px-1.5 py-0.5 text-xs font-medium",
                          entry.passed
                            ? "border border-emerald-500/40 bg-emerald-500/10 text-emerald-300"
                            : "border border-red-500/40 bg-red-500/10 text-red-300",
                        ].join(" ")}
                      >
                        {entry.passed ? (
                          <CheckCircle2 aria-hidden="true" className="h-3.5 w-3.5" />
                        ) : (
                          <XCircle aria-hidden="true" className="h-3.5 w-3.5" />
                        )}
                        {entry.passed ? t(strings.fleet.passed) : t(strings.fleet.failed)}
                      </span>
                    </td>
                    <td className="px-2 py-1.5 text-end tabular-nums">{entry.expected}</td>
                    <td className="px-2 py-1.5 text-end tabular-nums text-emerald-300">{entry.covered}</td>
                    <td className="px-2 py-1.5 text-end tabular-nums text-amber-300">{entry.waived}</td>
                    <td className="px-2 py-1.5 text-end tabular-nums text-red-300">{entry.uncovered}</td>
                    <td className="px-2 py-1.5">
                      {entry.covered > 0 ? (
                        <span className={["rounded px-1.5 py-0.5 text-xs font-semibold uppercase", tier.chipClass].join(" ")}>
                          {t(tier.labelKey)}
                        </span>
                      ) : (
                        <span className="text-app-muted-foreground">{t(strings.fleet.tier.none)}</span>
                      )}
                    </td>
                    <td className="px-2 py-1.5 text-end tabular-nums">{entry.measureCount}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
