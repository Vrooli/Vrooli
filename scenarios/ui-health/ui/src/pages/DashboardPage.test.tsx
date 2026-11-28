import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";

import { renderWithProviders, makeHealthResponse } from "../test-utils";
import { selectors } from "../consts/selectors";

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return {
    ...actual,
    fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()),
  };
});

vi.mock("../api/search", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/search")>();
  return {
    ...actual,
    searchStatus: vi.fn().mockResolvedValue({
      available: true,
      ollama: true,
      qdrant: true,
      indexedCount: 42,
      lastReconcileAt: "2026-05-20T10:00:00.000Z",
      lastReconcileOutcome: "succeeded",
    }),
  };
});

import { DashboardPage } from "./DashboardPage";
import { fetchHealth } from "../api/health";

beforeEach(() => {
  window.localStorage.clear();
  vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
});

describe("DashboardPage", () => {
  it("renders the three stat cards", async () => {
    renderWithProviders(<DashboardPage />);
    expect(screen.getByTestId(selectors.dashboard.stats.scenariosValidated)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dashboard.stats.surfacesIndexed)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dashboard.stats.openIssues)).toBeInTheDocument();
    await waitFor(() => {
      const card = screen.getByTestId(selectors.dashboard.stats.surfacesIndexed);
      expect(card.textContent).toContain("42");
    });
  });

  it("renders quick actions for every primary destination", () => {
    renderWithProviders(<DashboardPage />);
    expect(screen.getByTestId(selectors.dashboard.quickActions.search)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dashboard.quickActions.validate)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dashboard.quickActions.reindex)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dashboard.quickActions.inventory)).toBeInTheDocument();
  });

  it("renders the API-status card with service + readiness", async () => {
    renderWithProviders(<DashboardPage />);
    const card = await screen.findByTestId(selectors.dashboard.apiStatus.card);
    await waitFor(() => expect(card.textContent).toMatch(/react-vite-test/));
  });

  it("shows the empty-activity state when localStorage has no runs or jobs", async () => {
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByTestId(selectors.dashboard.activity.empty)).toBeInTheDocument();
  });

  it("renders the API-status error envelope when fetchHealth rejects", async () => {
    vi.mocked(fetchHealth).mockRejectedValueOnce(new Error("nope"));
    renderWithProviders(<DashboardPage />);
    const card = await screen.findByTestId(selectors.dashboard.apiStatus.card);
    await waitFor(() => expect(card.querySelector('[role="alert"]')).not.toBeNull());
  });

  it("renders dependency rows when the health response carries them", async () => {
    vi.mocked(fetchHealth).mockResolvedValueOnce(
      makeHealthResponse({
        dependencies: {
          postgres: { connected: true, latencyMs: 12, error: "" },
          ollama: { connected: false, latencyMs: 0, error: "down" },
        },
      }),
    );
    renderWithProviders(<DashboardPage />);
    await waitFor(() => {
      const rows = screen.getAllByTestId(selectors.dashboard.apiStatus.dependency);
      expect(rows).toHaveLength(2);
    });
  });

  it("renders merged activity rows when runs and jobs exist in storage", async () => {
    window.localStorage.setItem(
      "ui-health.validation.recent.v1",
      JSON.stringify([
        {
          scenario: "ui-health",
          passed: false,
          errors: 2,
          warnings: 1,
          infos: 0,
          ranAt: "2026-05-20T09:00:00.000Z",
        },
      ]),
    );
    window.localStorage.setItem(
      "ui-health.reindex.tracked.v1",
      JSON.stringify([
        {
          jobId: "job-1",
          scenario: "ui-health",
          dryRun: false,
          triggeredAt: "2026-05-20T10:00:00.000Z",
          plannedUpserts: 3,
          plannedDeletes: 0,
        },
      ]),
    );
    renderWithProviders(<DashboardPage />);
    const list = await screen.findByTestId(selectors.dashboard.activity.list);
    expect(within(list).getAllByRole("link")).toHaveLength(2);
    // openIssues stat reflects the run's 2 errors.
    expect(screen.getByTestId(selectors.dashboard.stats.openIssues).textContent).toContain("2");
  });
});
