/**
 * Routing smoke — for each canonical path (`/`, `/notes`, `/settings`) the
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

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the focus page at /focus", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/focus"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.focus)).toBeInTheDocument();
  });

  it("renders the convergence page at /convergence", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/convergence"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.convergence)).toBeInTheDocument();
  });

  it("renders the trials page at /trials", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/trials"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.trials)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
