import { beforeEach, describe, expect, it, vi } from "vitest";
const listCatalog = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listCatalog }) }));
vi.mock("./client", () => ({ transport: {} }));
import { listCatalogRevisions } from "./catalog";

describe("catalog API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("lists pinned product and food revisions for a workspace", async () => {
    const revisions = [{ id: "vitamin-d", revision: 3n, nutrients: [] }];
    listCatalog.mockResolvedValue({ revisions });
    await expect(listCatalogRevisions("w1")).resolves.toBe(revisions);
    expect(listCatalog).toHaveBeenCalledWith({ workspaceId: "w1" });
  });
});
