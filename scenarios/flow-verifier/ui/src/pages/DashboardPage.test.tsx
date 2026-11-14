import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders, makeHealthResponse } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return { ...actual, fetchRuns: vi.fn() };
});
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { DashboardPage } from "./DashboardPage";

describe("DashboardPage", () => {
  beforeEach(async () => {
    const { fetchRuns } = await import("../api/inventory");
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchRuns).mockReset();
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders header, CTA, and timeline composition", async () => {
    const { fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([]);
    renderWithProviders(<DashboardPage />);
    expect(screen.getByTestId("dashboard-page")).toBeInTheDocument();
    expect(screen.getByTestId("dashboard-cta-verify")).toHaveAttribute("href", "/flows");
    await waitFor(() =>
      expect(screen.getByTestId("dashboard-recent-empty")).toBeInTheDocument(),
    );
  });

  it("renders recent run links when runs are returned", async () => {
    const { fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([
      {
        id: "run-1",
        flowId: "alpha.flow",
        flowPath: "a/flow.json",
        root: ".",
        mode: "check",
        status: "passed",
        startedAt: "2026-05-12T00:00:00Z",
        finishedAt: "2026-05-12T00:00:01Z",
        durationMs: 1000,
      },
    ]);
    renderWithProviders(<DashboardPage />);
    await waitFor(() =>
      expect(screen.getByTestId("dashboard-recent-run-1")).toHaveAttribute(
        "href",
        "/runs/run-1",
      ),
    );
  });
});
