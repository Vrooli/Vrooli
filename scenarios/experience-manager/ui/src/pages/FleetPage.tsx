import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { RefreshCw } from "lucide-react";
import { Link } from "react-router-dom";

import type { FleetScenario } from "@vrooli/proto-types/experience-manager/v1/contract/contract_pb";
import { fetchFleet } from "../api/experience";
import { MetricCard } from "../components/MetricCard";
import { PageFrame } from "../components/PageFrame";
import { Button } from "../components/ui/button";
import { DataTable, type DataTableColumn, type DataTableFilter } from "../components/ui/data-table";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function FleetPage() {
  const { t } = useTranslation();
  const {
    data,
    dataUpdatedAt,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-fleet"],
    queryFn: fetchFleet,
    staleTime: 60_000,
  });
  const rows = data?.scenarios ?? [];
  const covered = data?.withExperienceCount ?? 0;
  const total = data?.scenarioCount ?? 0;
  const coveragePercent = total > 0 ? Math.round((covered / total) * 100) : 0;
  const stale = Boolean(data && dataUpdatedAt && Date.now() - dataUpdatedAt > 60_000);
  const columns = useMemo<Array<DataTableColumn<FleetScenario>>>(
    () => [
      {
        id: "scenario",
        header: t(strings.experience.common.scenario),
        accessor: (scenario) => (
          <Link
            data-testid={selectors.experience.fleet.scenarioLink}
            className="inline-flex min-h-11 items-center font-medium text-app-primary underline-offset-4 hover:underline"
            to={`/scenarios/${scenario.scenario}`}
          >
            {scenario.scenario}
          </Link>
        ),
        sortValue: (scenario) => scenario.scenario,
        searchValue: (scenario) => scenario.scenario,
      },
      {
        id: "depth",
        header: t(strings.experience.common.depth),
        accessor: (scenario) => scenario.maxDepth,
        sortValue: (scenario) => scenario.maxDepthValue,
        searchValue: (scenario) => scenario.maxDepth,
      },
      {
        id: "debt",
        header: t(strings.experience.common.debt),
        accessor: (scenario) => scenario.debtScore,
        sortValue: (scenario) => scenario.debtScore,
        searchValue: (scenario) => String(scenario.debtScore),
      },
      {
        id: "status",
        header: t(strings.experience.common.status),
        accessor: (scenario) => scenario.status,
        sortValue: (scenario) => scenario.status,
        searchValue: (scenario) => scenario.status,
      },
    ],
    [t],
  );
  const filters = useMemo<Array<DataTableFilter<FleetScenario>>>(
    () => [
      {
        id: "all",
        label: t(strings.experience.fleet.filterAll),
        predicate: () => true,
      },
      {
        id: "findings",
        label: t(strings.experience.fleet.filterFindings),
        predicate: (scenario) => scenario.findingCount > 0 || scenario.status.toLowerCase().includes("finding"),
      },
      {
        id: "clean",
        label: t(strings.experience.fleet.filterClean),
        predicate: (scenario) => scenario.findingCount === 0 && !scenario.status.toLowerCase().includes("finding"),
      },
    ],
    [t],
  );
  const tableRows = isLoading ? [] : rows;

  return (
    <PageFrame
      testId={selectors.pages.fleet}
      title={t(strings.experience.fleet.title)}
      description={t(strings.experience.fleet.description)}
      experienceSurface="fleet-results"
      experienceState={isLoading ? "loading" : isError ? "error" : rows.length === 0 ? "empty" : stale ? "partial" : "ready"}
    >
      <div
        data-testid={selectors.experience.fleet.depthSummary}
        role="region"
        aria-label={t(strings.experience.fleet.depthLabel)}
        className="grid gap-3 md:grid-cols-[1fr_1fr_auto]"
      >
        <MetricCard label={t(strings.experience.fleet.specCoverage)} value={`${covered} / ${total}`} />
        <div
          data-testid={selectors.experience.fleet.coverageMeter}
          role="meter"
          aria-label={t(strings.experience.fleet.specCoverage)}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={coveragePercent}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.experience.fleet.depthDistribution)}
          </p>
          <div className="mt-3 h-3 rounded-full bg-app-surface-muted">
            <div className="h-3 rounded-full bg-app-primary" style={{ width: `${coveragePercent}%` }} />
          </div>
          <p className="mt-2 text-sm text-app-muted-foreground">
            {isLoading
              ? t(strings.experience.fleet.loadingData)
              : stale
                ? t(strings.experience.fleet.staleData)
                : t(strings.experience.fleet.pagesTracked, { count: data?.totalPages ?? 0 })}
          </p>
        </div>
        <Button
          data-testid={selectors.experience.fleet.refreshAction}
          type="button"
          onClick={() => void refetch()}
        >
          <RefreshCw className="mr-2 size-4" aria-hidden="true" />
          {isFetching ? t(strings.experience.explorer.refreshing) : t(strings.experience.fleet.refresh)}
        </Button>
      </div>
      {isError ? (
        <div role="alert" className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
          {t(strings.experience.fleet.loadError)}
        </div>
      ) : null}
      <DataTable
        rows={tableRows}
        columns={columns}
        getRowKey={(scenario) => scenario.scenario}
        caption={t(strings.experience.fleet.tableLabel)}
        searchLabel={t(strings.experience.fleet.tableLabel)}
        searchPlaceholder={t(strings.experience.fleet.tableLabel)}
        filters={filters}
        filterGroupLabel={t(strings.experience.fleet.filterStatus)}
        sortLabel={(header) => t(strings.experience.fleet.sortBy, { column: header })}
        emptyMessage={isLoading ? t(strings.experience.fleet.loadingData) : t(strings.experience.fleet.emptyFleet)}
        tableTestId={selectors.experience.fleet.debtTable}
      />
    </PageFrame>
  );
}
