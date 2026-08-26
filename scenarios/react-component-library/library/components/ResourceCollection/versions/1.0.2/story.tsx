import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState } from "react";
import { ResourceCollection, type ResourceCollectionStatus } from "./ResourceCollection";
import type { DataTableColumn, DataTableFilter } from "@vrooli/react-component-library/DataTable/1.3.0";

interface ResourceRow {
  id: string;
  name: string;
  owner: string;
  status: "Ready" | "Needs attention" | "Paused";
  updated: string;
  health: number;
}

const resources: ResourceRow[] = [
  {
    id: "atlas",
    name: "Atlas workspace",
    owner: "Maya Chen",
    status: "Ready",
    updated: "8 min ago",
    health: 98,
  },
  {
    id: "beacon",
    name: "Beacon workspace",
    owner: "Noah Williams",
    status: "Needs attention",
    updated: "21 min ago",
    health: 72,
  },
  {
    id: "cinder",
    name: "Cinder workspace",
    owner: "Avery Singh",
    status: "Ready",
    updated: "42 min ago",
    health: 94,
  },
  {
    id: "drift",
    name: "Drift workspace",
    owner: "Lucía Santos",
    status: "Paused",
    updated: "1 hr ago",
    health: 84,
  },
  {
    id: "ember",
    name: "Ember workspace",
    owner: "Theo Okafor",
    status: "Ready",
    updated: "2 hrs ago",
    health: 99,
  },
  {
    id: "foundry",
    name: "Foundry workspace",
    owner: "Inez Park",
    status: "Needs attention",
    updated: "Yesterday",
    health: 68,
  },
];

const columns: Array<DataTableColumn<ResourceRow>> = [
  {
    id: "name",
    header: "Resource",
    accessor: (row) => <strong>{row.name}</strong>,
    sortValue: (row) => row.name,
  },
  {
    id: "owner",
    header: "Owner",
    accessor: (row) => row.owner,
    sortValue: (row) => row.owner,
  },
  {
    id: "status",
    header: "Status",
    accessor: (row) => <span data-story-status={row.status}>{row.status}</span>,
    sortValue: (row) => row.status,
  },
  {
    id: "updated",
    header: "Updated",
    accessor: (row) => row.updated,
    sortValue: (row) => row.updated,
  },
  {
    id: "health",
    header: "Health",
    accessor: (row) => `${row.health}%`,
    sortValue: (row) => row.health,
  },
];

const filters: Array<DataTableFilter<ResourceRow>> = [
  { id: "ready", label: "Ready", predicate: (row) => row.status === "Ready" },
  {
    id: "attention",
    label: "Needs attention",
    predicate: (row) => row.status === "Needs attention",
  },
];

const frame = {
  display: "grid",
  gap: "var(--space-lg)",
  width: "min(100%, 1040px)",
  minWidth: 0,
  boxSizing: "border-box" as const,
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
};

function Example({
  status = "idle",
  rows = resources,
}: {
  status?: ResourceCollectionStatus;
  rows?: ResourceRow[];
}) {
  const [query, setQuery] = useState("");
  const [lastAction, setLastAction] = useState("Ready to refine the resource library");
  return (
    <div style={frame}>
      <ResourceCollection
        title={resolveStrings("patterns.resource-collection.title", "Resources")}
        description={resolveStrings(
          "patterns.resource-collection.description",
          "Keep operational work visible, searchable, and ready for the next decision.",
        )}
        rows={rows}
        columns={columns}
        filters={filters}
        getRowKey={(row) => row.id}
        query={query}
        onQueryChange={setQuery}
        status={status}
        statusMessage={status === "refreshing" ? "Checking resource health" : undefined}
        errorMessage="The resource service could not be reached. Your search and filters are still here."
        emptyMessage="No resources match"
        emptyDetail="Clear the search or choose another filter to see more resources."
        onRetry={() => setLastAction("Retry requested")}
        onRefresh={() => setLastAction("Refresh requested")}
        onNavigate={(row) => setLastAction(`Opened ${row.name}`)}
        onExport={() => setLastAction("Export requested")}
        enableSelection
        pageSize={3}
        views={[
          { id: "all", label: "All resources", count: resources.length },
          { id: "attention", label: "My attention", count: 2 },
        ]}
        sortOptions={[
          { id: "updated", label: "Recently updated" },
          { id: "name", label: "Name" },
          { id: "health", label: "Health" },
        ]}
        defaultSortId="updated"
        headerAside={
          <span
            role="status"
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-caption)",
            }}
          >
            {lastAction}
          </span>
        }
      />
    </div>
  );
}

export function Default() {
  return <Example />;
}
export function Loading() {
  return <Example status="loading" />;
}
export function Refreshing() {
  return <Example status="refreshing" />;
}
export function Partial() {
  return <Example status="partial" />;
}
export function Stale() {
  return <Example status="stale" />;
}
export function Offline() {
  return <Example status="offline" />;
}
export function RequestError() {
  return <Example status="request-error" />;
}
export function PermissionDenied() {
  return <Example status="permission-denied" />;
}
export function Empty() {
  return <Example rows={[]} />;
}
