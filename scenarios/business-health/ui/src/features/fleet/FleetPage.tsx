import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AlertTriangle, ArrowDown, ArrowUp, RefreshCw } from "lucide-react";

import { StatusChip } from "../../components/StatusChip";
import {
  DEFAULT_SORT,
  EMPTY_FILTERS,
  filterEntries,
  sortEntries,
  toggleSort,
  type FleetFilters,
  type SortColumn,
  type SortState,
} from "./fleetModel";
import { useFleet } from "./useFleet";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { formatDate } from "../../i18n/format";
import { timestampToDate } from "../../lib/protoTime";
import { useTranslation } from "../../i18n";

// `label` holds the typed key path (referenced explicitly so the
// no-unused-keys audit sees every `strings.fleet.column.*` leaf).
const COLUMNS = [
  { col: "scenario", label: strings.fleet.column.scenario, numeric: false },
  { col: "status", label: strings.fleet.column.status, numeric: false },
  { col: "errors", label: strings.fleet.column.errors, numeric: true },
  { col: "warnings", label: strings.fleet.column.warnings, numeric: true },
  { col: "autofix", label: strings.fleet.column.autofix, numeric: true },
  { col: "orphans", label: strings.fleet.column.orphans, numeric: true },
  { col: "unproven", label: strings.fleet.column.unproven, numeric: true },
  { col: "template", label: strings.fleet.column.template, numeric: false },
  { col: "debt", label: strings.fleet.column.debt, numeric: true },
] as const satisfies readonly { col: SortColumn; label: string; numeric: boolean }[];

/**
 * Worst-first fleet grade — every discovered scenario ranked by business
 * contract debt. Auto-scans on mount (no scenario picker; the scan covers the
 * whole fleet), with client-side filtering and sortable columns. Clicking a row
 * deep-links into that scenario's traceability matrix.
 */
