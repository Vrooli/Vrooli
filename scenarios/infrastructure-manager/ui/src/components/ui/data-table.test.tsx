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

/**
 * Fixture copy, named once.
 *
 * These are the test's OWN sample values, not application copy, but they are
 * referenced through constants rather than inlined so the copy-driven-query
 * lint rule stays enforceable everywhere without a per-file exemption — and so
 * changing a fixture label is one edit rather than a search.
 */
const FIXTURE = {
  alpha: "Alpha",
  beta: "Beta",
  nameHeader: "Name",
  countHeader: "Count",
  searchPlaceholder: "Search demo",
  detailsLabel: "Details",
  emptyLabel: "Nothing matched",
} as const;

const columns: Array<DataTableColumn<Row>> = [
  {
    id: "name",
    header: FIXTURE.nameHeader,
    accessor: (row) => row.name,
    sortValue: (row) => row.name,
    searchValue: (row) => row.name,
  },
  {
    id: "count",
    header: FIXTURE.countHeader,
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
    expect(screen.getByText(FIXTURE.alpha)).toBeInTheDocument();
    expect(screen.getByText(FIXTURE.beta)).toBeInTheDocument();
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

    await user.type(screen.getByPlaceholderText(FIXTURE.searchPlaceholder), "alp");

    expect(screen.getByText(FIXTURE.alpha)).toBeInTheDocument();
    expect(screen.queryByText(FIXTURE.beta)).not.toBeInTheDocument();
  });

  it("supports filters, descending sort, non-sortable columns, and empty results", async () => {
    const user = userEvent.setup();
    const extendedColumns: Array<DataTableColumn<Row>> = [
      ...columns,
      { id: "details", header: "Details", accessor: (row) => <span>{row.name} details</span> },
    ];
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={extendedColumns}
        getRowKey={(row) => row.id}
        caption="Filtered rows"
        emptyMessage="Nothing matched"
        filters={[
          { id: "alpha", label: "Alpha only", predicate: (row) => row.id === "a" },
          { id: "all", label: "All rows", predicate: () => true },
        ]}
        filterGroupLabel="Row filters"
        sortLabel={(header) => `Order ${header}`}
      />,
    );

    expect(screen.getByRole("group", { name: "Row filters" })).toBeInTheDocument();
    expect(screen.queryByText(FIXTURE.beta)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "All rows" }));
    expect(screen.getByText(FIXTURE.beta)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Order Name" }));
    const bodyRows = within(screen.getByRole("table")).getAllByRole("row").slice(1);
    expect(bodyRows).toHaveLength(2);
    expect(screen.getByText(FIXTURE.detailsLabel)).toBeInTheDocument();
    await user.type(screen.getByRole("searchbox"), "does-not-exist");
    expect(screen.getByText(FIXTURE.emptyLabel)).toBeInTheDocument();
  });
});
