import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Play, RefreshCw, RotateCw, Server } from "lucide-react";
import { confirmStagingRecovery, getStagingRecovery, getStagingState, listStagingOperations, previewStagingRecovery, requestStagingStart, resumeStagingOperation } from "../../api/deployments";
import type { StagingOperation } from "../../api/deployments";
import { SettingsIntro } from "./primitives";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";

export default function DeploymentsSection({ open }: { open: boolean }) {
  const queryClient = useQueryClient();
  const profile = useQuery({ queryKey: ["staging-profile"], queryFn: getStagingState, enabled: open });
  const operations = useQuery({ queryKey: ["staging-operations"], queryFn: listStagingOperations, enabled: open, refetchInterval: open ? 15000 : false });
  const start = useMutation({ mutationFn: requestStagingStart });
  const resume = useMutation({ mutationFn: resumeStagingOperation, onSuccess: () => queryClient.invalidateQueries({ queryKey: ["staging-operations"] }) });
  return <div data-testid="settings-deployments">
    <SettingsIntro eyebrow="Deployment" title="Staging deployment" description="Web Console staging is orchestrated by Deployment Manager. Provider credentials and instance setup remain owned by Compute Manager." />
    {profile.isLoading ? <p role="status" className="text-sm text-wc-text-muted">Loading staging profile…</p> : profile.error ? <div role="alert" className="rounded-xl border border-wc-danger/30 p-4 text-sm">Staging profile status is unavailable. Retry by reopening this section.</div> : profile.data && <section className="mb-5 rounded-xl border border-wc-default bg-wc-surface-base p-4" data-testid="staging-profile-status">
      <div className="flex items-start gap-3"><Server aria-hidden="true" className="mt-1 h-5 w-5 text-wc-accent"/><div className="min-w-0 flex-1">
        <h3 className="font-semibold text-wc-text-primary">{profile.data.profile.domain}</h3>
        <p className="mt-1 text-sm text-wc-text-secondary">{profile.data.profile.provider} {profile.data.profile.offer} · {profile.data.profile.regions.join(" or ")} · cap {profile.data.profile.cap_currency} {(profile.data.profile.cap_minor_units / 100).toFixed(2)}/month before tax</p>
        <p className="mt-1 text-xs text-wc-text-muted">Health check: {profile.data.profile.health_path} · {profile.data.profile.approval}</p>
      </div></div>
      {profile.data.readiness.state === "blocked" && <div className="mt-4 rounded-lg bg-wc-surface-input p-3" role="status" data-testid="staging-blocked">
        <p className="text-sm font-medium text-wc-text-primary">Waiting for Compute Manager</p>
        <ul className="mt-2 list-disc space-y-1 ps-5 text-sm text-wc-text-secondary">{profile.data.readiness.reasons.map((reason) => <li key={reason}>{reason}</li>)}</ul>
      </div>}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <button type="button" className="inline-flex items-center gap-2 rounded-lg border border-wc-default px-3 py-2 text-sm text-wc-text-primary disabled:opacity-50" onClick={() => start.mutate()} disabled={start.isPending} data-testid="staging-start-check">
          <Play aria-hidden="true" className="h-4 w-4"/>{start.isPending ? "Checking prerequisites…" : "Check start prerequisites"}
        </button>
        {start.data?.state === "blocked" && <span role="status" className="text-sm text-wc-text-secondary" data-testid="staging-start-blocked">Start remains blocked: {start.data.reasons.join("; ")}</span>}
        {start.error && <span role="alert" className="text-sm text-wc-danger">Start was not accepted. No staging success is claimed.</span>}
      </div>
    </section>}
    <div className="mb-3 flex items-center justify-between gap-3"><h3 className="text-sm font-semibold text-wc-text-primary">Deployment operation history</h3><button type="button" className="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-wc-text-secondary hover:bg-wc-surface-input" onClick={() => { void operations.refetch(); }} disabled={operations.isFetching}><RefreshCw aria-hidden="true" className="h-4 w-4"/>Refresh</button></div>
    {operations.isLoading ? <p role="status" className="text-sm text-wc-text-muted">Reading durable operation status…</p> : operations.error ? <p role="alert" className="text-sm text-wc-text-secondary">Deployment Manager status is unavailable. No deployment action was attempted.</p> : operations.data?.length ? <SettingsList.Group>{operations.data.map((operation) => <article key={operation.operation_id} className="mb-3 rounded-xl border border-wc-default bg-wc-surface-base p-4" data-testid="staging-operation">
      <div className="flex flex-wrap items-center justify-between gap-2"><span className="font-medium text-wc-text-primary">{operation.status.replace("RELEASE_OPERATION_STATUS_", "").toLowerCase()}</span><time className="text-xs text-wc-text-muted">{operation.updated_at ? new Date(operation.updated_at).toLocaleString() : ""}</time></div>
      <p className="mt-1 break-all text-xs text-wc-text-muted">{operation.operation_id}</p>
      {operation.cloud_spend_quote_idempotency_key && <p className="mt-1 break-all text-xs text-wc-text-muted">Spend quote reference: {operation.cloud_spend_quote_idempotency_key}</p>}
      {operation.cloud_compute_manager_instance_id && <p className="mt-1 break-all text-xs text-wc-text-muted">Compute Manager instance: {operation.cloud_compute_manager_instance_id}</p>}
      {operation.active_stage && <p className="mt-2 text-sm text-wc-text-secondary">Stage: {operation.active_stage}</p>}
      {operation.error && <p className="mt-2 text-sm text-wc-danger">{operation.error}</p>}
      {operation.cloud_step?.reason && <p className="mt-2 text-sm text-wc-text-secondary">Cloud hand-off: {operation.cloud_step.reason}</p>}
      {operation.cloud_step?.state && <p className="mt-2 text-sm text-wc-text-secondary">Cloud step: {operation.cloud_step.state.replace(/_/g, " ")}</p>}
      {operation.cloud_step?.deployment_id && <p className="mt-1 break-all text-xs text-wc-text-muted">Scenario-to-Cloud deployment: {operation.cloud_step.deployment_id}</p>}
      <dl className="mt-3 grid gap-1 rounded-lg bg-wc-surface-input p-3 text-sm" data-testid="staging-tls-status">
        <div><dt className="inline font-medium text-wc-text-primary">TLS health: </dt><dd className="inline text-wc-text-secondary">{operation.tls?.state ?? "unavailable"}</dd></div>
        {operation.tls?.sampled_at && <div><dt className="inline font-medium text-wc-text-primary">TLS sample: </dt><dd className="inline text-wc-text-secondary"><time dateTime={operation.tls.sampled_at}>{new Date(operation.tls.sampled_at).toLocaleString()}</time></dd></div>}
        {operation.tls?.reason && <div className="text-wc-text-secondary">{operation.tls.reason}</div>}
      </dl>
      <dl className="mt-3 grid gap-1 rounded-lg bg-wc-surface-input p-3 text-sm" data-testid="staging-certificate-status">
        <div><dt className="inline font-medium text-wc-text-primary">Certificate: </dt><dd className="inline text-wc-text-secondary">{operation.certificate?.state ?? "unavailable"}</dd></div>
        {operation.certificate?.not_after && <div><dt className="inline font-medium text-wc-text-primary">Expires: </dt><dd className="inline text-wc-text-secondary"><time dateTime={operation.certificate.not_after}>{new Date(operation.certificate.not_after).toLocaleString()}</time></dd></div>}
        {operation.certificate?.sampled_at && <div><dt className="inline font-medium text-wc-text-primary">Sampled: </dt><dd className="inline text-wc-text-secondary"><time dateTime={operation.certificate.sampled_at}>{new Date(operation.certificate.sampled_at).toLocaleString()}</time></dd></div>}
        <div><dt className="inline font-medium text-wc-text-primary">Renewal: </dt><dd className="inline text-wc-text-secondary">{operation.certificate?.renewal ?? "blocked"}</dd></div>
        {(operation.certificate?.renewal_reason || operation.certificate?.reason) && <div className="text-wc-text-secondary">{operation.certificate.renewal_reason ?? operation.certificate.reason}</div>}
      </dl>
      {operation.cloud_step?.owner_operation_id && <p className="mt-1 break-all text-xs text-wc-text-muted">Scenario-to-Cloud operation: {operation.cloud_step.owner_operation_id}</p>}
      {operation.cloud_step?.request_key && <p className="mt-1 break-all text-xs text-wc-text-muted">Handoff request: {operation.cloud_step.request_key}</p>}
      {operation.cloud_step?.request_digest && <p className="mt-1 break-all text-xs text-wc-text-muted">Handoff request digest: {operation.cloud_step.request_digest}</p>}
      {operation.cloud_step?.verification_receipt_json && <details className="mt-3 rounded-lg bg-wc-surface-input p-3" data-testid="staging-verification-receipt">
        <summary className="cursor-pointer text-sm font-medium text-wc-text-primary">View stored verification receipt</summary>
        <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap break-all text-xs text-wc-text-secondary">{formatReceipt(operation.cloud_step.verification_receipt_json)}</pre>
      </details>}
      {operation.cloud_step?.state === "recovery_decision_required" && <StagingRecoveryAction operation={operation} />}
      {operation.resume_available && <button type="button" className="mt-3 inline-flex items-center gap-2 rounded-lg border border-wc-default px-3 py-2 text-sm text-wc-text-primary disabled:opacity-50" onClick={() => resume.mutate(operation.operation_id)} disabled={resume.isPending} data-testid="staging-resume">
        <RotateCw aria-hidden="true" className="h-4 w-4"/>{resume.isPending ? "Requesting safe resume…" : "Resume operation"}
      </button>}
      {!operation.resume_available && operation.resume_reason && <p className="mt-2 text-xs text-wc-text-muted" data-testid="staging-resume-blocked">Resume unavailable: {operation.resume_reason}</p>}
    </article>)}</SettingsList.Group> : <p className="rounded-xl border border-wc-default p-4 text-sm text-wc-text-secondary" data-testid="staging-no-operations">No Deployment Manager operation has been recorded for this profile.</p>}
    {resume.error && <p role="alert" className="mt-2 text-sm text-wc-danger">Deployment Manager refused the resume request or could not confirm it. Refresh to read durable status.</p>}
  </div>;
}

