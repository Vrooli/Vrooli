import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Table, type ColumnDef } from "./Table";

interface Row {
  id: string;
  name: string;
  count: number;
}

const NAME_A = "alpha-x";
const NAME_B = "beta-x";
const NAME_HEADER = "name-h";

const ROWS: Row[] = [
  { id: "a", name: NAME_A, count: 1 },
  { id: "b", name: NAME_B, count: 2 },
];

const COLS: ColumnDef<Row>[] = [
  { key: "name", header: NAME_HEADER, cell: (r) => r.name, sortable: true },
  { key: "count", header: "count-h", cell: (r) => r.count, align: "right" },
];

describe("Table", () => {
  it("renders one row per record with stable row keys", () => {
    render(<Table data-testid="t" columns={COLS} rows={ROWS} rowKey={(r) => r.id} />);
    const table = screen.getByTestId("t");
    expect(table.querySelector('[data-row-key="a"]')).not.toBeNull();
    expect(table.querySelector('[data-row-key="b"]')).not.toBeNull();
    expect(table.textContent).toContain(NAME_A);
    expect(table.textContent).toContain(NAME_B);
  });

  it("renders loading skeletons", () => {
    const { container } = render(
      <Table data-testid="t" columns={COLS} rows={[]} rowKey={(r) => r.id} loading />,
    );
    expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
  });

  it("renders error alert with the supplied message", () => {
    const errMsg = "kaboom-x";
    render(<Table data-testid="t" columns={COLS} rows={[]} rowKey={(r) => r.id} error={errMsg} />);
    expect(screen.getByRole("alert").textContent).toContain(errMsg);
  });

  it("renders empty state when no rows", () => {
    const emptyTitle = "empty-x";
    render(<Table data-testid="t" columns={COLS} rows={[]} rowKey={(r) => r.id} emptyTitle={emptyTitle} />);
    expect(screen.getByTestId("t").textContent).toContain(emptyTitle);
  });

  it("fires onRowClick for clickable rows", async () => {
    const onRowClick = vi.fn();
    render(
      <Table data-testid="t" columns={COLS} rows={ROWS} rowKey={(r) => r.id} onRowClick={onRowClick} />,
    );
    const row = screen.getByTestId("t").querySelector('[data-row-key="a"]') as HTMLElement;
    await userEvent.click(row);
    expect(onRowClick).toHaveBeenCalledWith(ROWS[0]);
  });

  it("cycles sort direction on sortable column", async () => {
    const onSortChange = vi.fn();
    render(
      <Table
        data-testid="t"
        columns={COLS}
        rows={ROWS}
        rowKey={(r) => r.id}
        sort={{ key: "name", direction: "asc" }}
        onSortChange={onSortChange}
      />,
    );
    const headerButton = screen.getByTestId("t").querySelector("thead button") as HTMLElement;
    await userEvent.click(headerButton);
    expect(onSortChange).toHaveBeenCalledWith({ key: "name", direction: "desc" });
  });
});
