/**
 * Routing smoke — for each canonical path the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

vi.mock("../api/gateway", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/gateway")>();
  const { makeGatewayApiMocks } = await import("../test-utils/mocks/gateway");
  return { ...actual, ...makeGatewayApiMocks() };
});

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

  it("renders provider inventory at /providers", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/providers"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.providers)).toBeInTheDocument();
  });

  it("renders route preview at /routes/preview", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/routes/preview"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.routePreview)).toBeInTheDocument();
  });

  it("renders conformance at /conformance", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/conformance"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.conformance)).toBeInTheDocument();
  });
});
