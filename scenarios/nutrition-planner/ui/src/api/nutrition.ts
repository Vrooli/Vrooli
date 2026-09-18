import { createClient } from "@connectrpc/connect";
import { NutritionService } from "@vrooli/proto-types/nutrition-planner/v1/nutrition/nutrition_pb";
import { transport } from "./client";

const client = createClient(NutritionService, transport);
export type NutritionTarget = { id: string; revision: bigint; nutrientId: string; lower: string; upper: string; period: string; scope: string; enforcement: string; provenance: string; effectiveFrom: string; effectiveTo: string; active: boolean };
export type IntakeEvent = { id: string; date: string; recipeId: string; recipeRevision: bigint; nutrientId: string; amount: string; unit: string; reason: string; correctionOf: string; recordedAt: string };

export async function listTargets(workspaceId: string): Promise<NutritionTarget[]> {
  const response = await client.listTargets({ workspaceId });
  return response.targets.map((target) => ({ id: target.id, revision: target.revision, nutrientId: target.nutrientId, lower: target.lower, upper: target.upper, period: target.period, scope: target.scope, enforcement: target.enforcement, provenance: target.provenance, effectiveFrom: target.effectiveFrom, effectiveTo: target.effectiveTo, active: target.active }));
}

export async function createTarget(input: { workspaceId: string; nutrientId: string; lower: string; upper: string; period: string; scope: string; enforcement: string; provenance: string; effectiveFrom: string }): Promise<NutritionTarget> {
  const response = await client.createTarget(input);
  if (!response.target) throw new Error("The API returned no nutrition target.");
  const target = response.target;
  return { id: target.id, revision: target.revision, nutrientId: target.nutrientId, lower: target.lower, upper: target.upper, period: target.period, scope: target.scope, enforcement: target.enforcement, provenance: target.provenance, effectiveFrom: target.effectiveFrom, effectiveTo: target.effectiveTo, active: target.active };
}

function mapIntake(event: { id: string; date: string; recipeId: string; recipeRevision: bigint; nutrientId: string; amount: string; unit: string; reason: string; correctionOf: string; recordedAt: string }): IntakeEvent {
  return { id: event.id, date: event.date, recipeId: event.recipeId, recipeRevision: event.recipeRevision, nutrientId: event.nutrientId, amount: event.amount, unit: event.unit, reason: event.reason, correctionOf: event.correctionOf, recordedAt: event.recordedAt };
}

export async function listIntakes(workspaceId: string): Promise<IntakeEvent[]> {
  const response = await client.listIntakes({ workspaceId });
  return response.events.map(mapIntake);
}

export async function recordIntake(input: { workspaceId: string; id: string; date: string; recipeId?: string; recipeRevision?: bigint; nutrientId: string; amount: string; unit: string; reason?: string; correctionOf?: string }): Promise<IntakeEvent> {
  const response = await client.recordIntake({ ...input, recipeId: input.recipeId ?? "", recipeRevision: input.recipeRevision ?? 0n, reason: input.reason ?? "", correctionOf: input.correctionOf ?? "" });
  if (!response.event) throw new Error("The API returned no intake event.");
  return mapIntake(response.event);
}
