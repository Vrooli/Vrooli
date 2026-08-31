import { DataTable, type DataTableColumn } from "./DataTable";

interface ResourceRow {
  id: string;
  name: string;
  owner: string;
  status: "Ready" | "Needs attention" | "Paused";
  updated: string;
  score: number;
}

const rows: ResourceRow[] = [
  {
    id: "atlas",
    name: "Atlas workspace",
    owner: "Maya Chen",
    status: "Ready",
    updated: "2 minutes ago",
    score: 98,
  },
  {
    id: "beacon",
    name: "Beacon workspace",
    owner: "Noah Williams",
    status: "Needs attention",
    updated: "18 minutes ago",
    score: 72,
  },
  {
    id: "cinder",
    name: "Cinder workspace",
    owner: "Avery Singh",
    status: "Ready",
    updated: "Yesterday",
    score: 94,
  },
  {
    id: "drift",
    name: "Drift workspace",
    owner: "Leila Ortiz",
    status: "Paused",
    updated: "Yesterday",
    score: 61,
  },
  {
    id: "ember",
    name: "Ember workspace",
    owner: "Owen Brooks",
    status: "Ready",
    updated: "2 days ago",
    score: 88,
  },
  {
    id: "foundry",
    name: "Foundry workspace",
    owner: "Riley James",
    status: "Needs attention",
    updated: "3 days ago",
    score: 67,
  },
];

const columns: Array<DataTableColumn<ResourceRow>> = [
  {
    id: "name",
    header: "Resource",
    accessor: (row) => row.name,
    sortValue: (row) => row.name,
    searchValue: (row) => row.name,
  },
  {
    id: "owner",
    header: "Owner",
    accessor: (row) => row.owner,
    sortValue: (row) => row.owner,
    searchValue: (row) => row.owner,
  },
  {
    id: "status",
    header: "Status",
    accessor: (row) => <span>{row.status}</span>,
    sortValue: (row) => row.status,
  },
  {
    id: "updated",
    header: "Updated",
    accessor: (row) => row.updated,
    mobileHidden: true,
  },
  {
    id: "score",
    header: "Health",
    accessor: (row) => `${row.score}%`,
    sortValue: (row) => row.score,
  },
];

const filters = [
  {
    id: "ready",
    label: "Ready",
    predicate: (row: ResourceRow) => row.status === "Ready",
  },
  {
    id: "attention",
    label: "Needs attention",
    predicate: (row: ResourceRow) => row.status === "Needs attention",
  },
];

function frame(children: React.ReactNode) {
  return (
    <section
      style={{
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 1040px)",
        minWidth: 0,
        boxSizing: "border-box",
        padding: "var(--space-xl)",
        border: "var(--border-hairline) solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      {children}
    </section>
  );
}

const actions = [
  { id: "open", label: "Open", onSelect: (row: ResourceRow) => void row },
  {
    id: "archive",
    label: "Archive",
    tone: "danger" as const,
    onSelect: (row: ResourceRow) => void row,
  },
];

export function Default() {
  return frame(
    <DataTable
      rows={rows}
      columns={columns}
      filters={filters}
      getRowKey={(row) => row.id}
      caption="Resource workspaces"
      enableSelection
      rowActions={actions}
      pageSize={3}
      filterGroupLabel="Resource status"
      tableTestId="resource-table"
    />,
  );
}

export function Refreshing() {
  return frame(
    <DataTable
      rows={rows}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="Resource workspaces"
      status="refreshing"
      statusMessage="Checking 6 resources for changes"
      onRetry={() => undefined}
    />,
  );
}

export function Partial() {
  return frame(
    <DataTable
      rows={rows.slice(0, 4)}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="Resource workspaces"
      status="partial"
      statusMessage="Two resources could not be refreshed"
      rowActions={actions}
    />,
  );
}

export function Empty() {
  return frame(
    <DataTable
      rows={[]}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="Resource workspaces"
      status="empty"
      emptyMessage="No resources match"
      emptyDetail="Try clearing a filter or searching for a different workspace."
    />,
  );
}

export function RequestError() {
  return frame(
    <DataTable
      rows={rows}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="Resource workspaces"
      status="request-error"
      errorMessage="The resource service did not respond. Your filters are still here to retry."
      onRetry={() => undefined}
    />,
  );
}

export function PermissionDenied() {
  return frame(
    <DataTable
      rows={[]}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="Resource workspaces"
      status="permission-denied"
      permissionMessage="Ask a workspace administrator for access to this resource collection."
    />,
  );
}
