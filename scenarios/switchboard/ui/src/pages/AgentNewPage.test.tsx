import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { bodyOf, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { AgentNewPage } from "./AgentNewPage";

const renderPage = () =>
  renderWithProviders(
    <MemoryRouter initialEntries={["/agents/new"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/agents/new" element={<AgentNewPage />} />
        <Route path="/agents/:agentId" element={<p data-testid="landed" />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("AgentNewPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P1-007]
  it("drafts from a description, then confirms and writes", async () => {
    const { calls } = stubConsoleFetch({
      "/api/v1/agents/draft": { display_name: "Concierge", description: "A concierge", scopes: ["read"], owner_only_scopes: [] },
      "/api/v1/agents": { id: "concierge", display_name: "Concierge" },
    });
    renderPage();
    expect(screen.getByTestId("agent-new-prepare")).toBeDisabled();
    fireEvent.change(screen.getByTestId("agent-new-description"), { target: { value: "A concierge" } });
    fireEvent.click(screen.getByTestId("agent-new-prepare"));
    expect(await screen.findByTestId("agent-new-name")).toHaveValue("Concierge");
    fireEvent.change(screen.getByTestId("agent-new-name"), { target: { value: "Front desk" } });
    fireEvent.click(screen.getByTestId("agent-new-confirm"));
    await waitFor(() => expect(screen.getByTestId("landed")).toBeInTheDocument());
    const write = calls.find((call) => call.path === "/api/v1/agents");
    expect(bodyOf(write?.init)).toMatchObject({ display_name: "Front desk", scopes: ["read"] });
  });

  it("explains when the profile source is not writable", async () => {
    stubConsoleFetch({
      "/api/v1/agents/draft": { display_name: "Concierge", description: "A concierge", scopes: ["read"], owner_only_scopes: [] },
      "/api/v1/agents": new Response("no create endpoint", { status: 501 }),
    });
    renderPage();
    fireEvent.change(screen.getByTestId("agent-new-description"), { target: { value: "A concierge" } });
    fireEvent.click(screen.getByTestId("agent-new-prepare"));
    fireEvent.click(await screen.findByTestId("agent-new-confirm"));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });
});
