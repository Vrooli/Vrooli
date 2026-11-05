import type { RemediationJob } from "../../lib/api";
import { Button } from "../ui/button";

interface RemediationJobDetailsProps {
  job: RemediationJob;
  onRetry?: (id: string) => void;
  retrying?: boolean;
}

const terminalRetryable = new Set(["failed", "cancelled"]);

// RemediationJobDetails is intentionally evidence-first: it renders the stored
// source and attempt timeline rather than inferring success from the current
// filesystem or a fresh finding scan.
export function RemediationJobDetails({ job, onRetry, retrying = false }: RemediationJobDetailsProps) {
  const requirementDelta = job.verification?.requirementDelta;
  const findingDelta = job.verification?.delta;
  return <article className="mt-4 rounded-lg border border-white/10 p-3 text-sm" aria-label={`Remediation job ${job.id}`}>
    <div className="flex flex-wrap items-start justify-between gap-3"><div><p className="text-xs uppercase tracking-wide text-slate-400">Remediation job</p><strong>{job.status.replace(/_/g, " ")}</strong><p className="mt-1 break-all text-xs text-slate-400">{job.id}</p></div>{terminalRetryable.has(job.status) && onRetry && <Button variant="outline" onClick={() => onRetry(job.id)} disabled={retrying}>Retry remediation</Button>}</div>
    <dl className="mt-3 grid gap-2 text-xs text-slate-300 md:grid-cols-2"><div><dt className="text-slate-500">Source execution</dt><dd className="break-all">{job.source?.sourceExecutionId || "unavailable"}</dd></div><div><dt className="text-slate-500">Source run</dt><dd className="break-all">{job.source?.sourceRunId || "unavailable"}</dd></div><div><dt className="text-slate-500">Selected findings</dt><dd>{job.selectedFindingIds?.join(", ") || "none"}</dd></div><div><dt className="text-slate-500">Selected requirements</dt><dd>{job.selectedRequirementIds?.join(", ") || "none"}</dd></div></dl>
    {job.attempts?.length ? <div className="mt-3"><p className="text-xs uppercase tracking-wide text-slate-400">Attempt timeline</p><ol className="mt-2 grid gap-2">{job.attempts.map((attempt) => <li key={attempt.id} className="rounded border border-white/10 px-2 py-1 text-xs"><strong>{attempt.kind} {attempt.state}</strong>{attempt.runId && <span className="ml-2 text-slate-400">run {attempt.runId}</span>}{attempt.detail && <span className="mt-1 block text-slate-300">{attempt.detail}</span>}</li>)}</ol></div> : <p className="mt-3 text-xs text-amber-200">No durable attempt history is available for this historical job.</p>}
    {job.verification && <div className="mt-3"><p className="text-xs uppercase tracking-wide text-slate-400">Verification</p>{job.verification.executionId && <p className="mt-1 text-xs text-slate-300">Execution {job.verification.executionId}</p>}<p className="mt-1 text-xs text-slate-300">Findings: {findingDelta?.resolved?.length ?? 0} resolved, {findingDelta?.remaining?.length ?? 0} remaining, {findingDelta?.unverifiable?.length ?? 0} unverifiable</p><p className="mt-1 text-xs text-slate-300">Requirements: {requirementDelta?.resolved?.length ?? 0} verified, {requirementDelta?.remaining?.length ?? 0} remaining, {requirementDelta?.unverifiable?.length ?? 0} unverifiable</p>{job.verification.degraded && <p className="mt-1 text-xs text-amber-200">Verification degraded: {job.verification.degraded}</p>}</div>}
  </article>;
}
