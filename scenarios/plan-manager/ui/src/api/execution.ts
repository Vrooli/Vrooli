import { createClient } from "@connectrpc/connect";
import {
  ExecutionService,
  type Execution,
  type PhaseContext,
  type CompletionNudge,
} from "@vrooli/proto-types/plan-manager/v1/execution/execution_pb";
import {
  type Decision,
  type Finding,
  type GuidedStep,
  type Handoff,
  type VelocityPoint,
  type PhaseStatus,
  type FindingTriage,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the ExecutionService — the guided runner. The operator
 * console (Phase 7) drives a run: start it, fetch the just-in-time context,
 * advance + transition phases, capture decisions/findings in-flow, complete +
 * read the canonical handoff, triage candidate findings, and chart the per-plan
 * velocity series. Each helper returns the proto-typed shape.
 */
export const executionClient = createClient(ExecutionService, transport);

export async function startExecution(
  planId: string,
  runId = "",
): Promise<{ execution: Execution | undefined; context: PhaseContext | undefined; step: GuidedStep | undefined }> {
  const resp = await executionClient.start({ planId, runId });
  return { execution: resp.execution, context: resp.context, step: resp.step };
}

export async function getStatus(
  executionId: string,
): Promise<{
  execution: Execution | undefined;
  context: PhaseContext | undefined;
  step: GuidedStep | undefined;
}> {
  const resp = await executionClient.getStatus({ executionId });
  return { execution: resp.execution, context: resp.context, step: resp.step };
}

export async function getNext(
  executionId: string,
): Promise<{ context: PhaseContext | undefined; complete: boolean; step: GuidedStep | undefined }> {
  const resp = await executionClient.getNext({ executionId });
  return { context: resp.context, complete: resp.complete, step: resp.step };
}

export async function getContext(
  executionId: string,
  phaseId = "",
): Promise<{
  execution: Execution | undefined;
  context: PhaseContext | undefined;
  step: GuidedStep | undefined;
}> {
  const resp = await executionClient.getContext({ executionId, phaseId });
  return { execution: resp.execution, context: resp.context, step: resp.step };
}

export async function resumeExecution(
  planOrExecution: string,
  phaseId = "",
  runId = "",
): Promise<{
  execution: Execution | undefined;
  context: PhaseContext | undefined;
  step: GuidedStep | undefined;
}> {
  const resp = await executionClient.resume({ planOrExecution, phaseId, runId });
  return { execution: resp.execution, context: resp.context, step: resp.step };
}

export async function transitionPhase(
  executionId: string,
  phaseId: string,
  toStatus: PhaseStatus,
): Promise<{ execution: Execution | undefined; step: GuidedStep | undefined }> {
  const resp = await executionClient.transitionPhase({ executionId, phaseId, toStatus });
  return { execution: resp.execution, step: resp.step };
}

export async function recordDecision(
  executionId: string,
  phaseId: string,
  summary: string,
  detail = "",
): Promise<{ decision: Decision | undefined; step: GuidedStep | undefined }> {
  const resp = await executionClient.recordDecision({ executionId, phaseId, summary, detail });
  return { decision: resp.decision, step: resp.step };
}

export async function recordFinding(
  executionId: string,
  phaseId: string,
  title: string,
  detail = "",
): Promise<{ finding: Finding | undefined; step: GuidedStep | undefined }> {
  const resp = await executionClient.recordFinding({ executionId, phaseId, title, detail });
  return { finding: resp.finding, step: resp.step };
}

export async function completeExecution(
  executionId: string,
  tokens = 0n,
  iterations = 0,
): Promise<{ handoff: Handoff | undefined; nudges: CompletionNudge[]; step: GuidedStep | undefined }> {
  const resp = await executionClient.complete({ executionId, tokens, iterations });
  return { handoff: resp.handoff, nudges: resp.nudges, step: resp.step };
}

export async function getHandoff(
  executionId: string,
): Promise<{ handoff: Handoff | undefined; step: GuidedStep | undefined }> {
  const resp = await executionClient.getHandoff({ executionId });
  return { handoff: resp.handoff, step: resp.step };
}

export async function listCandidateFindings(executionId = ""): Promise<Finding[]> {
  const resp = await executionClient.listCandidateFindings({ executionId });
  return resp.findings;
}

export async function triageFinding(findingId: string, triage: FindingTriage): Promise<Finding | undefined> {
  const resp = await executionClient.triageFinding({ findingId, triage });
  return resp.finding;
}

export async function getVelocity(planId: string): Promise<VelocityPoint[]> {
  const resp = await executionClient.getVelocity({ planId });
  return resp.points;
}
