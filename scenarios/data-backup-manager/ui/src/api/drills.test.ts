import { beforeEach, describe, expect, it, vi } from "vitest";

const rpc = vi.hoisted(() => ({ listDrills: vi.fn(), runDrill: vi.fn() }));

vi.mock("@connectrpc/connect", () => ({ createClient: () => rpc }));
vi.mock("./client", () => ({ transport: {} }));

import { listDrills } from "./drills";

beforeEach(() => vi.clearAllMocks());

describe("listDrills", () => {
  it("follows continuation tokens so every drill unit can be represented", async () => {
    rpc.listDrills
      .mockResolvedValueOnce({ drills: [{ id: "newest" }], nextPageToken: "page-2" })
      .mockResolvedValueOnce({ drills: [{ id: "older" }], nextPageToken: "" });

    const drills = await listDrills("plan-1", "target-1");

    expect(drills.map((drill) => drill.id)).toEqual(["newest", "older"]);
    expect(rpc.listDrills).toHaveBeenNthCalledWith(1, { planId: "plan-1", targetId: "target-1", pageSize: 100, pageToken: "" });
    expect(rpc.listDrills).toHaveBeenNthCalledWith(2, { planId: "plan-1", targetId: "target-1", pageSize: 100, pageToken: "page-2" });
  });

  it("fails closed if the server repeats a continuation cursor", async () => {
    rpc.listDrills
      .mockResolvedValueOnce({ drills: [], nextPageToken: "page-2" })
      .mockResolvedValueOnce({ drills: [], nextPageToken: "page-2" });

    await expect(listDrills()).rejects.toThrow("recovery drill history pagination repeated a cursor");
    expect(rpc.listDrills).toHaveBeenCalledTimes(2);
  });
});
