import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { renderWithProviders, makeHealthResponse } from "../test-utils";

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

describe("AppShell", () => {
  beforeEach(async () => {
    const { fetchFlows } = await import("../api/inventory");
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchFlows).mockResolvedValue([]);
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders shell, sidebar, mobile header/nav, and the child route content", async () => {
    renderShell(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div data-testid="child">hello</div>} />
        </Route>
      </Routes>,
    );
    expect(screen.getByTestId("flow-verifier-app-shell")).toBeInTheDocument();
    expect(screen.getByTestId("flow-verifier-app-shell-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("child")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId("health-pill")).toBeInTheDocument(),
    );
  });

  it("does not wrap content in a centered card or eyebrow text", () => {
    const { container } = renderShell(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
    );
    expect(container.querySelector(".max-w-xl")).toBeNull();
    expect(screen.getByTestId("flow-verifier-app-shell")).toHaveAttribute("data-rcl-app-shell");
  });
});
