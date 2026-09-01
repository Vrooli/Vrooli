import { afterEach, describe, expect, it, vi } from "vitest";

import { createBinding, listChannels } from "./channels";

describe("channel API helpers", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("decodes the channel catalogue", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify([]), { status: 200 })));
    await expect(listChannels()).resolves.toEqual([]);
  });

  it("reports catalogue failures", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("", { status: 503 })));
    await expect(listChannels()).rejects.toThrow("Unable to load channels (503)");
  });

  it("uses an empty thread key when one is omitted and reports attach failures", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("", { status: 409 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(createBinding({ agentId: "a", channelId: "in-app", address: "browser" })).rejects.toThrow("Unable to attach agent (409)");
    expect(JSON.parse((fetchMock.mock.calls[0]?.[1] as RequestInit).body as string)).toMatchObject({ threadKey: "" });
  });
});
