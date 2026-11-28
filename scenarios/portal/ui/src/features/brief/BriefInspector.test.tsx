import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { BriefInspector, BriefMessage } from "./BriefInspector";

const api = vi.hoisted(() => ({
  listPortalBriefs: vi.fn(),
  getPortalBrief: vi.fn(),
  recordPortalBriefUse: vi.fn().mockResolvedValue(true),
}));

vi.mock("../../api/brief", () => ({
  BriefConsumer: { UNSPECIFIED: 0, PORTAL_LLM: 1, PORTAL_AGENT: 2, EXTERNAL_HARNESS: 3 },
  BriefUseKind: { OPENED: 1, COPIED: 2, REJECTED: 4 },
  ...api,
}));

const delivered = {
  id: "brief-1",
  consumer: 3,
  verdict: 1,
  reason: "top item cleared threshold",
  latencyMs: 42n,
  maxTrustClass: 1,
  effectiveQuery: "portal brief",
  queriedProviders: ["cli-health.commands", "web-search.live"],
  items: [{ providerId: "cli-health.commands", title: "Portal brief build", snippet: "A safe suggestion", path: "portal brief build", trustClass: 1, suggestedCommand: "portal brief build test" }],
  createdAt: "2026-09-07T12:00:00Z",
};

describe("BriefInspector", () => {
  it("shows delivered evidence and keeps suggested commands copy-only", async () => {
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, "writeText");
    api.listPortalBriefs.mockResolvedValue([delivered]);
    api.getPortalBrief.mockResolvedValue(delivered);
    renderWithProviders(<BriefInspector chatId="chat-1" />);

    await waitFor(() => expect(screen.getByText("DELIVER · EXTERNAL HARNESS · 2026-09-07T12:00:00Z")).toBeInTheDocument());
    expect(api.listPortalBriefs).toHaveBeenCalledWith({ chatId: "chat-1", consumer: 0, limit: 20 });
    await user.selectOptions(screen.getByRole("combobox", { name: "Select context brief" }), "brief-1");
    await waitFor(() => expect(api.recordPortalBriefUse).toHaveBeenCalledWith("brief-1", -1, 1));
    expect(screen.getByText((value) => value.includes("Searched: cli-health.commands, web-search.live") && value.includes("never executes them"))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy command" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open portal brief build" }));
    expect(api.recordPortalBriefUse).toHaveBeenCalledWith("brief-1", 0, 1);
    await user.click(screen.getByRole("button", { name: "Copy command" }));
    expect(writeText).toHaveBeenCalledWith("portal brief build test");
    await user.click(screen.getByRole("button", { name: "Reject" }));
    expect(api.recordPortalBriefUse).toHaveBeenCalledWith("brief-1", 0, 4);
  });

  it("shows withheld reason and no rendered evidence", async () => {
    const user = userEvent.setup();
    const withheld = { ...delivered, id: "brief-2", verdict: 2, reason: "best item effective score 0.100000 is below threshold", items: [], rendered: "" };
    api.listPortalBriefs.mockResolvedValue([withheld]);
    api.getPortalBrief.mockResolvedValue(withheld);
    renderWithProviders(<BriefInspector chatId="chat-2" />);
    await waitFor(() => expect(screen.getByText("WITHHELD · EXTERNAL HARNESS · 2026-09-07T12:00:00Z")).toBeInTheDocument());
    await user.selectOptions(screen.getByRole("combobox", { name: "Select context brief" }), "brief-2");
    await waitFor(() => expect(screen.getByText("best item effective score 0.100000 is below threshold")).toBeInTheDocument());
    expect(screen.getByText("best item effective score 0.100000 is below threshold")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Copy command" })).not.toBeInTheDocument();
  });

  it("renders an unavailable state when listing fails", async () => {
    api.listPortalBriefs.mockRejectedValueOnce(new Error("brief service offline"));
    renderWithProviders(<BriefInspector chatId="chat-error" />);
    await waitFor(() => expect(screen.getByText("brief service offline")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Refresh" })).toBeInTheDocument();
  });

  it("renders a brief on the matching chat message and records opened items", async () => {
    const user = userEvent.setup();
    api.getPortalBrief.mockResolvedValueOnce(delivered);
    renderWithProviders(<BriefMessage briefId="brief-1" />);
    await waitFor(() => expect(screen.getByRole("region", { name: "Message context brief" })).toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "Open portal brief build" }));
    expect(api.recordPortalBriefUse).toHaveBeenCalledWith("brief-1", 0, 1);
  });

  it("shows a message-level unavailable state", async () => {
    api.getPortalBrief.mockRejectedValueOnce(new Error("brief lookup failed"));
    renderWithProviders(<BriefMessage briefId="brief-missing" />);
    await waitFor(() => expect(screen.getByText("Context brief unavailable: brief lookup failed")).toBeInTheDocument());
  });

  it("shows a selection error without losing the inspector", async () => {
    const user = userEvent.setup();
    api.listPortalBriefs.mockResolvedValueOnce([delivered]);
    api.getPortalBrief.mockRejectedValueOnce(new Error("brief detail failed"));
    renderWithProviders(<BriefInspector chatId="chat-selection-error" />);
    await waitFor(() => expect(screen.getByRole("combobox", { name: "Select context brief" })).toBeInTheDocument());
    await user.selectOptions(screen.getByRole("combobox", { name: "Select context brief" }), "brief-1");
    await waitFor(() => expect(screen.getByText("brief detail failed")).toBeInTheDocument());
  });

  it("renders degraded message briefs and item fallbacks", async () => {
    const degraded = { ...delivered, verdict: 2, reason: "", degraded: true, items: [{ providerId: "unknown", type: "", title: "", snippet: "", path: "", trustClass: 99, suggestedCommand: "" }] };
    api.getPortalBrief.mockResolvedValueOnce(degraded);
    renderWithProviders(<BriefMessage briefId="brief-degraded" />);
    await waitFor(() => expect(screen.getByText(/degraded · 42 ms/)).toBeInTheDocument());
    expect(screen.getByText(/unknown/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open item" })).not.toBeInTheDocument();
  });

  it("filters the list by consumer", async () => {
    const user = userEvent.setup();
    api.listPortalBriefs.mockResolvedValue([delivered]);
    renderWithProviders(<BriefInspector chatId="chat-filter" />);
    await waitFor(() => expect(screen.getByRole("combobox", { name: "Filter context briefs" })).toBeInTheDocument());
    await user.selectOptions(screen.getByRole("combobox", { name: "Filter context briefs" }), "2");
    await waitFor(() => expect(api.listPortalBriefs).toHaveBeenLastCalledWith({ chatId: "chat-filter", consumer: 2, limit: 20 }));
  });
});
