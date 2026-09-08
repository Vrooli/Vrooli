import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { BriefInspector } from "./BriefInspector";

const api = vi.hoisted(() => ({
  listPortalBriefs: vi.fn(),
  getPortalBrief: vi.fn(),
  recordPortalBriefUse: vi.fn().mockResolvedValue(true),
}));

vi.mock("../../api/brief", () => ({
  BriefConsumer: { UNSPECIFIED: 0 },
  BriefUseKind: { OPENED: 1, COPIED: 2 },
  ...api,
}));

const delivered = {
  id: "brief-1",
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

    await waitFor(() => expect(screen.getByText("DELIVER · 2026-09-07T12:00:00Z")).toBeInTheDocument());
    await user.selectOptions(screen.getByRole("combobox", { name: "Select context brief" }), "brief-1");
    await waitFor(() => expect(api.recordPortalBriefUse).toHaveBeenCalledWith("brief-1", -1, 1));
    expect(screen.getByText((value) => value.includes("Searched: cli-health.commands, web-search.live") && value.includes("never executes them"))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Copy command" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copy command" }));
    expect(writeText).toHaveBeenCalledWith("portal brief build test");
  });

  it("shows withheld reason and no rendered evidence", async () => {
    const user = userEvent.setup();
    const withheld = { ...delivered, id: "brief-2", verdict: 2, reason: "best item effective score 0.100000 is below threshold", items: [], rendered: "" };
    api.listPortalBriefs.mockResolvedValue([withheld]);
    api.getPortalBrief.mockResolvedValue(withheld);
    renderWithProviders(<BriefInspector chatId="chat-2" />);
    await waitFor(() => expect(screen.getByText("WITHHELD · 2026-09-07T12:00:00Z")).toBeInTheDocument());
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
});
