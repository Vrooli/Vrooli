import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { DataTable } from "./DataTable";

interface Row {
  id: string;
  name: string;
}

const rows: Row[] = [
  { id: "1", name: "alpha" },
  { id: "2", name: "beta" },
];

const columns = [
  { key: "name", header: "Name", cell: (r: Row) => r.name },
];

describe("DataTable", () => {
  it("renders empty message when no rows", () => {
    render(<DataTable rows={[]} columns={columns} getRowId={(r) => r.id} emptyMessage="none" />);
    expect(screen.getByTestId(selectors.shared.dataTable.empty)).toHaveTextContent("none");
  });

  it("renders rows with stable testids", () => {
    render(
      <DataTable rows={rows} columns={columns} getRowId={(r) => r.id} emptyMessage="none" />,
    );
    expect(screen.getByTestId(selectors.shared.dataTable.row({ id: "1" }))).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.shared.dataTable.cell({ id: "1", column: "name" })),
    ).toHaveTextContent("alpha");
    expect(
      screen.getByTestId(selectors.shared.dataTable.cell({ id: "2", column: "name" })),
    ).toHaveTextContent("beta");
  });

  it("fires onRowClick with the row data", () => {
    const onRowClick = vi.fn();
    render(
      <DataTable
        rows={rows}
        columns={columns}
        getRowId={(r) => r.id}
        emptyMessage="none"
        onRowClick={onRowClick}
      />,
    );
    fireEvent.click(screen.getByTestId(selectors.shared.dataTable.row({ id: "1" })));
    expect(onRowClick).toHaveBeenCalledWith(rows[0]);
  });
});
