import { screen, within } from "@testing-library/react";

import { Table } from "@vrooli/react-component-library/Table/1";
import { renderWithProviders } from "../test-utils";

describe("Table", () => {
  it("renders an empty table surface when no rows are supplied", () => {
    renderWithProviders(<Table />);
    expect(document.querySelector("[data-rcl-table]")).toBeInTheDocument();
    expect(document.querySelector("table")).toBeInTheDocument();
  });

  it("renders captioned object rows and preserves values", () => {
    renderWithProviders(
      <Table
        caption="People"
        rows={[
          { name: "Alice", role: "Designer" },
          { name: "Bob", role: "Engineer" },
        ]}
      />,
    );
    const table = screen.getByRole("table");
    expect(within(table).getByText("People")).toBeInTheDocument();
    expect(
      within(table)
        .getAllByRole("columnheader")
        .map((cell) => cell.textContent),
    ).toEqual(["name", "role"]);
    expect(within(table).getByText("Alice")).toBeInTheDocument();
    expect(within(table).getByText("Engineer")).toBeInTheDocument();
  });

  it("uses supplied children instead of generating an object table", () => {
    renderWithProviders(
      <Table rows={[{ hidden: "value" }]}>
        <div data-testid="custom-table-content">Custom surface</div>
      </Table>,
    );
    expect(screen.getByTestId("custom-table-content")).toHaveTextContent("Custom surface");
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });
});
