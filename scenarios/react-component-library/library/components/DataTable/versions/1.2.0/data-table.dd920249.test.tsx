import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { DataTable, type DataTableColumn } from "./DataTable.tsx";

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

afterEach(() => {
  cleanup();
});

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

  it("supports filters, numeric sorting, and the accessor search fallback", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={[
          ...columns,
          { id: "display", header: "Display", accessor: (row) => <span>{row.name}</span> },
        ]}
        getRowKey={(row) => row.id}
        caption="Filtered rows"
        filters={[{ id: "alpha", label: "Alpha only", predicate: (row) => row.name === "Alpha" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Alpha only" }));
    expect(screen.getAllByText("Alpha").length).toBeGreaterThan(0);
    expect(screen.queryByText("Beta")).not.toBeInTheDocument();

    await user.clear(screen.getByRole("searchbox"));
    await user.type(screen.getByRole("searchbox"), "beta");
    expect(screen.queryByText("Alpha")).not.toBeInTheDocument();
    expect(screen.getByText("No rows")).toBeInTheDocument();
  });

  it("renders an empty message and toggles a numeric sort back to ascending", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable rows={[]} columns={columns} getRowKey={(row) => row.id} caption="Empty rows" emptyMessage="Nothing here" />,
    );
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
    cleanup();

    renderWithProviders(
      <DataTable rows={rows} columns={columns} getRowKey={(row) => row.id} caption="Sorted rows" />,
    );
    const countButtons = screen.getAllByRole("button", { name: "Sort by Count" });
    const countButton = countButtons[countButtons.length - 1];
    if (!countButton) throw new Error("count sort button was not rendered");
    await user.click(countButton);
    await user.click(countButton);
    await user.click(countButton);
    const tables = screen.getAllByRole("table");
    const table = tables[tables.length - 1];
    if (!table) throw new Error("sorted table was not rendered");
    const bodyRows = within(table).getAllByRole("row").slice(1);
    expect(bodyRows[0]?.textContent).toContain("Alpha");
  });
});
