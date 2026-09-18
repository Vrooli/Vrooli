import { createClient } from "@connectrpc/connect";
import { PlanningService } from "@vrooli/proto-types/nutrition-planner/v1/planning/planning_pb";
import { transport } from "./client";

const client = createClient(PlanningService, transport);
export type PlanOccurrence = { date: string; recipeId: string; recipeName: string; reason: string; locked: boolean };
export type PlanDraft = { occurrences: PlanOccurrence[]; unresolved: { date: string; code: string; message: string }[]; inputReferences: string[]; runId: string; seed: number; currentRevision: bigint };

export async function generatePlan(input: { workspaceId: string; dates: string[] }): Promise<PlanDraft> {
  const response = await client.generatePlan({ workspaceId: input.workspaceId, dates: input.dates, lockedRecipeIds: {}, seed: 0n, costWeight: 0.34, effortWeight: 0.33, repetitionWeight: 0.33 });
  try { return { ...JSON.parse(response.draftJson) as Omit<PlanDraft, "currentRevision">, currentRevision: response.currentRevision }; } catch { throw new Error("The API returned an unreadable plan draft."); }
}

export async function applyPlan(input: { workspaceId: string; expectedRevision: bigint; draft: PlanDraft }): Promise<{ revision: bigint }> {
  const { currentRevision: _, ...draft } = input.draft;
  const response = await client.applyPlan({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision, draftJson: JSON.stringify(draft) });
  return { revision: response.revision };
}

export async function previewSwap(input: { workspaceId: string; expectedRevision: bigint; date: string; replacementRecipeId: string; replaceMatchingFuture?: boolean }): Promise<{ revision: bigint; preview: { draft: PlanDraft; changes: { date: string; beforeName: string; afterName: string }[] }; affectedDates: string[] }> {
  const response = await client.previewSwap({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision, date: input.date, replacementRecipeId: input.replacementRecipeId, replaceMatchingFuture: input.replaceMatchingFuture ?? false });
  try { const parsed = JSON.parse(response.previewJson) as { draft: Omit<PlanDraft, "currentRevision">; changes: { date: string; beforeName: string; afterName: string }[] }; return { revision: response.revision, preview: { draft: { ...parsed.draft, currentRevision: response.revision }, changes: parsed.changes }, affectedDates: response.affectedDates }; } catch { throw new Error("The API returned an unreadable swap preview."); }
}

export type ShoppingLine = { key: string; label: string; need: string; stock: string; missing: string; packageCount: string; price: string; sourceRecipeIds: string[]; checked: boolean };

export async function getShoppingPreview(input: { workspaceId: string; expectedRevision: bigint }): Promise<{ revision: bigint; lines: ShoppingLine[] }> {
  const response = await client.getShoppingPreview({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision });
  return { revision: response.revision, lines: response.lines.map((line) => ({ key: line.key, label: line.label, need: line.need, stock: line.stock, missing: line.missing, packageCount: line.packageCount, price: line.price, sourceRecipeIds: line.sourceRecipeIds, checked: line.checked })) };
}

export async function setShoppingChecked(input: { workspaceId: string; lineKey: string; checked: boolean }): Promise<boolean> {
  const response = await client.setShoppingChecked(input);
  return response.checked;
}

export async function recordFeedback(input: { workspaceId: string; expectedRevision: bigint; date: string; recipeId: string; portion?: string; minutes?: number }): Promise<void> {
  await client.recordFeedback({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision, date: input.date, recipeId: input.recipeId, portion: input.portion ?? "", minutes: input.minutes ?? 0 });
}

export async function undoFeedback(input: { workspaceId: string; date: string }): Promise<void> {
  await client.undoFeedback(input);
}
