import { CheckCircle2, GitPullRequestArrow } from "lucide-react";
import { Button } from "../ui/button";
import { formatDisplayText } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import type { AgentSessionProposal } from "../../types";

interface SessionProposalListProps {
  proposals: AgentSessionProposal[];
  isMutating: boolean;
  onApply: (proposalId: string) => Promise<void>;
  variant?: "panel" | "plain";
}

const statusClasses: Record<string, string> = {
  draft: "bg-slate-800 text-slate-300",
  ready: "bg-violet-500/15 text-violet-200",
  applied: "bg-emerald-500/15 text-emerald-200",
  rejected: "bg-slate-800 text-slate-400",
  superseded: "bg-slate-800 text-slate-400",
  failed: "bg-red-500/15 text-red-200",
  needs_revision: "bg-amber-500/15 text-amber-200",
};

export function SessionProposalList({ proposals, isMutating, onApply, variant = "panel" }: SessionProposalListProps) {
  return (
    <section className={cn(variant === "panel" && "rounded-lg border border-white/10 bg-slate-950/30 p-3")} data-testid="agent-session-proposals">
      <div className="flex items-center gap-2">
        <GitPullRequestArrow className="h-4 w-4 text-violet-300" />
        <h4 className="text-xs font-medium text-slate-300">Proposals</h4>
        <span className="ml-auto text-[11px] text-slate-500">{proposals.length}</span>
      </div>
      <div className="mt-3 space-y-2">
        {proposals.length > 0 ? (
          proposals.map((proposal) => (
            <article key={proposal.id} className="rounded-md border border-white/10 bg-slate-900/70 p-2">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate text-xs font-medium text-slate-100">{formatDisplayText(proposal.kind)}</p>
                  <p className="mt-1 line-clamp-4 text-[11px] leading-5 text-slate-400">{proposal.summary}</p>
                </div>
                <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px]", statusClasses[proposal.status] ?? "bg-slate-800 text-slate-300")}>
                  {formatDisplayText(proposal.status)}
                </span>
              </div>
              {proposal.status === "ready" && (
                <Button
                  className="mt-2 w-full"
                  size="sm"
                  onClick={() => void onApply(proposal.id)}
                  disabled={isMutating}
                  data-testid="agent-session-apply-proposal"
                >
                  <CheckCircle2 className="mr-2 h-4 w-4" />
                  Apply
                </Button>
              )}
            </article>
          ))
        ) : (
          <p className="py-8 text-center text-xs text-slate-500">No proposals yet.</p>
        )}
      </div>
    </section>
  );
}
