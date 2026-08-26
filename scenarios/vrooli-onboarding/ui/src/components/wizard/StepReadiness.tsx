import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { acknowledgeDegraded, applyOnboarding, fetchCapabilities, fetchV2ApplyPlan, fetchV2ApplyStatus, fetchV2Readiness, provisionCredential } from "../../lib/api";
import { pollApplyRun } from "../../lib/applyRun";
import { ApplyPlanDisclosure } from "./ApplyPlanDisclosure";
import type { CompletionBlocker, ReadinessItem } from "../../types";
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
  const [applyReconnecting, setApplyReconnecting] = useState(false);
  const [acknowledging, setAcknowledging] = useState(false);
  const [acknowledgeError, setAcknowledgeError] = useState<string | null>(null);
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
      await pollApplyRun(result, {
        fetchStatus: (runID) => fetchV2ApplyStatus(runID, controller.signal),
        onUpdate: (current) => setApplyState({
          run_id: current.run_id,
          status: current.status,
          items: current.items.map((item) => ({ name: item.name, outcome: item.outcome, error: item.error })),
        }),
        onConnectionChange: (connected) => setApplyReconnecting(!connected),
        wait: (ms) => new Promise((resolve) => window.setTimeout(resolve, ms)),
        now: () => Date.now(),
      });
      setApplyReconnecting(false);
      await refetch();
    } catch {
      setApplyReconnecting(false);
      setApplyError("The selection could not be applied. Resolve the reported host issue and try again.");
    }
  };

  const acceptDegraded = async () => {
    const digest = data?.degraded_digest;
    if (!digest) return;
    setAcknowledging(true);
    setAcknowledgeError(null);
    try {
      await acknowledgeDegraded(digest);
      await refetch();
    } catch {
      setAcknowledgeError("The acknowledgement was not recorded. Recheck readiness and try again.");
    } finally {
      setAcknowledging(false);
    }
  };

  return <div data-testid="step-readiness">
    <h1 className="text-xl font-semibold sm:text-2xl">{title}</h1>
    <p className="mt-2 text-sm text-muted">This report uses manifest metadata and provider status only. Credential values are never displayed.</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-muted" role="status"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</p>}
    {error && <p className="mt-6 text-danger" role="alert">Readiness could not be checked. Verify the control plane is available.</p>}
    {provisionError && <p className="mt-4 text-sm text-danger" role="alert">{provisionError}</p>}
    {applyError && <p className="mt-4 text-sm text-danger" role="alert">{applyError}</p>}
    {applyReconnecting && <p className="mt-4 text-sm text-muted" role="status" data-testid="apply-reconnecting">The onboarding API restarted while applying. The run is still going; reconnecting&hellip;</p>}
    {title === "Credentials" && capabilities && <CapabilityActions statuses={capabilities.capabilities} onRefresh={() => { void Promise.all([refetchCapabilities(), refetch()]); }} />}
    {data?.credential_diagnosis?.provider && <div data-testid="backend-diagnosis" className="mt-4 rounded-lg border border-warning/30 bg-warning-surface p-3 text-sm"><p className="font-medium text-warning">Credential provider diagnosis</p><p className="mt-1 text-foreground">{data.credential_diagnosis.provider.condition} · {data.credential_diagnosis.provider.backend}</p>{data.credential_diagnosis.provider.explanation && <p className="mt-1 text-xs text-muted">{data.credential_diagnosis.provider.explanation}</p>}{data.credential_diagnosis.provider.fix && <p className="mt-1 text-xs text-primary-soft">Next: {data.credential_diagnosis.provider.fix}</p>}{data.credential_diagnosis.provider.write_condition && <p className="mt-1 text-xs text-muted">Write reachability: {data.credential_diagnosis.provider.write_condition}. {data.credential_diagnosis.provider.write_fix}</p>}</div>}
      {data && <>
      <div className="mt-6 flex items-center gap-2" role="status" data-testid="readiness-summary">{data.status === "ready" ? <CheckCircle2 className="h-5 w-5 text-primary" /> : <CircleAlert className="h-5 w-5 text-warning" />}<span className={data.status === "ready" ? "text-primary" : "text-warning"}>{data.status === "ready" ? "Ready" : `${data.status}: action required`}</span></div>
      {title === "Apply" && <section className="mt-4" aria-label="Apply plan">
        <h2 className="text-lg font-medium">Review apply plan</h2>
        <ApplyPlanDisclosure items={plan?.items ?? []} />
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <Button data-testid="apply-confirm" type="button" onClick={() => { void apply(); }} className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary">Apply selection</Button>
          {applyState && <span data-testid="apply-progress" role="progressbar" aria-valuetext={`${applyState.status} · ${applyState.items.length} items`} className="text-sm text-muted">{applyState.status} · {applyState.items.length} items</span>}
        </div>
      </section>}
      {applyState && <table className="mt-3 w-full text-left text-xs text-muted" data-testid="apply-report" role="table"><caption className="sr-only">Apply results</caption><thead><tr><th scope="col">Item</th><th scope="col">Outcome</th></tr></thead><tbody>{applyState.items.map((item) => <tr key={item.name}><td className="py-1">{item.name}</td><td className="py-1">{item.outcome}{item.error ? ` — ${item.error}` : ""}</td></tr>)}</tbody></table>}
      {applyState?.items.some((item) => item.outcome === "blocked" || item.outcome === "failed") && <><p data-testid="skipped-note" role="note" className="mt-3 text-sm text-warning">Some items were skipped or failed. Resolve the reported dependency or host issue before trying again.</p><Button data-testid="retry" type="button" variant="outline" onClick={() => { void apply(); }}>Apply again</Button></>}
      {title === "Validation" && <>
        <BlockerList title="Blocking" testID="readiness-blockers" items={data.blockers ?? []} tone="danger" />
        <BlockerList title="Degraded (optional)" testID="readiness-degraded" items={data.degraded ?? []} tone="warning" />
        {acknowledgeError && <p className="mt-3 text-sm text-danger" role="alert">{acknowledgeError}</p>}
        <div className="mt-4 flex flex-wrap gap-3">
          <Button data-testid="recheck" type="button" variant="outline" onClick={() => { void refetch(); }}>Recheck</Button>
          {(data.blockers ?? []).length === 0 && (data.degraded ?? []).length > 0 && !data.degraded_acknowledged && <Button data-testid="readiness-continue-degraded" type="button" variant="outline" disabled={acknowledging} onClick={() => { void acceptDegraded(); }}>{acknowledging ? "Recording…" : "Accept the degraded items"}</Button>}
          {(data.degraded ?? []).length > 0 && data.degraded_acknowledged && <p role="status" className="text-sm text-warning">The degraded items above are recorded as accepted.</p>}
        </div>
        {(data.blockers ?? []).length > 0 && <p data-testid="finish-blocked" role="note" className="mt-3 text-sm text-danger">Configuration cannot be reported complete while a blocking item remains. Resolve the items above, then recheck.</p>}
      </>}
      <ul className="mt-4 space-y-2">{data.credentials.map((credential) => {
        const key = `${credential.logical_id}/${credential.field}`;
        const componentSupplied = credential.provisioning === "derived" || credential.provisioning === "generated";
        const canProvision = title === "Credentials" && !componentSupplied && credential.status !== "configured";
        return <li key={key} data-testid="credential-card" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><p data-testid="credential-purpose" role="note" className="font-medium">{credential.label || credential.field}</p><span data-testid="credential-status" role="status" className="ml-2 text-muted">{credential.required ? "Required" : "Optional"} · {credential.status}</span>{credential.provisioning === "derived" && <p className="mt-1 text-xs text-muted">Provided by {credential.derived_from || "the owning component"} after its source credential is available.</p>}{credential.provisioning === "generated" && <p className="mt-1 text-xs text-muted">Generated by the owning component on first start. There is nothing to enter.</p>}{credential.description && <p className="mt-1 text-xs text-muted">{credential.description}</p>}{credential.obtain_url && <a data-testid="credential-obtain-link" className="mt-1 inline-flex min-h-11 items-center rounded px-2 text-xs text-primary-soft underline" href={credential.obtain_url}>How to obtain this credential</a>}{credential.detail && <p className="mt-1 text-xs text-muted">{credential.detail}</p>}{canProvision && <div className="mt-3 flex flex-col gap-2 sm:flex-row"><input data-testid="credential-input" aria-label={`Value for ${credential.label || credential.field}`} type="password" autoComplete="off" value={values[key] ?? ""} onChange={(event) => setValues((current) => ({ ...current, [key]: event.target.value }))} className="min-h-11 min-w-0 flex-1 rounded-md border border-muted bg-surface px-3 py-2 text-sm" /><Button data-testid="credential-save" type="button" disabled={!values[key]?.trim() || provisioning === key} onClick={() => { void provision(credential.logical_id, credential.field); }} className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-on-primary disabled:opacity-50">{provisioning === key ? "Saving…" : "Save securely"}</Button></div>}</li>;
      })}</ul>
      {title === "Validation" && <><ReadinessGroup title="Host requirements" items={data.hosts} /><ReadinessGroup title="Integrations" items={data.integrations.filter((item) => item.category === "integration")} /><ReadinessGroup title="System checks" items={data.integrations.filter((item) => item.category === "system")} /></>}
    </>}
  </div>;
}

