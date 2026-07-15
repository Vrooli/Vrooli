import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DataTable, type DataTableColumn, type DataTableFilter } from "./data-table";

interface Row {
  id: string;
  name: string;
  score: number;
  active: boolean;
}

const rows: Row[] = [
  { id: "a", name: "Charlie", score: 30, active: true },
  { id: "b", name: "alice", score: 10, active: false },
  { id: "c", name: "Bravo", score: 20, active: true },
];

const columns: Array<DataTableColumn<Row>> = [
  {
    id: "name",
    header: "Name",
    // Accessor returns a React element so the searchableText fallback has to
    // recurse into element children rather than read an explicit searchValue.
    accessor: (row) => <strong>{row.name}</strong>,
    sortValue: (row) => row.name,
  },
  {
    id: "score",
    header: "Score",
    accessor: (row) => row.score,
    sortValue: (row) => row.score,
    searchValue: (row) => String(row.score),
  },
  {
    id: "status",
    header: "Status",
    // No sortValue: exercises the non-sortable header branch.
    accessor: (row) => (row.active ? "Active" : "Idle"),
  },
];

const filters: Array<DataTableFilter<Row>> = [
  { id: "all", label: "All", predicate: () => true },
  { id: "active", label: "Active", predicate: (row) => row.active },
];

const renderTable = (overrides: Partial<React.ComponentProps<typeof DataTable<Row>>> = {}) =>
  render(
    <DataTable
      rows={rows}
      columns={columns}
      getRowKey={(row) => row.id}
      caption="People"
      tableTestId="people-table"
      filterGroupLabel="Filter people"
      filters={filters}
      {...overrides}
    />,
  );

const bodyNames = () => {
  const table = screen.getByTestId("people-table");
  const bodyRows = within(table).getAllByRole("row").slice(1); // drop header row
  return bodyRows.map((r) => within(r).getAllByRole("cell")[0]?.textContent);
};

describe("DataTable", () => {
  afterEach(() => cleanup());

  it("renders every row and defaults to ascending sort on the first sortable column", () => {
    renderTable();
    // firstSortable is "name" → ascending, numeric+base-insensitive: alice, Bravo, Charlie
    expect(bodyNames()).toEqual(["alice", "Bravo", "Charlie"]);
    expect(screen.getByTestId("people-table")).toBeInTheDocument();
  });

  it("toggles sort direction on the active column and resets to asc on a new column", async () => {
    const user = userEvent.setup();
    renderTable();

    await user.click(screen.getByRole("button", { name: "Sort by Name" }));
    expect(bodyNames()).toEqual(["Charlie", "Bravo", "alice"]); // desc

    // Switch to the numeric Score column → asc by number.
    await user.click(screen.getByRole("button", { name: "Sort by Score" }));
    expect(bodyNames()).toEqual(["alice", "Bravo", "Charlie"]); // scores 10,20,30
  });

  it("uses a custom sortLabel formatter for sortable column buttons", () => {
    renderTable({ sortLabel: (header) => `order:${header}` });
    expect(screen.getByRole("button", { name: "order:Name" })).toBeInTheDocument();
    // The non-sortable Status column renders plain text, not a button.
    expect(screen.queryByRole("button", { name: /Status/ })).not.toBeInTheDocument();
  });

  it("searches by an explicit searchValue and by the rendered-content fallback", async () => {
    const user = userEvent.setup();
    renderTable();
    const search = screen.getByRole("searchbox");

    // searchValue path: Score column returns "20".
    await user.type(search, "20");
    expect(bodyNames()).toEqual(["Bravo"]);

    await user.clear(search);
    // searchableText fallback: Name accessor is <strong>{name}</strong>.
    await user.type(search, "charlie");
    expect(bodyNames()).toEqual(["Charlie"]);
  });

  it("applies the selected filter predicate", async () => {
    const user = userEvent.setup();
    renderTable();
    await user.click(screen.getByRole("button", { name: "Active" }));
    expect(bodyNames().sort()).toEqual(["Bravo", "Charlie"]);
  });

  it("shows the empty message when nothing matches", async () => {
    const user = userEvent.setup();
    const emptyMessage = "Nobody here";
    renderTable({ emptyMessage });
    await user.type(screen.getByRole("searchbox"), "zzz-no-match");
    // When nothing matches, the body collapses to a single spanning cell that
    // carries the empty message.
    const cell = within(screen.getByTestId("people-table")).getByRole("cell");
    expect(cell.textContent).toBe(emptyMessage);
  });

  it("labels the filter group via filterGroupLabel, overriding filterLabel", () => {
    renderTable({ filterLabel: "ignored", filterGroupLabel: "Filter people" });
    expect(screen.getByRole("group", { name: "Filter people" })).toBeInTheDocument();
  });

  it("extracts searchable text from mixed accessor nodes (arrays, elements, and empties)", async () => {
    const user = userEvent.setup();
    const mixedColumns: Array<DataTableColumn<Row>> = [
      {
        id: "mixed",
        header: "Mixed",
        // An array containing an element, a null, a boolean, and a string —
        // searchableText must recurse arrays, read element children, and skip
        // null/boolean without throwing.
        accessor: (row) => [<em key="e">{row.name}</em>, null, row.active, ` #${row.score}`],
      },
    ];
    render(
      <DataTable
        rows={rows}
        columns={mixedColumns}
        getRowKey={(row) => row.id}
        caption="Mixed"
        tableTestId="people-table"
        searchLabel="Find people"
        searchPlaceholder="Type a name"
        emptyMessage="No matches"
      />,
    );

    const matchingRows = () =>
      within(screen.getByTestId("people-table")).getAllByRole("row").slice(1);
    const search = screen.getByRole("searchbox");

    // Matches the element-child text inside the array.
    await user.type(search, "bravo");
    expect(matchingRows()).toHaveLength(1);
    expect(matchingRows()[0]).toHaveTextContent("Bravo");

    await user.clear(search);
    // Matches the trailing string fragment "#10" for alice.
    await user.type(search, "#10");
    expect(matchingRows()).toHaveLength(1);
    expect(matchingRows()[0]).toHaveTextContent("#10");
  });

  it("renders with all optional props defaulted (no filters, labels, or test id)", () => {
    // Exercises the default-parameter branches: no filters group, default
    // filterLabel/filterGroupLabel/sortLabel, and no tableTestId.
    render(
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        caption="Defaults"
      />,
    );
    // No filters passed → the filter button group is absent.
    expect(screen.queryByRole("group")).toBeNull();
    // Default sortLabel formats the sortable column's button as "Sort by <h>".
    expect(screen.getByRole("button", { name: "Sort by Name" })).toBeInTheDocument();
  });

  it("returns empty searchable text for element accessors with no children", async () => {
    const user = userEvent.setup();
    const emptyEl: Array<DataTableColumn<Row>> = [
      { id: "name", header: "Name", accessor: (row) => row.name },
      // An element with no children — searchableText must return "" for it
      // rather than throwing or recursing into undefined.
      { id: "icon", header: "Icon", accessor: () => <br /> },
    ];
    render(
      <DataTable rows={rows} columns={emptyEl} getRowKey={(row) => row.id} caption="Icons" tableTestId="people-table" />,
    );
    // Searching for a name still works (the childless element contributes no
    // searchable text, and does not break the search over other columns).
    await user.type(screen.getByRole("searchbox"), "alice");
    const bodyRows = within(screen.getByTestId("people-table")).getAllByRole("row").slice(1);
    expect(bodyRows).toHaveLength(1);
    expect(bodyRows[0]).toHaveTextContent("alice");
  });
});
