import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { DataTable, type DataTableColumn } from "../../components/DataTable";
import { EmptyState } from "../../components/EmptyState";
import type { DomainSpec } from "@vrooli/proto-types/architecture-cartographer/v1/manifest/manifest_pb";

export interface DomainInventoryProps {
  domains: readonly DomainSpec[];
}

export function DomainInventory({ domains }: DomainInventoryProps) {
  const { t } = useTranslation();

  if (domains.length === 0) {
    return (
      <div data-testid={selectors.features.manifest.inventory.empty}>
        <EmptyState title={t(strings.pages.targetManifest.noDomains)} />
      </div>
    );
  }

  const columns: ReadonlyArray<DataTableColumn<DomainSpec>> = [
    {
      key: "name",
      header: t(strings.pages.targetManifest.columns.name),
      cell: (row) => <span className="font-semibold">{row.name}</span>,
    },
    {
      key: "paths",
      header: t(strings.pages.targetManifest.columns.paths),
      cell: (row) => (
        <span className="font-mono text-xs">
          {row.paths.length === 0 ? "—" : row.paths.join(", ")}
        </span>
      ),
    },
    {
      key: "allowedDeps",
      header: t(strings.pages.targetManifest.columns.allowedDeps),
      cell: (row) => (
        <span className="text-xs text-app-muted-foreground">
          {row.allowedDependencies.length === 0 ? "—" : row.allowedDependencies.join(", ")}
        </span>
      ),
    },
  ];

  return (
    <div data-testid={selectors.features.manifest.inventory.root}>
      <DataTable
        rows={domains}
        getRowId={(d) => d.name}
        columns={columns}
        emptyMessage={t(strings.pages.targetManifest.noDomains)}
      />
    </div>
  );
}