/**
 * BlockerList shows the named reasons configuration is not complete. It renders
 * nothing when the list is empty, so a clean verdict stays quiet.
 */
function BlockerList({ title, testID, items, tone }: { title: string; testID: string; items: CompletionBlocker[]; tone: "danger" | "warning" }) {
  if (items.length === 0) return null;
  const border = tone === "danger" ? "border-danger/40" : "border-warning/40";
  const text = tone === "danger" ? "text-danger" : "text-warning";
  return <section className="mt-4" aria-label={title}>
    <h2 className={`text-lg font-medium ${text}`}>{title}</h2>
    <ul data-testid={testID} role="list" className={`mt-2 space-y-2 rounded-lg border ${border} p-3`}>
      {items.map((item) => <li key={`${item.kind}-${item.name}`} data-testid={`${testID}-item`} className="text-sm">
        <span className="font-medium">{item.kind}: {item.name}</span>
        <p className="mt-1 text-xs text-muted">{item.reason}</p>
        <p className="mt-1 text-xs text-primary-soft">Next: {item.remediation}</p>
      </li>)}
    </ul>
  </section>;
}

function ReadinessGroup({ title, items }: { title: string; items: ReadinessItem[] }) {
  return <section className="mt-6"><h2 className="text-lg font-medium">{title}</h2>{items.length === 0 ? <p className="mt-2 text-sm text-muted">No declared requirements.</p> : <ul className="mt-3 space-y-2">{items.map((item) => <li key={`${item.kind ?? "item"}-${item.name}`} data-testid="readiness-item" className="rounded-lg border border-muted bg-surface-muted p-3 text-sm"><span className="font-medium">{item.name}</span><span className="ml-2 text-muted">{item.kind ? `${item.kind} · ` : ""}{item.status}{item.required ? " · required" : ""}</span>{item.detail && <p className="mt-1 text-xs text-muted">{item.detail}</p>}{item.remediation && <p data-testid="remediation" className="mt-1 text-xs text-primary-soft">Next: {item.remediation}</p>}</li>)}</ul>}</section>;
}
