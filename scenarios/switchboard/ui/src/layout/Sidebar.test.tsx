import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeGate, makeOverview, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { strings } from "../consts/strings";
import { Sidebar } from "./Sidebar";

const renderSidebar = () =>
  renderWithProviders(
    <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Sidebar />
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("Sidebar", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("shows the pending count beside the overview link", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = makeOverview({ gates: [makeGate(), makeGate({ id: "gate-2" })] });
    stubConsoleFetch(routes);
    renderSidebar();
    await waitFor(() => expect(screen.getByText("2")).toBeInTheDocument());
    expect(screen.getByText(strings.console.shell.apiConnected)).toBeInTheDocument();
  });

  it("reports an unreachable API in the footer", async () => {
    stubConsoleFetch({});
    renderSidebar();
    await waitFor(() => expect(screen.getByText(strings.console.shell.apiUnreachable)).toBeInTheDocument(), { timeout: 4000 });
  });
});
