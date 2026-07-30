import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { BacklogService } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import { API_BASE } from "../lib/api-client";

export type ReviewDecision = "accept" | "fail" | "followup" | "drop";

export interface ReviewDecisionInput {
  kind: string;
  name: string;
  round: number;
  decision: ReviewDecision;
  actor: string;
  rationale: string;
  followUp?: { steering: string; disposition: "follow_up_run" | "replan" | "new_items" };
}

export interface AttemptDecisionInput {
  subjectKind: string;
  subjectRef: string;
  roundNum: number;
  decision: ReviewDecision;
  actor: string;
  rationale: string;
  acceptedProposalIds?: string[];
  followUp?: { steering: string; disposition: "follow_up_run" | "replan" | "new_items" };
}

function defaultClient() {
  return createClient(BacklogService, createConnectTransport({ baseUrl: API_BASE }));
}

export type ReviewDecisionClient = Pick<ReturnType<typeof defaultClient>, "decideAttempt">;

export interface IAttemptDecisionService {
  decide(input: AttemptDecisionInput): Promise<unknown>;
}

export function createAttemptDecisionService(client: ReviewDecisionClient = defaultClient()): IAttemptDecisionService {
  return {
    decide(input: AttemptDecisionInput) {
      return client.decideAttempt(input);
    },
  };
}

export function createReviewDecisionService(client: ReviewDecisionClient = defaultClient()) {
	const attempts = createAttemptDecisionService(client);
  return {
    decide(input: ReviewDecisionInput) {
      return attempts.decide({
        subjectKind: "backlog-item",
        subjectRef: `${input.kind}/${input.name}`,
        roundNum: input.round,
        decision: input.decision,
        actor: input.actor,
        rationale: input.rationale,
        followUp: input.followUp,
      });
    },
  };
}

export const reviewDecisionService = createReviewDecisionService();
