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

  it("combines a named filter, fallback rendered-text search, and descending numeric sorting", async () => {
    const user = userEvent.setup();
    const fallbackColumns: Array<DataTableColumn<Row>> = [
      { id: "name", header: "Name", accessor: (row) => <strong>{row.name}</strong> },
      { id: "count", header: "Count", accessor: (row) => row.count, sortValue: (row) => row.count },
    ];
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={fallbackColumns}
        getRowKey={(row) => row.id}
        caption="Filtered rows"
        emptyMessage="Nothing matches"
        filters={[{ id: "only-beta", label: "Only Beta", predicate: (row) => row.name === "Beta" }]}
        filterGroupLabel="Row status"
        sortLabel={(header) => `Order ${header}`}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Only Beta" }));
    expect(screen.getByText("Beta")).toBeInTheDocument();
    expect(screen.queryByText("Alpha")).not.toBeInTheDocument();

    await user.clear(screen.getByRole("searchbox"));
    await user.type(screen.getByRole("searchbox"), "beta");
    expect(screen.getByText("Beta")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Order Count" }));
    const bodyRows = within(screen.getByRole("table")).getAllByRole("row").slice(1);
    expect(bodyRows[0]?.textContent).toContain("Beta");
    expect(screen.getByRole("group", { name: "Row status" })).toBeInTheDocument();
  });

  it("renders the configured empty state when a search has no matches", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DataTable rows={rows} columns={columns} getRowKey={(row) => row.id} caption="Empty rows" emptyMessage="Nothing matches" />);

    await user.type(screen.getByRole("searchbox"), "missing");

    expect(screen.getByText("Nothing matches")).toBeInTheDocument();
  });

  it("supports a searchable table with no sortable column", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={[{ id: "name", header: "Name", accessor: (row) => [null, <span key={row.id}>{row.name}</span>] }]}
        getRowKey={(row) => row.id}
        caption="Simple rows"
      />,
    );

    expect(screen.queryByRole("button", { name: /sort/i })).not.toBeInTheDocument();
    await user.type(screen.getByRole("searchbox"), "alpha");
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.queryByText("Beta")).not.toBeInTheDocument();
  });
});
