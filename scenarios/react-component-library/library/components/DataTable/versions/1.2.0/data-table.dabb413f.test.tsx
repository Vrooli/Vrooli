import { describe, expect, it } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
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

describe("DataTable", () => {
  it("renders rows and exposes the table test id", () => {
    const { container } = renderWithProviders(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        caption="Demo rows"
        tableTestId="demo-table"
      />,
    );

    expect(screen.getByTestId("demo-table")).toBeInTheDocument();
    expect(container).toHaveTextContent("Alpha");
    expect(container).toHaveTextContent("Beta");
  });

  it("sorts by a sortable column", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DataTable rows={rows} columns={columns} getRowKey={(row) => row.id} caption="Demo rows" />,
    );

    await user.click(screen.getByRole("button", { name: /name/i }));

    await waitFor(() => {
      const bodyRows = within(screen.getByRole("table")).getAllByRole("row").slice(1);
      expect(bodyRows[0]?.textContent).toContain("Beta");
      expect(bodyRows[1]?.textContent).toContain("Alpha");
    });
  });

  it("filters rows through search", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        caption="Demo rows"
        searchPlaceholder="Search demo"
      />,
    );

    const search = container.querySelector<HTMLInputElement>('input[type="search"]');
    expect(search).not.toBeNull();
    await user.type(search as HTMLInputElement, "alp");

    expect(container).toHaveTextContent("Alpha");
    expect(container).not.toHaveTextContent("Beta");
  });

  it("supports filter buttons, non-sortable columns, and empty states", async () => {
    const user = userEvent.setup();
    const mixedColumns: Array<DataTableColumn<Row>> = [
      {
        id: "label",
        header: "Label",
        accessor: (row) => <span>{row.name}</span>,
      },
      ...columns,
    ];
    const { container } = renderWithProviders(
      <DataTable
        rows={rows}
        columns={mixedColumns}
        getRowKey={(row) => row.id}
        caption="Filtered rows"
        emptyMessage="Nothing matched"
        filters={[
          { id: "all", label: "All", predicate: () => true },
          { id: "large", label: "Large", predicate: (row) => row.count > 10 },
        ]}
      />,
    );

    expect(screen.getByRole("columnheader", { name: "Label" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Large" }));

    expect(container).toHaveTextContent("Nothing matched");
  });

  it("searches rendered node content and falls back for empty values", async () => {
    const user = userEvent.setup();
    const nodeRows = [
      { id: "empty", label: null },
      { id: "node", label: ["Nested", <span key="value">Value</span>, false] },
    ];
    const nodeColumns: Array<DataTableColumn<(typeof nodeRows)[number]>> = [
      {
        id: "label",
        header: "Label",
        accessor: (row) => row.label,
      },
    ];
    const { container } = renderWithProviders(
      <DataTable
        rows={nodeRows}
        columns={nodeColumns}
        getRowKey={(row) => row.id}
        caption="Node rows"
        searchPlaceholder="Search nodes"
      />,
    );

    const search = container.querySelector<HTMLInputElement>('input[type="search"]');
    expect(search).not.toBeNull();
    await user.type(search as HTMLInputElement, "nested value");

    expect(container).toHaveTextContent("Nested");
    expect(container).toHaveTextContent("Value");
    expect(container).not.toHaveTextContent("empty");
  });
});
