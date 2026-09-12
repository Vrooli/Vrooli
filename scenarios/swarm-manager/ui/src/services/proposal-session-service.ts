import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { buildQueryString } from "../lib/query-utils";
import { createAttemptDecisionService, type IAttemptDecisionService } from "./review-decision-service";

export type ProposalSessionTargetType = "goal" | "backlog_item" | "capture";

export interface ProposalMutationOutcome {
  mutation_id: string;
  op: string;
  target: string;
  applied: boolean;
  skipped: boolean;
  error?: string;
}

export interface ProposalSessionProposal {
  id: string;
  kind: "mutation_list" | "no_change_recommendation" | string;
  status: "draft" | "ready" | "applied" | "needs_revision" | "superseded" | "failed" | string;
  summary: string;
  payload_json: string;
  created_at: string;
  updated_at: string;
  parse_warnings?: string[];
  validation_errors?: string[];
  decisions?: Array<{
    accepted_mutation_ids?: string[];
    rejected_mutation_ids?: string[];
    note?: string;
    outcomes?: ProposalMutationOutcome[];
    decided_at: string;
  }>;
}

export interface ProposalSession {
  id: string;
  title: string;
  status: string;
  skill_id: string;
  proposal_target?: { type: ProposalSessionTargetType; ref: string; name: string };
  proposals?: ProposalSessionProposal[];
}

export interface CreateProposalSessionArgs {
  title: string;
  target: { type: ProposalSessionTargetType; ref: string; name: string };
  starterJobId?: string;
}

export interface IProposalSessionService {
  list(target?: { type: ProposalSessionTargetType; ref: string }): Promise<ProposalSession[]>;
  create(args: CreateProposalSessionArgs): Promise<ProposalSession>;
  decide(sessionId: string, proposalId: string, acceptedMutationIds: string[], note?: string): Promise<ProposalSession>;
  acceptKeep(sessionId: string, proposalId: string, note?: string): Promise<ProposalSession>;
  revise(sessionId: string, proposalId: string, note?: string): Promise<ProposalSession>;
}

export function createProposalSessionService(
  apiClient: IApiClient = defaultApiClient,
  decisions: IAttemptDecisionService = createAttemptDecisionService(),
): IProposalSessionService {
  return {
	async list(target) {
	  const data = await apiClient.get<{ sessions?: ProposalSession[] }>(`${API_ENDPOINTS.proposalSessions}${buildQueryString(target ? { target_type: target.type, target_ref: target.ref } : {})}`);
      return data.sessions ?? [];
    },
    create(args) {
      return apiClient.post<ProposalSession>(API_ENDPOINTS.proposalSessions, {
        kind: "swarm_operations",
        title: args.title,
        target: args.target,
        ...(args.starterJobId ? { starter_job_id: args.starterJobId } : {}),
      });
    },
    async decide(sessionId, proposalId, acceptedMutationIds, note) {
      await decisions.decide({
        subjectKind: "agent-session-proposal",
        subjectRef: `${sessionId}/${proposalId}`,
        roundNum: 1,
        decision: acceptedMutationIds.length > 0 ? "accept" : "drop",
        actor: "operator-ui",
        rationale: note ?? "",
        acceptedProposalIds: acceptedMutationIds,
      });
      return apiClient.get<ProposalSession>(API_ENDPOINTS.agentSessionById(sessionId));
    },
    acceptKeep(sessionId, proposalId, note) {
      return apiClient.post<ProposalSession>(API_ENDPOINTS.agentSessionAcceptKeepRecommendation(sessionId, proposalId), { note: note ?? "" });
    },
    revise(sessionId, proposalId, note) {
      return apiClient.post<ProposalSession>(API_ENDPOINTS.agentSessionReviseMutationProposal(sessionId, proposalId), { note: note ?? "" });
    },
  };
}

export const proposalSessionService = createProposalSessionService();
