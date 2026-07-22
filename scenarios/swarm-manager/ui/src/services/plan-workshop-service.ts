import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

export type PlanWorkshopSubject = {
  kind: "backlog_item";
  ref: string;
};

export type PlanWorkshopFinding = {
  id: string;
  severity: string;
  summary: string;
  evidence?: string;
  disposition?: { kind: "plan_revision" | "plan_authoring" | "follow_up" | "archive" | "supersede" | "attention"; rationale: string; confidence: "high" | "medium" | "low"; scope?: string };
};

export type PlanWorkshopQuestion = {
  id: string;
  question: string;
  options?: string[];
};

export type PlanWorkshopProposalRef = {
  session_id: string;
  proposal_id: string;
  apply_mode?: "direct" | "reconciliation" | "attention";
};

export type PlanWorkshopPacket = {
  findings?: PlanWorkshopFinding[];
  questions?: PlanWorkshopQuestion[];
  proposals?: PlanWorkshopProposalRef[];
};

export type PlanWorkshopResolution = {
  response_id: string;
  state: "direct_applied" | "reconciliation_required" | "candidate_ready" | "candidate_applied" | "candidate_discarded" | "needs_attention" | "stale" | "pending" | "integration_unavailable";
  reconciliation_id?: string;
  applied_at?: string;
  error?: string;
  candidate?: {
    id: string;
    plan_id: string;
    expected_base_content_hash: string;
    quality_status?: string;
    quality_findings?: string[];
    diff?: Array<{ field: string; before_json: string; after_json: string }>;
    diagnostics?: Array<{ severity: string; code: string; location?: string; message: string; guidance?: string }>;
    impact?: { before_grade?: string; after_grade?: string; added_issue_codes?: string[]; cleared_issue_codes?: string[]; execution_grade_regression?: boolean };
  };
};

export type PlanWorkshopReviewRun = {
  state: "pending" | "running" | "applied" | "stale" | "failed";
  workflow?: { execution_id?: string; run_id?: string; started_at?: string };
  agent_session_id?: string;
  error?: string;
  applied_at?: string;
};

export type PlanWorkshopSession = {
  id: string;
  subject: PlanWorkshopSubject;
  subject_version: string;
  plan_id?: string;
  plan_content_hash?: string;
  legacy_history?: { source_path: string; round_count: number; archived_at: string };
  packet: PlanWorkshopPacket;
  packet_history?: Array<{ id: string; subject_version: string; plan_content_hash?: string; created_at: string; packet: PlanWorkshopPacket }>;
  review?: PlanWorkshopReviewRun;
  responses?: Array<{ id: string; actor: string; subject_version: string; answers?: Record<string, string>; accepted_proposals?: PlanWorkshopProposalRef[]; submitted_at: string }>;
  resolutions?: PlanWorkshopResolution[];
};

export interface PlanWorkshopResponseInput {
  actor?: string;
  subject_version: string;
  answers?: Record<string, string>;
  accepted_proposals?: PlanWorkshopProposalRef[];
  idempotency_key: string;
}

export interface IPlanWorkshopService {
  open(subject: PlanWorkshopSubject, packet?: PlanWorkshopPacket): Promise<PlanWorkshopSession>;
  get(id: string): Promise<PlanWorkshopSession>;
  startReview(id: string): Promise<{ session: PlanWorkshopSession; review: PlanWorkshopReviewRun }>;
  applyReview(id: string): Promise<{ session: PlanWorkshopSession; review: PlanWorkshopReviewRun }>;
  submitResponse(id: string, response: PlanWorkshopResponseInput): Promise<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>;
  applyReconciliation(id: string, responseId: string): Promise<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>;
  applyCandidate(id: string, responseId: string): Promise<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>;
  discardCandidate(id: string, responseId: string, reason?: string): Promise<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>;
  acceptPlan(kind: string, name: string, planContentHash?: string): Promise<{ plan_acceptance: { actor: string; accepted_at: string; plan_content_hash: string; subject_version: string } }>;
  unacceptPlan(kind: string, name: string): Promise<void>;
}

export function createPlanWorkshopService(apiClient: IApiClient = defaultApiClient): IPlanWorkshopService {
  return {
    open(subject, packet) {
      return apiClient.post<PlanWorkshopSession>(API_ENDPOINTS.planWorkshops, {
        subject,
        ...(packet ? { packet } : {}),
      });
    },
    get(id) {
      return apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
    },
    startReview(id) {
      return apiClient.post<{ session: PlanWorkshopSession; review: PlanWorkshopReviewRun }>(API_ENDPOINTS.planWorkshopReview(id), {});
    },
    applyReview(id) {
      return apiClient.post<{ session: PlanWorkshopSession; review: PlanWorkshopReviewRun }>(API_ENDPOINTS.planWorkshopReviewApply(id), {});
    },
    submitResponse(id, response) {
      return apiClient.post<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>(API_ENDPOINTS.planWorkshopResponses(id), response);
    },
    applyReconciliation(id, responseId) {
      return apiClient.post<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>(API_ENDPOINTS.planWorkshopReconciliationApply(id, responseId), {});
    },
    applyCandidate(id, responseId) {
      return apiClient.post<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>(API_ENDPOINTS.planWorkshopCandidateApply(id, responseId), { acknowledge_quality_impact: true });
    },
    discardCandidate(id, responseId, reason) {
      return apiClient.post<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>(API_ENDPOINTS.planWorkshopCandidateDiscard(id, responseId), { reason: reason || "ignored by operator in Plan Workshop" });
    },
    acceptPlan(kind, name, planContentHash) {
      return apiClient.post<{ plan_acceptance: { actor: string; accepted_at: string; plan_content_hash: string; subject_version: string } }>(API_ENDPOINTS.backlogPlanAccept(kind, name), {
        ...(planContentHash ? { plan_content_hash: planContentHash } : {}),
      });
    },
    async unacceptPlan(kind, name) {
      await apiClient.delete<void>(API_ENDPOINTS.backlogPlanAccept(kind, name));
    },
  };
}

export const planWorkshopService = createPlanWorkshopService();
