import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return { ...actual, fetchFlows: vi.fn() };
});

import { SidebarFlowList } from "./SidebarFlowList";

describe("SidebarFlowList", () => {
  beforeEach(async () => {
    const { fetchFlows } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockReset();
  });
  afterEach(() => cleanup());

  it("renders the loading state while fetching", async () => {
    const { fetchFlows } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockReturnValue(new Promise(() => {}));
    renderWithProviders(<SidebarFlowList />);
    expect(screen.getByTestId("sidebar-flow-list-loading")).toBeInTheDocument();
  });

  it("renders an empty state when no flows are discovered", async () => {
    const { fetchFlows } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([]);
    renderWithProviders(<SidebarFlowList />);
    await waitFor(() =>
      expect(screen.getByTestId("sidebar-flow-list-empty")).toBeInTheDocument(),
    );
  });

  it("renders a NavLink per flow", async () => {
    const { fetchFlows } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([
      { flowId: "alpha.flow", contractPath: "a/flow.json", language: "ts", schemaVersion: 1 },
      { flowId: "beta.flow", contractPath: "b/flow.json", language: "go", schemaVersion: 1 },
    ]);
    renderWithProviders(<SidebarFlowList />);
    await waitFor(() =>
      expect(screen.getByTestId("sidebar-flow-alpha.flow")).toHaveAttribute(
        "href",
        "/flows/alpha.flow",
      ),
    );
    expect(screen.getByTestId("sidebar-flow-beta.flow")).toBeInTheDocument();
  });

  it("renders an error state on fetch failure", async () => {
    const { fetchFlows } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockRejectedValue(new Error("nope"));
    renderWithProviders(<SidebarFlowList />);
    await waitFor(() =>
      expect(screen.getByTestId("sidebar-flow-list-error")).toBeInTheDocument(),
    );
  });
});