function StagingRecoveryAction({ operation }: { operation: StagingOperation }) {
  const queryClient = useQueryClient();
  const eligible = operation.cloud_step?.state === "recovery_decision_required";
  const standing = useQuery({
    queryKey: ["staging-recovery", operation.operation_id],
    queryFn: () => getStagingRecovery(operation.operation_id),
    enabled: eligible,
    refetchInterval: eligible ? 15000 : false,
  });
  const preview = useMutation({
    mutationFn: () => previewStagingRecovery(operation.operation_id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["staging-recovery", operation.operation_id] }),
  });
  const confirmation = useMutation({
    mutationFn: (previewRef: string) => confirmStagingRecovery(operation.operation_id, previewRef),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["staging-recovery", operation.operation_id] }),
  });
  const [compatibilityConfirmed, setCompatibilityConfirmed] = useState(false);
  const [rollbackConfirmed, setRollbackConfirmed] = useState(false);
  const status = standing.data?.recovery;
  const previewView = preview.data?.recovery;
  const recoveryStarted = Boolean(status?.recovery_operation_id || status?.state === "pending" || status?.state === "succeeded" || status?.state === "failed" || status?.state === "ambiguous");
  return <div className="mt-3 rounded-lg border border-wc-danger/30 bg-wc-surface-input p-3" role="status" data-testid="staging-recovery-required">
    <p className="text-sm font-semibold text-wc-text-primary">Recovery decision required</p>
    <p className="mt-1 text-sm text-wc-text-secondary">The stored verification did not pass. This profile can request a rollback preview from Scenario-to-Cloud using your verified human session. Recovery never runs automatically.</p>
    {standing.isLoading && <p className="mt-2 text-sm text-wc-text-muted">Reading exact recovery status from Scenario-to-Cloud…</p>}
    {standing.error && <p className="mt-2 text-sm text-wc-danger" role="alert">Recovery status is unavailable or this human session lacks target authorization. No recovery action was attempted.</p>}
    {status && <RecoveryIdentity status={status} />}
    {recoveryStarted && status && <div className="mt-3 rounded border border-wc-default p-3" data-testid="staging-recovery-owner-standing">
      <p className="text-sm font-semibold text-wc-text-primary">Scenario-to-Cloud recovery: {status.recovery_status ?? status.state}</p>
      {status.recovery_operation_id && <p className="mt-1 break-all text-xs text-wc-text-muted">Owner operation: {status.recovery_operation_id}</p>}
      {status.recovery_error && <p className="mt-1 text-sm text-wc-danger">{status.recovery_error}</p>}
      {status.recovery_receipt_json && <details className="mt-2"><summary className="cursor-pointer text-sm">View owner receipt</summary><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all text-xs">{formatReceipt(status.recovery_receipt_json)}</pre></details>}
    </div>}
    {eligible && status?.state === "awaiting_preview" && !recoveryStarted && <div className="mt-3 space-y-3" data-testid="staging-recovery-preview-action">
      <label className="flex items-start gap-2 text-sm text-wc-text-secondary">
        <input type="checkbox" checked={compatibilityConfirmed} onChange={(event) => { setCompatibilityConfirmed(event.target.checked); setRollbackConfirmed(false); }} />
        <span>I have checked that data written by the current release is compatible with the last-good release. This is my assessment; the system does not infer compatibility.</span>
      </label>
      <button type="button" className="inline-flex items-center gap-2 rounded-lg border border-wc-default px-3 py-2 text-sm text-wc-text-primary disabled:opacity-50" onClick={() => { setRollbackConfirmed(false); preview.mutate(); }} disabled={!compatibilityConfirmed || preview.isPending} data-testid="staging-recovery-preview">
        {preview.isPending ? "Rechecking current release…" : "Preview rollback"}
      </button>
      {preview.error && <p role="alert" className="text-sm text-wc-danger">Scenario-to-Cloud could not produce a target-authorized current preview. No rollback was started.</p>}
      {previewView?.state === "preview_ready" && <div className="space-y-3 rounded-lg border border-wc-default p-3" data-testid="staging-recovery-preview-details">
        <p className="text-sm font-semibold text-wc-text-primary">Canonical rollback preview</p>
        <RecoveryIdentity status={previewView} />
        <p className="break-all text-xs text-wc-text-secondary">Last-good bundle: {previewView.last_good_bundle_sha256}</p>
        <p className="text-xs text-wc-text-secondary">Data compatibility: operator-declared compatible. Route: {previewView.route_kind}. Effect: {previewView.impact_summary}. This is a destructive action.</p>
        <p className="break-all text-xs text-wc-text-muted">Preview reference: {previewView.preview_ref}</p>
        <label className="flex items-start gap-2 text-sm text-wc-text-secondary">
          <input type="checkbox" checked={rollbackConfirmed} onChange={(event) => setRollbackConfirmed(event.target.checked)} />
          <span>I reviewed these exact release identities and authorize this rollback.</span>
        </label>
        <button type="button" className="rounded-lg border border-wc-danger/40 px-3 py-2 text-sm text-wc-danger disabled:opacity-50" onClick={() => { setRollbackConfirmed(false); confirmation.mutate(previewView.preview_ref ?? ""); }} disabled={!rollbackConfirmed || confirmation.isPending || !previewView.preview_ref} data-testid="staging-recovery-confirm">
          {confirmation.isPending ? "Submitting confirmed rollback…" : "Confirm rollback"}
        </button>
        {confirmation.error && <p role="alert" className="text-sm text-wc-danger">S2C did not confirm the recovery request. Refresh to read the exact owner operation before retrying.</p>}
      </div>}
    </div>}
    {status?.state === "stale_preview" && <p role="alert" className="mt-2 text-sm text-wc-danger">The current or last-good release changed. Review a fresh preview before confirming.</p>}
    {confirmation.data?.recovery && !recoveryStarted && <p className="mt-2 text-sm text-wc-text-secondary">Recovery request accepted. Status is being read back from the exact Scenario-to-Cloud owner operation.</p>}
  </div>;
}

function RecoveryIdentity({ status }: { status: { current_release_digest: string; current_bundle_sha256: string; last_good_release_digest: string; last_good_bundle_sha256?: string; deployment_id: string; review_ref: string } }) {
  return <dl className="mt-2 grid gap-2 text-xs text-wc-text-muted">
    <div><dt className="font-medium text-wc-text-secondary">Deployment</dt><dd className="break-all">{status.deployment_id}</dd></div>
    <div><dt className="font-medium text-wc-text-secondary">Current failed release</dt><dd className="break-all">{status.current_release_digest} · bundle {status.current_bundle_sha256}</dd></div>
    <div><dt className="font-medium text-wc-text-secondary">Exact last-good release</dt><dd className="break-all">{status.last_good_release_digest}{status.last_good_bundle_sha256 ? ` · bundle ${status.last_good_bundle_sha256}` : ""}</dd></div>
    <div><dt className="font-medium text-wc-text-secondary">Publication review</dt><dd className="break-all">{status.review_ref}</dd></div>
  </dl>;
}

function formatReceipt(value: string): string {
  try { return JSON.stringify(JSON.parse(value), null, 2); } catch { return value; }
}
