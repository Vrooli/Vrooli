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
    getGraphSnapshot: vi.fn().mockResolvedValue({ snapshot: undefined }),
  },
}));

vi.mock("../api/domains", () => ({
  domainsClient: {
    getDomainMap: vi.fn().mockResolvedValue({ domainMap: undefined }),
    extractDomains: vi.fn().mockResolvedValue({ domainMap: undefined }),
    convergenceReport: vi.fn().mockResolvedValue({ scenario: "", authority: 0, findings: [] }),
  },
}));

vi.mock("../api/apply", () => ({
  applyClient: {
    getBuildBaseline: vi.fn().mockResolvedValue({ baseline: undefined }),
    listApplyHistory: vi.fn().mockResolvedValue({ runs: [], nextPageToken: "" }),
    planApply: vi.fn().mockResolvedValue({ plan: undefined, dryRun: false }),
    runApply: vi.fn(),
  },
}));

vi.mock("../api/analytics", () => ({
  analyticsClient: {
    getStats: vi.fn().mockResolvedValue({ stats: undefined }),
    listEvents: vi.fn().mockResolvedValue({ events: [], nextPageToken: "" }),
    listPlacements: vi.fn().mockResolvedValue({ placements: [], nextPageToken: "" }),
    recordOverride: vi.fn(),
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

// Stub the conflicts Connect client — the conflicts page calls ListConflicts
// on mount and the workbench would otherwise need a live backend.
vi.mock("../api/conflicts", () => ({
  conflictsClient: {
    listConflicts: vi.fn().mockResolvedValue({ conflicts: [], nextPageToken: "" }),
    getConflict: vi.fn().mockResolvedValue({ conflict: undefined }),
    detectConflicts: vi.fn().mockResolvedValue({ conflicts: [] }),
    validateConflicts: vi.fn().mockResolvedValue({ conflicts: [], clean: true }),
    assignConflict: vi.fn(),
    resolveConflict: vi.fn(),
    reopenConflict: vi.fn(),
  },
}));

// Stub the migration Connect client — the migration page calls
// ListMigrations on mount and the workbench would otherwise need a backend.
vi.mock("../api/migration", () => ({
  migrationClient: {
    listMigrations: vi.fn().mockResolvedValue({ migrations: [] }),
    getMigrationStatus: vi.fn().mockResolvedValue({ status: undefined }),
    nextMigrationStep: vi.fn().mockResolvedValue({ findings: [] }),
    createMigration: vi.fn(),
    resolveFinding: vi.fn(),
    applyFinding: vi.fn(),
    reauditMigration: vi.fn(),
    closeMigration: vi.fn(),
  },
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

  it("renders the graph page at /targets/:encodedPath/graph", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/graph"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetGraph)).toBeInTheDocument();
  });

  it("renders the conflicts page at /targets/:encodedPath/conflicts", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/conflicts"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetConflicts)).toBeInTheDocument();
  });

  it("renders the conflict detail at /targets/:encodedPath/conflicts/:conflictId", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/conflicts/c-1"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetConflictDetail)).toBeInTheDocument();
  });

  it("renders the migration page at /targets/:encodedPath/migration", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/migration"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetMigration)).toBeInTheDocument();
  });

  it("renders the migration detail at /targets/:encodedPath/migration/:migrationId", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/migration/m-1"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetMigrationDetail)).toBeInTheDocument();
  });

  it("renders the domains page at /targets/:encodedPath/domains", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/domains"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetDomains)).toBeInTheDocument();
  });

  it("renders the apply page at /targets/:encodedPath/apply", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/apply"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetApply)).toBeInTheDocument();
  });

  it("renders the per-domain apply page at /targets/:encodedPath/apply/:domainKey", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/apply/foo"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetApplyDomain)).toBeInTheDocument();
  });

  it("renders the analytics page at /targets/:encodedPath/analytics", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/targets/demo/analytics"]} />,
      { withoutRouter: true },
    );
    expect(await screen.findByTestId(selectors.pages.targetAnalytics)).toBeInTheDocument();
  });

  it("renders the cross-target history page at /history", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/history"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.history)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
