// ============================================================================
// Commit proposal hooks — React Query over ProposalService
// ============================================================================

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { MutationPreview } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import type {
  ApplyProposalResponse,
  ListProposalsResponse,
  Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { queryKeys } from "./hooks-query-keys";
import {
  applyApprovedProposal,
  editProposal,
  getProposal,
  listProposals,
  prepareProposalApproval,
  refreshProposal,
  withdrawProposal,
  getProposalDiff,
  combineProposals,
  type EditProposalParams,
} from "./api-proposals";

/** Open proposals in queue order. Freshness is computed server-side on read. */
export function useProposals(
  repoId?: string | null,
  opts: {
    enabled?: boolean;
    states?: Parameters<typeof listProposals>[1];
  } = {},
) {
  return useQuery<ListProposalsResponse, Error>({
    queryKey: [...queryKeys.proposals(repoId), opts.states ?? "open"],
    queryFn: () => listProposals(repoId, opts.states),
    enabled: opts.enabled ?? true,
    refetchInterval: 30_000,
    staleTime: 5_000,
  });
}

export function useProposalDiff(
  proposal: Proposal | undefined,
  path: string | undefined,
  compareCurrent = false,
) {
  return useQuery({
    queryKey: [
      "proposal-diff",
      proposal?.id,
      proposal?.revision,
      path,
      compareCurrent,
    ],
    queryFn: () => {
      if (!proposal || !path)
        throw new Error("Select a captured proposal file");
      return getProposalDiff(
        proposal.repositoryId,
        proposal.id,
        proposal.revision,
        path,
        compareCurrent,
      );
    },
    enabled: Boolean(proposal?.captured && path),
    staleTime: compareCurrent ? 0 : Infinity,
  });
}

export function useCombineProposals(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation({
    mutationFn: ({
      sources,
      subject,
      body,
    }: {
      sources: Proposal[];
      subject: string;
      body: string;
    }) => combineProposals(sources, subject, body),
    onSuccess: invalidate,
  });
}

export function useProposal(id: string | null, revision = 0) {
  return useQuery<Proposal | undefined, Error>({
    queryKey: queryKeys.proposal(id ?? "", revision),
    queryFn: () => getProposal(id ?? "", revision),
    enabled: Boolean(id),
    refetchInterval: revision ? false : 15_000,
  });
}

function useProposalInvalidation(repoId?: string | null) {
  const queryClient = useQueryClient();
  return (proposal?: Proposal) => {
    if (proposal) {
      queryClient.setQueryData(queryKeys.proposal(proposal.id), proposal);
    }
    void queryClient.invalidateQueries({
      queryKey: queryKeys.proposals(repoId),
    });
  };
}

export function useEditProposal(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<Proposal | undefined, Error, EditProposalParams>({
    mutationFn: (params) => editProposal(params),
    onSuccess: invalidate,
  });
}

export function useWithdrawProposal(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<
    Proposal | undefined,
    Error,
    {
      id: string;
      reason: string;
      expectedRevision?: number;
      replacementId?: string;
    }
  >({
    mutationFn: ({ id, reason, expectedRevision, replacementId }) =>
      withdrawProposal(id, reason, expectedRevision, replacementId),
    onSuccess: invalidate,
  });
}

export function useRefreshProposal(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<
    Proposal | undefined,
    Error,
    { id: string; expectedRevision: number }
  >({
    mutationFn: ({ id, expectedRevision }) =>
      refreshProposal(id, expectedRevision),
    onSuccess: invalidate,
  });
}

export function usePrepareProposalApproval() {
  return useMutation<
    MutationPreview,
    Error,
    {
      proposal: Proposal;
      acknowledgePriorWork: boolean;
      keepLaterChanges?: boolean;
    }
  >({
    mutationFn: ({ proposal, acknowledgePriorWork, keepLaterChanges }) =>
      prepareProposalApproval(proposal, acknowledgePriorWork, keepLaterChanges),
  });
}

/** Confirms the exact preview and applies once. A commit changes repository
 *  status and history, so those queries refresh too. */
export function useApplyProposal(repoId?: string | null) {
  const queryClient = useQueryClient();
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<
    ApplyProposalResponse,
    Error,
    {
      proposal: Proposal;
      preview: MutationPreview;
      skipPrecommit?: boolean;
      acknowledgePriorWork?: boolean;
      keepLaterChanges?: boolean;
    }
  >({
    mutationFn: ({
      proposal,
      preview,
      skipPrecommit,
      acknowledgePriorWork,
      keepLaterChanges,
    }) =>
      applyApprovedProposal(proposal, preview, {
        skipPrecommit,
        acknowledgePriorWork,
        keepLaterChanges,
      }),
    onSettled: (result) => {
      invalidate(result?.proposal);
      void queryClient.invalidateQueries({
        queryKey: queryKeys.repoStatus(repoId),
      });
      void queryClient.invalidateQueries({
        queryKey: ["repo", "history", repoId ?? "default"],
      });
    },
  });
}
