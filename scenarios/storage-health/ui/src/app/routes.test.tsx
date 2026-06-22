/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this file's
 * job is to assert the router config.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { makeScanFleetResponse } from "../features/storage/mocks/factories";

// The data-backed pages fire RPCs on mount; stub the whole client so the router
// smoke never touches a live transport.
vi.mock("../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/storage")>();
  return {
    ...actual,
    storageClient: {
      ...actual.storageClient,
      getInventory: vi.fn().mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 })),
      scanFleet: vi.fn().mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 })),
      adviseEngines: vi.fn().mockResolvedValue({ candidates: [], scenarioCount: 0, errors: [] }),
      analyzeMigrations: vi
        .fn()
        .mockResolvedValue({ entries: [], scenarioCount: 0, withMigrationsCount: 0, debtCount: 0, errors: [] }),
    },
  };
});

import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the fleet page at /fleet", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/fleet"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.fleet)).toBeInTheDocument();
  });

  it("renders the validate page at /validate", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/validate"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.validate)).toBeInTheDocument();
  });

  it("renders the advisor page at /advisor", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/advisor"]} />, { withoutRouter: true });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.pages.advisor)).toBeInTheDocument();
    });
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
