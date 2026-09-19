import { beforeEach, describe, expect, it, vi } from "vitest";

const listEvents = vi.hoisted(() => vi.fn());
const recordEvent = vi.hoisted(() => vi.fn());
const prepareBatch = vi.hoisted(() => vi.fn());
const consumeBatchPortion = vi.hoisted(() => vi.fn());
const undoBatchPortion = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listEvents, recordEvent, prepareBatch, consumeBatchPortion, undoBatchPortion }) }));
vi.mock("./client", () => ({ transport: {} }));

import { listInventoryEvents, recordInventoryEvent, prepareInventoryBatch, consumeInventoryBatchPortion, undoInventoryBatchPortion } from "./inventory";

describe("inventory API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("maps persisted events", async () => {
    listEvents.mockResolvedValue({ events: [{ id: "p1", workspaceId: "w1", kind: "purchase", itemId: "rice", batchId: "", amount: "500", unit: "g", recipeId: "", createdAt: "2026-09-18T00:00:00Z" }] });
    await expect(listInventoryEvents("w1")).resolves.toEqual([expect.objectContaining({ id: "p1", amount: "500" })]);
    expect(listEvents).toHaveBeenCalledWith({ workspaceId: "w1" });
  });
  it("records an explicit purchase event", async () => {
    recordEvent.mockResolvedValue({ event: { id: "p1", workspaceId: "w1", kind: "purchase", itemId: "rice", batchId: "", amount: "500", unit: "g", recipeId: "", createdAt: "2026-09-18T00:00:00Z" } });
    await expect(recordInventoryEvent({ workspaceId: "w1", id: "p1", kind: "purchase", itemId: "rice", amount: "500", unit: "g" })).resolves.toMatchObject({ itemId: "rice" });
    expect(recordEvent).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", id: "p1", kind: "purchase", amount: "500", unit: "g" }));
  });
  it("rejects a response without an event", async () => {
    recordEvent.mockResolvedValue({});
    await expect(recordInventoryEvent({ workspaceId: "w1", id: "p1", kind: "purchase", itemId: "rice", amount: "500", unit: "g" })).rejects.toThrow("no inventory event");
  });
  it("prepares a batch through the typed command", async () => {
    prepareBatch.mockResolvedValue({ batch: { id: "b1", recipeId: "r1", recipeRevision: 2n, yieldAmount: "4", availableAmount: "4", unit: "serving" } });
    await expect(prepareInventoryBatch({ workspaceId: "w1", eventId: "prep-1", batchId: "b1", recipeId: "r1", recipeRevision: 2n, yieldAmount: "4", unit: "serving", requirements: [{ itemId: "rice", amount: "500", unit: "g" }] })).resolves.toMatchObject({ id: "b1", availableAmount: "4" });
  });
  it("consumes and undoes a batch portion", async () => {
    consumeBatchPortion.mockResolvedValue({ batch: { id: "b1", availableAmount: "3", yieldAmount: "4", unit: "serving" } });
    undoBatchPortion.mockResolvedValue({ batch: { id: "b1", availableAmount: "4", yieldAmount: "4", unit: "serving" } });
    await expect(consumeInventoryBatchPortion({ workspaceId: "w1", eventId: "eat-1", batchId: "b1", amount: "1", unit: "serving" })).resolves.toMatchObject({ availableAmount: "3" });
    await expect(undoInventoryBatchPortion({ workspaceId: "w1", eventId: "undo-1", batchId: "b1", amount: "1", unit: "serving" })).resolves.toMatchObject({ availableAmount: "4" });
  });
  it("rejects batch commands without a returned batch", async () => {
    prepareBatch.mockResolvedValue({}); consumeBatchPortion.mockResolvedValue({}); undoBatchPortion.mockResolvedValue({});
    await expect(prepareInventoryBatch({ workspaceId: "w1", eventId: "prep-1", batchId: "b1", recipeId: "r1", recipeRevision: 1n, yieldAmount: "1", unit: "serving", requirements: [] })).rejects.toThrow("no prepared batch");
    await expect(consumeInventoryBatchPortion({ workspaceId: "w1", eventId: "eat-1", batchId: "b1", amount: "1", unit: "serving" })).rejects.toThrow("no batch state");
    await expect(undoInventoryBatchPortion({ workspaceId: "w1", eventId: "undo-1", batchId: "b1", amount: "1", unit: "serving" })).rejects.toThrow("no batch state");
  });
});
