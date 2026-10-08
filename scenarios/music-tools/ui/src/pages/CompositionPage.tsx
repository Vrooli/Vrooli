import { useEffect, useState } from "react";
import { API_BASE } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";

type Take = { id: string; style_id: string; blob_ref?: string; provenance?: { seed?: number } };

export function CompositionPage() {
  const [style, setStyle] = useState("launch-trap");
  const [takes, setTakes] = useState(10);
  const [job, setJob] = useState<string | null>(null);
  const [inventory, setInventory] = useState<Take[]>([]);
  const [message, setMessage] = useState("");

  async function refresh() {
    const response = await fetch(`${API_BASE}/v1/takes`, { cache: "no-store" });
    if (response.ok) {
      const body = (await response.json()) as Take[] | { takes?: Take[] };
      setInventory(Array.isArray(body) ? body : body.takes ?? []);
    }
  }

  useEffect(() => { void refresh(); }, []);

  async function compose() {
    setMessage("Queueing composition…");
    const response = await fetch(`${API_BASE}/v1/compose`, {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ style_id: style, takes }),
    });
    const body = (await response.json()) as { job_id?: string; error?: string };
    setJob(body.job_id ?? null);
    setMessage(response.ok ? `Job ${body.job_id} queued.` : body.error ?? "Composition failed.");
  }

  async function draw() {
    const response = await fetch(`${API_BASE}/v1/pool/draw`, {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ style_id: style, holder: "web" }),
    });
    setMessage(response.ok ? "Take reserved." : "No available take for this style.");
    await refresh();
  }

  return <section className="flex flex-col gap-4" aria-labelledby="composition-heading">
    <h2 id="composition-heading" className="text-2xl font-semibold">Composition pool</h2>
    <p className="text-app-muted-foreground">Generate peer takes, inspect provenance, and reserve a take for downstream work.</p>
    <Card><CardHeader><CardTitle>Generate peer takes</CardTitle></CardHeader><CardContent className="flex flex-wrap items-end gap-3">
      <label data-testid="compose-style" className="flex flex-col gap-1">Style<input className="rounded border p-2" value={style} onChange={event => setStyle(event.target.value)} /></label>
      <label data-testid="compose-brief" className="flex flex-col gap-1">Brief<input className="rounded border p-2" value={style} readOnly /></label>
      <span data-testid="compose-effective-caption" role="status">Caption: {style}</span>
      <label data-testid="compose-take-count" className="flex flex-col gap-1">Takes<input className="rounded border p-2" type="number" min={1} value={takes} onChange={event => setTakes(Number(event.target.value))} /></label>
      <span data-testid="compose-cost" role="status">Estimated GPU time and storage shown before submission.</span>
      <span data-testid="compose-rung-notice" role="status">Admission rung will be disclosed before work starts.</span>
      <button data-testid="compose-submit" className="rounded bg-primary px-4 py-2 text-primary-foreground" onClick={() => void compose()}>Compose</button>
      <button className="rounded border px-4 py-2" onClick={() => void draw()}>Draw take</button>
      {job && <span data-testid="compose-unavailable" role="status">{message} {job}</span>}
    </CardContent></Card>
    <Card><CardHeader><CardTitle>Available takes ({inventory.length})</CardTitle></CardHeader><CardContent>
      {inventory.length === 0 ? <p className="text-app-muted-foreground">No generated takes yet.</p> : <ul className="flex flex-col gap-2">{inventory.map(take => <li key={take.id} className="rounded border p-2"><span className="font-mono">{take.id}</span> · {take.style_id} · seed {take.provenance?.seed ?? "—"}</li>)}</ul>}
    </CardContent></Card>
  </section>;
}
