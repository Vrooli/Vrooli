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

  it("renders the fleet page at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.fleet)).toBeInTheDocument();
  });

  it("renders the scenario explorer page", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scenarios/experience-manager"]} />, {
      withoutRouter: true,
    });
    expect(screen.getByTestId(selectors.pages.explorer)).toBeInTheDocument();
  });

  it("renders the evidence page", () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/scenarios/experience-manager/pages/fleet/evidence"]} />,
      { withoutRouter: true },
    );
    expect(screen.getByTestId(selectors.pages.evidence)).toBeInTheDocument();
  });

  it("renders the studio page", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/studio"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.studio)).toBeInTheDocument();
  });

  it("renders the findings page", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/findings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.findings)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
