import { createClient } from "@connectrpc/connect";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { onboardingTransport } from "./base";
import {
  ApplyService,
  type CancelApplyResponse,
  type GetApplyPlanResponse,
  type GetApplyRunResponse,
  type ReviewApplyResponse,
  type StartApplyResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";

const client = createClient(ApplyService, onboardingTransport());

export interface ApplyPlanItem {
  id: string;
  kind: string;
  name: string;
  required: boolean;
  privileged: boolean;
  state: string;
}

export interface ApplyPlanReview {
  target: string;
  plan_id: string;
  plan_digest: string;
  revision: string;
  consent_receipt_id: string;
  expires_at?: string;
}

export function reviewApply(request: { target: string; plan_id: string; plan_digest: string; expected_revision: string }): Promise<ReviewApplyResponse> {
  return client.reviewApply({
    target: request.target,
    planId: request.plan_id,
    planDigest: request.plan_digest,
    expectedRevision: request.expected_revision,
  }) as unknown as Promise<ReviewApplyResponse>;
}

export function startApply(request: { target: string; plan_id: string; plan_digest: string; expected_revision: string; consent_receipt_id: string; idempotency_key: string }): Promise<StartApplyResponse> {
  return client.startApply({
    target: request.target,
    planId: request.plan_id,
    planDigest: request.plan_digest,
    expectedRevision: request.expected_revision,
    consentReceiptId: request.consent_receipt_id,
    idempotencyKey: request.idempotency_key,
  }) as unknown as Promise<StartApplyResponse>;
}

export function fetchApplyRun(runId: string, target = "local"): Promise<GetApplyRunResponse> {
  return client.getApplyRun({ target, runId }) as unknown as Promise<GetApplyRunResponse>;
}

export function cancelApply(runId: string, target = "local"): Promise<CancelApplyResponse> {
  return client.cancelApply({ target, runId }) as unknown as Promise<CancelApplyResponse>;
}

export interface ApplyPlan {
  items: ApplyPlanItem[];
  target: string;
  plan_id: string;
  plan_digest: string;
  revision: string;
  expires_at?: string;
}

export async function fetchApplyPlan(target = "local"): Promise<ApplyPlan> {
  const response = await client.getApplyPlan({ target }) as unknown as GetApplyPlanResponse;
  return {
    items: response.items.map((item) => ({ id: item.id, kind: item.kind, name: item.name, required: item.required, privileged: item.privileged, state: item.observedState })),
    target: response.target,
    plan_id: response.planId,
    plan_digest: response.planDigest,
    revision: response.revision,
    expires_at: response.expiresAt ? timestampDate(response.expiresAt).toISOString() : undefined,
  };
}
