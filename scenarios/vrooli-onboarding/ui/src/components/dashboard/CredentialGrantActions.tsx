import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import {
  listCredentialGrantActions,
  writeCredentialGrantAction,
  type CredentialGrantAction,
  type CredentialGrantActionReceipt,
} from "../../api/credentialGrants";
import { i18n } from "../../i18n";

type Intent = { action: CredentialGrantAction; value: Record<string, unknown>; idempotencyKey: string };

export function CredentialGrantActions() {
  const query = useQuery({ queryKey: ["sealed-credential-grant-actions"], queryFn: listCredentialGrantActions, refetchInterval: 30_000 });
  const queryClient = useQueryClient();
  const [intent, setIntent] = useState<Intent>();
  const [receipt, setReceipt] = useState<CredentialGrantActionReceipt>();
  const [error, setError] = useState(false);
  const [saving, setSaving] = useState(false);

  const stage = (action: CredentialGrantAction, value: Record<string, unknown>) => {
    setReceipt(undefined);
    setError(false);
    setIntent({ action, value, idempotencyKey: newRequestID() });
  };

  const submit = async () => {
    if (!intent || saving) return;
    setSaving(true);
    setError(false);
    try {
      const result = await writeCredentialGrantAction(intent.action, intent.idempotencyKey, intent.value);
      setReceipt(result);
      await queryClient.invalidateQueries({ queryKey: ["sealed-credential-grant-metadata"] });
    } catch {
      setError(true);
    } finally {
      setSaving(false);
    }
  };

  const retry = async () => {
    if (!intent || !receipt || !retryableState(receipt.ownerState)) return;
    await submit();
  };

  return (
    <section className="mt-8 rounded-xl border border-border bg-surface p-5" aria-labelledby="credential-grant-actions-heading" data-testid="credential-grant-actions">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{i18n.t("onboarding.dashboard.credentialGrantActionsEyebrow")}</p>
          <h2 id="credential-grant-actions-heading" className="mt-1 text-lg font-semibold">{i18n.t("onboarding.dashboard.credentialGrantActionsHeading")}</h2>
          <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.credentialGrantActionsIntro")}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => void query.refetch()} disabled={query.isFetching}>{i18n.t("onboarding.dashboard.credentialGrantRefreshActions")}</Button>
      </div>
      {query.isLoading && <p className="mt-4 text-sm text-muted" role="status">{i18n.t("onboarding.dashboard.credentialGrantActionsLoading")}</p>}
      {query.error && <p className="mt-4 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.credentialGrantActionsError")}</p>}
      {!query.isLoading && !query.error && query.data?.length === 0 && <p className="mt-4 text-sm text-muted">{i18n.t("onboarding.dashboard.credentialGrantActionsEmpty")}</p>}
      {query.data && query.data.length > 0 && (
        <ul className="mt-4 divide-y divide-border" aria-label={i18n.t("onboarding.dashboard.credentialGrantActionsHeading")}>
          {query.data.map((action) => (
            <li key={action.permissionId} className="py-4" data-testid="credential-grant-action-row">
              <p className="font-medium">{action.operation === "create" ? i18n.t("onboarding.dashboard.credentialGrantCreate") : i18n.t("onboarding.dashboard.credentialGrantRevoke", { id: action.subjectId })}</p>
              <p className="mt-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantTarget", { machine: action.target.machineId, node: action.target.nodeId, generation: action.target.enrollmentGeneration })}</p>
              <p className="text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantExpectedRevision", { revision: action.expectedRevision })}</p>
              {action.operation === "create"
                ? <CreateGrantForm action={action} disabled={Boolean(intent)} onStage={stage} />
                : <Button variant="secondary" size="sm" className="mt-3" disabled={Boolean(intent)} onClick={() => stage(action, { operation: "revoke" })}>{i18n.t("onboarding.dashboard.credentialGrantReviewAction")}</Button>}
            </li>
          ))}
        </ul>
      )}
      {intent && (
        <div className="mt-4 rounded-lg border border-warning/50 bg-surface-muted p-4" data-testid="credential-grant-action-review">
          <p className="font-medium">{intent.action.operation === "create" ? i18n.t("onboarding.dashboard.credentialGrantConfirmCreate") : i18n.t("onboarding.dashboard.credentialGrantConfirmRevoke")}</p>
          <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.dashboard.credentialGrantTarget", { machine: intent.action.target.machineId, node: intent.action.target.nodeId, generation: intent.action.target.enrollmentGeneration })}</p>
          {intent.action.operation === "create" && <p className="text-sm text-muted">{String(intent.value.logical_id)} · {String(intent.value.field)} · {String(intent.value.class)} · {String(intent.value.retention)} · generation {String(intent.value.generation)}</p>}
          {error && <p className="mt-2 text-sm text-danger" role="alert">{i18n.t("onboarding.dashboard.credentialGrantActionError")}</p>}
          {receipt && <p className="mt-2 text-sm" role="status">{i18n.t("onboarding.dashboard.credentialGrantTargetEffect", { state: receipt.ownerState ?? i18n.t("onboarding.dashboard.notAvailable"), outcome: receipt.outcome })}</p>}
          <div className="mt-3 flex flex-wrap gap-2">
            {!receipt && <Button size="sm" onClick={() => void submit()} disabled={saving}>{saving ? i18n.t("onboarding.dashboard.credentialGrantSaving") : error ? i18n.t("onboarding.dashboard.credentialGrantRetry") : i18n.t("onboarding.dashboard.credentialGrantConfirm")}</Button>}
            {receipt && retryableState(receipt.ownerState) && <Button size="sm" onClick={() => void retry()} disabled={saving}>{i18n.t("onboarding.dashboard.credentialGrantRetry")}</Button>}
            {receipt && !retryableState(receipt.ownerState) && <Button size="sm" variant="secondary" onClick={() => setIntent(undefined)}>{i18n.t("onboarding.dashboard.credentialGrantClose")}</Button>}
            {!receipt && <Button size="sm" variant="secondary" onClick={() => setIntent(undefined)} disabled={saving}>{i18n.t("onboarding.dashboard.credentialGrantCancelAction")}</Button>}
          </div>
        </div>
      )}
    </section>
  );
}

