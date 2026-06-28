/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. This file's job is to assert the router config wires every route to
 * the right page; page-internal behaviour is exercised in per-feature card
 * tests. Rendering a page mounts its card; read-only cards fire a query that
 * errors without a server, but the page `<section>` still renders, which is all
 * this smoke asserts.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

// (path, page-selector) for every route in the canonical table. Kept in sync
// with `routes.tsx`; a missing route here means an untested page wrapper.
const ROUTES: ReadonlyArray<{ path: string; selector: string; label: string }> = [
  { path: "/", selector: selectors.pages.dashboard, label: "dashboard at /" },
  { path: "/brands", selector: selectors.pages.brands, label: "brands page at /brands" },
  { path: "/assignments", selector: selectors.pages.assignments, label: "assignments page at /assignments" },
  { path: "/assets", selector: selectors.pages.assets, label: "assets page at /assets" },
  { path: "/generation", selector: selectors.pages.generation, label: "generation page at /generation" },
  { path: "/apply", selector: selectors.pages.apply, label: "apply page at /apply" },
  { path: "/discovery", selector: selectors.pages.discovery, label: "discovery page at /discovery" },
  { path: "/design", selector: selectors.pages.design, label: "design page at /design" },
  { path: "/settings", selector: selectors.pages.settings, label: "settings page at /settings" },
];

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it.each(ROUTES)("renders the $label", ({ path, selector }) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(screen.getByTestId(selector)).toBeInTheDocument();
  });
});
