import { screen, within } from "@testing-library/react";

import { Table } from "@vrooli/react-component-library/Table/1";
import { renderWithProviders } from "../test-utils";

// Fixture caption and rows, named once so the queries match the fixture
// instead of restating its copy.
const tableCaption = "People";
type Person = {
  name: string;
  role: string;
};

const peopleRows: [Person, Person] = [
  { name: "Alice", role: "Designer" },
  { name: "Bob", role: "Engineer" },
];

describe("Table", () => {
  it("renders an empty table surface when no rows are supplied", () => {
    renderWithProviders(<Table />);
    expect(document.querySelector("[data-rcl-table]")).toBeInTheDocument();
    expect(document.querySelector("table")).toBeInTheDocument();
  });

  it("renders captioned object rows and preserves values", () => {
    renderWithProviders(<Table caption={tableCaption} rows={peopleRows} />);
    const table = screen.getByRole("table");
    expect(within(table).getByText(tableCaption)).toBeInTheDocument();
    expect(
      within(table)
        .getAllByRole("columnheader")
        .map((cell) => cell.textContent),
    ).toEqual(["name", "role"]);
    expect(within(table).getByText(peopleRows[0].name)).toBeInTheDocument();
    expect(within(table).getByText(peopleRows[1].role)).toBeInTheDocument();
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
