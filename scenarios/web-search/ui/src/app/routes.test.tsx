/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page / per-feature
 * tests; this file's job is to assert the router config. The domain pages mount
 * Connect clients, so the clients module is mocked to keep this a pure routing
 * check.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

vi.mock("../api/clients", () => ({
  findingsClient: { listFindings: vi.fn().mockResolvedValue({ findings: [] }) },
  liveSearchClient: { search: vi.fn() },
}));

import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders the search page at /search", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/search"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.search)).toBeInTheDocument();
  });

  it("renders the findings page at /findings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/findings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.findings)).toBeInTheDocument();
  });

  it("renders the ops page at /ops", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/ops"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.ops)).toBeInTheDocument();
  });
});
