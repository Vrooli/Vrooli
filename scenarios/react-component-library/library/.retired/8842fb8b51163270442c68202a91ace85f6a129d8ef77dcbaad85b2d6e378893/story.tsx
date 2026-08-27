import { DataTable, type DataTableColumn } from "./DataTable";

interface ScenarioRow {
  id: string;
  name: string;
  status: "Current" | "Behind";
}

const rows: ScenarioRow[] = [
  { id: "rcl", name: "React Component Library", status: "Current" },
  { id: "web", name: "Web Console", status: "Behind" },
];

const columns: Array<DataTableColumn<ScenarioRow>> = [
  {
    id: "name",
    header: "Scenario",
    accessor: (row) => row.name,
    sortValue: (row) => row.name,
    searchValue: (row) => row.name,
  },
  {
    id: "status",
    header: "Status",
    accessor: (row) => row.status,
    sortValue: (row) => row.status,
    searchValue: (row) => row.status,
  },
];

const filters = [
  {
    id: "current",
    label: "Current",
    predicate: (row: ScenarioRow) => row.status === "Current",
  },
];

export function Populated() {
  return (
    <DataTable
      rows={rows}
      columns={columns}
      filters={filters}
      getRowKey={(row) => row.id}
      caption="Scenario adoption table"
    />
  );
}

export function Empty() {
  return (
    <DataTable
      rows={[]}
      columns={[columns[0]]}
      getRowKey={(row) => row.id}
      caption="No rows table"
      emptyMessage="No scenarios match"
    />
  );
}
