import type { ReactNode } from "react";
import { useProposal, useProposals } from "../../lib/hooks-proposals";
import { ProposalDetail } from "./ProposalDetail";
import { ProposalQueue } from "./ProposalQueue";
import { useSessionState } from "./sessionState";

interface ProposalsSectionProps {
  repoId: string | null;
  canApprove: boolean;
  approveDisabledReason?: string;
  skipConfirmation?: boolean;
  /** Hidden while browsing history; the composer slot shows commit checks. */
  hidden?: boolean;
  onOpenDiff: (path: string) => void;
  /** The commit composer, rendered below the queue. */
  children: ReactNode;
}

/** Places the proposal queue above the commit composer. Opening a proposal
 *  replaces both with its detail view, which is a full-width subview on
 *  mobile because the commit panel already fills the screen there. */
export function ProposalsSection({ repoId, canApprove, approveDisabledReason, skipConfirmation, hidden = false, onOpenDiff, children }: ProposalsSectionProps) {
  const [openId, setOpenId] = useSessionState<string | null>(`gct.proposals.open.${repoId ?? "active"}`, null);
  const list = useProposals(repoId, { enabled: !hidden });
  const detail = useProposal(hidden ? null : openId);
  const listed = list.data?.proposals.find((proposal) => proposal.id === openId);
  const openProposal = detail.data ?? listed;

  if (hidden) return <>{children}</>;
  if (openId && openProposal) {
    return (
      <div className="h-full min-h-0 rounded-lg border border-slate-800 bg-slate-950/40">
        <ProposalDetail
          key={openProposal.id}
          proposal={openProposal}
          repoId={repoId}
          canApprove={canApprove}
          approveDisabledReason={approveDisabledReason}
          skipConfirmation={skipConfirmation}
          onBack={() => setOpenId(null)}
          onOpenDiff={onOpenDiff}
        />
      </div>
    );
  }
  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <ProposalQueue
        proposals={list.data?.proposals ?? []}
        openCount={list.data?.openCount ?? 0}
        isLoading={list.isLoading}
        error={list.error}
        onOpen={setOpenId}
      />
      <div className="min-h-0 flex-1">{children}</div>
    </div>
  );
}
