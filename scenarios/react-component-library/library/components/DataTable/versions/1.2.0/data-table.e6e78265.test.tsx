import { describe, expect, it } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { DataTable, type DataTableColumn } from "./DataTable.tsx";

interface Row {
  id: string;
  name: string;
  count: number;
}

const betaName = "Beta";
const alphaName = "Alpha";

const rows: Row[] = [
  { id: "b", name: betaName, count: 2 },
  { id: "a", name: alphaName, count: 1 },
];

const searchPlaceholder = "Search demo";
const noMatchingRows = "No matching demo rows";

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
    expect(screen.getByText(alphaName)).toBeInTheDocument();
    expect(screen.getByText(betaName)).toBeInTheDocument();
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
        searchPlaceholder={searchPlaceholder}
      />,
    );

    await user.type(screen.getByPlaceholderText(searchPlaceholder), "alp");

    expect(screen.getByText(alphaName)).toBeInTheDocument();
    expect(screen.queryByText(betaName)).not.toBeInTheDocument();
  });

  it("applies named filters, toggles descending sort, and explains an empty result", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        caption="Demo rows"
        searchPlaceholder={searchPlaceholder}
        emptyMessage={noMatchingRows}
        filters={[
          { id: "all", label: "All", predicate: () => true },
          { id: "large", label: "Large", predicate: (row) => row.count > 1 },
        ]}
        filterGroupLabel="Demo filters"
        sortLabel={(header) => `Order by ${header}`}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Large" }));
    expect(screen.getByText(betaName)).toBeInTheDocument();
    expect(screen.queryByText(alphaName)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Order by Name" }));
    await user.type(screen.getByPlaceholderText(searchPlaceholder), "missing");
    expect(screen.getByText(noMatchingRows)).toBeInTheDocument();
  });
});
