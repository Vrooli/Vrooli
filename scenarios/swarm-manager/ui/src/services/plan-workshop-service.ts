import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { transitionService, type ITransitionService } from "./transition-service";
import { createAttemptDecisionService, type IAttemptDecisionService } from "./review-decision-service";

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
  workflow?: { execution_id?: string; run_id?: string; started_at?: string };
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

export function createPlanWorkshopService(
  apiClient: IApiClient = defaultApiClient,
  transitions: ITransitionService = transitionService,
  decisions: IAttemptDecisionService = createAttemptDecisionService(),
): IPlanWorkshopService {
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
    async applyReview(id) {
      const session = await apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
      const executionId = session.review?.workflow?.execution_id;
      if (!executionId) throw new Error("Plan workshop review has no transition execution.");
      await transitions.apply("plan.workshop.review", executionId);
      const updated = await apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
      if (!updated.review) throw new Error("Plan workshop review disappeared after transition application.");
      return { session: updated, review: updated.review };
    },
    submitResponse(id, response) {
      return apiClient.post<{ session: PlanWorkshopSession; resolution: PlanWorkshopResolution }>(API_ENDPOINTS.planWorkshopResponses(id), response);
    },
    async applyReconciliation(id, responseId) {
      const session = await apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
      const resolution = session.resolutions?.find((candidate) => candidate.response_id === responseId);
      const executionId = resolution?.workflow?.execution_id;
      if (!executionId) throw new Error("Plan workshop reconciliation has no transition execution.");
      await transitions.apply("plan.workshop.reconcile", executionId);
      const updated = await apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
      const updatedResolution = updated.resolutions?.find((candidate) => candidate.response_id === responseId);
      if (!updatedResolution) throw new Error("Plan workshop reconciliation disappeared after transition application.");
      return { session: updated, resolution: updatedResolution };
    },
    async applyCandidate(id, responseId) {
      await decisions.decide({
        subjectKind: "plan-workshop-candidate",
        subjectRef: `${id}/${responseId}`,
        roundNum: 1,
        decision: "accept",
        actor: "operator-ui",
        rationale: "Candidate accepted after review.",
      });
      const session = await apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
      const resolution = session.resolutions?.find((candidate) => candidate.response_id === responseId);
      if (!resolution) throw new Error("Plan workshop candidate disappeared after decision.");
      return { session, resolution };
    },
    async discardCandidate(id, responseId, reason) {
      await decisions.decide({
        subjectKind: "plan-workshop-candidate",
        subjectRef: `${id}/${responseId}`,
        roundNum: 1,
        decision: "drop",
        actor: "operator-ui",
        rationale: reason || "Ignored by operator in Plan Workshop.",
      });
      const session = await apiClient.get<PlanWorkshopSession>(API_ENDPOINTS.planWorkshopById(id));
      const resolution = session.resolutions?.find((candidate) => candidate.response_id === responseId);
      if (!resolution) throw new Error("Plan workshop candidate disappeared after decision.");
      return { session, resolution };
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
