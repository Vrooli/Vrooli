import { useState } from "react";
import { ChevronDown, ChevronRight, FileCode2, GitPullRequestDraft, Loader2 } from "lucide-react";
import type { Proposal } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { Badge } from "../../components/ui/badge";
import { effortLabel, flagCounts, flagMeta, freshnessMeta } from "./model";

interface ProposalQueueProps {
  proposals: Proposal[];
  openCount: number;
  isLoading?: boolean;
  error?: Error | null;
  onOpen: (id: string) => void;
}

/** The "Proposals (n)" queue shown above the commit composer. Each card is a
 *  commit an agent proposed; opening it shows the exact files and message. */
export function ProposalQueue({ proposals, openCount, isLoading = false, error, onOpen }: ProposalQueueProps) {
  const [collapsed, setCollapsed] = useState(false);
  if (!isLoading && !error && proposals.length === 0) return null;
  return (
    <section className="rounded-lg border border-slate-800 bg-slate-950/40" aria-label="Commit proposals" data-testid="proposal-queue">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm font-medium text-slate-200 hover:bg-slate-900/60"
        onClick={() => setCollapsed((value) => !value)}
        aria-expanded={!collapsed}
      >
        {collapsed ? <ChevronRight className="h-3 w-3 text-slate-400" /> : <ChevronDown className="h-3 w-3 text-slate-400" />}
        <GitPullRequestDraft className="h-4 w-4 text-blue-300" />
        <span>Proposals ({openCount})</span>
        {isLoading && <Loader2 className="ml-auto h-3 w-3 animate-spin text-slate-500" />}
      </button>
      {!collapsed && (
        <div className="max-h-64 space-y-2 overflow-y-auto px-2 pb-2">
          {error && <p className="px-1 text-xs text-red-400" role="alert">Proposals unavailable: {error.message}</p>}
          {proposals.map((proposal) => (
            <ProposalCard key={proposal.id} proposal={proposal} onOpen={() => onOpen(proposal.id)} />
          ))}
        </div>
      )}
    </section>
  );
}

function ProposalCard({ proposal, onOpen }: { proposal: Proposal; onOpen: () => void }) {
  const freshness = freshnessMeta(proposal.freshness?.state);
  const work = proposal.work;
  return (
    <button
      type="button"
      onClick={onOpen}
      className="w-full rounded-md border border-slate-800 bg-slate-900/60 p-2.5 text-left transition-colors hover:border-slate-600 hover:bg-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
      data-testid="proposal-card"
      aria-label={`Open proposal: ${proposal.message?.subject ?? proposal.id}`}
    >
      <div className="flex items-start justify-between gap-2">
        <p className="min-w-0 flex-1 truncate text-sm font-medium text-slate-100" title={proposal.message?.subject}>
          {proposal.message?.subject || "(no subject)"}
        </p>
        <Badge variant={freshness.tone} title={freshness.hint} data-testid="proposal-freshness">{freshness.label}</Badge>
      </div>
      <div className="mt-1.5 flex flex-wrap items-center gap-1.5 text-[11px]">
        {work?.epoch && <span className="rounded border border-blue-900/70 bg-blue-950/40 px-1.5 py-0.5 font-mono text-blue-200">{work.epoch}</span>}
        {work?.effortRef && (
          <span className="max-w-[12rem] truncate rounded border border-slate-700 bg-slate-950 px-1.5 py-0.5 text-slate-300" title={work.effortRef}>
            {effortLabel(work.effortRef)}
          </span>
        )}
        <span className="inline-flex items-center gap-1 text-slate-400">
          <FileCode2 className="h-3 w-3" />
          {proposal.files.length} file{proposal.files.length === 1 ? "" : "s"}
        </span>
        {flagCounts(proposal).map(({ code, count }) => {
          const meta = flagMeta(code);
          return (
            <Badge key={code} variant={meta.tone} className="px-1.5 text-[10px]" title={meta.hint}>
              {meta.label} {count}
            </Badge>
          );
        })}
      </div>
    </button>
  );
}
