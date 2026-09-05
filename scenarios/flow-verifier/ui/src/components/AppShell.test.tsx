import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

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

describe("AppShell", () => {
  beforeEach(async () => {
    const { fetchFlows } = await import("../api/inventory");
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchFlows).mockResolvedValue([]);
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders shell, sidebar, mobile header/nav, and the child route content", async () => {
    renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div data-testid="child">hello</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    expect(screen.getByTestId("app-shell")).toBeInTheDocument();
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-header")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();
    expect(screen.getByTestId("child")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId("health-pill")).toBeInTheDocument(),
    );
  });

  it("does not wrap content in a centered card or eyebrow text", () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route element={<AppShell />}>
          <Route path="/" element={<div>page</div>} />
        </Route>
      </Routes>,
      { routerEntries: ["/"] },
    );
    expect(container.querySelector(".max-w-xl")).toBeNull();
    const shell = screen.getByTestId("app-shell");
    expect(shell.className).toContain("w-full");
  });
});
