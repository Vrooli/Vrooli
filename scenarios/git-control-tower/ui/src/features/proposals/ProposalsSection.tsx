import type { ReactNode } from "react";
import { ArrowUpRight } from "lucide-react";
import { useProposals } from "../../lib/hooks-proposals";
import { hasPriorWork } from "./model";

interface ProposalsSectionProps {
  repoId: string | null;
  hidden?: boolean;
  onOpen: (id: string) => void;
  children: ReactNode;
}

/** A compact entry into the feature-owned review workspace; no nested detail. */
export function ProposalsSection({
  repoId,
  hidden = false,
  onOpen,
  children,
}: ProposalsSectionProps) {
  const list = useProposals(repoId, { enabled: !hidden });
  const first = list.data?.proposals[0];
  if (hidden) return <>{children}</>;
  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <button
        className="flex min-h-10 shrink-0 items-center justify-between rounded-lg border border-slate-800 px-3 text-xs text-cyan-300 hover:bg-slate-900"
        onClick={() => onOpen("inbox")}
        data-testid="proposal-inbox-open"
      >
        Proposal inbox · {list.data?.openCount ?? 0} open{" "}
        <ArrowUpRight className="h-3.5 w-3.5" />
      </button>
      {list.error && (
        <p role="alert" className="px-1 text-xs text-amber-300">
          Proposal inbox is unavailable. Your working changes are still
          accessible.
        </p>
      )}
      {first && (
        <button
          className="shrink-0 truncate px-2 py-1 text-left text-xs text-slate-400 hover:text-slate-100"
          onClick={() => onOpen(first.id)}
          data-testid="proposal-card"
          title={first.message?.subject}
        >
          {first.message?.subject} · {first.files.length} files
          {first.files.some(hasPriorWork) ? " · includes earlier work" : ""}
        </button>
      )}
      <div className="min-h-0 flex-1">{children}</div>
    </div>
  );
}
