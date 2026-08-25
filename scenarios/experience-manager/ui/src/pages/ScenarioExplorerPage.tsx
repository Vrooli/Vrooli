import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { Wand2 } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import {
  fetchScenarioSpec,
  type ExperienceClaimSpec,
  type ScenarioSpecPage,
} from "../api/experience";
import { PageFrame } from "../components/PageFrame";
import { Button } from "../components/ui/button";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1.2.0";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { claimEvidencePath, machineClaims, stateDepth } from "./experiencePageUtils";

export function ScenarioExplorerPage() {
  const { t } = useTranslation();
  const params = useParams();
  const scenario = params.scenario ?? "experience-manager";
  const {
    data: pages,
    dataUpdatedAt,
    isError,
    isFetching,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["experience-scenario-spec", scenario],
    queryFn: () => fetchScenarioSpec(scenario),
    staleTime: 60_000,
  });
  const rows = useMemo(() => pages ?? [], [pages]);
  const claims = machineClaims(rows);
  const stale = Boolean(dataUpdatedAt && Date.now() - dataUpdatedAt > 60_000);
  const columns = useMemo<Array<DataTableColumn<ScenarioSpecPage>>>(
    () => [
      {
        id: "page",
        header: t(strings.experience.common.page),
        accessor: (row) => row.document.title || row.spec.page.title,
        sortValue: (row) => row.document.title || row.spec.page.title,
        searchValue: (row) => `${row.document.title} ${row.spec.page.title} ${row.document.id}`,
      },
      {
        id: "default",
        header: t(strings.experience.explorer.defaultState),
        accessor: (row) => stateDepth(row, "default"),
        sortValue: (row) => stateDepth(row, "default"),
      },
      {
        id: "empty",
        header: t(strings.experience.explorer.emptyState),
        accessor: (row) => stateDepth(row, "empty"),
        sortValue: (row) => stateDepth(row, "empty"),
      },
      {
        id: "stale",
        header: t(strings.experience.explorer.staleState),
        accessor: (row) => stateDepth(row, "stale"),
        sortValue: (row) => stateDepth(row, "stale"),
      },
      {
        id: "claims",
        header: t(strings.experience.common.claims),
        accessor: (row) => row.spec.claims?.length ?? 0,
        sortValue: (row) => row.spec.claims?.length ?? 0,
      },
    ],
    [t],
  );
  const tableRows = isLoading ? [] : rows;

  return (
    <PageFrame
      testId={selectors.pages.explorer}
      title={t(strings.experience.explorer.title)}
      description={t(strings.experience.explorer.description)}
      experienceSurface="scenario-spec"
      experienceState={isLoading ? "loading" : isError ? "error" : rows.length === 0 ? "empty" : stale ? "partial" : "ready"}
    >
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <DataTable
          rows={tableRows}
          columns={columns}
          getRowKey={(row) => row.document.id}
          caption={t(strings.experience.explorer.gridLabel)}
          searchLabel={t(strings.experience.explorer.gridLabel)}
          searchPlaceholder={t(strings.experience.explorer.gridLabel)}
          emptyMessage={isLoading ? t(strings.experience.explorer.loadingSpec) : t(strings.experience.explorer.emptySpec)}
          tableTestId={selectors.experience.explorer.depthGrid}
        />
        <aside
          data-testid={selectors.experience.explorer.gapPanel}
          role={isError ? "alert" : "region"}
          aria-label={t(strings.experience.explorer.gapsLabel)}
          className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.experience.explorer.gapsLabel)}</h3>
          {isError ? (
            <p className="mt-2 text-sm text-app-muted-foreground">
              {t(strings.experience.explorer.loadError)}
            </p>
          ) : rows.length === 0 && !isLoading ? (
            <p className="mt-2 text-sm text-app-muted-foreground">
              {t(strings.experience.explorer.emptyGap)}
            </p>
          ) : (
            <p className="mt-2 text-sm text-app-muted-foreground">
              {stale
                ? t(strings.experience.explorer.staleData)
                : t(strings.experience.explorer.summary, {
                    claims: claims.length,
                    pages: rows.length,
                  })}
            </p>
          )}
          <Button
            data-testid={selectors.experience.explorer.studioAction}
            type="button"
            className="mt-4"
            onClick={() => void refetch()}
          >
            <Wand2 className="mr-2 size-4" aria-hidden="true" />
            {isFetching ? t(strings.experience.explorer.refreshing) : t(strings.experience.explorer.openStudio)}
          </Button>
        </aside>
      </div>
      <ul
        data-testid={selectors.experience.explorer.claimList}
        aria-label={t(strings.experience.explorer.claimsLabel)}
        className="grid gap-3 md:grid-cols-2"
      >
        {isLoading ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.explorer.loadingClaims)}
          </li>
        ) : claims.length === 0 ? (
          <li className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground">
            {t(strings.experience.explorer.emptyClaims)}
          </li>
        ) : (
          claims.map(({ page, claim }: { page: ScenarioSpecPage; claim: ExperienceClaimSpec }) => (
            <li key={`${page.document.id}:${claim.id}`} className="rounded-panel border border-app-border bg-app-surface p-4">
              <span
                data-testid={selectors.experience.explorer.tierLabel}
                role="note"
                aria-label={`${t(strings.experience.common.tier)} ${claim.tier}`}
                className="text-xs font-semibold uppercase text-app-primary"
              >
                {claim.tier}
              </span>
              <p className="mt-2 font-medium">{claim.id}</p>
              <p className="mt-1 text-sm text-app-muted-foreground">{claim.type}</p>
              <Link
                data-testid={selectors.experience.explorer.evidenceLink}
                to={claimEvidencePath(scenario, page.document.id)}
                aria-label={`${t(strings.experience.common.viewEvidence)} ${page.document.title} ${claim.id}`}
                className="mt-3 inline-flex text-sm text-app-primary underline-offset-4 hover:underline"
              >
                {t(strings.experience.common.viewEvidence)}
              </Link>
            </li>
          ))
        )}
      </ul>
    </PageFrame>
  );
}
