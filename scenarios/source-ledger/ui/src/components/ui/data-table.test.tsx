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
    expect(within(screen.getByRole("table")).getAllByText("Alpha")[0]).toBeInTheDocument();
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

  it("supports filters, reverse sorting, non-sortable columns, and empty results", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={[
          ...columns,
          { id: "label", header: "Label", accessor: (row) => <strong>{row.name}</strong> },
        ]}
        getRowKey={(row) => row.id}
        caption="Filtered demo rows"
        filters={[{ id: "first", label: "First only", predicate: (row) => row.id === "a" }]}
        filterGroupLabel="Row filters"
        sortLabel={(header) => `Order ${header}`}
        emptyMessage="Nothing matched"
      />,
    );

    await user.click(screen.getByRole("button", { name: "First only" }));
    expect(within(screen.getByRole("table")).getAllByText("Alpha")[0]).toBeInTheDocument();
    expect(screen.queryByText("Beta")).not.toBeInTheDocument();

    const nameSort = screen.getByRole("button", { name: "Order Name" });
    await user.click(nameSort);
    await user.click(nameSort);
    expect(screen.getByRole("columnheader", { name: "Label" })).toBeInTheDocument();

    await user.type(screen.getByRole("searchbox"), "does-not-exist");
    expect(screen.getByText("Nothing matched")).toBeInTheDocument();
  });

  it("renders the empty state when no sortable column is configured", () => {
    renderWithProviders(
      <DataTable
        rows={[]}
        columns={[{ id: "label", header: "Label", accessor: () => "" }]}
        getRowKey={() => "empty"}
        caption="Empty rows"
      />,
    );

    expect(screen.getByText("No rows")).toBeInTheDocument();
  });
});
