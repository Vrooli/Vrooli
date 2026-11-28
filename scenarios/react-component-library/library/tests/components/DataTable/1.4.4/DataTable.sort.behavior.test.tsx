import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DataTable } from "@vrooli/react-component-library/DataTable/1.4.4";

type Row = { id: string; name: string; size: number };

const rows: Row[] = [
  { id: "a", name: "Beta", size: 2 },
  { id: "b", name: "Alpha", size: 1 },
];

const columns = [
  { id: "name", header: "Name", accessor: (row: Row) => row.name, sortValue: (row: Row) => row.name },
  { id: "size", header: "Size", accessor: (row: Row) => row.size, sortValue: (row: Row) => row.size },
  { id: "note", header: "Note", accessor: () => "-" },
];

describe("DataTable sort state", () => {
  it("exposes the active sort on its column header and moves it with the user's choice", () => {
    render(<DataTable caption="Files" rows={rows} columns={columns} getRowKey={(row: Row) => row.id} />);

    const name = screen.getByRole("columnheader", { name: /Name/ });
    const size = screen.getByRole("columnheader", { name: /Size/ });
    const note = screen.getByRole("columnheader", { name: "Note" });
    expect(name).toHaveAttribute("aria-sort", "ascending");
    expect(size).toHaveAttribute("aria-sort", "none");
    expect(note).not.toHaveAttribute("aria-sort");

    fireEvent.click(screen.getByRole("button", { name: "Sort by Name" }));
    expect(name).toHaveAttribute("aria-sort", "descending");

    fireEvent.click(screen.getByRole("button", { name: "Sort by Size" }));
    expect(size).toHaveAttribute("aria-sort", "ascending");
    expect(name).toHaveAttribute("aria-sort", "none");
  });
});
