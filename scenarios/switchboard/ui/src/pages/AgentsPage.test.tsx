import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeAgent, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { AgentDetailPage } from "./AgentDetailPage";
import { AgentsPage, agentReachesTerminal } from "./AgentsPage";

const renderAt = (path: string) =>
  renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/agents" element={<AgentsPage />} />
        <Route path="/agents/:agentId" element={<AgentDetailPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("AgentsPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P0-008]
  it("renders the roster with reachability and grant summary", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/agents");
    expect(await screen.findByTestId("agents-card")).toBeInTheDocument();
    expect(screen.getByTestId("agents-live")).toBeInTheDocument();
    expect(screen.getByTestId("agents-channels")).toHaveTextContent("Switchboard app");
    expect(screen.getByTestId("agents-grant")).toHaveTextContent("read");
    expect(screen.getByTestId("agents-roster-region")).toHaveAttribute("data-experience-state", "ready");
  });

  // [REQ:SWBD-P1-007]
  it("keeps a broken reference visible with its reason and marks the roster partial when the source is down", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/agents"] = { source: { ok: false, reason: "prompt-manager unreachable" }, agents: [makeAgent({ id: "ghost", display_name: "ghost", broken: "agent ghost is not known to prompt-manager", bindings: [] })] };
    stubConsoleFetch(routes);
    renderAt("/agents");
    expect(await screen.findByTestId("agents-card")).toHaveTextContent("agent ghost is not known to prompt-manager");
    expect(screen.getByTestId("agents-roster-region")).toHaveAttribute("data-experience-state", "partial");
    expect(screen.getByRole("alert")).toHaveTextContent("prompt-manager unreachable");
  });

  it("shows the empty state with a create call to action", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/agents"] = { source: { ok: true }, agents: [] };
    stubConsoleFetch(routes);
    renderAt("/agents");
    expect(await screen.findByTestId("agents-empty-cta")).toBeInTheDocument();
  });

  it("flags terminal-reaching grants", () => {
    expect(agentReachesTerminal(makeAgent({ grant: { scopes: ["read", "terminal.exec"], owner_only: [], source: "descriptor" } }))).toBe(true);
    expect(agentReachesTerminal(makeAgent())).toBe(false);
  });
});

describe("AgentDetailPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P0-011]
  it("renders the grant as facts and never as a disabled control", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt("/agents/household-planner");
    expect(await screen.findByTestId("agent-grant")).toHaveTextContent("read");
    expect(screen.getByTestId("agent-owner-only")).toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("agent-activity")).toBeInTheDocument());
    expect(screen.getByTestId("agent-profile-link")).toHaveAttribute("href", expect.stringContaining("household-planner"));
  });
});

describe("AgentsPage card variants", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("renders paused bindings, terminal-reaching grants and descriptor grants", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/agents"] = {
      source: { ok: true },
      agents: [
        makeAgent({ id: "ops", display_name: "Ops", bindings: [{ id: "b", channel_id: "telegram", channel_display_name: "Telegram", address: "chat", thread_key: "", live: false }], grant: { scopes: ["read", "terminal.exec"], owner_only: [], source: "descriptor" }, activity: undefined, description: undefined }),
      ],
    };
    stubConsoleFetch(routes);
    renderAt("/agents");
    const card = await screen.findByTestId("agents-card");
    expect(card).toHaveTextContent("terminal.exec");
    expect(screen.getByTestId("agents-live")).toBeInTheDocument();
    expect(screen.getByTestId("agents-channels")).toHaveTextContent("Telegram");
  });
});

describe("AgentDetailPage not found", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("renders the grant region as ready with a not-found notice", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/agents/ghost"] = new Response("not found", { status: 404 });
    stubConsoleFetch(routes);
    renderAt("/agents/ghost");
    await waitFor(() => expect(screen.getByTestId("agent-detail-grant-region")).toHaveAttribute("data-experience-state", "ready"));
    expect(screen.queryByTestId("agent-grant")).not.toBeInTheDocument();
  });
});
