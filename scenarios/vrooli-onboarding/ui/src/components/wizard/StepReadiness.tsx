import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { applyOnboarding, fetchV2Readiness, provisionCredential } from "../../lib/api";
import type { ReadinessItem } from "../../types";
import { Button } from "../ui/button";

export function StepReadiness({ title = "Validation" }: { title?: "Credentials" | "Validation" }) {
  const { data, isLoading, error, refetch } = useQuery({ queryKey: ["v2-readiness"], queryFn: fetchV2Readiness });
  const [values, setValues] = useState<Record<string, string>>({});
  const [provisioning, setProvisioning] = useState<string | null>(null);
  const [provisionError, setProvisionError] = useState<string | null>(null);
  const [applyState, setApplyState] = useState<{ status: string; items: Array<{ name: string; outcome: string; error?: string }> } | null>(null);
  const [applyError, setApplyError] = useState<string | null>(null);
  const [degradedAcknowledged, setDegradedAcknowledged] = useState(false);

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
    try {
      const result = await applyOnboarding();
      setApplyState({ status: result.status, items: result.items.map((item) => ({ name: item.name, outcome: item.outcome, error: item.error })) });
      await refetch();
    } catch {
      setApplyError("The selection could not be applied. Resolve the reported host issue and try again.");
    }
  };

  return <div data-testid="step-readiness">
    <h1 className="text-xl font-semibold sm:text-2xl">{title}</h1>
    <p className="mt-2 text-sm text-slate-300">This report uses manifest metadata and provider status only. Credential values are never displayed.</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-slate-300" role="status"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</p>}
    {error && <p className="mt-6 text-red-400" role="alert">Readiness could not be checked. Verify the control plane is available.</p>}
    {provisionError && <p className="mt-4 text-sm text-red-400" role="alert">{provisionError}</p>}
    {applyError && <p className="mt-4 text-sm text-red-400" role="alert">{applyError}</p>}
    {data?.credential_diagnosis?.provider && <div data-testid="backend-diagnosis" className="mt-4 rounded-lg border border-amber-300/30 bg-amber-300/10 p-3 text-sm"><p className="font-medium text-amber-200">Credential provider diagnosis</p><p className="mt-1 text-slate-200">{data.credential_diagnosis.provider.condition} · {data.credential_diagnosis.provider.backend}</p>{data.credential_diagnosis.provider.explanation && <p className="mt-1 text-xs text-slate-300">{data.credential_diagnosis.provider.explanation}</p>}{data.credential_diagnosis.provider.fix && <p className="mt-1 text-xs text-emerald-200">Next: {data.credential_diagnosis.provider.fix}</p>}{data.credential_diagnosis.provider.write_condition && <p className="mt-1 text-xs text-slate-300">Write reachability: {data.credential_diagnosis.provider.write_condition}. {data.credential_diagnosis.provider.write_fix}</p>}</div>}
    {data?.recovery && <div data-testid="store-init-guidance" className={`mt-4 rounded-lg border p-3 text-sm ${data.recovery.receipt_exists && data.recovery.uncovered.length === 0 ? "border-emerald-300/30 bg-emerald-300/10" : "border-amber-300/30 bg-amber-300/10"}`}><p className="font-medium text-amber-100">Recovery backup</p>{data.recovery.receipt_exists ? <p className="mt-1 text-slate-200">Verified bundle covers {data.recovery.entry_count} credential entries{data.recovery.uncovered.length ? `; ${data.recovery.uncovered.length} configured entries still need coverage.` : "."}</p> : <p className="mt-1 text-slate-200">Step: create and verify a recovery bundle with <code>secrets-manager backup export --output &lt;bundle&gt; &lt; passphrase</code>, then store the bundle and passphrase separately.</p>}</div>}
      {data && <>
      <div className="mt-6 flex items-center gap-2" role="status" data-testid="readiness-summary">{data.status === "ready" ? <CheckCircle2 className="h-5 w-5 text-emerald-400" /> : <CircleAlert className="h-5 w-5 text-amber-300" />}<span className={data.status === "ready" ? "text-emerald-300" : "text-amber-200"}>{data.status === "ready" ? "Ready" : `${data.status}: action required`}</span></div>
      {title === "Validation" && <section className="mt-4" aria-label="Apply plan">
        <h2 className="text-lg font-medium">Review apply plan</h2>
        <ul data-testid="apply-plan" role="list" className="mt-2 space-y-1 text-sm text-slate-300">
          {[...data.resources.map((name) => `Resource: ${name}`), ...data.scenarios.map((name) => `Scenario: ${name}`), ...data.hosts.filter((item) => item.status !== "deferred").map((item) => `${item.kind ?? "Host"}: ${item.name}`)].map((item) => <li key={item}>{item}</li>)}
          {data.resources.length === 0 && data.scenarios.length === 0 && data.hosts.every((item) => item.status === "deferred") && <li>Current selection has no host changes.</li>}
        </ul>
        <p data-testid="privilege-warning" role="note" className="mt-2 text-sm text-amber-200">Review the named host items above; safeguards and elevated tools may require privilege. No undeclared host action is performed.</p>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Button data-testid="apply-confirm" type="button" onClick={() => { void apply(); }} className="rounded-md bg-emerald-500 px-3 py-2 text-sm font-medium text-slate-950">Apply selection</Button>
          {applyState && <span data-testid="apply-progress" role="progressbar" aria-valuetext={`${applyState.status} · ${applyState.items.length} items`} className="text-sm text-slate-300">{applyState.status} · {applyState.items.length} items</span>}
        </div>
      </section>}
      {applyState && <table className="mt-3 w-full text-left text-xs text-slate-300" data-testid="apply-report" role="table"><caption className="sr-only">Apply results</caption><thead><tr><th scope="col">Item</th><th scope="col">Outcome</th></tr></thead><tbody>{applyState.items.map((item) => <tr key={item.name}><td className="py-1">{item.name}</td><td className="py-1">{item.outcome}{item.error ? ` — ${item.error}` : ""}</td></tr>)}</tbody></table>}
      {applyState?.items.some((item) => item.outcome === "blocked" || item.outcome === "failed") && <><p data-testid="skipped-note" role="note" className="mt-3 text-sm text-amber-200">Some items were skipped or failed. Resolve the reported dependency or host issue before trying again.</p><Button data-testid="retry" type="button" variant="outline" onClick={() => { void apply(); }}>Apply again</Button></>}
      {title === "Validation" && <div className="mt-4 flex flex-wrap gap-3"><Button data-testid="recheck" type="button" variant="outline" onClick={() => { void refetch(); }}>Recheck</Button>{data.status === "degraded" && !degradedAcknowledged && <Button data-testid="readiness-continue-degraded" type="button" variant="outline" onClick={() => setDegradedAcknowledged(true)}>Continue with degraded</Button>}{degradedAcknowledged && <p role="status" className="text-sm text-amber-200">Degraded continuation recorded for this session.</p>}</div>}
      <ul className="mt-4 space-y-2">{data.credentials.map((credential) => {
        const key = `${credential.logical_id}/${credential.field}`;
        const canProvision = title === "Credentials" && credential.status !== "configured";
        return <li key={key} data-testid="credential-card" className="rounded-lg border border-white/10 bg-white/5 p-3 text-sm"><p data-testid="credential-purpose" role="note" className="font-medium">{credential.label || credential.field}</p><span data-testid="credential-status" role="status" className="ml-2 text-slate-300">{credential.required ? "Required" : "Optional"} · {credential.status}</span>{credential.description && <p className="mt-1 text-xs text-slate-300">{credential.description}</p>}{credential.obtain_url && <a data-testid="credential-obtain-link" className="mt-1 inline-flex min-h-11 items-center rounded px-2 text-xs text-emerald-200 underline" href={credential.obtain_url}>How to obtain this credential</a>}{credential.detail && <p className="mt-1 text-xs text-slate-300">{credential.detail}</p>}{canProvision && <div className="mt-3 flex flex-col gap-2 sm:flex-row"><input data-testid="credential-input" aria-label={`Value for ${credential.label || credential.field}`} type="password" autoComplete="off" value={values[key] ?? ""} onChange={(event) => setValues((current) => ({ ...current, [key]: event.target.value }))} className="min-h-11 min-w-0 flex-1 rounded-md border border-white/20 bg-slate-950 px-3 py-2 text-sm" /><Button data-testid="credential-save" type="button" disabled={!values[key]?.trim() || provisioning === key} onClick={() => { void provision(credential.logical_id, credential.field); }} className="rounded-md bg-emerald-500 px-3 py-2 text-sm font-medium text-slate-950 disabled:opacity-50">{provisioning === key ? "Saving…" : "Save securely"}</Button></div>}</li>;
      })}</ul>
      {title === "Validation" && <><ReadinessGroup title="Host requirements" items={data.hosts} /><ReadinessGroup title="Integrations" items={data.integrations} /></>}
    </>}
  </div>;
}

function ReadinessGroup({ title, items }: { title: string; items: ReadinessItem[] }) {
  return <section className="mt-6"><h2 className="text-lg font-medium">{title}</h2>{items.length === 0 ? <p className="mt-2 text-sm text-slate-300">No declared requirements.</p> : <ul className="mt-3 space-y-2">{items.map((item) => <li key={`${item.kind ?? "item"}-${item.name}`} data-testid="readiness-item" className="rounded-lg border border-white/10 bg-white/5 p-3 text-sm"><span className="font-medium">{item.name}</span><span className="ml-2 text-slate-300">{item.kind ? `${item.kind} · ` : ""}{item.status}{item.required ? " · required" : ""}</span>{item.detail && <p className="mt-1 text-xs text-slate-300">{item.detail}</p>}{item.remediation && <p data-testid="remediation" className="mt-1 text-xs text-emerald-200">Next: {item.remediation}</p>}</li>)}</ul>}</section>;
}
