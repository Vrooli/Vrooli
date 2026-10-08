/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { Providers } from "./providers";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the dashboard at /", async () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
    await screen.findByText(/Readiness is unavailable|No governed scenarios were returned/);
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/settings"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
