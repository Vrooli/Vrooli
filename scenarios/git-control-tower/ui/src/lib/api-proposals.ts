// ============================================================================
// Commit proposals API — typed wrappers over ProposalService
// ============================================================================
//
// Proposals are drafts bound to exact content. Reads and edits are plain
// Connect calls. Approval is the human-control path used by commits: prepare
// an exact preview for "repo.apply_proposal" with subject context
// "proposal:<id>@<revision>", confirm it into a single-use intent, then apply.

import { create } from "@bufbuild/protobuf";
import {
  ConfirmMutationRequestSchema,
  GetAuthorityStatusRequestSchema,
  PrepareMutationRequestSchema,
  type MutationPreview,
} from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import {
  ProposalState,
  type ApplyProposalResponse,
  type ListProposalsResponse,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { humanControlClient, proposalClient } from "./connect";

export const APPLY_PROPOSAL_OPERATION = "repo.apply_proposal";

export function proposalSubjectContext(proposal: Pick<Proposal, "id" | "revision">): string {
  return `proposal:${proposal.id}@${proposal.revision}`;
}

export async function listProposals(repoId?: string | null, states: ProposalState[] = [ProposalState.OPEN]): Promise<ListProposalsResponse> {
  return proposalClient.listProposals({ repositoryId: repoId ?? "", states });
}

export async function getProposal(id: string): Promise<Proposal | undefined> {
  const response = await proposalClient.getProposal({ id });
  return response.proposal;
}

export interface EditProposalParams {
  id: string;
  expectedRevision: number;
  subject?: string;
  body?: string;
  removePaths?: string[];
}

export async function editProposal(params: EditProposalParams): Promise<Proposal | undefined> {
  const response = await proposalClient.editProposal({
    id: params.id,
    expectedRevision: params.expectedRevision,
    subject: params.subject,
    body: params.body,
    removePaths: params.removePaths ?? [],
  });
  return response.proposal;
}

export async function withdrawProposal(id: string, reason: string): Promise<Proposal | undefined> {
  const response = await proposalClient.withdrawProposal({ id, reason });
  return response.proposal;
}

export async function refreshProposal(id: string, expectedRevision: number): Promise<Proposal | undefined> {
  const response = await proposalClient.refreshProposal({ id, expectedRevision });
  return response.proposal;
}

/** Step 1 of approval: an exact server preview of this proposal revision. */
export async function prepareProposalApproval(proposal: Proposal): Promise<MutationPreview> {
  const authority = await humanControlClient.getAuthorityStatus(create(GetAuthorityStatusRequestSchema, {}));
  if (!authority.canMutate) {
    throw new Error(authority.reason || "approval unavailable: only the operator can approve proposals");
  }
  return humanControlClient.prepareMutation(create(PrepareMutationRequestSchema, {
    repositoryId: proposal.repositoryId,
    operation: APPLY_PROPOSAL_OPERATION,
    subjectContext: proposalSubjectContext(proposal),
  }));
}

/** Step 2 of approval: confirm the exact preview and apply once. */
export async function applyApprovedProposal(
  proposal: Proposal,
  preview: MutationPreview,
  options: { skipPrecommit?: boolean } = {},
): Promise<ApplyProposalResponse> {
  const subjectContext = proposalSubjectContext(proposal);
  const intent = await humanControlClient.confirmMutation(create(ConfirmMutationRequestSchema, {
    repositoryId: preview.repositoryId,
    operation: APPLY_PROPOSAL_OPERATION,
    expectedRevision: preview.expectedRevision,
    subjectDigest: preview.subjectDigest,
    subjectContext,
  }));
  if (!intent.intentId) {
    throw new Error("mutation intent response did not contain an intent id");
  }
  return proposalClient.applyProposal({
    repositoryId: preview.repositoryId,
    intentId: intent.intentId,
    id: proposal.id,
    revision: proposal.revision,
    skipPrecommitOnce: options.skipPrecommit ?? false,
  });
}
