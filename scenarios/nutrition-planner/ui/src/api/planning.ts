import { createClient } from "@connectrpc/connect";
import { PlanningService } from "@vrooli/proto-types/nutrition-planner/v1/planning/planning_pb";
import { transport } from "./client";

const client = createClient(PlanningService, transport);
export type ExploreFitReason = { code: string; rule: string; reference: string; message: string };
export type ExploreRecipe = { recipeId: string; name: string; recipeRevision: bigint; fitReasons: ExploreFitReason[]; summary: string };
export async function exploreRecipes(workspaceId: string): Promise<{ planRevision: bigint; profileRevision: bigint; profileConfigured: boolean; candidates: ExploreRecipe[]; blockingReasons: ExploreFitReason[]; savedRecipeCount: number }> {
  const response = await client.exploreRecipes({ workspaceId });
  const mapReason = (reason: ExploreFitReason) => ({ code: reason.code, rule: reason.rule, reference: reason.reference, message: reason.message });
  return { planRevision: response.planRevision, profileRevision: response.profileRevision, profileConfigured: response.profileConfigured, candidates: response.candidates.map((item) => ({ recipeId: item.recipeId, name: item.name, recipeRevision: item.recipeRevision, fitReasons: item.fitReasons.map(mapReason), summary: item.summary })), blockingReasons: response.blockingReasons.map(mapReason), savedRecipeCount: response.savedRecipeCount };
}
export type PlanOccurrence = { date: string; slotName?: string; mode?: string; quantity?: string; recipeId: string; recipeRevision?: number; recipeName: string; reason: string; locked: boolean };
export type PlanDraft = { occurrences: PlanOccurrence[]; unresolved: { date: string; code: string; message: string }[]; inputReferences: string[]; runId: string; seed: number; currentRevision: bigint };

export async function generatePlan(input: { workspaceId: string; dates: string[]; mealSlots?: { date: string; slotName: string; mode?: string; quantity?: string; lockedRecipeId?: string }[]; lockedRecipeIds?: Record<string, string> }): Promise<PlanDraft> {
  const response = await client.generatePlan({ workspaceId: input.workspaceId, dates: input.dates, mealSlots: (input.mealSlots ?? []).map((slot) => ({ date: slot.date, slotName: slot.slotName, mode: slot.mode ?? "flexible", quantity: slot.quantity ?? "1", lockedRecipeId: slot.lockedRecipeId ?? "" })), lockedRecipeIds: input.lockedRecipeIds ?? {}, seed: 0n, costWeight: 0.34, effortWeight: 0.33, repetitionWeight: 0.33 });
  try { return { ...JSON.parse(response.draftJson) as Omit<PlanDraft, "currentRevision">, currentRevision: response.currentRevision }; } catch { throw new Error("The API returned an unreadable plan draft."); }
}

export async function getPlan(input: { workspaceId: string; fromDate: string; toDate: string }): Promise<{ draft: PlanDraft; hasPlan: boolean }> {
  const response = await client.getPlan({ workspaceId: input.workspaceId, fromDate: input.fromDate, toDate: input.toDate });
  try {
    const draft = JSON.parse(response.draftJson) as Omit<PlanDraft, "currentRevision">;
    return { draft: { ...draft, currentRevision: response.currentRevision }, hasPlan: response.hasPlan };
  } catch {
    throw new Error("The API returned an unreadable saved plan.");
  }
}

export async function applyPlan(input: { workspaceId: string; expectedRevision: bigint; draft: PlanDraft }): Promise<{ revision: bigint }> {
  const { currentRevision: _, ...draft } = input.draft;
  const response = await client.applyPlan({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision, draftJson: JSON.stringify(draft) });
  return { revision: response.revision };
}

