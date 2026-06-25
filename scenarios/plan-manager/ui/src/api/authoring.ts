import { createClient } from "@connectrpc/connect";
import {
  AuthoringService,
  type AuthoringSession,
  type AutofillResult,
  type Section,
  type StructureViolation,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";
import { type Plan } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

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
): Promise<AuthoringSession | undefined> {
  const resp = await authoringClient.startSession({ title, slug, templateId });
  return resp.session;
}

export async function getSection(
  sessionId: string,
  sectionKey: string,
): Promise<Section | undefined> {
  const resp = await authoringClient.getSection({ sessionId, sectionKey });
  return resp.section;
}

export async function submitSection(
  sessionId: string,
  sectionKey: string,
  content: string,
): Promise<{ session: AuthoringSession | undefined; violations: StructureViolation[] }> {
  const resp = await authoringClient.submitSection({ sessionId, sectionKey, content });
  return { session: resp.session, violations: resp.violations };
}

export async function nextSection(
  sessionId: string,
): Promise<{ section: Section | undefined; complete: boolean }> {
  const resp = await authoringClient.next({ sessionId });
  return { section: resp.section, complete: resp.complete };
}

export async function validateStructure(
  sessionId: string,
): Promise<{ valid: boolean; violations: StructureViolation[] }> {
  const resp = await authoringClient.validateStructure({ sessionId });
  return { valid: resp.valid, violations: resp.violations };
}

export async function autofill(
  sessionId: string,
  sources: string[] = [],
): Promise<{ session: AuthoringSession | undefined; results: AutofillResult[] }> {
  const resp = await authoringClient.autofill({ sessionId, sources });
  return { session: resp.session, results: resp.results };
}

export async function finalize(sessionId: string): Promise<Plan | undefined> {
  const resp = await authoringClient.finalize({ sessionId });
  return resp.plan;
}
