import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { applyOnboarding, changeCredentialStorePassphrase, fetchCredentialStoreStatus, fetchOperatorInputs, fetchV2ApplyPlan, fetchV2ApplyStatus, fetchV2Readiness, initializeCredentialStore, provisionCredential, reselectCredentialBackend, rewrapCredentialStore, resolveOperatorInputs, selectCredentialBackend, unlockCredentialStore } from "../../lib/api";
import type { ReadinessItem } from "../../types";
import { Button } from "../ui/button";

export function StepReadiness({ title = "Validation" }: { title?: "Credentials" | "Apply" | "Validation" }) {
  const { data, isLoading, error, refetch } = useQuery({ queryKey: ["v2-readiness"], queryFn: fetchV2Readiness });
  const { data: operatorInputs, refetch: refetchOperatorInputs } = useQuery({ queryKey: ["operator-inputs"], queryFn: fetchOperatorInputs, enabled: title === "Credentials" });
  const { data: credentialStore } = useQuery({ queryKey: ["credential-store-status"], queryFn: fetchCredentialStoreStatus, enabled: title === "Credentials" });
  const { data: plan } = useQuery({ queryKey: ["v2-apply-plan"], queryFn: fetchV2ApplyPlan, enabled: title === "Apply" });
  const [values, setValues] = useState<Record<string, string>>({});
  const [provisioning, setProvisioning] = useState<string | null>(null);
  const [provisionError, setProvisionError] = useState<string | null>(null);
  const [applyState, setApplyState] = useState<{ run_id: string; status: string; items: Array<{ name: string; outcome: string; error?: string }> } | null>(null);
  const [applyError, setApplyError] = useState<string | null>(null);
  const [degradedAcknowledged, setDegradedAcknowledged] = useState(false);
  const [operatorValues, setOperatorValues] = useState<Record<string, string>>({});
  const [operatorInputError, setOperatorInputError] = useState<string | null>(null);
  const [storeSecret, setStoreSecret] = useState("");
  const [storeCurrent, setStoreCurrent] = useState("");
  const [storeNew, setStoreNew] = useState("");
  const [storeError, setStoreError] = useState<string | null>(null);
  const applyController = useRef<AbortController | null>(null);

  useEffect(() => () => applyController.current?.abort(), []);

  const provision = async (logicalID: string, field: string) => {
    const key = `${logicalID}/${field}`;
    const value = values[key]?.trim() ?? "";
    if (!value) return;
    setProvisioning(key);
    setProvisionError(null);
    try {
      await provisionCredential({ logical_id: logicalID, field, value });
      setValues((current) => ({ ...current, [key]: "" }));
      await refetch();
    } catch {
      setProvisionError("Credential provisioning failed. Verify that the native credential authority is available.");
    } finally {
      setProvisioning(null);
    }
  };

  const apply = async () => {
    setApplyError(null);
    applyController.current?.abort();
    const controller = new AbortController();
    applyController.current = controller;
    try {
      const result = await applyOnboarding(controller.signal);
      const update = (current: typeof result) => setApplyState({ run_id: current.run_id, status: current.status, items: current.items.map((item) => ({ name: item.name, outcome: item.outcome, error: item.error })) });
      update(result);
      let current = result;
      while (current.status === "pending" || current.status === "applying") {
        await new Promise((resolve) => window.setTimeout(resolve, 250));
        current = await fetchV2ApplyStatus(result.run_id, controller.signal);
        update(current);
      }
      await refetch();
    } catch {
      setApplyError("The selection could not be applied. Resolve the reported host issue and try again.");
    }
  };

  const resolveInputs = async () => {
    setOperatorInputError(null);
    try {
      await resolveOperatorInputs((operatorInputs?.requests ?? []).map((request) => ({ request_id: request.id, value: operatorValues[request.id] ?? request.default ?? "" })));
      setOperatorValues({});
      await Promise.all([refetchOperatorInputs(), refetch()]);
    } catch {
      setOperatorInputError("The operator input could not be applied. Check each required answer and try again.");
    }
  };

  const storeAction = async (action: () => Promise<unknown>) => {
    setStoreError(null);
    try {
      await action();
      setStoreSecret("");
      setStoreCurrent("");
      setStoreNew("");
      await refetch();
    } catch {
      setStoreError("Credential protection could not be changed. Verify the passphrase and try again.");
    }
  };

  return <div data-testid="step-readiness">
    <h1 className="text-xl font-semibold sm:text-2xl">{title}</h1>
    <p className="mt-2 text-sm text-muted">This report uses manifest metadata and provider status only. Credential values are never displayed.</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</p>}
    {error && <p className="mt-6 text-danger" role="alert">Readiness could not be checked. Verify the control plane is available.</p>}
    {provisionError && <p className="mt-4 text-sm text-danger" role="alert">{provisionError}</p>}
    {applyError && <p className="mt-4 text-sm text-danger" role="alert">{applyError}</p>}
    {operatorInputError && <p className="mt-4 text-sm text-danger" role="alert">{operatorInputError}</p>}
    {storeError && <p className="mt-4 text-sm text-danger" role="alert">{storeError}</p>}
    {title === "Credentials" && <section data-testid="credential-store-card" className="mt-4 rounded-lg border border-muted bg-surface-muted p-4" aria-label="Credential protection"><h2 className="font-medium">Credential protection</h2>{credentialStore ? <p className="mt-1 text-sm text-foreground">{credentialStore.initialized ? `${credentialStore.entries} credential${credentialStore.entries === 1 ? "" : "s"} protected by ${credentialStore.active_wrap || "the encrypted store"}${credentialStore.active_key_store ? ` (${credentialStore.active_key_store})` : ""}.` : "The encrypted store is not initialized yet."}</p> : <p className="mt-1 text-sm text-muted">Checking the credential authority…</p>}{credentialStore?.host_bound_blocked && <p className="mt-1 text-xs text-muted">Unattended protection is unavailable: {credentialStore.host_bound_blocked}</p>}<details className="mt-3"><summary className="cursor-pointer text-sm font-medium">Manage protection</summary><div className="mt-3 space-y-3"><div className="flex flex-wrap gap-2"><Button type="button" variant="outline" disabled={(credentialStore?.entries ?? 0) > 0} title={(credentialStore?.entries ?? 0) > 0 ? "Use verified reselect and migration when credentials exist" : undefined} onClick={() => { void storeAction(() => selectCredentialBackend("native")); }}>Use native authority</Button><Button type="button" variant="outline" disabled={(credentialStore?.entries ?? 0) > 0} title={(credentialStore?.entries ?? 0) > 0 ? "Use verified reselect and migration when credentials exist" : undefined} onClick={() => { void storeAction(() => selectCredentialBackend("encrypted-file")); }}>Use encrypted authority</Button></div>{(credentialStore?.entries ?? 0) > 0 && <><p className="text-xs text-muted">Backend selection is locked while credentials exist. Re-evaluate the host and migrate the manifest-backed credentials with read-back verification.</p><Button type="button" variant="outline" onClick={() => { void storeAction(() => reselectCredentialBackend()); }}>Re-evaluate and migrate safely</Button></>}{!credentialStore?.initialized && <div className="flex gap-2"><input aria-label="New store passphrase" type="password" autoComplete="new-password" value={storeSecret} onChange={(event) => setStoreSecret(event.target.value)} className="min-h-11 min-w-0 flex-1 rounded-md border border-muted bg-surface px-3 py-2" /><Button type="button" disabled={!storeSecret} onClick={() => { void storeAction(() => initializeCredentialStore(storeSecret)); }}>Initialize</Button></div>}{credentialStore?.initialized && <><div className="flex gap-2"><input aria-label="Store passphrase" type="password" autoComplete="current-password" value={storeSecret} onChange={(event) => setStoreSecret(event.target.value)} className="min-h-11 min-w-0 flex-1 rounded-md border border-muted bg-surface px-3 py-2" /><Button type="button" disabled={!storeSecret} onClick={() => { void storeAction(() => unlockCredentialStore(storeSecret)); }}>Unlock</Button><Button type="button" disabled={!storeSecret} onClick={() => { void storeAction(() => rewrapCredentialStore(storeSecret)); }}>Rewrap</Button></div><div className="flex gap-2"><input aria-label="Current store passphrase" type="password" autoComplete="current-password" value={storeCurrent} onChange={(event) => setStoreCurrent(event.target.value)} className="min-h-11 min-w-0 flex-1 rounded-md border border-muted bg-surface px-3 py-2" /><input aria-label="New store passphrase" type="password" autoComplete="new-password" value={storeNew} onChange={(event) => setStoreNew(event.target.value)} className="min-h-11 min-w-0 flex-1 rounded-md border border-muted bg-surface px-3 py-2" /><Button type="button" disabled={!storeCurrent || !storeNew} onClick={() => { void storeAction(() => changeCredentialStorePassphrase(storeCurrent, storeNew)); }}>Change passphrase</Button></div></>}</div></details></section>}
    {title === "Credentials" && (operatorInputs?.requests.length ?? 0) > 0 && <section data-testid="operator-input-card" className="mt-4 rounded-lg border border-warning/30 bg-warning-surface p-4" aria-label="Pending operator decisions"><h2 className="font-medium text-warning">Complete protected setup</h2><p className="mt-1 text-sm text-foreground">These decisions were queued by non-interactive setup. Answers are used in memory and are not saved in this queue.</p><div className="mt-3 space-y-3">{operatorInputs?.requests.map((request) => <label key={request.id} className="block text-sm"><span className="font-medium">{request.title}</span>{request.description && <span className="mt-1 block text-xs text-muted">{request.description}</span>}<input data-testid="operator-input" type={request.kind === "secret" ? "password" : "text"} autoComplete="off" value={operatorValues[request.id] ?? ""} placeholder={request.default ?? ""} onChange={(event) => setOperatorValues((current) => ({ ...current, [request.id]: event.target.value }))} className="mt-2 min-h-11 w-full rounded-md border border-muted bg-surface px-3 py-2" /></label>)}</div><Button data-testid="operator-input-resolve" type="button" onClick={() => { void resolveInputs(); }} className="mt-4 rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary">Continue setup</Button></section>}
    {data?.credential_diagnosis?.provider && <div data-testid="backend-diagnosis" className="mt-4 rounded-lg border border-warning/30 bg-warning-surface p-3 text-sm"><p className="font-medium text-warning">Credential provider diagnosis</p><p className="mt-1 text-foreground">{data.credential_diagnosis.provider.condition} · {data.credential_diagnosis.provider.backend}</p>{data.credential_diagnosis.provider.explanation && <p className="mt-1 text-xs text-muted">{data.credential_diagnosis.provider.explanation}</p>}{data.credential_diagnosis.provider.fix && <p className="mt-1 text-xs text-primary-soft">Next: {data.credential_diagnosis.provider.fix}</p>}{data.credential_diagnosis.provider.write_condition && <p className="mt-1 text-xs text-muted">Write reachability: {data.credential_diagnosis.provider.write_condition}. {data.credential_diagnosis.provider.write_fix}</p>}</div>}
    {data?.recovery && <div data-testid="store-init-guidance" className={`mt-4 rounded-lg border p-3 text-sm ${data.recovery.receipt_exists && data.recovery.uncovered.length === 0 ? "border-primary-soft/30 bg-primary-soft/10" : "border-warning/30 bg-warning-surface"}`}><p className="font-medium text-warning">Recovery backup</p>{data.recovery.receipt_exists ? <p className="mt-1 text-foreground">Verified bundle covers {data.recovery.entry_count} credential entries{data.recovery.uncovered.length ? `; ${data.recovery.uncovered.length} configured entries still need coverage.` : "."}</p> : <p className="mt-1 text-foreground">Step: create and verify a recovery bundle with <code>secrets-manager backup export --output &lt;bundle&gt; &lt; passphrase</code>, then store the bundle and passphrase separately.</p>}</div>}
      {data && <>
      <div className="mt-6 flex items-center gap-2" role="status" data-testid="readiness-summary">{data.status === "ready" ? <CheckCircle2 className="h-5 w-5 text-primary" /> : <CircleAlert className="h-5 w-5 text-warning" />}<span className={data.status === "ready" ? "text-primary" : "text-warning"}>{data.status === "ready" ? "Ready" : `${data.status}: action required`}</span></div>
      {title === "Apply" && <section className="mt-4" aria-label="Apply plan">
        <h2 className="text-lg font-medium">Review apply plan</h2>
        <ul data-testid="apply-plan" role="list" className="mt-2 space-y-1 text-sm text-muted">
          {(plan?.items ?? []).map((item) => <li key={item.id}>{item.kind}: {item.name}{item.required ? " · required" : ""}</li>)}
          {(plan?.items ?? []).length === 0 && <li>Current selection has no consented host changes.</li>}
        </ul>
        <p data-testid="privilege-warning" role="note" className="mt-2 text-sm text-warning">Review the named host items above; safeguards and elevated tools may require privilege. No undeclared host action is performed.</p>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Button data-testid="apply-confirm" type="button" onClick={() => { void apply(); }} className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary">Apply selection</Button>
          {applyState && <span data-testid="apply-progress" role="progressbar" aria-valuetext={`${applyState.status} · ${applyState.items.length} items`} className="text-sm text-muted">{applyState.status} · {applyState.items.length} items</span>}
        </div>
      </section>}
      {applyState && <table className="mt-3 w-full text-left text-xs text-muted" data-testid="apply-report" role="table"><caption className="sr-only">Apply results</caption><thead><tr><th scope="col">Item</th><th scope="col">Outcome</th></tr></thead><tbody>{applyState.items.map((item) => <tr key={item.name}><td className="py-1">{item.name}</td><td className="py-1">{item.outcome}{item.error ? ` — ${item.error}` : ""}</td></tr>)}</tbody></table>}
      {applyState?.items.some((item) => item.outcome === "blocked" || item.outcome === "failed") && <><p data-testid="skipped-note" role="note" className="mt-3 text-sm text-warning">Some items were skipped or failed. Resolve the reported dependency or host issue before trying again.</p><Button data-testid="retry" type="button" variant="outline" onClick={() => { void apply(); }}>Apply again</Button></>}
      {title === "Validation" && <div className="mt-4 flex flex-wrap gap-3"><Button data-testid="recheck" type="button" variant="outline" onClick={() => { void refetch(); }}>Recheck</Button>{data.status === "degraded" && !degradedAcknowledged && <Button data-testid="readiness-continue-degraded" type="button" variant="outline" onClick={() => setDegradedAcknowledged(true)}>Continue with degraded</Button>}{degradedAcknowledged && <p role="status" className="text-sm text-warning">Degraded continuation recorded for this session.</p>}</div>}
      <ul className="mt-4 space-y-2">{data.credentials.map((credential) => {
        const key = `${credential.logical_id}/${credential.field}`;
        const canProvision = title === "Credentials" && credential.provisioning !== "derived" && credential.status !== "configured";
        return <li key={key} data-testid="credential-card" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><p data-testid="credential-purpose" role="note" className="font-medium">{credential.label || credential.field}</p><span data-testid="credential-status" role="status" className="ml-2 text-muted">{credential.required ? "Required" : "Optional"} · {credential.status}</span>{credential.provisioning === "derived" && <p className="mt-1 text-xs text-muted">Provided by {credential.derived_from || "the owning component"} after its source credential is available.</p>}{credential.description && <p className="mt-1 text-xs text-muted">{credential.description}</p>}{credential.obtain_url && <a data-testid="credential-obtain-link" className="mt-1 inline-flex min-h-11 items-center rounded px-2 text-xs text-primary-soft underline" href={credential.obtain_url}>How to obtain this credential</a>}{credential.detail && <p className="mt-1 text-xs text-muted">{credential.detail}</p>}{canProvision && <div className="mt-3 flex flex-col gap-2 sm:flex-row"><input data-testid="credential-input" aria-label={`Value for ${credential.label || credential.field}`} type="password" autoComplete="off" value={values[key] ?? ""} onChange={(event) => setValues((current) => ({ ...current, [key]: event.target.value }))} className="min-h-11 min-w-0 flex-1 rounded-md border border-muted bg-surface px-3 py-2 text-sm" /><Button data-testid="credential-save" type="button" disabled={!values[key]?.trim() || provisioning === key} onClick={() => { void provision(credential.logical_id, credential.field); }} className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary disabled:opacity-50">{provisioning === key ? "Saving…" : "Save securely"}</Button></div>}</li>;
      })}</ul>
      {title === "Validation" && <><ReadinessGroup title="Host requirements" items={data.hosts} /><ReadinessGroup title="Integrations" items={data.integrations} /></>}
    </>}
  </div>;
}

function ReadinessGroup({ title, items }: { title: string; items: ReadinessItem[] }) {
  return <section className="mt-6"><h2 className="text-lg font-medium">{title}</h2>{items.length === 0 ? <p className="mt-2 text-sm text-muted">No declared requirements.</p> : <ul className="mt-3 space-y-2">{items.map((item) => <li key={`${item.kind ?? "item"}-${item.name}`} data-testid="readiness-item" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><span className="font-medium">{item.name}</span><span className="ml-2 text-muted">{item.kind ? `${item.kind} · ` : ""}{item.status}{item.required ? " · required" : ""}</span>{item.detail && <p className="mt-1 text-xs text-muted">{item.detail}</p>}{item.remediation && <p data-testid="remediation" className="mt-1 text-xs text-primary-soft">Next: {item.remediation}</p>}</li>)}</ul>}</section>;
}
