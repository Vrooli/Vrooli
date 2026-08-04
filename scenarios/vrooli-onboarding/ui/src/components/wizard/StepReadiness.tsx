import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { fetchV2Readiness, provisionCredential } from "../../lib/api";

export function StepReadiness({ title = "Validation" }: { title?: "Credentials" | "Validation" }) {
  const { data, isLoading, error, refetch } = useQuery({ queryKey: ["v2-readiness"], queryFn: fetchV2Readiness });
  const [values, setValues] = useState<Record<string, string>>({});
  const [provisioning, setProvisioning] = useState<string | null>(null);
  const [provisionError, setProvisionError] = useState<string | null>(null);

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
  return <div data-testid="step-readiness">
    <h1 className="text-xl font-semibold sm:text-2xl">{title}</h1>
    <p className="mt-2 text-sm text-slate-300">This report uses manifest metadata and provider status only. Credential values are never displayed.</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-slate-300" role="status"><Loader2 className="h-4 w-4 animate-spin" />Checking readiness…</p>}
    {error && <p className="mt-6 text-red-400" role="alert">Readiness could not be checked. Verify the control plane is available.</p>}
    {provisionError && <p className="mt-4 text-sm text-red-400" role="alert">{provisionError}</p>}
    {data?.credential_diagnosis?.provider && <div className="mt-4 rounded-lg border border-amber-300/30 bg-amber-300/10 p-3 text-sm"><p className="font-medium text-amber-200">Credential provider diagnosis</p><p className="mt-1 text-slate-200">{data.credential_diagnosis.provider.condition} · {data.credential_diagnosis.provider.backend}</p>{data.credential_diagnosis.provider.explanation && <p className="mt-1 text-xs text-slate-300">{data.credential_diagnosis.provider.explanation}</p>}{data.credential_diagnosis.provider.fix && <p className="mt-1 text-xs text-emerald-200">Next: {data.credential_diagnosis.provider.fix}</p>}{data.credential_diagnosis.provider.write_condition && <p className="mt-1 text-xs text-slate-300">Write reachability: {data.credential_diagnosis.provider.write_condition}. {data.credential_diagnosis.provider.write_fix}</p>}</div>}
    {data && <><div className="mt-6 flex items-center gap-2" role="status">{data.status === "ready" ? <CheckCircle2 className="h-5 w-5 text-emerald-400" /> : <CircleAlert className="h-5 w-5 text-amber-300" />}<span className={data.status === "ready" ? "text-emerald-300" : "text-amber-200"}>{data.status === "ready" ? "Ready" : `${data.status}: action required`}</span></div><ul className="mt-4 space-y-2">{data.credentials.map((credential) => {
      const key = `${credential.logical_id}/${credential.field}`;
      const canProvision = title === "Credentials" && credential.status !== "configured";
	  return <li key={key} className="rounded-lg border border-white/10 bg-white/5 p-3 text-sm"><span className="font-medium">{credential.label || credential.field}</span><span className="ml-2 text-slate-300">{credential.required ? "Required" : "Optional"} · {credential.status}</span>{credential.detail && <p className="mt-1 text-xs text-slate-300">{credential.detail}</p>}{canProvision && <div className="mt-3 flex flex-col gap-2 sm:flex-row"><input aria-label={`Value for ${credential.label || credential.field}`} type="password" autoComplete="off" value={values[key] ?? ""} onChange={(event) => setValues((current) => ({ ...current, [key]: event.target.value }))} className="min-w-0 flex-1 rounded-md border border-white/20 bg-slate-950 px-3 py-2 text-sm" /><button type="button" disabled={!values[key]?.trim() || provisioning === key} onClick={() => { void provision(credential.logical_id, credential.field); }} className="rounded-md bg-emerald-500 px-3 py-2 text-sm font-medium text-slate-950 disabled:opacity-50">{provisioning === key ? "Saving…" : "Save securely"}</button></div>}</li>;
    })}</ul>{title === "Validation" && <><ReadinessGroup title="Host requirements" items={data.hosts} /><ReadinessGroup title="Integrations" items={data.integrations} /></>}</>}
  </div>;
}

function ReadinessGroup({ title, items }: { title: string; items: Array<{ name: string; status: string; detail?: string; remediation?: string; kind?: string; required?: boolean }> }) {
  return <section className="mt-6"><h2 className="text-lg font-medium">{title}</h2>{items.length === 0 ? <p className="mt-2 text-sm text-slate-300">No declared requirements.</p> : <ul className="mt-3 space-y-2">{items.map((item) => <li key={`${item.kind ?? "item"}-${item.name}`} className="rounded-lg border border-white/10 bg-white/5 p-3 text-sm"><span className="font-medium">{item.name}</span><span className="ml-2 text-slate-300">{item.kind ? `${item.kind} · ` : ""}{item.status}{item.required ? " · required" : ""}</span>{item.detail && <p className="mt-1 text-xs text-slate-300">{item.detail}</p>}{item.remediation && <p className="mt-1 text-xs text-emerald-200">Next: {item.remediation}</p>}</li>)}</ul>}</section>;
}
