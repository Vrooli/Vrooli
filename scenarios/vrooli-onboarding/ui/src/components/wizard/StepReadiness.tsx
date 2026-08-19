import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { applyOnboarding, fetchCapabilities, fetchV2ApplyPlan, fetchV2ApplyStatus, fetchV2Readiness, provisionCredential } from "../../lib/api";
import type { ReadinessItem } from "../../types";
import { Button } from "../ui/button";
import { CapabilityActions } from "./CapabilityActions";

export function StepReadiness({ title = "Validation" }: { title?: "Credentials" | "Apply" | "Validation" }) {
  const { data, isLoading, error, refetch } = useQuery({ queryKey: ["v2-readiness"], queryFn: fetchV2Readiness });
  const { data: capabilities, refetch: refetchCapabilities } = useQuery({ queryKey: ["capabilities"], queryFn: fetchCapabilities, enabled: title === "Credentials" });
  const { data: plan } = useQuery({ queryKey: ["v2-apply-plan"], queryFn: fetchV2ApplyPlan, enabled: title === "Apply" });
  const [values, setValues] = useState<Record<string, string>>({});
  const [provisioning, setProvisioning] = useState<string | null>(null);
  const [provisionError, setProvisionError] = useState<string | null>(null);
  const [applyState, setApplyState] = useState<{ run_id: string; status: string; items: Array<{ name: string; outcome: string; error?: string }> } | null>(null);
  const [applyError, setApplyError] = useState<string | null>(null);
  const [degradedAcknowledged, setDegradedAcknowledged] = useState(false);
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

  return <div data-testid="step-readiness">
    <h1 className="text-xl font-semibold sm:text-2xl">{title}</h1>
    <p className="mt-2 text-sm text-muted">This report uses manifest metadata and provider status only. Credential values are never displayed.</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</p>}
    {error && <p className="mt-6 text-danger" role="alert">Readiness could not be checked. Verify the control plane is available.</p>}
    {provisionError && <p className="mt-4 text-sm text-danger" role="alert">{provisionError}</p>}
    {applyError && <p className="mt-4 text-sm text-danger" role="alert">{applyError}</p>}
    {title === "Credentials" && capabilities && <CapabilityActions statuses={capabilities.capabilities} onRefresh={() => { void Promise.all([refetchCapabilities(), refetch()]); }} />}
    {data?.credential_diagnosis?.provider && <div data-testid="backend-diagnosis" className="mt-4 rounded-lg border border-warning/30 bg-warning-surface p-3 text-sm"><p className="font-medium text-warning">Credential provider diagnosis</p><p className="mt-1 text-foreground">{data.credential_diagnosis.provider.condition} · {data.credential_diagnosis.provider.backend}</p>{data.credential_diagnosis.provider.explanation && <p className="mt-1 text-xs text-muted">{data.credential_diagnosis.provider.explanation}</p>}{data.credential_diagnosis.provider.fix && <p className="mt-1 text-xs text-primary-soft">Next: {data.credential_diagnosis.provider.fix}</p>}{data.credential_diagnosis.provider.write_condition && <p className="mt-1 text-xs text-muted">Write reachability: {data.credential_diagnosis.provider.write_condition}. {data.credential_diagnosis.provider.write_fix}</p>}</div>}
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
      {title === "Validation" && <><ReadinessGroup title="Host requirements" items={data.hosts} /><ReadinessGroup title="Integrations" items={data.integrations.filter((item) => item.category === "integration")} /><ReadinessGroup title="System checks" items={data.integrations.filter((item) => item.category === "system")} /></>}
    </>}
  </div>;
}

function ReadinessGroup({ title, items }: { title: string; items: ReadinessItem[] }) {
  return <section className="mt-6"><h2 className="text-lg font-medium">{title}</h2>{items.length === 0 ? <p className="mt-2 text-sm text-muted">No declared requirements.</p> : <ul className="mt-3 space-y-2">{items.map((item) => <li key={`${item.kind ?? "item"}-${item.name}`} data-testid="readiness-item" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><span className="font-medium">{item.name}</span><span className="ml-2 text-muted">{item.kind ? `${item.kind} · ` : ""}{item.status}{item.required ? " · required" : ""}</span>{item.detail && <p className="mt-1 text-xs text-muted">{item.detail}</p>}{item.remediation && <p data-testid="remediation" className="mt-1 text-xs text-primary-soft">Next: {item.remediation}</p>}</li>)}</ul>}</section>;
}
