import { describe, expect, it } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1.2.0";

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
    expect(screen.getByText(/Alpha/)).toBeInTheDocument();
    expect(screen.getByText(/Beta/)).toBeInTheDocument();
  });

  it("exposes adopter hooks for the table, headers, and rows", () => {
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={columns.map((column, index) => ({ ...column, headerTestId: index === 0 ? "name-header" : undefined }))}
        getRowKey={(row) => row.id}
        getRowTestId={(row) => `demo-row-${row.id}`}
        caption="Hooked rows"
        tableId="hooked-table"
      />,
    );

    expect(screen.getByRole("table")).toHaveAttribute("id", "hooked-table");
    expect(screen.getByTestId("name-header")).toBeInTheDocument();
    expect(screen.getByTestId("demo-row-a")).toBeInTheDocument();
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

    await user.type(screen.getByPlaceholderText(/Search demo/), "alp");

    expect(screen.getByText(/Alpha/)).toBeInTheDocument();
    expect(screen.queryByText(/Beta/)).not.toBeInTheDocument();
  });

  it("covers empty rows, filters, fallback search text, and sort direction changes", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={[...columns, { id: "plain", header: "Plain", accessor: (row) => <span>{row.name}</span> }]}
        getRowKey={(row) => row.id}
        caption="Filtered rows"
        filters={[{ id: "alpha", label: "Alpha only", predicate: (row) => row.name === "Alpha" }]}
        filterGroupLabel="Row filters"
        sortLabel={(header) => `Order ${header}`}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Alpha only/ }));
    expect(screen.queryByText(/Beta/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Order Name/ }));
    await user.click(screen.getByRole("button", { name: /Order Count/ }));
    await user.click(screen.getByRole("button", { name: /Order Count/ }));
    await user.type(screen.getByRole("searchbox"), "alpha");
    expect(screen.getAllByText(/Alpha/).length).toBeGreaterThan(0);

    const empty = renderWithProviders(
      <DataTable<Row> rows={[]} columns={[{ id: "plain", header: "Plain", accessor: (row) => row.name }]} getRowKey={(row) => row.id} caption="Empty rows" emptyMessage="Nothing here" />,
    );
    expect(empty.getByText(/Nothing here/)).toBeInTheDocument();
  });
});
