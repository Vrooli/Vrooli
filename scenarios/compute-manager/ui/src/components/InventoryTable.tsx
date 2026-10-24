import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1.4.2";
import type { Instance } from "@vrooli/proto-types/compute-manager/v1/instance/instance_pb";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export type InventoryInstance = Pick<Instance, "id" | "state" | "region" | "size" | "remainingSeconds">;

type InventoryTableProps = {
  instances: InventoryInstance[];
  loading: boolean;
  error?: string;
};

export function InventoryTable({ instances, loading, error }: InventoryTableProps) {
  const { t } = useTranslation();
  const columns: DataTableColumn<InventoryInstance>[] = [
    { id: "state", header: t(strings.pages.dashboard.state), accessor: (row) => row.state, sortValue: (row) => row.state },
    { id: "region", header: t(strings.pages.dashboard.region), accessor: (row) => row.region },
    { id: "size", header: t(strings.pages.dashboard.size), accessor: (row) => row.size },
    { id: "remaining", header: t(strings.pages.dashboard.remaining), accessor: (row) => `${Number(row.remainingSeconds)}s`, sortValue: (row) => Number(row.remainingSeconds) },
  ];

  return (
    <div data-testid={selectors.pages.dashboardPlaceholder} className="rounded-card border border-border bg-surface p-space-md">
      <p className="sr-only">{t(strings.pages.dashboard.inventoryDescription)}</p>
      <h2 className="text-lg font-semibold">{t(strings.pages.dashboard.inventoryTitle)}</h2>
      {loading && <p role="status" className="mt-space-sm">{t(strings.pages.dashboard.loading)}</p>}
      {error && <p role="alert" className="mt-space-sm text-danger">{error}</p>}
      {!loading && !error && instances.length === 0 && <p className="mt-space-sm text-muted">{t(strings.pages.dashboard.empty)}</p>}
      {!loading && !error && instances.length > 0 && (
        <div className="mt-space-sm overflow-x-auto">
          <DataTable rows={instances} columns={columns} getRowKey={(row) => row.id} caption={t(strings.pages.dashboard.inventoryTitle)} status="success" hideQueryControls hideDensityControl />
        </div>
      )}
    </div>
  );
}
