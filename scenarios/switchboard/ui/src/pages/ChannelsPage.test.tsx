import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { ChannelsPage } from "./ChannelsPage";

describe("ChannelsPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("turns the attach action into a binding request", async () => {
    const fetchMock = vi.fn<typeof fetch>((input) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
      if (url.includes("/api/v1/channels")) {
        return Promise.resolve(new Response(JSON.stringify([{ descriptor: { id: "in-app", displayName: "In-app", setup: { friction: 0 } }, availability: "available", implemented: true }]), { status: 200 }));
      }
      return Promise.resolve(new Response("{}", { status: 200 }));
    });
    vi.stubGlobal("fetch", fetchMock);
    renderWithProviders(<ChannelsPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: strings.console.channels.attachAgent })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: strings.console.channels.attachAgent }));
    fireEvent.change(screen.getByLabelText(strings.console.channels.agentId), { target: { value: "agent-1" } });
    fireEvent.change(screen.getByLabelText(strings.console.channels.address), { target: { value: "browser" } });
    fireEvent.click(screen.getByRole("button", { name: strings.console.channels.confirmAttachment }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent(strings.console.channels.attached));
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("CreateBinding"), expect.objectContaining({ method: "POST" }));
  });
});
