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

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders the snapshot page at /snapshots", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/snapshots"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.snapshots)).toBeInTheDocument();
  });

  it("renders the resolver page at /resolver", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/resolver"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.resolver)).toBeInTheDocument();
  });

  it("renders the devices page at /devices", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/devices"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.devices)).toBeInTheDocument();
  });

  it("renders the optimization page at /optimization", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/optimization"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.optimization)).toBeInTheDocument();
  });
});
