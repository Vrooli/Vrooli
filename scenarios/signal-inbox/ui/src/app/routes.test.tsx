/**
 * Routing smoke — for each canonical path (`/`, `/signals`, `/settings`) the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";
import { Providers } from "./providers";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the signals page at /signals", () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/signals"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.signals)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<Providers><TestAppRouter initialEntries={["/settings"]} /></Providers>, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
