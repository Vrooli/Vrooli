import { memo } from "react";
import { ExternalLink, FileText, GitPullRequestArrow, Layers3 } from "lucide-react";
import { formatDisplayText } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import type { AgentSessionArtifact, AgentSessionProposal, AgentSessionProposalTarget } from "../../types";
import { nodeIdForSessionArtifact } from "./session-artifact-routing";

interface SessionArtifactListProps {
  artifacts: AgentSessionArtifact[];
  proposals?: AgentSessionProposal[];
  proposalTarget?: AgentSessionProposalTarget;
  onOpenArtifact: (artifact: AgentSessionArtifact) => void;
  onOpenProposal?: (proposal: AgentSessionProposal) => void;
  onApplyProposal?: (proposal: AgentSessionProposal) => void;
  applyingProposalId?: string;
  variant?: "panel" | "plain";
}

interface StartTransitionPayload {
  transitionKey: string;
  subjectRef: string;
  projectionAction: string;
  projectionAgrees: boolean;
  reason?: string;
}

function parseStartTransitionPayload(payloadJson: string): StartTransitionPayload | null {
  try {
    const value = JSON.parse(payloadJson) as Record<string, unknown>;
    if (typeof value.transition_key !== "string" || typeof value.subject_ref !== "string" || typeof value.projection_action !== "string") {
      return null;
    }
    return {
      transitionKey: value.transition_key,
      subjectRef: value.subject_ref,
      projectionAction: value.projection_action,
      projectionAgrees: value.projection_agrees === true,
      reason: typeof value.reason === "string" ? value.reason : undefined,
    };
  } catch {
    return null;
  }
}

