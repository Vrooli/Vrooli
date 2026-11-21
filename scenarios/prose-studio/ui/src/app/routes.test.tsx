/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { AppRouter, TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders the job readiness workbench at /jobs", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const path = String(input);
      const data = path.includes("/openings") ? { openings: [] } : path.includes("/facts") ? { facts: [] } : path.includes("/resumes") ? { resumes: [] } : { cover_letters: [] };
      return new Response(JSON.stringify(data), { status: 200, headers: { "content-type": "application/json" } });
    });
    renderWithProviders(<TestAppRouter initialEntries={["/jobs"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.jobs)).toBeInTheDocument();
  });

  it.each(["/styles/style-1", "/sessions/session-1"])("renders dynamic operator surfaces at %s", (path) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(screen.getByTestId(path.startsWith("/styles") ? selectors.pages.styles : selectors.pages.variation)).toBeInTheDocument();
  });

  it("mounts the production browser router", () => {
    renderWithProviders(<AppRouter />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });
});
