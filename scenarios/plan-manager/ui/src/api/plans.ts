import { createClient } from "@connectrpc/connect";
import {
  PlansService,
  type CreatePlanRequest,
  type ListPlansResponse,
  type PlanTemplate,
} from "@vrooli/proto-types/plan-manager/v1/plans/plans_pb";
import {
  type Plan,
  type PlanEdge,
  type Phase,
  PlanStatus,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the PlansService — the structured-plan SSOT. The
 * operator console (Phase 7 boards) consumes these typed helpers; each returns
 * the proto-typed message so callers branch on typed fields, not strings.
 */
export const plansClient = createClient(PlansService, transport);

export async function listPlans(options?: {
  status?: PlanStatus;
  includeArchived?: boolean;
}): Promise<Plan[]> {
  const resp: ListPlansResponse = await plansClient.listPlans({
    status: options?.status ?? PlanStatus.UNSPECIFIED,
    includeArchived: options?.includeArchived ?? false,
  });
  return resp.plans;
}

export async function getPlan(id: string): Promise<Plan | undefined> {
  const resp = await plansClient.getPlan({ id });
  return resp.plan;
}

export async function createPlan(plan: CreatePlanRequest["plan"]): Promise<Plan | undefined> {
  const resp = await plansClient.createPlan({ plan });
  return resp.plan;
}

export async function archivePlan(id: string): Promise<Plan | undefined> {
  const resp = await plansClient.archivePlan({ id });
  return resp.plan;
}

export async function renderPlan(id: string): Promise<string> {
  const resp = await plansClient.renderMarkdown({ id });
  return resp.markdown;
}

export async function addPhase(planId: string, phase: Phase): Promise<Plan | undefined> {
  const resp = await plansClient.addPhase({ planId, phase });
  return resp.plan;
}

export async function getGraph(planId?: string): Promise<PlanEdge[]> {
  const resp = await plansClient.getGraph({ planId: planId ?? "" });
  return resp.edges;
}

export async function listTemplates(): Promise<PlanTemplate[]> {
  const resp = await plansClient.listTemplates({});
  return resp.templates;
}

export async function createFromTemplate(
  templateId: string,
  title: string,
  slug = "",
): Promise<Plan | undefined> {
  const resp = await plansClient.createFromTemplate({ templateId, title, slug });
  return resp.plan;
}
