import { useMemo, useState } from "react";
import {
  planWorkshopService,
  type PlanWorkshopSession,
  type PlanWorkshopSubject,
} from "../../services/plan-workshop-service";

interface PlanWorkshopPanelProps {
  subject: PlanWorkshopSubject;
  disabled?: boolean;
}

function responseKey() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

// The visible workshop is deliberately a single response surface. Agent
// Sessions remain available for open-ended discussion; their proposal records
// are referenced here rather than copied into a separate store.
export function PlanWorkshopPanel({ subject, disabled = false }: PlanWorkshopPanelProps) {
  const [session, setSession] = useState<PlanWorkshopSession | null>(null);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [accepted, setAccepted] = useState<Record<string, boolean>>({});
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const selectedProposals = useMemo(
    () => (session?.packet.proposals ?? []).filter((proposal) => accepted[`${proposal.session_id}/${proposal.proposal_id}`]),
    [accepted, session?.packet.proposals],
  );

  const open = async () => {
    setPending(true);
    setError(null);
    try {
      const next = await planWorkshopService.open(subject);
      setSession(next);
      setAnswers({});
      setAccepted({});
      setNotice(next.packet.findings?.length || next.packet.questions?.length || next.packet.proposals?.length
        ? "Loaded the current review packet."
        : "Session opened. Start or continue an Agent Session for this subject to add review findings and proposals.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to open the Plan Workshop.");
    } finally {
      setPending(false);
    }
  };

  const submit = async () => {
    if (!session) return;
    setPending(true);
    setError(null);
    try {
      const result = await planWorkshopService.submitResponse(session.id, {
        actor: "operator",
        subject_version: session.subject_version,
        answers,
        accepted_proposals: selectedProposals,
        idempotency_key: responseKey(),
      });
      setSession(result.session);
      const state = result.resolution.state;
      setNotice(state === "reconciliation_required"
        ? "Your response started one reconciliation engagement. Its progress is retained in this workshop."
        : state === "direct_applied"
          ? "Your response was recorded and applied."
          : state === "stale"
            ? "This review packet is stale. Open the workshop again for the current subject version."
            : "Your response was recorded, but a required integration is unavailable.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to submit the Plan Workshop response.");
    } finally {
      setPending(false);
    }
  };

  const startReview = async () => {
    if (!session) return;
    setPending(true);
    setError(null);
    try {
      const result = await planWorkshopService.startReview(session.id);
      setSession(result.session);
      setNotice(result.review.state === "running"
        ? "Review is running. When the Agent Manager run completes, apply its typed result here."
        : result.review.state === "stale"
          ? "This workshop is stale. Reopen it to capture the current subject and plan."
          : result.review.error || "Unable to start the review.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to start the Plan Workshop review.");
    } finally {
      setPending(false);
    }
  };

  const applyReview = async () => {
    if (!session) return;
    setPending(true);
    setError(null);
    try {
      const result = await planWorkshopService.applyReview(session.id);
      setSession(result.session);
      setAnswers({});
      setAccepted({});
      setNotice(result.review.state === "applied"
        ? "The typed review packet is ready for your decisions."
        : result.review.state === "stale"
          ? "This workshop changed while the review was running. Reopen it before trying again."
          : result.review.error || "The review result is not ready to apply yet.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to apply the Plan Workshop review.");
    } finally {
      setPending(false);
    }
  };

  const applyReconciliation = async (responseId: string) => {
    if (!session) return;
    setPending(true);
    setError(null);
    try {
      const result = await planWorkshopService.applyReconciliation(session.id, responseId);
      setSession(result.session);
      setNotice(result.resolution.state === "candidate_ready"
        ? `Candidate ${result.resolution.candidate?.id ?? "revision"} is ready for Plan Manager preview and guarded application.`
        : result.resolution.state === "needs_attention"
          ? result.resolution.error || "Reconciliation needs attention before it can create a candidate."
          : result.resolution.state === "stale"
            ? "The canonical plan changed while reconciliation was running. Reopen the workshop."
            : "The reconciliation result is not ready to apply yet.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to apply the Plan Workshop reconciliation.");
    } finally {
      setPending(false);
    }
  };

  const applyCandidate = async (responseId: string, candidateId: string) => {
    if (!session || !window.confirm(`Apply candidate ${candidateId} to the canonical plan? This acknowledges its quality impact.`)) return;
    setPending(true);
    setError(null);
    try {
      const result = await planWorkshopService.applyCandidate(session.id, responseId);
      setSession(result.session);
      setNotice("Candidate applied to the canonical plan. Any prior plan acceptance was cleared and must be renewed.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to apply the Plan Manager candidate.");
    } finally {
      setPending(false);
    }
  };

  const acceptPlan = async () => {
    const [kind, name] = subject.ref.split("/", 2);
    if (!kind || !name) {
      setError("This backlog workshop does not have a valid item reference for plan acceptance.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      await planWorkshopService.acceptPlan(kind, name, session?.plan_content_hash);
      setNotice("Canonical plan accepted for this work item. Queueing will recheck the accepted revision and scope.");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to accept the current canonical plan.");
    } finally {
      setPending(false);
    }
  };

  const discardCandidate = async (responseId: string, candidateId: string) => {
    setPending(true);
    setError(null);
    try {
      const result = await planWorkshopService.discardCandidate(session!.id, responseId);
      setSession(result.session);
      setNotice(`Candidate ${candidateId} was ignored. The canonical plan was not changed.`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to ignore the Plan Manager candidate.");
    } finally {
      setPending(false);
    }
  };

  return (
    <section className="rounded-lg border border-violet-500/25 bg-violet-500/5 p-4" data-testid="plan-workshop-panel">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-violet-100">Plan Workshop</h3>
          <p className="mt-1 text-xs text-slate-400">Review findings, decisions, and proposals together before accepting the canonical plan.</p>
        </div>
        {!session && (
          <button className="rounded-md bg-violet-400 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-violet-300 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void open()}>
            {pending ? "Opening…" : "Open Plan Workshop"}
          </button>
        )}
      </div>
      {error && <p className="mt-3 text-xs text-rose-300">{error}</p>}
      {notice && <p className="mt-3 text-xs text-slate-300">{notice}</p>}
      {session && (
        <div className="mt-4 space-y-4">
          {session.legacy_history && <p className="rounded-md border border-slate-700/70 bg-slate-950/30 p-3 text-xs text-slate-400">Historical workshop: {session.legacy_history.round_count} archived round{session.legacy_history.round_count === 1 ? "" : "s"} remain read-only at <code>{session.legacy_history.source_path}</code>.</p>}
          {!session.packet.findings?.length && !session.packet.questions?.length && !session.packet.proposals?.length && session.review?.state !== "running" && (
            <button className="rounded-md bg-violet-400 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-violet-300 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void startReview()}>
              {pending ? "Starting…" : "Run plan review"}
            </button>
          )}
          {session.review?.state === "running" && (
            <div className="flex flex-wrap items-center gap-3 rounded-md border border-sky-500/25 bg-sky-500/5 p-3">
              <p className="text-xs text-sky-100">Review run {session.review.workflow?.execution_id ? `(${session.review.workflow.execution_id}) ` : ""}is in progress.</p>
              <button className="rounded-md border border-sky-400/50 px-3 py-1.5 text-sm text-sky-100 hover:bg-sky-400/10 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void applyReview()}>
                {pending ? "Applying…" : "Apply completed review"}
              </button>
            </div>
          )}
          {session.review?.state === "failed" && <p className="text-xs text-rose-300">Review failed: {session.review.error || "unknown error"}</p>}
          {(session.packet.findings ?? []).map((finding) => (
            <div key={finding.id} className="rounded-md border border-slate-700/70 bg-slate-950/30 p-3">
              <p className="text-xs font-medium uppercase tracking-wide text-violet-300">{finding.severity || "finding"}</p>
              <p className="mt-1 text-sm text-slate-200">{finding.summary}</p>
              {finding.evidence && <p className="mt-1 text-xs text-slate-400">{finding.evidence}</p>}
              {finding.disposition && <p className="mt-1 text-xs text-amber-200">Recommended: {finding.disposition.kind.replaceAll("_", " ")} ({finding.disposition.confidence} confidence). {finding.disposition.rationale}{finding.disposition.scope ? ` Scope: ${finding.disposition.scope}` : ""}</p>}
            </div>
          ))}
          {(session.packet.questions ?? []).map((question) => (
            <label key={question.id} className="block text-sm text-slate-200">
              <span>{question.question}</span>
              {question.options?.length ? (
                <select className="mt-1 block w-full rounded-md border border-slate-700 bg-slate-950 px-2 py-1.5 text-sm" value={answers[question.id] ?? ""} onChange={(event) => setAnswers((current) => ({ ...current, [question.id]: event.target.value }))}>
                  <option value="">Choose an answer…</option>
                  {question.options.map((option) => <option key={option} value={option}>{option}</option>)}
                </select>
              ) : (
                <input className="mt-1 block w-full rounded-md border border-slate-700 bg-slate-950 px-2 py-1.5 text-sm" value={answers[question.id] ?? ""} onChange={(event) => setAnswers((current) => ({ ...current, [question.id]: event.target.value }))} />
              )}
            </label>
          ))}
          {(session.packet.proposals ?? []).map((proposal) => {
            const key = `${proposal.session_id}/${proposal.proposal_id}`;
            return <label key={key} className="flex cursor-pointer items-center gap-2 text-sm text-slate-200"><input type="checkbox" checked={!!accepted[key]} onChange={(event) => setAccepted((current) => ({ ...current, [key]: event.target.checked }))} /> Accept proposal {proposal.proposal_id}{proposal.apply_mode ? ` (${proposal.apply_mode.replaceAll("_", " ")})` : ""}</label>;
          })}
          {(session.packet.findings?.length || session.packet.questions?.length || session.packet.proposals?.length) ? (
            <button className="rounded-md bg-violet-400 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-violet-300 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void submit()}>{pending ? "Submitting…" : "Submit workshop response"}</button>
          ) : null}
          {session.resolutions?.at(-1) ? (
            <div className="flex flex-wrap items-center gap-3 text-xs text-slate-400">
              <p>Latest resolution: {session.resolutions.at(-1)?.state.replaceAll("_", " ")}</p>
              {session.resolutions.at(-1)?.state === "reconciliation_required" && (
                <button className="rounded-md border border-violet-400/50 px-2.5 py-1 text-xs text-violet-100 hover:bg-violet-400/10 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void applyReconciliation(session.resolutions!.at(-1)!.response_id)}>
                  {pending ? "Applying…" : "Apply completed reconciliation"}
                </button>
              )}
              {session.resolutions.at(-1)?.candidate && <span>Candidate: {session.resolutions.at(-1)?.candidate?.id} ({session.resolutions.at(-1)?.candidate?.quality_status || "quality pending"})</span>}
              {session.resolutions.at(-1)?.state === "candidate_ready" && session.resolutions.at(-1)?.candidate && (
                <div className="w-full space-y-2 rounded-md border border-amber-400/20 bg-slate-950/30 p-3 text-xs text-slate-300" data-testid="plan-workshop-candidate-preview">
                  <p className="font-medium text-amber-100">Candidate review: inspect before choosing apply or ignore.</p>
                  <p>Quality impact: {session.resolutions.at(-1)!.candidate!.impact?.before_grade || "unknown"} → {session.resolutions.at(-1)!.candidate!.impact?.after_grade || session.resolutions.at(-1)!.candidate!.quality_status || "pending"}{session.resolutions.at(-1)!.candidate!.impact?.execution_grade_regression ? " (execution-grade regression)" : ""}</p>
                  <div>
                    <p className="font-medium text-slate-200">Structured diff</p>
                    {session.resolutions.at(-1)!.candidate!.diff?.length ? (
                      <ul className="mt-1 space-y-1">
                        {session.resolutions.at(-1)!.candidate!.diff!.map((change) => <li key={change.field}><span className="text-violet-200">{change.field}</span>: <code>{change.before_json}</code> → <code>{change.after_json}</code></li>)}
                      </ul>
                    ) : <p className="mt-1 text-slate-500">No authored field changes were reported.</p>}
                  </div>
                  <div>
                    <p className="font-medium text-slate-200">Validation</p>
                    {session.resolutions.at(-1)!.candidate!.diagnostics?.length ? (
                      <ul className="mt-1 space-y-1">{session.resolutions.at(-1)!.candidate!.diagnostics!.map((diagnostic, index) => <li key={`${diagnostic.code}-${index}`}>[{diagnostic.severity}] {diagnostic.message}{diagnostic.guidance ? ` — ${diagnostic.guidance}` : ""}</li>)}</ul>
                    ) : <p className="mt-1 text-emerald-200">No validation diagnostics reported.</p>}
                  </div>
                </div>
              )}
              {session.resolutions.at(-1)?.state === "candidate_ready" && session.resolutions.at(-1)?.candidate && (
                <>
                  <button className="rounded-md border border-amber-400/50 px-2.5 py-1 text-xs text-amber-100 hover:bg-amber-400/10 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void applyCandidate(session.resolutions!.at(-1)!.response_id, session.resolutions!.at(-1)!.candidate!.id)}>
                    Apply candidate to canonical plan
                  </button>
                  <button className="rounded-md border border-slate-500/50 px-2.5 py-1 text-xs text-slate-200 hover:bg-slate-500/10 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void discardCandidate(session.resolutions!.at(-1)!.response_id, session.resolutions!.at(-1)!.candidate!.id)}>
                    Ignore candidate
                  </button>
                </>
              )}
            </div>
          ) : null}
          {subject.kind === "backlog_item" && session?.plan_id && (
            <button className="rounded-md border border-sky-400/50 px-2.5 py-1 text-xs text-sky-100 hover:bg-sky-400/10 disabled:cursor-not-allowed disabled:opacity-50" disabled={disabled || pending} onClick={() => void acceptPlan()}>
              Accept current canonical plan
            </button>
          )}
        </div>
      )}
    </section>
  );
}
