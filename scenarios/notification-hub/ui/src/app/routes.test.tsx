/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { routes, TestAppRouter } from "./routes";

vi.mock("../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [] })) },
  notificationsClient: { get: vi.fn(() => Promise.resolve({ notification: { id: "record-1" }, receipts: [] })) },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));
vi.mock("../pages/SettingsPage", () => ({ SettingsPage: () => <div data-testid="page-settings" /> }));
vi.mock("../pages/DashboardPage", () => ({ DashboardPage: () => <div data-testid="page-dashboard" /> }));

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("opens the authenticated Needs you queue at /", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.asks)).toBeInTheDocument();
  });

  it("keeps Updates at its own destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/updates"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("preserves both ask route contracts", () => {
    const children = routes[0]?.children ?? [];
    expect(children.some((route) => route.path === "asks")).toBe(true);
    expect(children.some((route) => route.path === "asks/:askId")).toBe(true);
  });

  it("renders history and delivery on their canonical routes", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/history"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.history)).toBeInTheDocument();
    cleanup();
    renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.delivery)).toBeInTheDocument();
    cleanup();
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/record-1"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.deliveryDetail)).toBeInTheDocument();
  });
});
