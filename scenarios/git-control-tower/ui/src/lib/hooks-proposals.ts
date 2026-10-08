// ============================================================================
// Commit proposal hooks — React Query over ProposalService
// ============================================================================

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { MutationPreview } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import type { ApplyProposalResponse, ListProposalsResponse, Proposal } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { queryKeys } from "./hooks-query-keys";
import {
  applyApprovedProposal,
  editProposal,
  getProposal,
  listProposals,
  prepareProposalApproval,
  refreshProposal,
  withdrawProposal,
  type EditProposalParams,
} from "./api-proposals";

/** Open proposals in queue order. Freshness is computed server-side on read. */
export function useProposals(repoId?: string | null, opts: { enabled?: boolean } = {}) {
  return useQuery<ListProposalsResponse, Error>({
    queryKey: queryKeys.proposals(repoId),
    queryFn: () => listProposals(repoId),
    enabled: opts.enabled ?? true,
    refetchInterval: 30_000,
    staleTime: 5_000,
  });
}

export function useProposal(id: string | null) {
  return useQuery<Proposal | undefined, Error>({
    queryKey: queryKeys.proposal(id ?? ""),
    queryFn: () => getProposal(id ?? ""),
    enabled: Boolean(id),
    refetchInterval: 15_000,
  });
}

function useProposalInvalidation(repoId?: string | null) {
  const queryClient = useQueryClient();
  return (proposal?: Proposal) => {
    if (proposal) {
      queryClient.setQueryData(queryKeys.proposal(proposal.id), proposal);
    }
    void queryClient.invalidateQueries({ queryKey: queryKeys.proposals(repoId) });
  };
}

export function useEditProposal(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<Proposal | undefined, Error, EditProposalParams>({
    mutationFn: editProposal,
    onSuccess: invalidate,
  });
}

export function useWithdrawProposal(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<Proposal | undefined, Error, { id: string; reason: string }>({
    mutationFn: ({ id, reason }) => withdrawProposal(id, reason),
    onSuccess: invalidate,
  });
}

export function useRefreshProposal(repoId?: string | null) {
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<Proposal | undefined, Error, { id: string; expectedRevision: number }>({
    mutationFn: ({ id, expectedRevision }) => refreshProposal(id, expectedRevision),
    onSuccess: invalidate,
  });
}

export function usePrepareProposalApproval() {
  return useMutation<MutationPreview, Error, Proposal>({ mutationFn: prepareProposalApproval });
}

/** Confirms the exact preview and applies once. A commit changes repository
 *  status and history, so those queries refresh too. */
export function useApplyProposal(repoId?: string | null) {
  const queryClient = useQueryClient();
  const invalidate = useProposalInvalidation(repoId);
  return useMutation<ApplyProposalResponse, Error, { proposal: Proposal; preview: MutationPreview; skipPrecommit?: boolean }>({
    mutationFn: ({ proposal, preview, skipPrecommit }) => applyApprovedProposal(proposal, preview, { skipPrecommit }),
    onSettled: (result) => {
      invalidate(result?.proposal);
      void queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus(repoId) });
      void queryClient.invalidateQueries({ queryKey: ["repo", "history", repoId ?? "default"] });
    },
  });
}
