/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";

import { en } from "../i18n/catalog";
import { setLocale } from "../i18n";
import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { routes } from "./routeTable";
import { TestAppRouter } from "./routes";

vi.mock("../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [], totalAwaiting: 0 })) },
  notificationsClient: { get: vi.fn(() => Promise.resolve({ notification: { id: "record-1" }, receipts: [] })) },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));
// The settings screens have their own route tests; here only the mount point matters.
vi.mock("../features/settings/routes", () => ({
  settingsRoute: {
    path: "settings",
    children: [
      { index: true, element: <div data-testid="page-settings" /> },
      { path: "reach", element: <div data-testid="page-reach" /> },
    ],
  },
}));
vi.mock("../pages/DashboardPage", () => ({ DashboardPage: () => <div data-testid="page-dashboard" /> }));

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("opens the read-only Updates feed at /", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("says so inside the shell when an address is not served, with one way back", async () => {
    await setLocale("en");
    renderWithProviders(<TestAppRouter initialEntries={["/nowhere"]} />, { withoutRouter: true });
    const page = await screen.findByTestId(selectors.pages.notFound);
    expect(within(page).getByRole("heading", { level: 1, name: en.pages.notFound.title })).toBeInTheDocument();
    expect(within(page).getByRole("link", { name: en.pages.notFound.home })).toHaveAttribute("href", "/updates");
    expect(screen.getByTestId(selectors.layout.shell)).toBeInTheDocument();
  });

  it("keeps Updates at its own destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/updates"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders Reach as a nested Settings destination", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings/reach"]} />, { withoutRouter: true });
    expect(await screen.findByTestId("page-reach")).toBeInTheDocument();
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
