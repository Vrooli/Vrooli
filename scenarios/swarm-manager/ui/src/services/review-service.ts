/**
 * Review Evidence Service
 *
 * Thin API client for review evidence endpoints. Follows the same
 * pattern as backlog-service.ts and execution-service.ts.
 */

import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { BacklogService } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import { API_BASE } from "../lib/api-client";

// --- Types ---

export type EvidenceType =
  | "screenshot"
  | "api_test"
  | "cli_output"
  | "config_diff"
  | "workflow_recording"
  | "custom";

export type ReviewRoundStatus = "pending" | "gathering" | "complete" | "failed";

export interface EvidenceTestResult {
  name: string;
  passed: boolean;
  output_summary?: string;
}

export interface EvidenceItem {
  id: string;
  criterion_id?: string;
  type: EvidenceType;
  producer?: string;
  trust?: string;
  settlement?: "settled" | "refuted" | "unavailable" | "pending";
  unavailable_reason?: string;
  title: string;
  description: string;
  capture_path?: string;
  verified: boolean;
  verified_at?: string;
  before_capture_path?: string;
  test_results?: EvidenceTestResult[];
}

export interface RequestMessage {
  role: "user" | "assistant";
  content: string;
  timestamp: string;
  added_evidence_ids?: string[];
}

export interface RequestThread {
  id: string;
  evidence_id?: string;
  status: "pending" | "fulfilled" | "dismissed";
  messages: RequestMessage[];
  created_at: string;
  run_id?: string;
}

export interface ReviewRound {
  round: number;
  generated_at: string;
  execution_id: string;
  status: ReviewRoundStatus;
  current_run_status?: string;
  failure_reason?: string;
  agent_assessment?: string;
  classification?: string;
  notes?: string[];
  disposition?: { kind: string; rationale: string; confidence: "high" | "medium" | "low"; scope?: string };
  evidence: EvidenceItem[];
  request_threads?: RequestThread[];
  run_id?: string;
}

// --- Service ---

export interface IReviewService {
  listRounds(kind: string, name: string): Promise<ReviewRound[]>;
  verifyEvidence(
    kind: string,
    name: string,
    round: number,
    evidenceId: string,
    verified: boolean,
    executionId?: string,
	actor?: string,
	reason?: string,
  ): Promise<void>;
  requestMoreEvidence(
    kind: string,
    name: string,
    round: number,
    message: string,
    evidenceId?: string,
  ): Promise<{ thread_id: string }>;
  continueRequest(
    kind: string,
    name: string,
    round: number,
    threadId: string,
    message: string,
  ): Promise<void>;
  dismissRequest(
    kind: string,
    name: string,
    round: number,
    threadId: string,
  ): Promise<void>;
  triggerReviewAgent(executionId: string): Promise<void>;
  getCaptureUrl(kind: string, name: string, capturePath: string): string;
}

function defaultVerificationClient() {
  return createClient(BacklogService, createConnectTransport({ baseUrl: API_BASE }));
}

export type ReviewVerificationClient = Pick<ReturnType<typeof defaultVerificationClient>, "verifyAttemptEvidence">;

export function createReviewService(
  apiClient: IApiClient = defaultApiClient,
  verificationClient: ReviewVerificationClient = defaultVerificationClient(),
): IReviewService {
  return {
    async listRounds(kind, name) {
      const data = await apiClient.get<{ rounds: ReviewRound[] }>(
        API_ENDPOINTS.reviewRounds(kind, name),
      );
      return data.rounds ?? [];
    },

    async verifyEvidence(kind, name, round, evidenceId, verified, executionId, actor, reason) {
		void executionId; // Attempt correlation is resolved server-side.
      await verificationClient.verifyAttemptEvidence({
        subjectKind: "backlog-item",
        subjectRef: `${kind}/${name}`,
        roundNum: round,
        evidenceId,
        verified,
        actor: actor?.trim() ?? "",
        reason: reason?.trim() ?? "",
      });
    },

    async requestMoreEvidence(kind, name, round, message, evidenceId) {
      return apiClient.post<{ thread_id: string }>(
        API_ENDPOINTS.reviewRequest(kind, name, round),
        { message, evidence_id: evidenceId },
      );
    },

    async continueRequest(kind, name, round, threadId, message) {
      await apiClient.post(
        API_ENDPOINTS.reviewContinueRequest(kind, name, round, threadId),
        { message },
      );
    },

    async dismissRequest(kind, name, round, threadId) {
      await apiClient.post(
        API_ENDPOINTS.reviewDismissRequest(kind, name, round, threadId),
        {},
      );
    },

    async triggerReviewAgent(executionId) {
      await apiClient.post(
        API_ENDPOINTS.executionTriggerReviewAgent(executionId),
        {},
      );
    },

    getCaptureUrl(kind, name, capturePath) {
      // Returns the relative API path; the api-client will resolve the full URL.
      return API_ENDPOINTS.reviewCapture(kind, name, capturePath);
    },
  };
}

export const reviewService = createReviewService();
