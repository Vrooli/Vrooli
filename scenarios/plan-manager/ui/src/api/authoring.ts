import { createClient } from "@connectrpc/connect";
import {
  AuthoringService,
  type AuthoringMutationSummary,
  type AuthoringProgress,
  type AuthoringSession,
  type AutofillResult,
  type PhaseDraft,
  type Section,
  type StructureViolation,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";
import {
  type GuidedStep,
  type Plan,
  type RelevantContextItem,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the AuthoringService — the guided composer wizard. The
 * operator console walks a plan's sections, runs the structure gate, autofills the
 * mechanical sections, and finalizes into a structured plan.
 *
 * Response contract: normal mutations return only a compact {@link AuthoringProgress}
 * snapshot plus a mutation summary, violations, and the next guided step — never
 * the full {@link AuthoringSession}. The UI hydrates full state explicitly through
 * {@link getSession} (read-after-write), so a growing plan never echoes its whole
 * graph on every keystroke.
 */
export const authoringClient = createClient(AuthoringService, transport);

/** getSession is the explicit full-state read used for read-after-write hydration. */
export async function getSession(
  sessionId: string,
): Promise<{ session: AuthoringSession | undefined; step: GuidedStep | undefined }> {
  const resp = await authoringClient.getSession({ sessionId });
  return { session: resp.session, step: resp.step };
}

export async function startSession(
  title: string,
  slug = "",
  templateId = "",
): Promise<{ session: AuthoringSession | undefined; step: GuidedStep | undefined }> {
  const resp = await authoringClient.startSession({ title, slug, templateId });
  return { session: resp.session, step: resp.step };
}

export async function getSection(
  sessionId: string,
  sectionKey: string,
): Promise<{ section: Section | undefined; step: GuidedStep | undefined }> {
  const resp = await authoringClient.getSection({ sessionId, sectionKey });
  return { section: resp.section, step: resp.step };
}

export async function submitSection(
  sessionId: string,
  sectionKey: string,
  content: string,
): Promise<{
  summary: AuthoringMutationSummary | undefined;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.submitSection({ sessionId, sectionKey, content });
  return { summary: resp.summary, progress: resp.progress, violations: resp.violations, step: resp.step };
}

export async function nextSection(
  sessionId: string,
): Promise<{ section: Section | undefined; complete: boolean; step: GuidedStep | undefined }> {
  const resp = await authoringClient.next({ sessionId });
  return { section: resp.section, complete: resp.complete, step: resp.step };
}

export async function validateStructure(
  sessionId: string,
): Promise<{ valid: boolean; violations: StructureViolation[]; step: GuidedStep | undefined }> {
  const resp = await authoringClient.validateStructure({ sessionId });
  return { valid: resp.valid, violations: resp.violations, step: resp.step };
}

export async function autofill(
  sessionId: string,
  sources: string[] = [],
): Promise<{ results: AutofillResult[]; progress: AuthoringProgress | undefined; step: GuidedStep | undefined }> {
  const resp = await authoringClient.autofill({ sessionId, sources });
  return { results: resp.results, progress: resp.progress, step: resp.step };
}

export async function submitRelevantContextItem(
  sessionId: string,
  phaseId: string,
  item: RelevantContextItem,
): Promise<{
  item: RelevantContextItem | undefined;
  summary: AuthoringMutationSummary | undefined;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.submitRelevantContextItem({ sessionId, phaseId, item });
  return { item: resp.item, summary: resp.summary, progress: resp.progress, violations: resp.violations, step: resp.step };
}

export async function updateRelevantContextItem(
  sessionId: string,
  phaseId: string,
  itemId: string,
  item: RelevantContextItem,
): Promise<{
  item: RelevantContextItem | undefined;
  summary: AuthoringMutationSummary | undefined;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.updateRelevantContextItem({ sessionId, phaseId, itemId, item });
  return { item: resp.item, summary: resp.summary, progress: resp.progress, violations: resp.violations, step: resp.step };
}

export async function removeRelevantContextItem(
  sessionId: string,
  phaseId: string,
  itemId: string,
): Promise<{
  summary: AuthoringMutationSummary | undefined;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.removeRelevantContextItem({ sessionId, phaseId, itemId });
  return { summary: resp.summary, progress: resp.progress, violations: resp.violations, step: resp.step };
}

export async function listRelevantContext(
  sessionId: string,
  phaseId = "",
): Promise<{ items: RelevantContextItem[]; step: GuidedStep | undefined }> {
  const resp = await authoringClient.listRelevantContext({ sessionId, phaseId });
  return { items: resp.items, step: resp.step };
}

export async function discoverSkillPack(
  sessionId: string,
  concepts: string[],
  complexity = "",
): Promise<{
  addedItems: RelevantContextItem[];
  keptItems: RelevantContextItem[];
  readCommand: string;
  recommendedReadCommand: string;
  budgetStatus: string;
  resultsSummary: string;
  degraded: boolean;
  degradedReason: string;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.discoverSkillPack({ sessionId, concepts, complexity });
  return {
    addedItems: resp.addedItems,
    keptItems: resp.keptItems,
    readCommand: resp.readCommand,
    recommendedReadCommand: resp.recommendedReadCommand,
    budgetStatus: resp.budgetStatus,
    resultsSummary: resp.resultsSummary,
    degraded: resp.degraded,
    degradedReason: resp.degradedReason,
    progress: resp.progress,
    violations: resp.violations,
    step: resp.step,
  };
}

export async function finalize(sessionId: string): Promise<{ plan: Plan | undefined; step: GuidedStep | undefined }> {
  const resp = await authoringClient.finalize({ sessionId });
  return { plan: resp.plan, step: resp.step };
}

export async function addPhase(
  sessionId: string,
  title: string,
  intent = "",
): Promise<{
  phase: PhaseDraft | undefined;
  summary: AuthoringMutationSummary | undefined;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.addPhase({ sessionId, title, intent });
  return {
    phase: resp.phase,
    summary: resp.summary,
    progress: resp.progress,
    violations: resp.violations,
    step: resp.step,
  };
}

export async function getPhase(
  sessionId: string,
  phaseId: string,
): Promise<{ phase: PhaseDraft | undefined; step: GuidedStep | undefined }> {
  const resp = await authoringClient.getPhase({ sessionId, phaseId });
  return { phase: resp.phase, step: resp.step };
}

export async function submitPhaseField(
  sessionId: string,
  phaseId: string,
  field: string,
  content: string,
): Promise<{
  phase: PhaseDraft | undefined;
  summary: AuthoringMutationSummary | undefined;
  progress: AuthoringProgress | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.submitPhaseField({ sessionId, phaseId, field, content });
  return { phase: resp.phase, summary: resp.summary, progress: resp.progress, violations: resp.violations, step: resp.step };
}

export async function nextPhase(
  sessionId: string,
): Promise<{ phase: PhaseDraft | undefined; complete: boolean; step: GuidedStep | undefined }> {
  const resp = await authoringClient.nextPhase({ sessionId });
  return { phase: resp.phase, complete: resp.complete, step: resp.step };
}
