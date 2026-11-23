/**
 * Routing smoke — for each canonical path the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { resolveRouterBasename, TestAppRouter } from "./routes";

const proxyBase = "/apps/ai-gateway/proxy";
const appMonitorProxyInfo = {
  appId: "ai-gateway",
  targetScenario: "ai-gateway",
  primary: {
    port: 20000,
    label: "UI_PORT",
    normalizedLabel: "ui_port",
    slug: "ui-port",
    source: "port_mappings",
    isPrimary: true,
    path: proxyBase,
    aliases: [],
  },
};

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

  it("keeps the direct owner root at the root basename", () => {
    expect(resolveRouterBasename("/", null)).toBe("/");
  });

  it("uses the matching injected proxy base for a nested deep link", () => {
    const basename = resolveRouterBasename(`${proxyBase}/settings`, appMonitorProxyInfo);
    expect(basename).toBe(proxyBase);

    renderWithProviders(
      <TestAppRouter initialEntries={[`${proxyBase}/settings`]} basename={basename} />,
      { withoutRouter: true },
    );
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("rejects proxy metadata for another app or an unrelated browser path", () => {
    expect(resolveRouterBasename(`${proxyBase}/`, {
      ...appMonitorProxyInfo,
      targetScenario: "another-app",
    })).toBe("/");
    expect(resolveRouterBasename("/settings", appMonitorProxyInfo)).toBe("/");
  });
});
