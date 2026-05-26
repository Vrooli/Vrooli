/**
 * Plans domain — UI ↔ API boundary over PlansService. A plan binds a set of
 * targets to a set of destinations with a schedule and a retention policy; the
 * scheduler fires it on cadence and operators can run it on demand. A plan
 * requires at least one target and one destination (enforced by the API).
 */
import { createClient, type Client } from "@connectrpc/connect";
import { PlansService } from "@vrooli/proto-types/data-backup-manager/v1/plans/plans_pb";
import type { Plan } from "@vrooli/proto-types/data-backup-manager/v1/plans/plans_pb";

import { transport } from "./client";

export const plansClient: Client<typeof PlansService> = createClient(PlansService, transport);

export interface PlanInput {
  name: string;
  targetIds: string[];
  destinationIds: string[];
  schedule: string;
  keepLatest: number;
  enabled: boolean;
}

export async function listPlans(): Promise<Plan[]> {
  const res = await plansClient.listPlans({});
  return res.plans;
}

export async function getPlan(id: string): Promise<Plan | undefined> {
  const res = await plansClient.getPlan({ id });
  return res.plan;
}

export async function createPlan(input: PlanInput): Promise<Plan | undefined> {
  const res = await plansClient.createPlan({
    name: input.name,
    targetIds: input.targetIds,
    destinationIds: input.destinationIds,
    schedule: input.schedule,
    retention: { keepLatest: input.keepLatest },
    enabled: input.enabled,
  });
  return res.plan;
}

export async function updatePlan(id: string, input: PlanInput): Promise<Plan | undefined> {
  const res = await plansClient.updatePlan({
    id,
    name: input.name,
    targetIds: input.targetIds,
    destinationIds: input.destinationIds,
    schedule: input.schedule,
    retention: { keepLatest: input.keepLatest },
    enabled: input.enabled,
  });
  return res.plan;
}

export async function deletePlan(id: string): Promise<void> {
  await plansClient.deletePlan({ id });
}

export type { Plan };
