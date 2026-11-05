import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { ListToolbar } from "./ListToolbar";

describe("ListToolbar", () => {
  afterEach(() => cleanup());

  it("calls onSearchChange when the user types", async () => {
    const onSearchChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <ListToolbar
        testId="probe"
        searchValue=""
        onSearchChange={onSearchChange}
      />,
    );
    await user.type(screen.getByTestId("probe-search"), "x");
    expect(onSearchChange).toHaveBeenLastCalledWith("x");
  });

  it("renders the sort dropdown and flips direction on click", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <ListToolbar
        testId="probe"
        searchValue=""
        onSearchChange={() => {}}
        sort={{
          options: [{ value: "name", label: "Name" }],
          value: { key: "name", dir: "asc" },
          onChange,
          testIdPrefix: "probe",
        }}
      />,
    );
    await user.click(screen.getByTestId("probe-sort-dir"));
    expect(onChange).toHaveBeenLastCalledWith({ key: "name", dir: "desc" });
  });

  it("slots the filters and actions content through", () => {
    renderWithProviders(
      <ListToolbar
        testId="probe"
        searchValue=""
        onSearchChange={() => {}}
        filters={<span data-testid="probe-filter-slot">filters</span>}
        actions={<span data-testid="probe-action-slot">actions</span>}
        summary={<p data-testid="probe-summary-slot">summary</p>}
      />,
    );
    expect(screen.getByTestId("probe-filter-slot")).toBeInTheDocument();
    expect(screen.getByTestId("probe-action-slot")).toBeInTheDocument();
    expect(screen.getByTestId("probe-summary-slot")).toBeInTheDocument();
  });
});
