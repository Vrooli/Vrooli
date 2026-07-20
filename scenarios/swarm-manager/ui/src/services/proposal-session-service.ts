import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { buildQueryString } from "../lib/query-utils";

export type ProposalSessionTargetType = "initiative" | "backlog_item";

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
  kind: "mutation_list" | string;
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
}

export interface IProposalSessionService {
  list(target?: { type: ProposalSessionTargetType; ref: string }): Promise<ProposalSession[]>;
  create(args: CreateProposalSessionArgs): Promise<ProposalSession>;
  decide(sessionId: string, proposalId: string, acceptedMutationIds: string[], note?: string): Promise<ProposalSession>;
  revise(sessionId: string, proposalId: string, note?: string): Promise<ProposalSession>;
}

export function createProposalSessionService(apiClient: IApiClient = defaultApiClient): IProposalSessionService {
  return {
	async list(target) {
	  const data = await apiClient.get<{ sessions?: ProposalSession[] }>(`${API_ENDPOINTS.proposalSessions}${buildQueryString(target ? { target_type: target.type, target_ref: target.ref } : {})}`);
      return data.sessions ?? [];
    },
    create(args) {
      return apiClient.post<ProposalSession>(API_ENDPOINTS.proposalSessions, { kind: "swarm_operations", ...args });
    },
    decide(sessionId, proposalId, acceptedMutationIds, note) {
      return apiClient.post<ProposalSession>(API_ENDPOINTS.agentSessionDecideMutationProposal(sessionId, proposalId), {
        accepted_mutation_ids: acceptedMutationIds,
        note: note ?? "",
      });
    },
    revise(sessionId, proposalId, note) {
      return apiClient.post<ProposalSession>(API_ENDPOINTS.agentSessionReviseMutationProposal(sessionId, proposalId), { note: note ?? "" });
    },
  };
}

export const proposalSessionService = createProposalSessionService();
