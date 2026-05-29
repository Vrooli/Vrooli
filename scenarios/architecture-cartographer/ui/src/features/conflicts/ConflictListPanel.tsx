import { Link } from "react-router-dom";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { SeverityBadge } from "../../components/SeverityBadge";
import { Button } from "../../components/ui/button";
import {
  useDetectConflicts,
  useListConflicts,
  useValidateConflicts,
} from "./controllers/useConflictsController";
import { severityToLevel } from "./severity";
import type { Conflict } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

const SEVERITY_LABEL_KEY = {
  info: strings.shared.severity.info,
  low: strings.shared.severity.low,
  medium: strings.shared.severity.medium,
  high: strings.shared.severity.high,
  critical: strings.shared.severity.critical,
} as const;

export interface ConflictListPanelProps {
  scenario: string;
  /** When provided, the row's "open" link points here; otherwise to the detail page. */
  selectedId?: string;
}

export function ConflictListPanel({ scenario, selectedId }: ConflictListPanelProps) {
  const { t } = useTranslation();
  const list = useListConflicts({ scenario });
  const detect = useDetectConflicts(scenario);
  const validate = useValidateConflicts(scenario);

  const columns: ReadonlyArray<DataTableColumn<Conflict>> = [
    {
      key: "id",
      header: t(strings.pages.conflicts.columns.id),
      cell: (row) => {
        const isSelected = selectedId !== undefined && row.id === selectedId;
        return (
          <Link
            to={`/targets/${encodeScenarioPath(scenario)}/conflicts/${encodeURIComponent(row.id)}`}
            data-testid={selectors.features.conflicts.list.openButton({ id: row.id })}
            className={`block font-mono text-xs ${isSelected ? "text-app-primary" : "text-app-foreground hover:underline"}`}
          >
            {row.id}
          </Link>
        );
      },
    },
    {
      key: "type",
      header: t(strings.pages.conflicts.columns.type),
      cell: (row) => <span className="text-sm">{row.subtype ? `${row.type} / ${row.subtype}` : row.type}</span>,
    },
    {
      key: "severity",
      header: t(strings.pages.conflicts.columns.severity),
      cell: (row) => {
        const level = severityToLevel(row.severity);
        return <SeverityBadge level={level} label={t(SEVERITY_LABEL_KEY[level])} />;
      },
    },
    {
      key: "domains",
      header: t(strings.pages.conflicts.columns.domains),
      cell: (row) => (
        <span className="text-sm text-app-muted-foreground">
          {row.domains.length === 0 ? "—" : row.domains.join(", ")}
        </span>
      ),
    },
  ];

  if (list.isPending) {
    return (
      <div data-testid={selectors.features.conflicts.list.loading}>
        <LoadingState label={t(strings.pages.conflicts.loading)} />
      </div>
    );
  }

  if (list.isError) {
    return (
      <div data-testid={selectors.features.conflicts.list.error}>
        <ErrorState
          title={t(strings.pages.conflicts.errorTitle)}
          message={list.error instanceof Error ? list.error.message : String(list.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void list.refetch();
          }}
        />
      </div>
    );
  }

  const conflicts = list.data.conflicts;

  return (
    <section
      data-testid={selectors.features.conflicts.list.root}
      aria-label={t(strings.pages.conflicts.title)}
      className="flex flex-col gap-3"
    >
      <header className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="default"
          size="sm"
          data-testid={selectors.features.conflicts.list.detectButton}
          onClick={() => detect.mutate()}
          disabled={detect.isPending}
        >
          {detect.isPending
            ? t(strings.pages.conflicts.detecting)
            : t(strings.pages.conflicts.detectButton)}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid={selectors.features.conflicts.list.validateButton}
          onClick={() => validate.mutate()}
          disabled={validate.isPending}
        >
          {validate.isPending
            ? t(strings.pages.conflicts.validating)
            : t(strings.pages.conflicts.validateButton)}
        </Button>
      </header>
      {conflicts.length === 0 ? (
        <div data-testid={selectors.features.conflicts.list.empty}>
          <EmptyState
            title={t(strings.pages.conflicts.listEmptyTitle)}
            description={t(strings.pages.conflicts.listEmptyDescription)}
          />
        </div>
      ) : (
        <DataTable<Conflict>
          rows={conflicts}
          getRowId={(row) => row.id}
          columns={columns}
          emptyMessage={t(strings.pages.conflicts.listEmptyTitle)}
          caption={t(strings.pages.conflicts.title)}
        />
      )}
    </section>
  );
}
