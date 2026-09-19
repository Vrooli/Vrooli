import { AlertTriangle, Check, CornerDownLeft, X } from "lucide-react";
import { useState } from "react";
import type { BacklogCriterion } from "@vrooli/proto-types/swarm-manager/v1/shared/backlog_pb";
import { Button } from "../../ui/button";
import { EvidenceItemCard } from "../evidence-item-card";
import { readOperatorIdentity, rememberOperatorIdentity } from "../../../lib/operator-identity";
import { summarizeCriteria, type SettlementState } from "../../../lib/review-settlement";
import { reviewDecisionService } from "../../../services/review-decision-service";
import { reviewService, type ReviewRound } from "../../../services/review-service";

type Decision = "accept" | "followup" | "fail";
type FollowUpDisposition = "follow_up_run" | "replan" | "new_items";

const SETTLEMENT_TONE: Record<SettlementState, string> = {
  settled: "text-emerald-200",
  refuted: "text-rose-200",
  pending: "text-amber-100",
  unsettled: "text-amber-100",
};

export function ReviewDecisionCard({
  kind, name, round, criteria = [], onDecided, onSendBack,
}: {
  kind: string;
  name: string;
  round: ReviewRound | undefined;
  criteria?: BacklogCriterion[];
  onDecided: () => void;
  onSendBack?: () => void;
}) {
  const [pending, setPending] = useState<Decision | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [actor, setActor] = useState(readOperatorIdentity);
  const [rationale, setRationale] = useState("");
  const [agree, setAgree] = useState(false);
  const [followUpDisposition, setFollowUpDisposition] = useState<FollowUpDisposition | "">("");

  if (!round || round.status === "pending" || round.status === "gathering") {
    return <section className="rounded-xl border border-amber-400/20 bg-amber-400/[0.06] p-4"><p className="text-sm font-medium text-amber-100">Review is gathering evidence</p><p className="mt-1 text-xs text-slate-300">The result will appear here when the review is ready for your decision.</p></section>;
  }

  const evidence = round.evidence ?? [];
  const unverified = evidence.filter((item) => !item.verified).length;
  const decide = async (decision: Decision) => {
    const resolvedRationale = rationale.trim() || (agree ? `Agree with the review's assessment: ${round.agent_assessment ?? "no assessment supplied"}` : "");
    if (!actor.trim() || !resolvedRationale) {
      setError("Operator identity and a rationale (or explicit agreement) are required.");
      return;
    }
    if (decision === "followup" && (!rationale.trim() || !followUpDisposition)) {
      setError("Send back requires steering text and a follow-up disposition.");
      return;
    }
    setPending(decision); setError(null);
    try {
      await reviewDecisionService.decide({
        kind, name, round: round.round, decision, actor: actor.trim(), rationale: resolvedRationale,
        ...(decision === "followup" ? { followUp: { steering: rationale.trim(), disposition: followUpDisposition as FollowUpDisposition } } : {}),
      });
      rememberOperatorIdentity(actor);
      onDecided();
      if (decision === "followup") onSendBack?.();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to save this review decision.");
    } finally { setPending(null); }
  };
  const verifyEvidence = async (evidenceId: string, verified: boolean) => {
		if (verified && (!actor.trim() || !rationale.trim())) {
			setError("Operator identity and a verification reason are required.");
			return;
		}
    try {
		await reviewService.verifyEvidence(kind, name, round.round, evidenceId, verified, round.execution_id, actor.trim(), rationale.trim());
      onDecided();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to update evidence verification.");
    }
  };
  const verdict = round.classification === "not_assessable" ? "Inconclusive" : round.classification?.replace(/-/g, " ") || "Review ready";
  const criterionSummary = summarizeCriteria(criteria, evidence);

  return <section className="rounded-xl border border-violet-400/25 bg-violet-400/[0.06] p-4" aria-label="Review decision">
    <div className="flex items-center gap-2"><AlertTriangle className="h-4 w-4 text-violet-200" /><h2 className="text-sm font-semibold text-white">Review decision required</h2><span className="rounded-full border border-violet-300/25 bg-violet-300/10 px-2 py-0.5 text-[11px] capitalize text-violet-100">{verdict}</span></div>
    <p className="mt-2 max-w-3xl text-sm text-slate-200">{round.agent_assessment || "Review evidence is ready for your judgment."}</p>
    <div className="mt-2 flex flex-wrap gap-2 text-xs text-slate-400"><span>{evidence.length} evidence item{evidence.length === 1 ? "" : "s"} · Review round {round.round}</span>{round.execution_id ? <a className="text-cyan-200 underline-offset-2 hover:underline" href={`/executions/${round.execution_id}?tab=changes`}>View changed files</a> : null}</div>
    {round.disposition ? <p className="mt-3 text-xs text-sky-100">Advisory: {round.disposition.kind.replace(/_/g, " ")} — {round.disposition.rationale}</p> : null}
    {criteria.length === 0 ? <p className="mt-3 rounded border border-amber-300/25 bg-amber-300/[0.08] p-2 text-xs text-amber-100">This item has no typed criteria. Define criteria before relying on this review as a definition of done.</p> : <>
      <div className="mt-3 overflow-x-auto rounded border border-slate-700"><table className="w-full text-left text-xs"><thead className="bg-slate-900 text-slate-300"><tr><th className="p-2">Criterion</th><th className="p-2">Settlement</th><th className="p-2">Evidence</th></tr></thead><tbody>{criterionSummary.rows.map((row) => <tr key={row.criterion.id} className="border-t border-slate-700 align-top"><td className="p-2 text-slate-100">{row.criterion.gherkin}</td><td className={`p-2 capitalize ${SETTLEMENT_TONE[row.state]}`}>{row.state}</td><td className="p-2 tabular-nums text-slate-300">{row.evidenceCount}</td></tr>)}</tbody></table></div>
      {criterionSummary.unsettled > 0 ? <p className="mt-2 rounded border border-amber-300/25 bg-amber-300/[0.08] p-2 text-xs text-amber-100">{criterionSummary.unsettled} of {criteria.length} criteria unsettled. Accepting closes them as delivered.</p> : <p className="mt-2 rounded border border-emerald-300/20 bg-emerald-300/[0.06] p-2 text-xs text-emerald-100">All {criteria.length} criteria settled by evidence.</p>}
    </>}
    {evidence.length > 0 ? <div className="mt-3 space-y-2">{evidence.map((item) => <EvidenceItemCard key={item.id} item={item} backlogKind={kind} backlogName={name} onVerify={(id, verified) => void verifyEvidence(id, verified)} />)}</div> : null}
    <div className="mt-4 grid gap-2"><input aria-label="Operator identity" value={actor} onChange={(event) => setActor(event.target.value)} placeholder="Operator identity" className="rounded border border-slate-600 bg-slate-950 px-2 py-1.5 text-xs text-slate-100" /><textarea aria-label="Decision rationale" value={rationale} onChange={(event) => setRationale(event.target.value)} placeholder="Your rationale or follow-up steering" className="min-h-16 rounded border border-slate-600 bg-slate-950 px-2 py-1.5 text-xs text-slate-100" /><label className="flex items-center gap-2 text-xs text-slate-300"><input type="checkbox" checked={agree} onChange={(event) => setAgree(event.target.checked)} />Agree with the review&apos;s assessment</label><label className="text-xs text-slate-300">Follow-up disposition<select aria-label="Follow-up disposition" value={followUpDisposition} onChange={(event) => setFollowUpDisposition(event.target.value as FollowUpDisposition | "")} className="ml-2 rounded border border-slate-600 bg-slate-950 p-1 text-slate-100"><option value="">Choose when sending back</option><option value="follow_up_run">Run follow-up</option><option value="replan">Replan</option><option value="new_items">Create new items</option></select></label></div>
    <div className="mt-4 flex flex-wrap gap-2"><Button size="sm" disabled={pending !== null} onClick={() => void decide("accept")}><Check className="mr-1.5 h-3.5 w-3.5" />{pending === "accept" ? "Accepting…" : `Accept (${unverified} unverified)`}</Button><Button size="sm" variant="outline" disabled={pending !== null} onClick={() => void decide("followup")}><CornerDownLeft className="mr-1.5 h-3.5 w-3.5" />{pending === "followup" ? "Sending back…" : "Send back"}</Button><Button size="sm" variant="destructive" disabled={pending !== null} onClick={() => void decide("fail")}><X className="mr-1.5 h-3.5 w-3.5" />{pending === "fail" ? "Failing…" : "Fail"}</Button></div>{error ? <p className="mt-3 text-xs text-rose-200">{error}</p> : null}
  </section>;
}
