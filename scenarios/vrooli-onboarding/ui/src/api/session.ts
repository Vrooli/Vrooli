import { createClient } from "@connectrpc/connect";
import type { JsonObject } from "@bufbuild/protobuf";
import { onboardingTransport } from "./base";
import {
  SessionService,
  type AdvanceSessionStepRequest,
  type GetSessionRequest,
  type GetSessionResponse,
  type GetStepModelResponse,
  type ProfileSession as WireProfileSession,
  type Step,
} from "@vrooli/proto-types/vrooli-onboarding/v1/session/session_pb";

export type WizardStep = Pick<Step, "id" | "ordinal" | "title" | "route" | "deferred">;

export type WizardDraft = {
  target: string;
  actor: string;
  baseRevision: string;
  revision: string;
  stepId: string;
  choices: Record<string, string>;
  updatedAt?: { seconds?: bigint; nanos?: number };
};

export type WizardDraftSaveRequest = {
  target: string;
  actor: string;
  expectedRevision?: string;
  baseRevision: string;
  stepId?: string;
  choices?: Record<string, string>;
};

export type WizardProfileSession = {
  target: string;
  actor: string;
  mode: "guided" | "manual";
  profileId?: string;
  profileVersion?: string;
  catalogRevision?: string;
  consequenceDigest?: string;
  baseRevision: string;
  answers: Record<string, unknown>;
  manualDecisions: Record<string, boolean>;
  targetContext: Record<string, string>;
  updatedAt?: WireProfileSession["updatedAt"];
  revision?: string;
  reconciliationState?: string;
  currentProfileVersion?: string;
  reconciliationReasons?: string[];
  reconciliationChanges?: Array<{
    kind: string;
    field: string;
    before: string;
    after: string;
    impact: string;
    requiresReview: boolean;
  }>;
  nextQuestionId?: string;
  nextAction?: string;
};

export type WizardProfileSessionSaveRequest = Omit<WizardProfileSession, "actor" | "updatedAt" | "revision"> & {
  expectedRevision: string;
};

const client = createClient(
  SessionService,
  onboardingTransport(),
);

export function fetchSession(target = "local"): Promise<GetSessionResponse> {
  return client.getSession({ target } as GetSessionRequest) as unknown as Promise<GetSessionResponse>;
}

export function advanceSessionStep(
  stepId: string,
  target = "local",
): Promise<GetSessionResponse> {
  return client.advanceSessionStep({ target, stepId } as AdvanceSessionStepRequest) as unknown as Promise<GetSessionResponse>;
}

export function fetchStepModel(target = "local"): Promise<GetStepModelResponse> {
  return client.getStepModel({ target }) as unknown as Promise<GetStepModelResponse>;
}

// The compatibility cast lets projected consumers roll forward independently
// while keeping the server-owned draft contract typed at this boundary.
export function fetchDraft(target: string, actor: string): Promise<WizardDraft> {
  return (client as unknown as { getDraft(request: { target: string; actor: string }): Promise<WizardDraft> }).getDraft({ target, actor });
}

export function saveDraft(request: WizardDraftSaveRequest): Promise<WizardDraft> {
  return (client as unknown as { saveDraft(request: WizardDraftSaveRequest): Promise<WizardDraft> }).saveDraft(request);
}

export function discardDraft(target: string, actor: string): Promise<WizardDraft> {
  return (client as unknown as { discardDraft(request: { target: string; actor: string }): Promise<WizardDraft> }).discardDraft({ target, actor });
}

export async function fetchProfileSession(target = "local"): Promise<WizardProfileSession | null> {
  const response = await (client as unknown as { getProfileSession(request: { target: string }): Promise<{ session?: WireProfileSession }> }).getProfileSession({ target });
  const value = response.session;
  if (!value) return null;
  const metadata = value as WireProfileSession & { revision?: string; reconciliationState?: string; currentProfileVersion?: string; reconciliationReasons?: string[] };
  return {
    target: value.target,
    actor: value.actor,
    mode: value.mode === "manual" ? "manual" : "guided",
    profileId: value.profileId || undefined,
    profileVersion: value.profileVersion || undefined,
    catalogRevision: value.catalogRevision || undefined,
    consequenceDigest: value.consequenceDigest || undefined,
    baseRevision: value.baseRevision,
    answers: value.answers ? { ...value.answers } : {},
    manualDecisions: { ...value.manualDecisions },
    targetContext: stringContext(value.targetContext),
    updatedAt: value.updatedAt,
    revision: metadata.revision || undefined,
    reconciliationState: metadata.reconciliationState || undefined,
    currentProfileVersion: metadata.currentProfileVersion || undefined,
    reconciliationReasons: [...(metadata.reconciliationReasons ?? [])],
    reconciliationChanges: (value.reconciliationChanges ?? []).map((change) => ({
      kind: change.kind, field: change.field, before: change.before, after: change.after,
      impact: change.impact, requiresReview: change.requiresReview,
    })),
    nextQuestionId: value.nextQuestionId || undefined,
    nextAction: value.nextAction || undefined,
  };
}

export async function saveProfileSession(request: WizardProfileSessionSaveRequest): Promise<WizardProfileSession> {
  const response = await (client as unknown as { saveProfileSession(request: unknown): Promise<{ session?: WireProfileSession }> }).saveProfileSession({
    target: request.target,
    expectedRevision: request.expectedRevision,
    session: {
      target: request.target,
      mode: request.mode,
      profileId: request.profileId ?? "",
      profileVersion: request.profileVersion ?? "",
      catalogRevision: request.catalogRevision ?? "",
      consequenceDigest: request.consequenceDigest ?? "",
      baseRevision: request.baseRevision,
      answers: request.answers as JsonObject,
      manualDecisions: request.manualDecisions,
      targetContext: request.targetContext as JsonObject,
    },
  });
  if (!response.session) throw new Error("profile session save returned no session");
  const value = response.session;
  const metadata = value as WireProfileSession & { revision?: string; reconciliationState?: string; currentProfileVersion?: string; reconciliationReasons?: string[] };
  return {
    target: value.target,
    actor: value.actor,
    mode: value.mode === "manual" ? "manual" : "guided",
    profileId: value.profileId || undefined,
    profileVersion: value.profileVersion || undefined,
    catalogRevision: value.catalogRevision || undefined,
    consequenceDigest: value.consequenceDigest || undefined,
    baseRevision: value.baseRevision,
    answers: value.answers ? { ...value.answers } : {},
    manualDecisions: { ...value.manualDecisions },
    targetContext: stringContext(value.targetContext),
    updatedAt: value.updatedAt,
    revision: metadata.revision || undefined,
    reconciliationState: metadata.reconciliationState || undefined,
    currentProfileVersion: metadata.currentProfileVersion || undefined,
    reconciliationReasons: [...(metadata.reconciliationReasons ?? [])],
    reconciliationChanges: (value.reconciliationChanges ?? []).map((change) => ({
      kind: change.kind, field: change.field, before: change.before, after: change.after,
      impact: change.impact, requiresReview: change.requiresReview,
    })),
    nextQuestionId: value.nextQuestionId || undefined,
    nextAction: value.nextAction || undefined,
  };
}

function stringContext(value: JsonObject | undefined): Record<string, string> {
  if (!value) return {};
  const result: Record<string, string> = {};
  for (const [key, item] of Object.entries(value)) {
    if (typeof item === "string") result[key] = item;
  }
  return result;
}
