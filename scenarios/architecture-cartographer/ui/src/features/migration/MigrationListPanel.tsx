import * as React from "react";
import { Link, useNavigate } from "react-router-dom";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { encodeScenarioPath } from "../../hooks/useScenarioPath";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { useListMigrations } from "./controllers/useMigrationController";
import { CreateMigrationForm } from "./CreateMigrationForm";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import {
  MigrationLifecycle,
  type Migration,
} from "@vrooli/proto-types/architecture-cartographer/v1/migration/migration_pb";

export interface MigrationListPanelProps {
  scenario: string;
  /** When provided, the matching row renders as selected. */
  selectedId?: string;
}

function lifecycleLabelKey(status: MigrationLifecycle) {
  return status === MigrationLifecycle.CLOSED
    ? strings.migration.lifecycle.closed
    : strings.migration.lifecycle.open;
}

export function MigrationListPanel({ scenario, selectedId }: MigrationListPanelProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const list = useListMigrations({ scenario });
  const [creating, setCreating] = React.useState(false);

  const columns: ReadonlyArray<DataTableColumn<Migration>> = [
    {
      key: "id",
      header: t(strings.pages.migration.columns.id),
      cell: (row) => {
        const isSelected = selectedId !== undefined && row.id === selectedId;
        return (
          <Link
            to={`/targets/${encodeScenarioPath(scenario)}/migration/${encodeURIComponent(row.id)}`}
            data-testid={selectors.features.migration.list.openButton({ id: row.id })}
            className={`block font-mono text-xs ${isSelected ? "text-app-primary" : "text-app-foreground hover:underline"}`}
          >
            {row.id.slice(0, 8)}
          </Link>
        );
      },
    },
    {
      key: "name",
      header: t(strings.pages.migration.columns.name),
      cell: (row) => <span className="text-sm">{row.name || t(strings.pages.migration.unnamed)}</span>,
    },
    {
      key: "status",
      header: t(strings.pages.migration.columns.status),
      cell: (row) => (
        <Badge variant={row.status === MigrationLifecycle.CLOSED ? "default" : "info"}>
          {t(lifecycleLabelKey(row.status))}
        </Badge>
      ),
    },
    {
      key: "created",
      header: t(strings.pages.migration.columns.created),
      cell: (row) => (
        <span className="text-xs text-app-muted-foreground">
          {row.createdAt ? formatDate(timestampDate(row.createdAt), { dateStyle: "medium", timeStyle: "short" }) : "—"}
        </span>
      ),
    },
  ];

  if (list.isPending) {
    return (
      <div data-testid={selectors.features.migration.list.loading}>
        <LoadingState label={t(strings.pages.migration.loading)} />
      </div>
    );
  }

  if (list.isError) {
    return (
      <div data-testid={selectors.features.migration.list.error}>
        <ErrorState
          title={t(strings.pages.migration.errorTitle)}
          message={list.error instanceof Error ? list.error.message : String(list.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void list.refetch();
          }}
        />
      </div>
    );
  }

  const migrations = list.data.migrations;

  return (
    <section
      data-testid={selectors.features.migration.list.root}
      aria-label={t(strings.pages.migration.title)}
      className="flex flex-col gap-3"
    >
      <header className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="default"
          size="sm"
          data-testid={selectors.features.migration.list.newButton}
          onClick={() => setCreating((v) => !v)}
        >
          {t(strings.pages.migration.newButton)}
        </Button>
      </header>

      {creating ? (
        <CreateMigrationForm
          scenario={scenario}
          onCreated={(id) => {
            setCreating(false);
            navigate(`/targets/${encodeScenarioPath(scenario)}/migration/${encodeURIComponent(id)}`);
          }}
          onCancel={() => setCreating(false)}
        />
      ) : null}

      {migrations.length === 0 ? (
        <div data-testid={selectors.features.migration.list.empty}>
          <EmptyState
            title={t(strings.pages.migration.listEmptyTitle)}
            description={t(strings.pages.migration.listEmptyDescription)}
          />
        </div>
      ) : (
        <DataTable<Migration>
          rows={migrations}
          getRowId={(row) => row.id}
          columns={columns}
          emptyMessage={t(strings.pages.migration.listEmptyTitle)}
          caption={t(strings.pages.migration.title)}
        />
      )}
    </section>
  );
}