function CreateGrantForm({ action, disabled, onStage }: { action: CredentialGrantAction; disabled: boolean; onStage: (action: CredentialGrantAction, value: Record<string, unknown>) => void }) {
  const [logicalID, setLogicalID] = useState("");
  const [field, setField] = useState("");
  const [grantClass, setGrantClass] = useState("user_prompt");
  const [retention, setRetention] = useState("ephemeral");
  const [generation, setGeneration] = useState("1");
  const parsedGeneration = Number(generation);
  const valid = Boolean(logicalID.trim() && field.trim() && Number.isSafeInteger(parsedGeneration) && parsedGeneration > 0);
  return (
    <form className="mt-3 grid gap-2 sm:grid-cols-2" onSubmit={(event) => { event.preventDefault(); if (valid) onStage(action, { operation: "create", logical_id: logicalID.trim(), field: field.trim(), class: grantClass, retention, generation: parsedGeneration }); }}>
      <label className="grid gap-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantLogicalID")}<input className="rounded border border-border bg-surface px-2 py-1 text-sm text-primary" value={logicalID} onChange={(event) => setLogicalID(event.target.value)} required /></label>
      <label className="grid gap-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantField")}<input className="rounded border border-border bg-surface px-2 py-1 text-sm text-primary" value={field} onChange={(event) => setField(event.target.value)} required /></label>
      <label className="grid gap-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantClass")}<select className="rounded border border-border bg-surface px-2 py-1 text-sm text-primary" value={grantClass} onChange={(event) => setGrantClass(event.target.value)}><option value="infrastructure">{i18n.t("onboarding.dashboard.credentialGrantClassInfrastructure")}</option><option value="per_install_generated">{i18n.t("onboarding.dashboard.credentialGrantClassGenerated")}</option><option value="user_prompt">{i18n.t("onboarding.dashboard.credentialGrantClassPrompt")}</option><option value="remote_fetch">{i18n.t("onboarding.dashboard.credentialGrantClassRemoteFetch")}</option></select></label>
      <label className="grid gap-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantRetention")}<select className="rounded border border-border bg-surface px-2 py-1 text-sm text-primary" value={retention} onChange={(event) => setRetention(event.target.value)}><option value="ephemeral">{i18n.t("onboarding.dashboard.credentialGrantRetentionEphemeral")}</option><option value="durable">{i18n.t("onboarding.dashboard.credentialGrantRetentionDurable")}</option></select></label>
      <label className="grid gap-1 text-xs text-muted">{i18n.t("onboarding.dashboard.credentialGrantGeneration")}<input type="number" min="1" step="1" className="rounded border border-border bg-surface px-2 py-1 text-sm text-primary" value={generation} onChange={(event) => setGeneration(event.target.value)} required /></label>
      <div className="flex items-end"><Button type="submit" size="sm" disabled={disabled || !valid}>{i18n.t("onboarding.dashboard.credentialGrantReviewAction")}</Button></div>
    </form>
  );
}

function retryableState(value?: string): boolean {
  return value === "pending" || value === "dispatched" || value === "retryable";
}

function newRequestID(): string {
  return globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
