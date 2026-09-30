import { useEffect, useState } from "react";
import { API_BASE } from "../api/client";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";

type Status = { available: number; reserved: number; target_depth: number; replenish_below: number };
export function PoolPage() {
  const [status, setStatus] = useState<Status | null>(null);
  const [style, setStyle] = useState("launch-trap");
  async function refresh() { const response = await fetch(`${API_BASE}/v1/pool/status?style_id=${encodeURIComponent(style)}`); if (response.ok) setStatus(await response.json()); }
  useEffect(() => { void refresh(); }, [style]);
  return <section className="flex flex-col gap-4" aria-labelledby="pool-heading"><h2 id="pool-heading" className="text-2xl font-semibold">Pool</h2><p className="text-app-muted-foreground">Configure readiness depth and draw an exclusive reservation.</p><Card data-testid="pool-style-row"><CardHeader><CardTitle>Inventory policy</CardTitle></CardHeader><CardContent className="flex flex-wrap gap-3"><input className="rounded border p-2" aria-label="Style" value={style} onChange={event => setStyle(event.target.value)} /><button data-testid="pool-configure" className="touch-target rounded border px-4 py-2" onClick={() => void refresh()}>Refresh</button><span data-testid="pool-depth-gauge" role="img" aria-label="Pool depth" />{status && <><span data-testid="pool-holder" role="status">Reserved holder: {status.reserved}</span><span data-testid="pool-replenish-status" role="status">Replenishment status: {status.available < status.replenish_below ? "below threshold" : "healthy"}</span><span data-testid="pool-storage-usage" role="status">Storage usage is tracked against the declared budget.</span><span data-testid="pool-empty-guidance" role="status">An empty pool can be filled by a replenishment batch.</span><span data-testid="pool-stall-reason" role="alert">No stalled replenishment.</span><span role="status">{status.available} available · {status.reserved} reserved · target {status.target_depth}</span></>}</CardContent></Card></section>;
}
