import { createClient } from "@connectrpc/connect";
import {
  AuthoringService,
  type AuthoringSession,
  type AutofillResult,
  type PhaseDraft,
  type Section,
  type StructureViolation,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";
import { type GuidedStep, type Plan } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the AuthoringService — the guided composer wizard. The
 * operator console (Phase 7) walks a plan's sections, runs the structure gate,
 * autofills the mechanical sections, and finalizes into a structured plan. Each
 * helper returns the proto-typed shape so callers branch on typed fields.
 */
export const authoringClient = createClient(AuthoringService, transport);

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
  session: AuthoringSession | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.submitSection({ sessionId, sectionKey, content });
  return { session: resp.session, violations: resp.violations, step: resp.step };
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
): Promise<{ session: AuthoringSession | undefined; results: AutofillResult[]; step: GuidedStep | undefined }> {
  const resp = await authoringClient.autofill({ sessionId, sources });
  return { session: resp.session, results: resp.results, step: resp.step };
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
  session: AuthoringSession | undefined;
  phase: PhaseDraft | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.addPhase({ sessionId, title, intent });
  return {
    session: resp.session,
    phase: resp.phase,
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
  session: AuthoringSession | undefined;
  violations: StructureViolation[];
  step: GuidedStep | undefined;
}> {
  const resp = await authoringClient.submitPhaseField({ sessionId, phaseId, field, content });
  return { session: resp.session, violations: resp.violations, step: resp.step };
}

export async function nextPhase(
  sessionId: string,
): Promise<{ phase: PhaseDraft | undefined; complete: boolean; step: GuidedStep | undefined }> {
  const resp = await authoringClient.nextPhase({ sessionId });
  return { phase: resp.phase, complete: resp.complete, step: resp.step };
}
