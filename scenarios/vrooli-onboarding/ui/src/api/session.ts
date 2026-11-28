import { Code, createClient } from "@connectrpc/connect";
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
  const draftRPC = client as unknown as {
    getDraft(request: { target: string; actor: string }): Promise<WizardDraft>;
    saveDraft(request: WizardDraftSaveRequest): Promise<WizardDraft>;
  };
  return draftRPC.saveDraft(request).catch(async (error: unknown) => {
    // A remote target can commit the draft and lose its reply. Read once from
    // the same owner and accept only an exact payload match; never replay it.
    if (!isAmbiguousDelivery(error)) throw error;
    try {
      const current = await draftRPC.getDraft({ target: request.target, actor: request.actor });
      if (draftMatches(current, request)) return current;
    } catch {
      // A failed read does not establish whether the write committed.
    }
    throw error;
  });
}

export function discardDraft(target: string, actor: string): Promise<WizardDraft> {
  return (client as unknown as { discardDraft(request: { target: string; actor: string }): Promise<WizardDraft> }).discardDraft({ target, actor });
}

export async function fetchProfileSession(target = "local"): Promise<WizardProfileSession | null> {
  const response = await (client as unknown as { getProfileSession(request: { target: string }): Promise<{ session?: WireProfileSession }> }).getProfileSession({ target });
  const value = response.session;
  if (!value) return null;
  return profileSessionFromWire(value);
}

export async function saveProfileSession(request: WizardProfileSessionSaveRequest): Promise<WizardProfileSession> {
  const profileRPC = client as unknown as { saveProfileSession(request: unknown): Promise<{ session?: WireProfileSession }> };
  const wireRequest = {
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
  };
  let response: { session?: WireProfileSession };
  try {
    response = await profileRPC.saveProfileSession(wireRequest);
  } catch (error) {
    // A timeout or transport loss can happen after the owner commits. Read the
    // same target once and reconcile by exact user payload; never replay a write.
    if (!isAmbiguousDelivery(error)) throw error;
    try {
      const latest = await fetchProfileSession(request.target);
      if (latest && profileSessionMatches(latest, request)) return latest;
    } catch {
      // Preserve the original write failure; failed reconciliation is not proof
      // that the commit succeeded or that retrying would be safe.
    }
    throw error;
  }
  if (!response.session) throw new Error("profile session save returned no session");
  return profileSessionFromWire(response.session);
}

function profileSessionFromWire(value: WireProfileSession): WizardProfileSession {
  const metadata = value as unknown as { revision?: string; reconciliationState?: string; currentProfileVersion?: string; reconciliationReasons?: string[] };
  const reconciliationChanges = (value as unknown as { reconciliationChanges?: WireProfileSession["reconciliationChanges"] }).reconciliationChanges ?? [];
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
    reconciliationChanges: reconciliationChanges.map((change) => ({
      kind: change.kind, field: change.field, before: change.before, after: change.after,
      impact: change.impact, requiresReview: change.requiresReview,
    })),
    nextQuestionId: value.nextQuestionId || undefined,
    nextAction: value.nextAction || undefined,
  };
}

function isAmbiguousDelivery(error: unknown): boolean {
  if (error instanceof TypeError) return true;
  if (!error || typeof error !== "object" || !("code" in error)) return false;
  const code = (error as { code: unknown }).code;
  return code === Code.Unavailable || code === Code.DeadlineExceeded;
}

function draftMatches(draft: WizardDraft, request: WizardDraftSaveRequest): boolean {
  return draft.target === request.target
    && draft.actor === request.actor
    && draft.baseRevision === request.baseRevision
    && draft.stepId === (request.stepId ?? "")
    && stringMapsEqual(draft.choices, request.choices ?? {});
}

function stringMapsEqual(left: Record<string, string>, right: Record<string, string>): boolean {
  const leftKeys = Object.keys(left).sort();
  const rightKeys = Object.keys(right).sort();
  return leftKeys.length === rightKeys.length
    && leftKeys.every((key, index) => key === rightKeys[index] && left[key] === right[key]);
}

function profileSessionMatches(value: WizardProfileSession, request: WizardProfileSessionSaveRequest): boolean {
  return value.target === request.target
    && value.mode === request.mode
    && value.profileId === (request.profileId || undefined)
    && value.profileVersion === (request.profileVersion || undefined)
    && value.catalogRevision === (request.catalogRevision || undefined)
    && value.consequenceDigest === (request.consequenceDigest || undefined)
    && value.baseRevision === request.baseRevision
    && stableJSON(value.answers) === stableJSON(request.answers)
    && stableJSON(value.manualDecisions) === stableJSON(request.manualDecisions)
    && stableJSON(value.targetContext) === stableJSON(request.targetContext);
}

function stableJSON(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stableJSON).join(",")}]`;
  if (value && typeof value === "object") {
    const entries = Object.entries(value).sort(([left], [right]) => left.localeCompare(right));
    return `{${entries.map(([key, item]) => `${JSON.stringify(key)}:${stableJSON(item)}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

function stringContext(value: JsonObject | undefined): Record<string, string> {
  if (!value) return {};
  const result: Record<string, string> = {};
  for (const [key, item] of Object.entries(value)) {
    if (typeof item === "string") result[key] = item;
  }
  return result;
}