export function FleetPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const query = useFleet();
  const response = query.data;

  const [sort, setSort] = useState<SortState>(DEFAULT_SORT);
  const [filters, setFilters] = useState<FleetFilters>(EMPTY_FILTERS);

  const rows = useMemo(() => {
    if (!response) return [];
    return sortEntries(filterEntries(response.entries, filters), sort);
  }, [response, filters, sort]);

  const asOf = response ? timestampToDate(response.asOf) : undefined;
  const setFilter = <K extends keyof FleetFilters>(key: K, value: FleetFilters[K]) =>
    setFilters((prev) => ({ ...prev, [key]: value }));

  return (
    <section
      data-testid={selectors.pages.fleet}
      aria-labelledby="fleet-heading"
      className="flex min-h-0 flex-col gap-4"
    >
      <header className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h2 id="fleet-heading" className="text-2xl font-semibold text-app-foreground">
            {t(strings.fleet.title)}
          </h2>
          <p className="text-sm text-app-muted-foreground">{t(strings.fleet.description)}</p>
          {asOf && (
            <p data-testid={selectors.fleet.asOf} className="text-xs text-app-muted-foreground">
              {t(strings.fleet.asOf, { when: formatDate(asOf) })}
            </p>
          )}
        </div>
        <button
          type="button"
          data-testid={selectors.fleet.refresh}
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
          aria-busy={query.isFetching}
          className="inline-flex min-h-[2.25rem] items-center gap-2 rounded-panel border border-app-border bg-app-surface px-3 py-1.5 text-sm font-medium text-app-foreground hover:bg-app-surface-muted disabled:opacity-60"
        >
          <RefreshCw
            aria-hidden="true"
            className={query.isFetching ? "h-4 w-4 animate-spin" : "h-4 w-4"}
          />
          {t(strings.fleet.refresh)}
        </button>
      </header>

      {query.isLoading && (
        <p
          data-testid={selectors.fleet.loading}
          className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground"
        >
          {t(strings.fleet.loading)}
        </p>
      )}

      {query.isError && (
        <div
          data-testid={selectors.fleet.error}
          role="alert"
          className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger"
        >
          <p>{t(strings.fleet.error)}</p>
          <p className="mt-1 text-xs opacity-80">{errorMessage(query.error, t)}</p>
        </div>
      )}

      {response && response.scenarioCount === 0 && (
        <p
          data-testid={selectors.fleet.empty}
          className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground"
        >
          {t(strings.fleet.empty)}
        </p>
      )}

      {response && response.scenarioCount > 0 && (
        <div className="flex flex-col gap-4">
          <div data-testid={selectors.fleet.tiles} className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <StatTile
              testId={selectors.fleet.tileScanned}
              label={t(strings.fleet.tile.scanned)}
              value={response.scenarioCount}
            />
            <StatTile
              testId={selectors.fleet.tilePassing}
              label={t(strings.fleet.tile.passing)}
              value={response.passingCount}
            />
            <StatTile
              testId={selectors.fleet.tileStarter}
              label={t(strings.fleet.tile.starter)}
              value={response.starterRegistryCount}
            />
            <StatTile
              testId={selectors.fleet.tileLaggard}
              label={t(strings.fleet.tile.laggard)}
              value={response.templateLaggardCount}
            />
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <input
              type="search"
              data-testid={selectors.fleet.filterText}
              value={filters.text}
              onChange={(event) => setFilter("text", event.target.value)}
              placeholder={t(strings.fleet.filterPlaceholder)}
              className="min-h-[2.25rem] min-w-0 flex-1 rounded-panel border border-app-border bg-app-surface px-3 py-1.5 text-sm text-app-foreground placeholder:text-app-muted-foreground"
            />
            <FilterToggle
              testId={selectors.fleet.filterStarter}
              label={t(strings.fleet.filter.starter)}
              checked={filters.starter}
              onChange={(next) => setFilter("starter", next)}
            />
            <FilterToggle
              testId={selectors.fleet.filterLaggard}
              label={t(strings.fleet.filter.laggard)}
              checked={filters.laggard}
              onChange={(next) => setFilter("laggard", next)}
            />
            <FilterToggle
              testId={selectors.fleet.filterUnproven}
              label={t(strings.fleet.filter.unproven)}
              checked={filters.unproven}
              onChange={(next) => setFilter("unproven", next)}
            />
          </div>

          {rows.length === 0 ? (
            <p className="rounded-panel bg-app-surface-muted p-6 text-sm text-app-muted-foreground">
              {t(strings.fleet.noMatches)}
            </p>
          ) : (
            <div className="overflow-x-auto rounded-panel border border-app-border">
              <table
                data-testid={selectors.fleet.table}
                className="w-full min-w-[48rem] border-collapse text-sm"
              >
                <thead>
                  <tr className="border-b border-app-border bg-app-surface-muted text-start">
                    {COLUMNS.map((column) => {
                      const active = sort.column === column.col;
                      return (
                        <th
                          key={column.col}
                          scope="col"
                          aria-sort={
                            active ? (sort.direction === "asc" ? "ascending" : "descending") : "none"
                          }
                          className={column.numeric ? "px-3 py-2 text-end" : "px-3 py-2 text-start"}
                        >
                          <button
                            type="button"
                            onClick={() => setSort((prev) => toggleSort(prev, column.col))}
                            className={
                              column.numeric
                                ? "inline-flex w-full items-center justify-end gap-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground"
                                : "inline-flex w-full items-center gap-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground"
                            }
                          >
                            {t(column.label)}
                            {active &&
                              (sort.direction === "asc" ? (
                                <ArrowUp aria-hidden="true" className="h-3 w-3" />
                              ) : (
                                <ArrowDown aria-hidden="true" className="h-3 w-3" />
                              ))}
                          </button>
                        </th>
                      );
                    })}
                  </tr>
                </thead>
                <tbody className="divide-y divide-app-border">
                  {rows.map((entry) => (
                    <tr
                      key={entry.scenario}
                      data-testid={selectors.fleet.row({ scenario: entry.scenario })}
                      className="hover:bg-app-surface-muted"
                    >
                      <td className="px-3 py-2">
                        <button
                          type="button"
                          onClick={() =>
                            navigate("/matrix?scenario=" + encodeURIComponent(entry.scenario))
                          }
                          className="min-h-[2rem] text-start font-medium text-app-primary hover:underline"
                        >
                          {entry.scenario}
                        </button>
                      </td>
                      <td className="px-3 py-2">
                        <StatusChip tone={entry.passed ? "success" : "danger"}>
                          {entry.passed ? t(strings.fleet.passed) : t(strings.fleet.failed)}
                        </StatusChip>
                      </td>
                      <NumberCell value={entry.errorCount} emphasize={entry.errorCount > 0} />
                      <NumberCell value={entry.warningCount} />
                      <NumberCell value={entry.autofixableCount} />
                      <NumberCell value={entry.orphanedTargets} emphasize={entry.orphanedTargets > 0} />
                      <NumberCell value={entry.unprovenClaims} emphasize={entry.unprovenClaims > 0} />
                      <td className="px-3 py-2 text-app-muted-foreground">
                        {entry.templateVersion === "" ? (
                          <span aria-hidden="true">—</span>
                        ) : (
                          <span className="font-mono text-xs">{entry.templateVersion}</span>
                        )}
                        {entry.templateLaggard && (
                          <StatusChip tone="warning" className="ms-2">
                            {t(strings.fleet.filter.laggard)}
                          </StatusChip>
                        )}
                      </td>
                      <td className="px-3 py-2 text-end font-semibold text-app-foreground">
                        {entry.debtScore}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {response.errors.length > 0 && (
            <div
              data-testid={selectors.fleet.errors}
              className="rounded-panel border border-app-warning/40 bg-app-warning/10 p-4"
            >
              <h3 className="flex items-center gap-2 text-sm font-semibold text-app-warning">
                <AlertTriangle aria-hidden="true" className="h-4 w-4" />
                {t(strings.fleet.errorsHeading, { count: response.errors.length })}
              </h3>
              <ul className="mt-2 flex flex-col gap-1 text-xs text-app-warning">
                {response.errors.map((error) => (
                  <li key={error.scenario} className="flex flex-wrap gap-1">
                    <span className="font-mono">{error.scenario}</span>
                    <span className="opacity-80">{error.reason}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </section>
  );
}

function StatTile({ testId, label, value }: { testId: string; label: string; value: number }) {
  return (
    <div data-testid={testId} className="rounded-panel border border-app-border bg-app-surface p-3">
      <p className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p className="mt-1 text-xl font-semibold text-app-foreground">{value}</p>
    </div>
  );
}

function FilterToggle({
  testId,
  label,
  checked,
  onChange,
}: {
  testId: string;
  label: string;
  checked: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <label className="inline-flex min-h-[2.25rem] items-center gap-2 text-sm text-app-foreground">
      <input
        type="checkbox"
        data-testid={testId}
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="h-4 w-4 rounded border-app-border"
      />
      {label}
    </label>
  );
}

function NumberCell({ value, emphasize = false }: { value: number; emphasize?: boolean }) {
  return (
    <td
      className={
        emphasize
          ? "px-3 py-2 text-end font-medium text-app-danger"
          : "px-3 py-2 text-end text-app-foreground"
      }
    >
      {value}
    </td>
  );
}
