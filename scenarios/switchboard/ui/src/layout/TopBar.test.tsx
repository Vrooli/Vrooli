import { fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeGate, makeOverview, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { SessionProvider } from "../features/session/SessionProvider";
import { TopBar } from "./TopBar";

describe("TopBar", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("shows the attention pill when a gate is pending and opens sign-in on demand", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = makeOverview({ gates: [makeGate()] });
    stubConsoleFetch(routes);
    renderWithProviders(
      <SessionProvider>
        <TopBar />
      </SessionProvider>,
    );
    expect(await screen.findByTestId("topbar-attention")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("topbar-session"));
    expect(await screen.findByTestId("session-sign-in")).toBeInTheDocument();
  });
});