export type RelatedOccurrence = { date: string; slotName: string; recipeRevision: number; recipeName: string; locked: boolean };
export type PreparedBatchImpact = { batchId: string; recipeId: string; recipeName: string; recipeRevision: number; available: string; unit: string };
export async function previewSwap(input: { workspaceId: string; expectedRevision: bigint; date: string; slotName: string; replacementRecipeId: string; replaceMatchingFuture?: boolean }): Promise<{ revision: bigint; preview: { draft: PlanDraft; changes: { date: string; slotName: string; beforeName: string; afterName: string }[]; relatedOccurrences: RelatedOccurrence[]; preparedBatchImpacts: PreparedBatchImpact[]; shoppingChanges: { key: string; before?: ShoppingLine; after?: ShoppingLine }[] }; affectedDates: string[] }> {
  const response = await client.previewSwap({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision, date: input.date, slotName: input.slotName, replacementRecipeId: input.replacementRecipeId, replaceMatchingFuture: input.replaceMatchingFuture ?? false });
  try { const parsed = JSON.parse(response.previewJson) as { draft: Omit<PlanDraft, "currentRevision">; changes: { date: string; slotName: string; beforeName: string; afterName: string }[]; relatedOccurrences?: RelatedOccurrence[]; preparedBatchImpacts?: PreparedBatchImpact[]; shoppingChanges?: { key: string; before?: ShoppingLine; after?: ShoppingLine }[] }; return { revision: response.revision, preview: { draft: { ...parsed.draft, currentRevision: response.revision }, changes: parsed.changes, relatedOccurrences: parsed.relatedOccurrences ?? [], preparedBatchImpacts: parsed.preparedBatchImpacts ?? [], shoppingChanges: parsed.shoppingChanges ?? [] }, affectedDates: response.affectedDates }; } catch { throw new Error("The API returned an unreadable swap preview."); }
}

export type ShoppingLine = { key: string; label: string; need: string; stock: string; missing: string; packageCount: string; price: string; portionCost?: string; checkoutTotal?: string; actualSpend?: string; sourceRecipeIds: string[]; checked: boolean; haveThis: boolean; actualQuantity: string; actualUnit: string; actualPrice: string; purchaseOmitted: boolean };

export async function getShoppingPreview(input: { workspaceId: string; expectedRevision: bigint }): Promise<{ revision: bigint; lines: ShoppingLine[] }> {
  const response = await client.getShoppingPreview({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision });
  return { revision: response.revision, lines: response.lines.map((line) => ({ key: line.key, label: line.label, need: line.need, stock: line.stock, missing: line.missing, packageCount: line.packageCount, price: line.price, portionCost: line.portionCost || "unknown", checkoutTotal: line.checkoutTotal || "unknown", actualSpend: line.actualSpend || "unknown", sourceRecipeIds: line.sourceRecipeIds, checked: line.checked, haveThis: line.haveThis, actualQuantity: line.actualQuantity, actualUnit: line.actualUnit, actualPrice: line.actualPrice, purchaseOmitted: line.purchaseOmitted })) };
}

export async function setShoppingChecked(input: { workspaceId: string; lineKey: string; checked: boolean }): Promise<boolean> {
  const response = await client.setShoppingChecked(input);
  return response.checked;
}

export async function setShoppingHaveThis(input: { workspaceId: string; lineKey: string; haveThis: boolean }): Promise<boolean> {
  const response = await client.setShoppingHaveThis(input);
  return response.haveThis;
}

export async function confirmShoppingPurchases(input: { workspaceId: string; reviewId: string; lines: { lineKey: string; itemId: string; amount: string; unit: string; price: string; omitted: boolean }[] }): Promise<void> {
  await client.confirmShoppingPurchases(input);
}

export async function recordFeedback(input: { workspaceId: string; expectedRevision: bigint; date: string; recipeId: string; portion?: string; minutes?: number }): Promise<void> {
  await client.recordFeedback({ workspaceId: input.workspaceId, expectedRevision: input.expectedRevision, date: input.date, recipeId: input.recipeId, portion: input.portion ?? "", minutes: input.minutes ?? 0 });
}

export async function undoFeedback(input: { workspaceId: string; date: string }): Promise<void> {
  await client.undoFeedback(input);
}
