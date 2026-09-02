import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { TestAppRouter } from "../app/routes";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeGate, makeOverview, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { isNavItemActive, NAV_ITEMS } from "./navItems";

describe("shell navigation", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("navigates from the bottom nav and shows the pending badge on the overview item", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = makeOverview({ gates: [makeGate()] });
    stubConsoleFetch(routes);
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId("topbar-attention")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId(selectors.layout.bottomNavLink({ key: "agents" })));
    await waitFor(() => expect(screen.getByTestId("agents-roster-region")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("topbar-settings"));
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
