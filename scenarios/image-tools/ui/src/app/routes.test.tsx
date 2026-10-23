/**
 * Routing smoke — for each canonical path (`/`, `/workspace`, `/library`,
 * `/activity`, `/models`, `/settings`) the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this file's
 * job is to assert the router config.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { makeListJobsResponse } from "../features/jobs/mocks/factories";

// Hoisted so the mock factory (lifted above the imports) can read it without a
// temporal-dead-zone error.
const mocks = vi.hoisted(async () => {
  const { makeHealthResponse: make } = await import("../test-utils/factories");
  return { health: make() };
});

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn().mockResolvedValue((await mocks).health) };
});

vi.mock("../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/jobs")>();
  return {
    ...actual,
    jobsClient: { listJobs: vi.fn(), cancelJob: vi.fn(), watchJob: vi.fn() },
  };
});

import { AppRouter, TestAppRouter } from "./routes";
import { jobsClient } from "../api/jobs";

describe("AppRouter", () => {
  beforeEach(() => {
    // Home/Library/Activity pages fire the jobs query on mount; give it a
    // resolved empty snapshot so its settling doesn't emit a console error.
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the home page at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.home)).toBeInTheDocument();
  });

  it("renders the workspace page at /workspace", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/workspace"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.workspace)).toBeInTheDocument();
  });

  it("renders the library page at /library", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/library"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.library)).toBeInTheDocument();
  });

  it("renders the select page at /select", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/select"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.select)).toBeInTheDocument();
  });

  it("renders the compare page at /compare", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/compare"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.compare)).toBeInTheDocument();
  });

  it("renders the activity page at /activity", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/activity"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.activity)).toBeInTheDocument();
  });

  it("renders the models page at /models", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/models"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.models)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});

describe("AppRouter (production browser router)", () => {
  beforeEach(() => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("builds a real createBrowserRouter and renders Home at jsdom's default '/'", async () => {
    // Exercise real browser history while the canonical helper supplies only
    // the surrounding providers. It must not insert a second router.
    renderWithProviders(<AppRouter />, { withoutRouter: true, initialTheme: "light" });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.pages.home)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
  });
});
