import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { expectNoA11yViolations, renderWithProviders, makeHealthResponse } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return { ...actual, fetchFlows: vi.fn() };
});
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { AppShell } from "./AppShell";

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    const { fetchFlows } = await import("../api/inventory");
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchFlows).mockResolvedValue([]);
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<section aria-label="content">page</section>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId("health-pill")).toHaveTextContent(/ok/i),
    );
    await waitFor(() =>
      expect(screen.getByTestId("sidebar-flow-list-empty")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
