import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { usePlacements } from "./controllers/useAnalyticsController";
import type { Placement } from "@vrooli/proto-types/architecture-cartographer/v1/analytics/analytics_pb";

export interface PlacementsTableProps {
  scenario: string;
}

export function PlacementsTable({ scenario }: PlacementsTableProps) {
  const { t } = useTranslation();
  const placements = usePlacements(scenario);

  if (placements.isPending) {
    return (
      <div data-testid={selectors.features.analytics.placements.loading}>
        <LoadingState label={t(strings.pages.targetAnalytics.placementsLoading)} />
      </div>
    );
  }
  if (placements.isError) {
    return (
      <div data-testid={selectors.features.analytics.placements.error}>
        <ErrorState
          title={t(strings.pages.targetAnalytics.placementsError)}
          message={placements.error instanceof Error ? placements.error.message : String(placements.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void placements.refetch();
          }}
        />
      </div>
    );
  }
  const rows = placements.data.placements;
  if (rows.length === 0) {
    return (
      <div data-testid={selectors.features.analytics.placements.empty}>
        <EmptyState title={t(strings.pages.targetAnalytics.placementsEmpty)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<Placement>> = [
    {
      key: "chunkId",
      header: t(strings.pages.targetAnalytics.columns.chunkId),
      cell: (row) => <span className="font-mono text-xs">{row.chunkId}</span>,
    },
    {
      key: "chunkPath",
      header: t(strings.pages.targetAnalytics.columns.chunkPath),
      cell: (row) => <span className="font-mono text-xs">{row.chunkPath}</span>,
    },
    {
      key: "outcome",
      header: t(strings.pages.targetAnalytics.columns.outcome),
      cell: (row) => <span className="text-sm">{row.outcome || "—"}</span>,
    },
  ];

  return (
    <div data-testid={selectors.features.analytics.placements.root}>
      <DataTable
        rows={rows}
        getRowId={(p) => p.id}
        columns={columns}
        emptyMessage={t(strings.pages.targetAnalytics.placementsEmpty)}
      />
    </div>
  );
}
