/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { routes, TestAppRouter } from "./routes";

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

  it.each([
    ["/catalog", selectors.pages.catalog],
    ["/styles/cyanotype-arcade", selectors.pages.style],
    ["/sweep", selectors.pages.sweep],
    ["/remix", selectors.pages.remix],
    ["/compose", selectors.pages.compose],
    ["/placements", selectors.pages.placements],
    ["/candidates", selectors.pages.candidates],
    ["/backdrops", selectors.pages.backdrops],
    ["/surfaces", selectors.pages.surfaces],
  ])("renders its own page at %s", (path, testId) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(screen.getByTestId(testId)).toBeInTheDocument();
  });

  /**
   * The route table's load-bearing property, asserted directly.
   *
   * Eleven routes used to resolve to one `WorkbenchPage`, so the navigation
   * promised a studio the app did not have and no test could tell. Comparing
   * the element types makes the collapse impossible to reintroduce by reusing a
   * component for "just one more" route.
   */
  it("maps no two routes to the same component", () => {
    const children = routes[0]?.children ?? [];
    const seen = new Map<unknown, string>();
    for (const child of children) {
      const element = child.element as { type?: unknown } | undefined;
      const type = element?.type;
      if (!type) continue;
      const path = child.path ?? "(index)";
      const prior = seen.get(type);
      expect(prior, `${path} and ${prior} resolve to the same component`).toBeUndefined();
      seen.set(type, path);
    }
    expect(seen.size).toBeGreaterThanOrEqual(10);
  });
});
