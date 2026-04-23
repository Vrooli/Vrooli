/**
 * Initiative Review Service — data access for the initiative-scoped review
 * phase that gates `review_pending` → terminal transitions.
 *
 * Separate from the per-item review-service because the initiative surface
 * has its own verdict vocabulary (accept/fail/followup), a different route
 * namespace, and a dedicated decisions audit log.
 */

import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  InitiativeReviewDecideResponse,
  InitiativeReviewDecision,
  InitiativeReviewRound,
  InitiativeReviewTriggerResult,
  InitiativeReviewVerdict,
} from "../types";

export interface DecideInitiativeReviewArgs {
  verdict: InitiativeReviewVerdict;
  rationale?: string;
  decidedBy?: string;
}

export interface IInitiativeReviewService {
  listRounds(name: string): Promise<InitiativeReviewRound[]>;
  getRound(name: string, round: number): Promise<InitiativeReviewRound>;
  trigger(name: string): Promise<InitiativeReviewTriggerResult>;
  decide(name: string, args: DecideInitiativeReviewArgs): Promise<InitiativeReviewDecideResponse>;
  listDecisions(name: string): Promise<InitiativeReviewDecision[]>;
}

export function createInitiativeReviewService(
  apiClient: IApiClient = defaultApiClient,
): IInitiativeReviewService {
  return {
    async listRounds(name) {
      const resp = await apiClient.get<{ rounds?: InitiativeReviewRound[] }>(
        API_ENDPOINTS.initiativeReviewRounds(name),
      );
      return resp.rounds ?? [];
    },

    async getRound(name, round) {
      return apiClient.get<InitiativeReviewRound>(
        API_ENDPOINTS.initiativeReviewRound(name, round),
      );
    },

    async trigger(name) {
      return apiClient.post<InitiativeReviewTriggerResult>(
        API_ENDPOINTS.initiativeReviewTrigger(name),
        {},
      );
    },

    async decide(name, args) {
      return apiClient.post<InitiativeReviewDecideResponse>(
        API_ENDPOINTS.initiativeReviewDecide(name),
        {
          verdict: args.verdict,
          rationale: args.rationale,
          decided_by: args.decidedBy,
        },
      );
    },

    async listDecisions(name) {
      const resp = await apiClient.get<{ decisions?: InitiativeReviewDecision[] }>(
        API_ENDPOINTS.initiativeReviewDecisions(name),
      );
      return resp.decisions ?? [];
    },
  };
}

export const initiativeReviewService = createInitiativeReviewService();
