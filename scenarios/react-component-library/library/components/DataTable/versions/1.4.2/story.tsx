import { DataTable } from "./DataTable";

type Row = { id: string; name: string; status: string };

const rows: Row[] = [
  { id: "rcl-1", name: "Preview workspace", status: "Ready" },
  { id: "rcl-2", name: "Release evidence", status: "Review" },
];

export function Default() {
  return (
    <DataTable<Row>
      rows={rows}
      getRowKey={(row) => row.id}
      caption="Preview workspace records"
      columns={[
        { id: "name", header: "Name", accessor: (row) => row.name, sortValue: (row) => row.name },
        { id: "status", header: "Status", accessor: (row) => row.status },
      ]}
    />
  );
}
