import { describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({
  list: vi.fn(),
  get: vi.fn(),
  recordUse: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => client),
}));

import { BriefConsumer, BriefUseKind, getPortalBrief, listPortalBriefs, recordPortalBriefUse } from "./brief";

describe("api/brief Connect wrappers", () => {
  it("maps list defaults and explicit chat filters", async () => {
    client.list.mockResolvedValueOnce({ briefs: [{ id: "brief-1" }] });
    await expect(listPortalBriefs()).resolves.toEqual([{ id: "brief-1" }]);
    expect(client.list).toHaveBeenCalledWith({ consumer: BriefConsumer.UNSPECIFIED, chatId: "", sessionRef: "", limit: 20 });

    client.list.mockResolvedValueOnce({ briefs: [] });
    await listPortalBriefs({ consumer: BriefConsumer.PORTAL_LLM, chatId: "chat-1", limit: 3 });
    expect(client.list).toHaveBeenLastCalledWith({ consumer: BriefConsumer.PORTAL_LLM, chatId: "chat-1", sessionRef: "", limit: 3 });
  });

  it("requires a brief in get responses and records use", async () => {
    client.get.mockResolvedValueOnce({ brief: { id: "brief-1" } });
    await expect(getPortalBrief("brief-1")).resolves.toEqual({ id: "brief-1" });

    client.get.mockResolvedValueOnce({});
    await expect(getPortalBrief("missing")).rejects.toThrow("brief response did not include a brief");

    client.recordUse.mockResolvedValueOnce({ recorded: true });
    await expect(recordPortalBriefUse("brief-1", 0, BriefUseKind.COPIED)).resolves.toBe(true);
    expect(client.recordUse).toHaveBeenCalledWith({ briefId: "brief-1", itemIndex: 0, kind: BriefUseKind.COPIED });
  });
});
