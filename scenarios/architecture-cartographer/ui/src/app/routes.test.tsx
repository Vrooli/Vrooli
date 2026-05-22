/**
 * Routing smoke — for each canonical path the matching page selector is in
 * the document. Page-internal behaviour is exercised in per-page tests;
 * this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

// Stub the snapshot list — the overview page calls ListGraphSnapshots on
// mount; without a stub the cartographer API would need to be running
// during unit tests. We only assert that the page rendered.
vi.mock("../api/graph", () => ({
  graphClient: {
    listGraphSnapshots: vi.fn().mockResolvedValue({ snapshots: [], nextPageToken: "" }),
    extractGraph: vi.fn().mockResolvedValue({ snapshot: undefined, fromCache: false }),
  },
}));

// Stub the health REST probe so the overview's HealthCard doesn't hit the
// network during unit tests.
vi.mock("../api/health", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "architecture-cartographer-api",
    timestamp: new Date(0).toISOString(),
  }),
}));

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the overview at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.overview)).toBeInTheDocument();
  });

  it("renders the new-target page at /targets/new", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/targets/new"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.newTarget)).toBeInTheDocument();
  });

  it("renders the target workspace at /targets/:encodedPath", () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/architecture-cartographer"]} />,
      { withoutRouter: true },
    );
    expect(screen.getByTestId(selectors.pages.targetWorkspace)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
