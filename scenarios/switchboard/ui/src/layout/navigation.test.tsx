import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeGate, makeOverview, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { isNavItemActive, NAV_ITEMS } from "./navItems";
import { SessionProvider } from "../features/session/SessionProvider";

describe("shell navigation", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("keeps pending decisions and sign-in reachable after the chrome migration", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = makeOverview({ gates: [makeGate()] });
    stubConsoleFetch(routes);
    renderWithProviders(<SessionProvider><TestAppRouter initialEntries={["/conversations"]} /></SessionProvider>, { withoutRouter: true });
    const attention = await screen.findByTestId("conversations-attention");
    expect(attention).toHaveAttribute("href", "/");
    fireEvent.click(screen.getByTestId("shell-session"));
    expect(await screen.findByTestId("session-sign-in")).toBeInTheDocument();
  });

  it("navigates from the bottom nav and shows the pending badge on the overview item", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = makeOverview({ gates: [makeGate()] });
    stubConsoleFetch(routes);
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId(selectors.layout.navLink({ key: "dashboard" }))).toHaveTextContent("1"));
    fireEvent.click(screen.getByTestId(`${selectors.layout.navLink({ key: "agents" })}-tab`));
    await waitFor(() => expect(screen.getByTestId("agents-roster-region")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("shell-settings"));
    await waitFor(() => expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument());
  });

  it("marks the index route exact and nested routes by prefix", () => {
    const dashboard = NAV_ITEMS.find((item) => item.key === "dashboard");
    const agents = NAV_ITEMS.find((item) => item.key === "agents");
    expect(dashboard && isNavItemActive(dashboard, "/")).toBe(true);
    expect(dashboard && isNavItemActive(dashboard, "/agents")).toBe(false);
    expect(agents && isNavItemActive(agents, "/agents/x")).toBe(true);
    expect(agents && isNavItemActive(agents, "/agentsx")).toBe(false);
  });
});
