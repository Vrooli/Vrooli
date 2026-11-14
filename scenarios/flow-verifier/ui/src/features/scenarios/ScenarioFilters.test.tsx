import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  ScenarioFilters,
  type ScenarioFilterState,
} from "./ScenarioFilters";

const baseState: ScenarioFilterState = {
  search: "",
  flows: "any",
  errors: "any",
  sort: { key: "name", dir: "asc" },
};

function defaultProps(overrides: Partial<Parameters<typeof ScenarioFilters>[0]> = {}) {
  return {
    value: baseState,
    onChange: vi.fn(),
    onReload: vi.fn(),
    onGenerateAll: vi.fn(),
    onClearAll: vi.fn(),
    scenariosCount: 3,
    selectedCount: 0,
    ...overrides,
  };
}

describe("ScenarioFilters", () => {
  afterEach(() => cleanup());

  it("emits search updates as the user types", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<ScenarioFilters {...defaultProps({ onChange })} />);
    await user.type(screen.getByTestId("scenario-search"), "a");
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ search: "a" }));
  });

  it("flips sort direction", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<ScenarioFilters {...defaultProps({ onChange })} />);
    await user.click(screen.getByTestId("scenario-sort-dir"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ sort: { key: "name", dir: "desc" } }),
    );
  });

  it("disables bulk actions when there are no scenarios", () => {
    renderWithProviders(<ScenarioFilters {...defaultProps({ scenariosCount: 0 })} />);
    expect(screen.getByTestId("scenario-generate-all")).toBeDisabled();
    expect(screen.getByTestId("scenario-clear-all")).toBeDisabled();
  });

  it("invokes onReload when reload is clicked", async () => {
    const onReload = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<ScenarioFilters {...defaultProps({ onReload })} />);
    await user.click(screen.getByTestId("scenario-reload"));
    expect(onReload).toHaveBeenCalledOnce();
  });

  it("changes filter dropdowns", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<ScenarioFilters {...defaultProps({ onChange })} />);
    await user.selectOptions(screen.getByTestId("scenario-flows"), "has");
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ flows: "has" }));
    await user.selectOptions(screen.getByTestId("scenario-errors"), "with");
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ errors: "with" }));
  });
});