function SessionArtifactListImpl({
  artifacts,
  proposals = [],
  proposalTarget,
  onOpenArtifact,
  onOpenProposal,
  onApplyProposal,
  applyingProposalId,
  variant = "panel",
}: SessionArtifactListProps) {
  return (
    <section className={cn(variant === "panel" && "rounded-lg border border-white/10 bg-slate-950/30 p-3")} data-testid="agent-session-artifacts">
      <div className="flex items-center gap-2">
        <Layers3 className="h-4 w-4 text-cyan-300" />
        <h4 className="text-xs font-medium text-slate-300">Artifacts</h4>
        <span className="ml-auto text-[11px] text-slate-500">{artifacts.length + proposals.length}</span>
      </div>
      <div className="mt-3 space-y-2">
        {proposals.length > 0 && (
          <div className="space-y-2">
            <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Proposals</p>
            {proposals.map((proposal) => {
              const canOpen = Boolean(proposalTarget && onOpenProposal);
              if (proposal.kind === "start_transition") {
                const transition = parseStartTransitionPayload(proposal.payloadJson);
                const disagrees = transition?.projectionAgrees === false;
                const canApply = proposal.status === "ready" && Boolean(onApplyProposal) && !applyingProposalId;
                return (
                  <div
                    key={proposal.id}
                    className={cn(
                      "rounded-md border bg-violet-500/5 p-3",
                      disagrees ? "border-amber-400/40" : "border-violet-400/20",
                    )}
                    data-testid="agent-session-start-transition-proposal"
                  >
                    <div className="flex items-start gap-2">
                      <GitPullRequestArrow className="mt-0.5 h-3.5 w-3.5 shrink-0 text-violet-300" />
                      <div className="min-w-0 flex-1">
                        <p className="text-xs font-medium text-slate-100">Start registered transition</p>
                        <p className="mt-1 break-all text-[11px] text-slate-300">{transition?.transitionKey ?? "Invalid transition payload"}</p>
                        <p className="mt-0.5 break-all text-[11px] text-slate-500">Subject: {transition?.subjectRef ?? "unknown"}</p>
                        <p className="mt-1 text-[11px] text-slate-400" data-testid="agent-session-transition-verdicts">
                          Projection: {transition?.projectionAction ?? "unknown"} · Session: {transition?.projectionAgrees ? "agrees" : "disagrees"}
                        </p>
                        {disagrees && (
                          <p className="mt-1 text-[11px] text-amber-200" data-testid="agent-session-transition-disagreement">
                            Reason: {transition?.reason || "No reason supplied"}
                          </p>
                        )}
                        <div className="mt-3 flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => onApplyProposal?.(proposal)}
                            disabled={!canApply}
                            className="rounded border border-emerald-400/30 px-2 py-1 text-[11px] font-medium text-emerald-200 hover:bg-emerald-400/10 disabled:pointer-events-none disabled:opacity-50"
                            data-testid="agent-session-approve-transition"
                          >
                            {applyingProposalId === proposal.id ? "Approving…" : disagrees ? "Approve despite projection" : "Approve transition"}
                          </button>
                          {canOpen && (
                            <button
                              type="button"
                              onClick={() => onOpenProposal?.(proposal)}
                              className="rounded border border-white/10 px-2 py-1 text-[11px] text-slate-300 hover:bg-white/5"
                              data-testid="agent-session-open-projection"
                            >
                              Review projection
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              }
              return (
                <button
                  key={proposal.id}
                  type="button"
                  onClick={() => onOpenProposal?.(proposal)}
                  className="w-full rounded-md border border-violet-400/20 bg-violet-500/5 p-2 text-left transition-colors hover:border-violet-300/40 hover:bg-violet-500/10 disabled:pointer-events-none disabled:opacity-60"
                  disabled={!canOpen}
                  data-testid="agent-session-proposal-artifact"
                >
                  <div className="flex items-start gap-2">
                    <GitPullRequestArrow className="mt-0.5 h-3.5 w-3.5 shrink-0 text-violet-300" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium text-slate-100">{proposal.summary || formatDisplayText(proposal.kind)}</p>
                      <p className="mt-0.5 text-[11px] text-slate-500">
                        {formatDisplayText(proposal.kind)} · {formatDisplayText(proposal.status)}
                      </p>
                    </div>
                    {canOpen && <ExternalLink className="h-3.5 w-3.5 shrink-0 text-slate-500" />}
                  </div>
                </button>
              );
            })}
          </div>
        )}
        {artifacts.length > 0 && (
          <div className="space-y-2">
            {proposals.length > 0 && <p className="pt-2 text-[11px] font-medium uppercase tracking-wide text-slate-500">Other artifacts</p>}
            {artifacts.map((artifact) => {
            const canOpen = nodeIdForSessionArtifact(artifact) !== null;
            return (
              <button
                key={artifact.id}
                type="button"
                onClick={() => onOpenArtifact(artifact)}
                className="w-full rounded-md border border-white/10 bg-slate-900/70 p-2 text-left transition-colors hover:border-slate-700 hover:bg-slate-800/70 disabled:pointer-events-none disabled:opacity-60"
                disabled={!canOpen}
                data-testid="agent-session-artifact"
              >
                <div className="flex items-start gap-2">
                  <FileText className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-400" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium text-slate-100">{artifact.title || artifact.entityRef}</p>
                    <p className="mt-0.5 text-[11px] text-slate-500">
                      {formatDisplayText(artifact.action)} {formatDisplayText(artifact.artifactType)}
                    </p>
                  </div>
                  {canOpen && <ExternalLink className="h-3.5 w-3.5 shrink-0 text-slate-500" />}
                </div>
              </button>
            );
            })}
          </div>
        )}
        {artifacts.length === 0 && proposals.length === 0 && (
          <p className="py-8 text-center text-xs text-slate-500">No artifacts linked yet.</p>
        )}
      </div>
    </section>
  );
}

/**
 * Memoized: The artifact list is inspector content: nothing about it depends on the composer draft, yet it sat on the same render path as every keystroke.
 * Its props are stabilised at the call site in SessionDetailsPage.
 */
export const SessionArtifactList = memo(SessionArtifactListImpl);
