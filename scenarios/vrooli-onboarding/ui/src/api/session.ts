import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  SessionService,
  type AdvanceSessionStepRequest,
  type GetSessionRequest,
  type GetSessionResponse,
  type GetStepModelResponse,
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
