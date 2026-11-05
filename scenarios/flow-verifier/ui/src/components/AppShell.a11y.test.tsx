import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

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
import { ThemeProvider } from "./theme/ThemeProvider";

function renderShell(ui: ReactNode) {
  return renderWithProviders(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <MemoryRouter initialEntries={["/"]}><ThemeProvider>{ui}</ThemeProvider></MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    const { fetchFlows } = await import("../api/inventory");
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchFlows).mockResolvedValue([]);
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderShell(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<section aria-label="content">page</section>} />
        </Route>
      </Routes>,
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
