import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { bodyOf, defaultRoutes, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { ChannelsPage } from "./ChannelsPage";

const renderAt = (path = "/channels") =>
  renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/channels" element={<ChannelsPage />} />
        <Route path="/channels/:channelId" element={<ChannelsPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );

describe("ChannelsPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P1-012]
  it("shows available channels first and hides ones that need setup behind a reveal", async () => {
    stubConsoleFetch(defaultRoutes());
    renderAt();
    expect(await screen.findByTestId("channels-row")).toHaveAttribute("data-channel-id", "in-app");
    expect(screen.getAllByTestId("channels-row")).toHaveLength(1);
    fireEvent.click(screen.getByTestId("channels-show-all"));
    expect(screen.getAllByTestId("channels-row")).toHaveLength(2);
    const telegram = screen.getAllByTestId("channels-row")[1];
    expect(telegram).toHaveTextContent("configure a Telegram bot token");
    expect(screen.getByTestId("channels-requirement")).toBeInTheDocument();
    expect(screen.getByTestId("channels-catalog-region")).toHaveAttribute("data-experience-state", "partial");
  });

  // [REQ:SWBD-P0-001]
  it("turns the attach action into a binding request", async () => {
    const routes = defaultRoutes();
    routes["/vrooli.switchboard.v1.channels.ChannelService/CreateBinding"] = { binding: { id: "b2" } };
    const { calls } = stubConsoleFetch(routes);
    renderAt();
    fireEvent.click(await screen.findByTestId("channels-attach"));
    const agentOptions = await screen.findAllByRole("radio");
    fireEvent.click(agentOptions[0] as HTMLElement);
    fireEvent.change(screen.getByTestId("channels-binding-address"), { target: { value: "owner" } });
    fireEvent.click(screen.getByTestId("channels-attach-confirm"));
    await waitFor(() => expect(calls.some((call) => call.path.endsWith("/CreateBinding"))).toBe(true));
    const binding = calls.find((call) => call.path.endsWith("/CreateBinding"));
    expect(bodyOf(binding?.init)).toMatchObject({ agentId: "household-planner", channelId: "in-app", address: "owner" });
    expect(await screen.findByTestId("channels-attached")).toBeInTheDocument();
  });
});
