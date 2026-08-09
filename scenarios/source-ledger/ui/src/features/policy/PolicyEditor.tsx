import { useEffect, useState } from "react";

import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "../../api/client";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";

type Snapshot = {
  frontierTarget: number; wakeBudgetLines: number; wakeBudgetChars: number; maxEntryLines: number; maxEntryChars: number;
  frontierTargetOrigin: string; wakeBudgetLinesOrigin: string; wakeBudgetCharsOrigin: string; maxEntryLinesOrigin: string; maxEntryCharsOrigin: string;
};
type Liveness = { unsummarizedLeafCount: number; oldestUnsummarizedLeafAt: string; lastSummaryAt: string };
type PolicyResponse = { effective: Snapshot; defaults: Snapshot; liveness?: Liveness };
type Key = "frontierTarget" | "wakeBudgetLines" | "wakeBudgetChars" | "maxEntryLines" | "maxEntryChars";
const fields: { key: Key; label: string }[] = [
  { key: "frontierTarget", label: "Frontier target" }, { key: "wakeBudgetLines", label: "Whole-view lines" },
  { key: "wakeBudgetChars", label: "Whole-view characters" }, { key: "maxEntryLines", label: "Per-entry lines" },
  { key: "maxEntryChars", label: "Per-entry characters" },
];

async function post<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body), cache: "no-store" });
  if (!response.ok) throw new Error(`Source Ledger request failed (${response.status})`);
  return (await response.json()) as T;
}

export function PolicyEditor({ scope }: { scope: string }) {
  const [data, setData] = useState<PolicyResponse>();
  const [values, setValues] = useState<Partial<Record<Key, number>>>({});
  const [dirty, setDirty] = useState<Set<Key>>(new Set());
  const [message, setMessage] = useState<string>();
  const [error, setError] = useState<string>();
  const load = async () => {
    try { setError(undefined); const response = await post<PolicyResponse>("/vrooli.source_ledger.v1.scopes.ScopesService/GetPolicy", { scope }); setData(response); setValues(response.effective); setDirty(new Set()); }
    catch (cause) { setError(String(cause)); }
  };
  useEffect(() => { void load(); }, [scope]);
  const save = async () => {
    const body: Record<string, number | string> = { scope };
    for (const key of dirty) { const value = values[key]; if (value !== undefined) body[key] = value; }
    try { const response = await post<PolicyResponse>("/vrooli.source_ledger.v1.scopes.ScopesService/SetPolicy", body); setData(response); setValues(response.effective); setDirty(new Set()); setMessage("Policy saved; the next wake uses it."); }
    catch (cause) { setError(String(cause)); }
  };
  const reset = async () => {
    try { const response = await post<PolicyResponse>("/vrooli.source_ledger.v1.scopes.ScopesService/ResetPolicy", { scope }); setData(response); setValues(response.effective); setDirty(new Set()); setMessage("Policy reset; this scope inherits file defaults."); }
    catch (cause) { setError(String(cause)); }
  };
  const effective = data?.effective;
  const defaults = data?.defaults;
  if (!effective || !defaults) return <Card aria-labelledby="policy-heading"><CardHeader><CardTitle id="policy-heading">Context policy</CardTitle></CardHeader><CardContent><p className="text-sm text-app-muted-foreground">Loading policy…</p></CardContent></Card>;
  return <Card aria-labelledby="policy-heading"><CardHeader><CardTitle id="policy-heading">Context policy</CardTitle><p className="text-sm text-app-muted-foreground">Change only the fields you intend to override; all other values inherit the file default.</p></CardHeader><CardContent className="space-y-3">
    {error && <p role="alert" className="text-app-danger">{error}</p>}{message && <p role="status" className="text-app-success">{message}</p>}
    {fields.map(({ key, label }) => <label key={key} className="grid gap-1"><span className="text-sm font-medium">{label}</span><span className="flex flex-wrap items-center gap-2"><input aria-label={label} type="number" min="1" value={values[key] ?? ""} onChange={(event) => { setValues((current) => ({ ...current, [key]: Number(event.target.value) })); setDirty((current) => new Set(current).add(key)); }} className="w-40 rounded-control border border-app-border bg-app-background px-3 py-2" /><span className="text-xs text-app-muted-foreground">origin: {effective ? String(effective[`${key}Origin` as keyof Snapshot]) : "loading"} · default: {defaults ? defaults[key] : "loading"}</span></span></label>)}
    {data.liveness && <div className="rounded-control border border-app-border p-3 text-sm"><div className="font-medium">Compaction liveness</div><div className="text-xs text-app-muted-foreground">Unsummarized leaves: {data.liveness.unsummarizedLeafCount} · oldest: {data.liveness.oldestUnsummarizedLeafAt || "none"} · last summary: {data.liveness.lastSummaryAt || "none"}</div></div>}
    <div className="flex flex-wrap gap-2"><Button onClick={() => void save()} disabled={dirty.size === 0}>Save overrides</Button><Button variant="secondary" onClick={() => void reset()}>Reset to defaults</Button></div>
  </CardContent></Card>;
}
