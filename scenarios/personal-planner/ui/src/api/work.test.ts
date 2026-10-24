import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ listWorkItems: vi.fn(), createWorkItem: vi.fn(), getTodayPlan: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { completeWorkItem, createWorkItem, fetchTodayPlan, fetchWorkItems, snoozeWorkItem, updateWorkEstimate } from "./work";

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

  it("reads the server-owned day projection", async () => {
    client.getTodayPlan.mockResolvedValueOnce({ plannedMinutes: 45 });
    expect(await fetchTodayPlan()).toEqual({ plannedMinutes: 45 });
  });

  it("persists a deferred work item through the REST seam", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(snoozeWorkItem({ id: "work/1", until: "2026-09-22", reason: "blocked" })).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("work%2F1/snooze"), expect.objectContaining({ method: "POST", body: JSON.stringify({ until: "2026-09-22", reason: "blocked" }) }));
  });

	it("persists completion through the REST seam", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(completeWorkItem("work/1")).resolves.toBeUndefined();
		expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("work%2F1/complete"), expect.objectContaining({ method: "POST" }));
	});

	it("persists an estimate change with its reason", async () => {
		const fetchMock = vi.fn().mockResolvedValueOnce(new Response(null, { status: 204 }));
		vi.stubGlobal("fetch", fetchMock);
		await expect(updateWorkEstimate({ id: "work/1", remainingMinutes: 75, reason: "scope_changed" })).resolves.toBeUndefined();
		expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining("work%2F1/estimate"), expect.objectContaining({ method: "PUT", body: JSON.stringify({ remaining_minutes: 75, reason: "scope_changed" }) }));
	});
});
