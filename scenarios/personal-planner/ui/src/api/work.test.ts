import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listWorkItems: vi.fn(), createWorkItem: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { createWorkItem, fetchWorkItems } from "./work";

afterEach(() => vi.clearAllMocks());

describe("work API transport", () => {
  it("lists captured work and creates a name-first task", async () => {
    const item = { id: "w-1", title: "Write release note" } as never;
    client.listWorkItems.mockResolvedValueOnce({ workItems: [item] });
    client.createWorkItem.mockResolvedValueOnce({ workItem: item });

    expect(await fetchWorkItems()).toEqual([item]);
    expect(await createWorkItem({ title: "Write release note", description: "Explain the workflow", remainingMinutes: 45, sourceLabel: "Planner" })).toBe(item);
    expect(client.createWorkItem).toHaveBeenCalledWith({ title: "Write release note", description: "Explain the workflow", remainingMinutes: 45, sourceLabel: "Planner" });
  });

  it("rejects a mutation response without its durable item", async () => {
    client.createWorkItem.mockResolvedValueOnce({});
    await expect(createWorkItem({ title: "Missing response" })).rejects.toThrow("work item was not returned");
  });
});
