import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { AlertTriangle, RefreshCw } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState, ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { storageClient, type FleetScenarioEntry, type ScanFleetResponse } from "../../api/storage";
import { EngineChips, IsolationBadge } from "../storage/format";

type FleetView = "all" | "isolation" | "no-backup" | "engines" | "stages";
const VIEWS: readonly FleetView[] = ["all", "isolation", "no-backup", "engines", "stages"];
type DataSource = "scan" | "snapshot";
type FleetScenarioEntryWithDataDir = FleetScenarioEntry & {
  dataDirBytes?: bigint | number;
  dataDirOverBudget?: boolean;
};

const VIEW_LABEL: Record<FleetView, (typeof strings.fleet.view)[keyof typeof strings.fleet.view]> = {
  all: strings.fleet.view.all,
  isolation: strings.fleet.view.isolation,
  "no-backup": strings.fleet.view.noBackup,
  engines: strings.fleet.view.engines,
  stages: strings.fleet.view.stages,
};

/** Offenders-first ordering: isolation-unready scenarios sort to the top. */
const byIsolationFirst = (a: FleetScenarioEntry, b: FleetScenarioEntry) => {
  if (a.isolationReady !== b.isolationReady) return a.isolationReady ? 1 : -1;
  return a.scenario.localeCompare(b.scenario);
};

/**
 * Fleet inventory workflow. Backed by FleetService — a live `ScanFleet` or the
 * persisted `GetInventory` snapshot — rendering the cross-scenario views the CLI
 * exposes (`all · isolation · no-backup · engines · stages`). The isolation view
 * is the safety scorecard: a checklist of real-data risks.
 */
