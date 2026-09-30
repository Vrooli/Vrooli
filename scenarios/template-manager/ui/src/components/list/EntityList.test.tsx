import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";

import { EntityList, EntityRowBody } from "./EntityList";

interface Item {
  id: string;
}

const rows: Item[] = Array.from({ length: 20 }, (_, i) => ({ id: `item-${i}` }));

function renderList(pageSize = 8) {
  return renderWithProviders(
    <EntityList
      rows={rows}
      pageSize={pageSize}
      tableTestId="list"
      getRowKey={(row) => row.id}
      getRowHref={(row) => `/items/${row.id}`}
      getRowTestId={(row) => `row-${row.id}`}
      getRowLabel={(row) => `View ${row.id}`}
      renderRow={(row) => <EntityRowBody primary={row.id} secondary="meta" trailing={null} />}
      searchText={(row) => row.id}
      searchLabel="Search items"
      searchPlaceholder="Search"
      emptyLabel="No items"
    />,
  );
}

describe("EntityList", () => {
  it("renders only one page of rows at a time", () => {
    renderList(8);
    const list = screen.getByTestId("list");
    expect(within(list).getAllByRole("link")).toHaveLength(8);
    expect(screen.getByTestId("row-item-0")).toBeInTheDocument();
    expect(screen.queryByTestId("row-item-8")).not.toBeInTheDocument();
  });

  it("advances to the next page", () => {
    renderList(8);
    fireEvent.click(screen.getByRole("button", { name: /next/i }));
    expect(screen.queryByTestId("row-item-0")).not.toBeInTheDocument();
    expect(screen.getByTestId("row-item-8")).toBeInTheDocument();
  });

  it("filters via the search box after paging and clamps back into range", () => {
    renderList(8);
    fireEvent.click(screen.getByRole("button", { name: /next/i }));
    // A single-match search collapses to one page; the clamped page shows it.
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "item-19" } });
    const list = screen.getByTestId("list");
    expect(within(list).getAllByRole("link")).toHaveLength(1);
    expect(screen.getByTestId("row-item-19")).toBeInTheDocument();
  });

  it("shows the empty label when nothing matches", () => {
    renderList(8);
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "zzz" } });
    expect(screen.getByTestId("list")).toHaveTextContent("No items");
  });
});
