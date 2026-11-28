/**
 * Routing smoke — for each canonical path the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the overview page at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.overview)).toBeInTheDocument();
  });

  it("renders the inventory page at /inventory", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/inventory"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.inventory)).toBeInTheDocument();
  });

  it("renders the search page at /search", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/search"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.search)).toBeInTheDocument();
  });

  it("renders the runs page at /runs", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/runs"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.runs)).toBeInTheDocument();
  });

  it("renders the findings page at /findings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/findings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.findings)).toBeInTheDocument();
  });

  it("renders the fixes page at /fixes", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/fixes"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.fixes)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
