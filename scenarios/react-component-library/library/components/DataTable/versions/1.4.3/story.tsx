import { useState } from "react";
import { DataTable, type DataTableStatus } from "./DataTable";
const rows = [
  { id: "alpha", name: "Alpha" },
  { id: "beta", name: "Beta" },
];
const columns = [
  {
    id: "name",
    header: "Name",
    accessor: (row: (typeof rows)[number]) => row.name,
    sortValue: (row: (typeof rows)[number]) => row.name,
  },
];
function Specimen({ initialStatus = "success" }: { initialStatus?: DataTableStatus }) {
  const [status, setStatus] = useState(initialStatus);
  return (
    <DataTable
      rows={status === "empty" ? [] : rows}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="Work items"
      status={status}
      searchLabel="Search work items"
      enableSelection
      pageSize={1}
      statusMessage={
        status === "refreshing"
          ? "Refreshing work items"
          : status === "partial"
            ? "Some work items are unavailable"
            : undefined
      }
      emptyMessage="No work items"
      emptyDetail="Create a work item to begin."
      errorMessage="Work items could not be refreshed"
      permissionMessage="Request access to work items"
      onRetry={() => setStatus("success")}
    />
  );
}
export const Default = () => <Specimen />;
export const Refreshing = () => <Specimen initialStatus="refreshing" />;
export const Partial = () => <Specimen initialStatus="partial" />;
export const Empty = () => <Specimen initialStatus="empty" />;
export const RequestError = () => <Specimen initialStatus="request-error" />;
export const PermissionDenied = () => <Specimen initialStatus="permission-denied" />;