export function FleetView() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const initialView = searchParams.get("view") as FleetView | null;
  const [view, setView] = useState<FleetView>(
    initialView && VIEWS.includes(initialView) ? initialView : "all",
  );
  const [source, setSource] = useState<DataSource>("snapshot");

  const query = useQuery<ScanFleetResponse>({
    queryKey: ["fleet", source],
    queryFn: () => (source === "scan" ? storageClient.scanFleet({}) : storageClient.getInventory()),
  });

  const data = query.data;
  const entries = [...(data?.entries ?? [])].sort(byIsolationFirst);
  const scanErrors = data?.errors ?? [];

  const changeView = (next: FleetView) => {
    setView(next);
    const params = new URLSearchParams(searchParams);
    params.set("view", next);
    setSearchParams(params, { replace: true });
  };

  return (
    <section
      data-testid={selectors.pages.fleet}
      aria-labelledby="fleet-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-wrap items-start gap-3">
        <div className="flex flex-col gap-1">
          <h2 id="fleet-heading" className="text-2xl font-semibold">
            {t(strings.fleet.title)}
          </h2>
          <p className="text-app-muted-foreground">{t(strings.fleet.description)}</p>
        </div>
        <div className="ms-auto flex items-center gap-2">
          <div
            data-testid={selectors.fleet.sourceSwitcher}
            role="radiogroup"
            aria-label={t(strings.fleet.view.label)}
            className="flex gap-1"
          >
            {(["snapshot", "scan"] as const).map((s) => (
              <button
                key={s}
                type="button"
                role="radio"
                aria-checked={source === s}
                data-testid={selectors.fleet.sourceTab({ source: s })}
                onClick={() => setSource(s)}
                className={
                  source === s
                    ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                    : "rounded-control border border-app-border px-3 py-1 text-sm hover:bg-app-surface-muted"
                }
              >
                {s === "scan" ? t(strings.fleet.source.scan) : t(strings.fleet.source.snapshot)}
              </button>
            ))}
          </div>
          <Button
            data-testid={selectors.fleet.scanButton}
            variant="outline"
            size="sm"
            onClick={() => void query.refetch()}
            disabled={query.isFetching}
            aria-label={source === "scan" ? t(strings.common.scanNow) : t(strings.common.useLastSnapshot)}
          >
            <RefreshCw
              aria-hidden="true"
              className={["h-4 w-4", query.isFetching ? "animate-spin" : ""].join(" ")}
            />
          </Button>
        </div>
      </header>

      <div
        data-testid={selectors.fleet.viewSwitcher}
        role="radiogroup"
        aria-label={t(strings.fleet.view.label)}
        className="flex flex-wrap gap-1"
      >
        {VIEWS.map((v) => (
          <button
            key={v}
            type="button"
            role="radio"
            aria-checked={view === v}
            data-testid={selectors.fleet.viewTab({ view: v })}
            onClick={() => changeView(v)}
            className={
              view === v
                ? "rounded-control bg-app-primary px-3 py-1 text-sm font-medium text-app-primary-foreground"
                : "rounded-control border border-app-border px-3 py-1 text-sm hover:bg-app-surface-muted"
            }
          >
            {t(VIEW_LABEL[v])}
          </button>
        ))}
      </div>

      {query.isLoading && (
        <LoadingState
          testId={selectors.fleet.loading}
          title={source === "scan" ? t(strings.common.scanning) : t(strings.fleet.loadingTitle)}
          skeleton={
            <div className="flex flex-col gap-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          }
        />
      )}

      {query.error && (
        <ErrorState
          testId={selectors.fleet.error}
          title={t(strings.fleet.errorTitle)}
          message={errorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
        />
      )}

      {data && !query.error && (
        <FleetBody view={view} entries={entries} data={data} />
      )}

      {data && scanErrors.length > 0 && (
        <section
          data-testid={selectors.fleet.errors}
          aria-label={t(strings.fleet.errors.title)}
          className="rounded-panel border border-app-danger/40 bg-app-danger/5 p-4"
        >
          <h3 className="flex items-center gap-2 text-sm font-medium text-app-danger">
            <AlertTriangle aria-hidden="true" className="h-4 w-4" />
            {t(strings.fleet.errors.title)}
          </h3>
          <ul className="mt-3 flex flex-col gap-1 text-sm">
            {scanErrors.map((e) => (
              <li key={e.scenario} className="flex flex-wrap gap-2">
                <span className="font-medium text-app-foreground">{e.scenario}</span>
                <span className="text-app-muted-foreground">{e.reason}</span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </section>
  );
}

function FleetBody({
  view,
  entries,
  data,
}: {
  view: FleetView;
  entries: FleetScenarioEntry[];
  data: ScanFleetResponse;
}) {
  const { t } = useTranslation();

  if (view === "engines") {
    return (
      <DistributionList
        items={data.engineDistribution.map((d) => ({ label: d.engine, value: d.scenarioCount }))}
        emptyMessage={t(strings.fleet.empty.engines)}
        countLabel={t(strings.fleet.col.count)}
      />
    );
  }
  if (view === "stages") {
    return (
      <DistributionList
        items={data.stageDistribution.map((d) => ({ label: d.stage, value: d.scenarioCount }))}
        emptyMessage={t(strings.fleet.empty.stages)}
        countLabel={t(strings.fleet.col.count)}
      />
    );
  }

  let rows = entries;
  let emptyMessage = t(strings.fleet.empty.all);
  if (view === "isolation") {
    rows = entries.filter((e) => !e.isolationReady);
    emptyMessage = t(strings.fleet.empty.isolation);
  } else if (view === "no-backup") {
    rows = entries.filter((e) => !e.hasBackupTarget);
    emptyMessage = t(strings.fleet.empty.noBackup);
  }

  if (rows.length === 0) {
    return <EmptyState testId={selectors.fleet.empty} message={emptyMessage} />;
  }

  return view === "isolation" ? (
    <IsolationScorecard rows={rows} />
  ) : (
    <FleetTable rows={rows} showBackup={view === "no-backup"} />
  );
}

function FleetTable({ rows, showBackup }: { rows: FleetScenarioEntry[]; showBackup: boolean }) {
  const { t } = useTranslation();
  return (
    <div data-testid={selectors.fleet.table} className="overflow-x-auto rounded-panel border border-app-border bg-app-surface">
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
            <th scope="col" className="px-3 py-2 text-start font-medium">{t(strings.fleet.col.scenario)}</th>
            <th scope="col" className="px-3 py-2 text-start font-medium">{t(strings.fleet.col.engines)}</th>
            <th scope="col" className="px-3 py-2 text-start font-medium">{t(strings.fleet.col.stage)}</th>
            <th scope="col" className="px-3 py-2 text-start font-medium">{t(strings.fleet.col.isolation)}</th>
            <th scope="col" className="px-3 py-2 text-end font-medium">{t(strings.fleet.col.findings)}</th>
            <th scope="col" className="px-3 py-2 text-end font-medium">{t(strings.fleet.col.data)}</th>
            {showBackup && (
              <th scope="col" className="px-3 py-2 text-start font-medium">{t(strings.fleet.col.backup)}</th>
            )}
          </tr>
        </thead>
        <tbody>
          {rows.map((e) => (
            <tr
              key={e.scenario}
              data-testid={selectors.fleet.row({ scenario: e.scenario })}
              className="border-t border-app-border"
            >
              <td className="px-3 py-2">
                <Link
                  to={`/validate?scenario=${encodeURIComponent(e.scenario)}`}
                  className="font-medium text-app-primary underline"
                >
                  {e.scenario}
                </Link>
              </td>
              <td className="px-3 py-2"><EngineChips engines={e.engines} /></td>
              <td className="px-3 py-2 text-app-muted-foreground">{e.storageStage || "—"}</td>
              <td className="px-3 py-2"><IsolationBadge ready={e.isolationReady} /></td>
              <td className="px-3 py-2 text-end tabular-nums">{e.findingCount}</td>
              <td className={["px-3 py-2 text-end tabular-nums", dataDirOverBudget(e) ? "text-app-danger" : "text-app-muted-foreground"].join(" ")}>
                {formatBytes(dataDirBytes(e))}
              </td>
              {showBackup && (
                <td className="px-3 py-2 text-app-muted-foreground">
                  {e.hasBackupTarget ? t(strings.fleet.backup.present) : t(strings.fleet.backup.missing)}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function dataDirBytes(entry: FleetScenarioEntry): number {
  const value = (entry as FleetScenarioEntryWithDataDir).dataDirBytes;
  if (typeof value === "bigint") return Number(value);
  return typeof value === "number" ? value : 0;
}

function dataDirOverBudget(entry: FleetScenarioEntry): boolean {
  return (entry as FleetScenarioEntryWithDataDir).dataDirOverBudget === true;
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function IsolationScorecard({ rows }: { rows: FleetScenarioEntry[] }) {
  const { t } = useTranslation();
  return (
    <ul data-testid={selectors.fleet.list} className="flex flex-col gap-3">
      {rows.map((e) => (
        <li
          key={e.scenario}
          data-testid={selectors.fleet.row({ scenario: e.scenario })}
          className="rounded-panel border border-app-danger/40 bg-app-danger/5 p-4"
        >
          <div className="flex flex-wrap items-center gap-3">
            <Link
              to={`/validate?scenario=${encodeURIComponent(e.scenario)}`}
              className="font-medium text-app-primary underline"
            >
              {e.scenario}
            </Link>
            <IsolationBadge ready={false} />
          </div>
          <p className="mt-2 text-sm text-app-foreground">
            {e.isolationReason || t(strings.isolation.unreadyLabel)}
          </p>
          <Link
            to={`/validate?scenario=${encodeURIComponent(e.scenario)}`}
            className="mt-2 inline-block text-sm text-app-primary underline"
          >
            {t(strings.fleet.howToFix)}
          </Link>
        </li>
      ))}
    </ul>
  );
}

function DistributionList({
  items,
  emptyMessage,
  countLabel,
}: {
  items: { label: string; value: number }[];
  emptyMessage: string;
  countLabel: string;
}) {
  if (items.length === 0) {
    return <EmptyState testId={selectors.fleet.empty} message={emptyMessage} />;
  }
  const max = Math.max(...items.map((i) => i.value), 1);
  return (
    <ul data-testid={selectors.fleet.list} className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4">
      {items.map((i) => (
        <li key={i.label} className="flex items-center gap-3">
          <span className="w-28 shrink-0 truncate text-sm text-app-foreground">{i.label}</span>
          <div className="h-2 flex-1 overflow-hidden rounded-control bg-app-surface-muted">
            <div className="h-full bg-app-primary" style={{ width: `${(i.value / max) * 100}%` }} />
          </div>
          <span className="w-10 shrink-0 text-end text-sm tabular-nums text-app-muted-foreground" aria-label={countLabel}>
            {i.value}
          </span>
        </li>
      ))}
    </ul>
  );
}
