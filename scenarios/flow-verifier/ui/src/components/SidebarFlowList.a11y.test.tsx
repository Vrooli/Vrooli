import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return { ...actual, fetchFlows: vi.fn() };
});

import { SidebarFlowList } from "./SidebarFlowList";

function renderFlowList() {
  return renderWithProviders(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <MemoryRouter initialEntries={["/"]}><SidebarFlowList /></MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("SidebarFlowList accessibility", () => {
  beforeEach(async () => {
    const { fetchFlows } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([
      { flowId: "alpha.flow", contractPath: "a/flow.json", language: "ts", schemaVersion: 1, kind: "temporal" },
    ]);
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderFlowList();
    await waitFor(() =>
      expect(screen.getByTestId("sidebar-flow-alpha.flow")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
