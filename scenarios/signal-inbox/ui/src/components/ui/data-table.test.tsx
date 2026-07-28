import { describe, expect, it } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { DataTable, type DataTableColumn } from "./data-table";

interface Row {
  id: string;
  name: string;
  count: number;
}

const rows: Row[] = [
  { id: "b", name: "Beta", count: 2 },
  { id: "a", name: "Alpha", count: 1 },
];

const columns: Array<DataTableColumn<Row>> = [
  {
    id: "name",
    header: "Name",
    accessor: (row) => row.name,
    sortValue: (row) => row.name,
    searchValue: (row) => row.name,
  },
  {
    id: "count",
    header: "Count",
    accessor: (row) => row.count,
    sortValue: (row) => row.count,
    searchValue: (row) => String(row.count),
  },
];

describe("DataTable", () => {
  it("renders rows and exposes the table test id", () => {
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        caption="Demo rows"
        tableTestId="demo-table"
      />,
    );

    expect(screen.getByTestId("demo-table")).toBeInTheDocument();
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.getByText("Beta")).toBeInTheDocument();
  });

  it("sorts by a sortable column", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable rows={rows} columns={columns} getRowKey={(row) => row.id} caption="Demo rows" />,
    );

    await user.click(screen.getByRole("button", { name: /name/i }));
    const bodyRows = within(screen.getByRole("table")).getAllByRole("row").slice(1);

    expect(bodyRows[0]?.textContent).toContain("Beta");
    expect(bodyRows[1]?.textContent).toContain("Alpha");
  });

  it("filters rows through search", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        caption="Demo rows"
        searchPlaceholder="Search demo"
      />,
    );

    await user.type(screen.getByPlaceholderText("Search demo"), "alp");

    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.queryByText("Beta")).not.toBeInTheDocument();
  });

  it("supports fallback search, filters, descending sort, and the empty state", async () => {
    const user = userEvent.setup();
    const fallbackColumns: Array<DataTableColumn<Row>> = [
      { id: "name", header: "Name", accessor: (row) => <strong>{row.name}</strong>, className: "name-column" },
      { id: "count", header: "Count", accessor: (row) => row.count, sortValue: (row) => row.count },
    ];
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={fallbackColumns}
        getRowKey={(row, index) => `${row.id}-${index}`}
        caption="Filtered rows"
        searchLabel="Find rows"
        searchPlaceholder="Find a row"
        emptyMessage="Nothing matched"
        filterLabel="Row state"
        filterGroupLabel="Choose row state"
        sortLabel={(header) => `Order ${header}`}
        filters={[{ id: "one", label: "Only one", predicate: (row) => row.count === 1 }]}
        className="table-shell"
      />,
    );

    await user.click(screen.getByRole("button", { name: "Only one" }));
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.queryByText("Beta")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Order Count" }));
    await user.type(screen.getByPlaceholderText("Find a row"), "missing");
    expect(screen.getByText("Nothing matched")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Choose row state" })).toBeInTheDocument();
  });
});
