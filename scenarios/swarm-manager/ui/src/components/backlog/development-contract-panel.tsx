import { useEffect, useState } from "react";
import { fromJsonString, toJsonString } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { PreviewDevelopmentRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import type { PreviewDevelopmentRequest, PreviewDevelopmentResponse } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import type { DevelopmentResponse } from "@vrooli/proto-types/swarm-manager/v1/api/development_pb";
import { developmentService, type DevelopmentClient } from "../../services/development-service";
import { Button } from "../ui/button";
import { DevelopmentGoalDrawer } from "./development-goal-drawer";
import { DevelopmentAmendmentReview } from "./development-amendment-review";
import type { PlanRef } from "../../types";

type Reviewed = { proposal: PreviewDevelopmentRequest; review: PreviewDevelopmentResponse };
type PanelProps = { workItem: string; planRef?: PlanRef; readOnly?: boolean; client?: DevelopmentClient };

/** Operator review surface. No execution method exists on this component's client. */
export function DevelopmentContractPanel(props: PanelProps) {
  // A review/rationale belongs to one item. Navigation must never carry a
  // pending response or decision control over to a different item's drawer.
  return <DevelopmentContractItemPanel key={props.workItem} {...props} />;
}

function DevelopmentContractItemPanel({ workItem, planRef, readOnly = false, client = developmentService }: PanelProps) {
  const [open, setOpen] = useState(false);
  const [state, setState] = useState<DevelopmentResponse>();
  const [draft, setDraft] = useState("");
  const [reviewed, setReviewed] = useState<Reviewed>();
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [artifact, setArtifact] = useState<{ path: string; content: string }>();
  const [refresh, setRefresh] = useState(0);
  const [configuration, setConfiguration] = useState<PreviewDevelopmentRequest>();
  const [acknowledgedAmendment, setAcknowledgedAmendment] = useState("");

  useEffect(() => {
    if (!open) return;
    let active = true;
    setLoaded(false);
    setError("");
    setState(undefined);
    setReviewed(undefined);
    setArtifact(undefined);
    setConfiguration(undefined);
    setAcknowledgedAmendment("");
    client.getDevelopment({ workItem }).then((value) => {
      if (!active) return;
      setState(value);
      if (value.approvedProposal) setDraft(toJsonString(PreviewDevelopmentRequestSchema, value.approvedProposal, { prettySpaces: 2, useProtoFieldName: true }));
      setLoaded(true);
    }).catch((failure: unknown) => {
      if (!active) return;
      if (ConnectError.from(failure).code === Code.NotFound) {
        setDraft(JSON.stringify({ scenario: "", work_item: workItem, objective: "", max_tokens: "0", max_wall_seconds: "0", execution_strategy: "adaptive-improvement", ...(planRef ? { plan_ref: { provider: planRef.provider, plan_id: planRef.planId, slug: planRef.slug, role: planRef.role } } : {}) }, null, 2));
        setLoaded(true);
      } else setError(ConnectError.from(failure).message);
    });
    return () => { active = false; };
  }, [open, workItem, client, refresh]);

  async function perform(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    try { await action(); } catch (failure) { setError(ConnectError.from(failure).message); }
    finally { setBusy(false); }
  }

  const preview = () => perform(async () => {
    setReviewed(undefined);
    const proposal = fromJsonString(PreviewDevelopmentRequestSchema, draft);
    if (proposal.workItem !== workItem) throw new Error("Proposal must name this backlog item.");
    const review = await client.previewDevelopment(proposal);
    setReviewed({ proposal, review });
  });
  const approve = () => perform(async () => {
    if (!reviewed?.review.reviewComplete || !reason.trim()) return;
    if (state && acknowledgedAmendment !== amendmentKey) return;
    if (reviewed.proposal.workItem !== workItem) throw new Error("Proposal must name this backlog item.");
    const value = await client.approveDevelopment({ proposal: reviewed.proposal, reviewedDigest: reviewed.review.proposalDigest, expectedVersion: state?.version ?? 0n, reason });
    setState(value);
    setReviewed(undefined);
    setReason("");
  });
  const revoke = () => perform(async () => {
    if (!state || !reason.trim()) return;
    setState(await client.revokeDevelopment({ workItem, expectedVersion: state.version, reason }));
    setReason("");
  });
  const review = reviewed?.review;
  const blockers = review?.launchBlockers ?? state?.launchBlockers ?? [];
  const amendmentKey = state && review ? `${state.version}:${review.proposalDigest}` : "";

  return <section className="space-y-3 rounded-lg border border-slate-700 p-3" aria-label="Adaptive plan">
    <button type="button" className="text-sm font-semibold text-slate-200" aria-expanded={open} onClick={() => setOpen(!open)}>
      Adaptive plan {open ? "▾" : "▸"}
    </button>
    {open && <div className="space-y-3 text-sm">
      <p>Review and retain the adaptive execution strategy for this canonical plan. One approval can authorize successive repairs. Approval does not launch an agent; product acceptance requires outcome evidence.</p>
      <p>Decisions require verified human authentication and the <code>swarm-manager:write</code> capability. An agent identity or an anonymous local request cannot approve work.</p>
      <Button size="sm" variant="outline" disabled={busy} onClick={() => setRefresh((value) => value + 1)}>Reload retained state</Button>
      {error && <p role="alert" className="text-red-300">{error}</p>}
      {!loaded && !error && <p role="status">Loading retained approval…</p>}
      {state && <div className="space-y-2">
        <p>Strategy: adaptive improvement · Status: {state.status} · Version: {state.version.toString()}</p>
        {state.approvedProposal?.planRef && <p className="break-all">Canonical plan: {state.approvedProposal.planRef.slug || state.approvedProposal.planRef.planId} ({state.approvedProposal.planRef.planId})</p>}
        <p className="break-all">Approved target: {state.digest}</p>
        <p>Tokens incurred / reserved / limit: {state.used?.tokens.toString() ?? "0"} / {state.reserved?.tokens.toString() ?? "0"} / {state.approvedProposal?.maxTokens.toString()}</p>
        <p>Wall seconds incurred / reserved / limit: {state.used?.wallSeconds.toString() ?? "0"} / {state.reserved?.wallSeconds.toString() ?? "0"} / {state.approvedProposal?.maxWallSeconds.toString()}</p>
        {state.stopReason && <p role="status">{state.stopReason}</p>}
        {state.checkpoint && <pre className="whitespace-pre-wrap">{state.checkpoint}</pre>}
        {state.campaign && <details open><summary>Campaign checkpoint</summary><div className="space-y-1 break-all">
          <p>Approval: {state.campaign.approvalDigest} · {state.campaign.pending ? "owner operation pending" : "owner operation settled"}</p>
          <p>Attempt: {state.campaign.attemptKey || "none"} · Owner: {state.campaign.ownerExecutionId || "unbound"}</p>
          <p>Remaining outcomes: {state.campaign.remainingOutcomeIds.join(", ") || "none"}</p>
          {state.campaign.lastCheckpoint && <p>Last checkpoint: {state.campaign.lastCheckpoint.kind} · {state.campaign.lastCheckpoint.value}</p>}
          {state.campaign.noProgressCycles > 0 && <p role="status">No useful progress observed for {state.campaign.noProgressCycles} settled cycle(s); remaining outcomes stay authorized work.</p>}
        </div></details>}
        {state.cancellation && <p role="status">Cancellation {state.cancellation.state}: {state.cancellation.reason}</p>}
        <details><summary>Approval history ({state.approvals.length})</summary><ul>
          {state.approvals.map((entry, index) => <li key={`${entry.digest}-${index}`} className="break-all">{entry.actor}: {entry.reason} · {entry.digest}</li>)}
        </ul></details>
        <details><summary>Attempts ({state.attempts.length}) and accepted evidence ({state.evidence.length})</summary><ul>
          {state.attempts.map((attempt) => <li key={attempt.key}>{attempt.key}: {attempt.mode} · {attempt.executionId || "owner response unresolved"} · {attempt.settledAt ? "settled" : "reservation held"}</li>)}
          {state.evidence.map((entry) => <li key={entry.outcomeId}>{entry.outcomeId}: {entry.receiptId} · product revision {entry.subjectRevision}</li>)}
        </ul></details>
        <details><summary>Retained target artifacts ({state.artifacts.length})</summary><ul>
          {state.artifacts.map((entry) => <li key={entry.path}>
            <button type="button" className="text-left text-blue-300 underline break-all" disabled={busy} onClick={() => perform(async () => {
              const value = await client.getDevelopmentArtifact({ workItem, digest: state.digest, path: entry.path });
              setArtifact({ path: entry.path, content: new TextDecoder().decode(value.content) });
            })}>{entry.path}</button> <span className="break-all">sha256:{entry.sha256}</span>
          </li>)}
        </ul></details>
        {artifact && <details open><summary>{artifact.path} — retained bytes</summary><pre className="max-h-96 overflow-auto whitespace-pre-wrap">{artifact.content}</pre></details>}
      </div>}
      {loaded && !readOnly && <>
        <Button size="sm" variant="outline" disabled={busy} onClick={() => {
          try {
            const proposal = fromJsonString(PreviewDevelopmentRequestSchema, draft);
            if (proposal.workItem !== workItem) throw new Error("Proposal must name this backlog item.");
            setConfiguration(proposal); setError("");
          }
          catch (failure) { setError(ConnectError.from(failure).message); }
        }}>Configure adaptive strategy</Button>
        <label className="block">Proposal JSON
          <textarea className="mt-1 h-52 w-full rounded bg-slate-950 p-2 font-mono text-xs" value={draft} disabled={busy} onChange={(event) => { setDraft(event.target.value); setReviewed(undefined); }} />
        </label>
        <Button size="sm" disabled={busy} onClick={preview}>Preview plan strategy and goal message</Button>
        <label className="block">Decision rationale
          <textarea className="mt-1 w-full rounded bg-slate-950 p-2" value={reason} disabled={busy} onChange={(event) => setReason(event.target.value)} />
        </label>
        {state?.approvedProposal && reviewed && <>
          <DevelopmentAmendmentReview previous={state.approvedProposal} proposed={reviewed.proposal} retained={state.artifacts} resolved={reviewed.review.artifacts} />
          <label className="flex gap-2"><input type="checkbox" disabled={busy} checked={acknowledgedAmendment === amendmentKey} onChange={(event) => setAcknowledgedAmendment(event.target.checked ? amendmentKey : "")} />I reviewed these changes and the new goal message</label>
        </>}
        <div className="flex flex-wrap gap-2">
          <Button size="sm" disabled={busy || !review?.reviewComplete || !reason.trim() || (!!state && acknowledgedAmendment !== amendmentKey)} onClick={approve}>{state ? "Approve reviewed amendment (no launch)" : "Approve reviewed target (no launch)"}</Button>
          {state && state.status !== "accepted" && <Button size="sm" variant="outline" disabled={busy || !reason.trim()} onClick={revoke}>Revoke new dispatch</Button>}
        </div>
      </>}
      {review && <><p className="break-all">Reviewed fingerprint: {review.proposalDigest}</p><ul>{review.findings.map((finding, index) => <li key={index}>{finding.code}: {finding.detail}</li>)}</ul></>}
      {(review?.goalMessage || state?.goalMessage) && <details open><summary>Goal message for review</summary><pre className="max-h-96 overflow-auto whitespace-pre-wrap">{review?.goalMessage ?? state?.goalMessage}</pre></details>}
      {!!blockers.length && <div role="status"><p>Launch blocked</p><ul className="list-disc pl-5">{blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}</ul></div>}
      {configuration && <DevelopmentGoalDrawer initial={configuration} client={client} onClose={() => setConfiguration(undefined)} onReviewed={(proposal, goalReview) => {
        setDraft(toJsonString(PreviewDevelopmentRequestSchema, proposal, { prettySpaces: 2, useProtoFieldName: true }));
        setReviewed({ proposal, review: goalReview });
        setConfiguration(undefined);
      }} />}
    </div>}
  </section>;
}
