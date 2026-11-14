import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  InventoryFilters,
  type InventoryFilterState,
} from "./InventoryFilters";

const baseState: InventoryFilterState = {
  scenarioId: "",
  search: "",
  language: "all",
  status: [],
  sort: { key: "flowId", dir: "asc" },
};

describe("InventoryFilters", () => {
  afterEach(() => cleanup());

  it("emits search updates as the user types", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <InventoryFilters
        value={baseState}
        scenarios={[]}
        onChange={onChange}
        onReload={() => {}}
        onVerifyAll={() => {}}
        flowsCount={3}
      />,
    );
    await user.type(screen.getByTestId("inventory-search"), "n");
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ search: "n" }),
    );
  });

  it("toggles status filters as a multi-select", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <InventoryFilters
        value={baseState}
        scenarios={[]}
        onChange={onChange}
        onReload={() => {}}
        onVerifyAll={() => {}}
        flowsCount={3}
      />,
    );
    await user.click(screen.getByTestId("inventory-status-passed"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ status: ["passed"] }),
    );
  });

  it("flips sort direction", async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <InventoryFilters
        value={baseState}
        scenarios={[]}
        onChange={onChange}
        onReload={() => {}}
        onVerifyAll={() => {}}
        flowsCount={3}
      />,
    );
    await user.click(screen.getByTestId("inventory-sort-dir"));
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ sort: { key: "flowId", dir: "desc" } }),
    );
  });

  it("disables Verify all when there are no flows", () => {
    renderWithProviders(
      <InventoryFilters
        value={baseState}
        scenarios={[]}
        onChange={() => {}}
        onReload={() => {}}
        onVerifyAll={() => {}}
        flowsCount={0}
      />,
    );
    expect(screen.getByTestId("inventory-verify-all")).toBeDisabled();
  });

  it("invokes onReload when reload is clicked", async () => {
    const onReload = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <InventoryFilters
        value={baseState}
        scenarios={[]}
        onChange={() => {}}
        onReload={onReload}
        onVerifyAll={() => {}}
        flowsCount={3}
      />,
    );
    await user.click(screen.getByTestId("inventory-reload"));
    expect(onReload).toHaveBeenCalledOnce();
  });
});
