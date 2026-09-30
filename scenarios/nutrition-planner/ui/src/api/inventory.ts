import { createClient } from "@connectrpc/connect";
import { InventoryService } from "@vrooli/proto-types/nutrition-planner/v1/inventory/inventory_pb";
import { transport } from "./client";

const client = createClient(InventoryService, transport);

export type InventoryEvent = { id: string; workspaceId: string; kind: string; itemId: string; batchId: string; amount: string; unit: string; recipeId: string; createdAt: string };
export type InventoryBatch = { id: string; recipeId: string; recipeRevision: bigint; yieldAmount: string; availableAmount: string; unit: string };

function mapEvent(event: { id: string; workspaceId: string; kind: string; itemId: string; batchId: string; amount: string; unit: string; recipeId: string; createdAt: string }): InventoryEvent {
  return { id: event.id, workspaceId: event.workspaceId, kind: event.kind, itemId: event.itemId, batchId: event.batchId, amount: event.amount, unit: event.unit, recipeId: event.recipeId, createdAt: event.createdAt };
}

export async function listInventoryEvents(workspaceId: string): Promise<InventoryEvent[]> {
  const response = await client.listEvents({ workspaceId });
  return response.events.map(mapEvent);
}

export async function recordInventoryEvent(input: { workspaceId: string; id: string; kind: string; itemId: string; amount: string; unit: string }): Promise<InventoryEvent> {
  const response = await client.recordEvent({ ...input, batchId: "", recipeId: "", createdAt: new Date().toISOString() });
  if (!response.event) throw new Error("The API returned no inventory event.");
  return mapEvent(response.event);
}

export async function prepareInventoryBatch(input: { workspaceId: string; eventId: string; batchId: string; recipeId: string; recipeRevision: bigint; yieldAmount: string; unit: string; requirements: Array<{ itemId: string; amount: string; unit: string }> }): Promise<InventoryBatch> {
  const response = await client.prepareBatch(input);
  if (!response.batch) throw new Error("The API returned no prepared batch.");
  return response.batch;
}

export async function consumeInventoryBatchPortion(input: { workspaceId: string; eventId: string; batchId: string; amount: string; unit: string; recipeId?: string }): Promise<InventoryBatch> {
  const response = await client.consumeBatchPortion({ ...input, recipeId: input.recipeId ?? "" });
  if (!response.batch) throw new Error("The API returned no batch state.");
  return response.batch;
}

export async function undoInventoryBatchPortion(input: { workspaceId: string; eventId: string; batchId: string; amount: string; unit: string; recipeId?: string }): Promise<InventoryBatch> {
  const response = await client.undoBatchPortion({ ...input, recipeId: input.recipeId ?? "" });
  if (!response.batch) throw new Error("The API returned no batch state.");
  return response.batch;
}
