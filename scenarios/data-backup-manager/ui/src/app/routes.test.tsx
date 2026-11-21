/**
 * Routing smoke — for each canonical path (`/`, `/settings`) the matching page
 * selector is in the document. Page-internal behaviour is exercised in per-page
 * tests; this file's job is to assert the router config.
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

  it("renders the overview at /", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.overview)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it.each([
    ["/targets", selectors.pages.targets],
    ["/destinations", selectors.pages.destinations],
    ["/plans", selectors.pages.plans],
    ["/runs", selectors.pages.runs],
    ["/restores", selectors.pages.restores],
  ] as const)("renders the lazy route at %s", async (path, selector) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selector)).toBeInTheDocument();
  });
});
