import { useCallback, useEffect, useState } from "react";
import { ChevronDown, Copy, ExternalLink, ShieldAlert } from "lucide-react";

import {
  BriefConsumer,
  BriefUseKind,
  getPortalBrief,
  listPortalBriefs,
  recordPortalBriefUse,
  type Brief,
} from "../../api/brief";
import { Button } from "../../components/ui/button";
import { cn } from "../../lib/utils";

function verdictLabel(brief: Brief): string {
  return brief.verdict === 1 ? "DELIVER" : "WITHHELD";
}

function consumerLabel(value: BriefConsumer): string {
  return ["ALL CONSUMERS", "PORTAL LLM", "PORTAL AGENT", "EXTERNAL HARNESS"][value] ?? "ALL CONSUMERS";
}

function trustLabel(value: number): string {
  return ["UNSPECIFIED", "FIRST_PARTY", "QUOTED", "EXTERNAL"][value] ?? "UNSPECIFIED";
}

function TrustChip({ value }: { value: number }) {
  return <span className="rounded-control bg-app-surface-muted px-2 py-0.5 text-xs font-medium">{trustLabel(value)}</span>;
}

export function BriefInspector({ chatId }: { chatId: string }) {
  const [briefs, setBriefs] = useState<Brief[]>([]);
  const [selected, setSelected] = useState<Brief | null>(null);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [consumer, setConsumer] = useState(BriefConsumer.UNSPECIFIED);

  const refresh = useCallback(async () => {
    if (!chatId) return;
    try {
      const next = await listPortalBriefs({ chatId, consumer, limit: 20 });
      setBriefs(next);
      setSelected((current) => current && next.some((brief) => brief.id === current.id) ? current : next[0] ?? null);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Brief inspector unavailable");
    }
  }, [chatId, consumer]);

  useEffect(() => { void refresh(); }, [refresh]);

  const choose = async (id: string) => {
    try {
      setSelected(await getPortalBrief(id));
      setOpen(true);
      await recordPortalBriefUse(id, -1, BriefUseKind.OPENED);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Brief unavailable");
    }
  };

  const copyItem = async (brief: Brief, index: number, text: string) => {
    await navigator.clipboard.writeText(text);
    await recordPortalBriefUse(brief.id, index, BriefUseKind.COPIED);
  };

  const openItem = async (brief: Brief, index: number) => {
    await recordPortalBriefUse(brief.id, index, BriefUseKind.OPENED);
  };

  const rejectItem = async (brief: Brief, index: number) => {
    await recordPortalBriefUse(brief.id, index, BriefUseKind.REJECTED);
  };

  if (!chatId || (!selected && briefs.length === 0 && !error)) return null;

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-3" aria-label="Context brief inspector">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">Context brief</p>
          <p className="text-sm text-app-muted-foreground">One Portal-owned verdict for this chat</p>
        </div>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => void refresh()}>Refresh</Button>
          <select
            aria-label="Filter context briefs"
            className="h-9 rounded-control border border-app-border bg-app-background px-2 text-sm"
            value={consumer}
            onChange={(event) => setConsumer(Number(event.target.value) as BriefConsumer)}
          >
            {[BriefConsumer.UNSPECIFIED, BriefConsumer.PORTAL_LLM, BriefConsumer.PORTAL_AGENT, BriefConsumer.EXTERNAL_HARNESS].map((value) => <option key={value} value={value}>{consumerLabel(value)}</option>)}
          </select>
          <select
            aria-label="Select context brief"
            className="h-9 rounded-control border border-app-border bg-app-background px-2 text-sm"
            value={selected?.id ?? ""}
            onChange={(event) => void choose(event.target.value)}
          >
            <option value="" disabled>Select brief</option>
            {briefs.map((brief) => <option key={brief.id} value={brief.id}>{verdictLabel(brief)} · {consumerLabel(brief.consumer)} · {brief.createdAt}</option>)}
          </select>
          <Button type="button" variant="outline" size="sm" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
            <ChevronDown className={cn("h-4 w-4 transition-transform", open && "rotate-180")} aria-hidden />
            Details
          </Button>
        </div>
      </div>
      {error ? <p className="mt-2 text-sm text-app-danger">{error}</p> : null}
      {selected ? (
        <div className="mt-3 grid gap-3 text-sm sm:grid-cols-4">
          <div><span className="text-app-muted-foreground">Verdict</span><p className="font-semibold">{verdictLabel(selected)}</p></div>
          <div><span className="text-app-muted-foreground">Consumer</span><p>{consumerLabel(selected.consumer)}</p></div>
          <div><span className="text-app-muted-foreground">Latency</span><p>{selected.latencyMs.toString()} ms</p></div>
          <div><span className="text-app-muted-foreground">Trust</span><p><TrustChip value={selected.maxTrustClass} /></p></div>
          <div><span className="text-app-muted-foreground">Providers</span><p>{selected.queriedProviders.length}</p></div>
          <div><span className="text-app-muted-foreground">Health</span><p>{selected.degraded ? "DEGRADED" : "HEALTHY"}</p></div>
        </div>
      ) : null}
      {open && selected ? (
        <div className="mt-3 space-y-3 border-t border-app-border pt-3">
          <div className="flex items-start gap-2 rounded-control bg-app-surface-muted p-2 text-sm">
            {selected.verdict === 1 ? <ExternalLink className="mt-0.5 h-4 w-4 text-app-success" aria-hidden /> : <ShieldAlert className="mt-0.5 h-4 w-4 text-app-warning" aria-hidden />}
            <div><p className="font-medium">{selected.reason || "No additional verdict reason"}</p><p className="text-app-muted-foreground">Effective query: {selected.effectiveQuery || "not applicable"}</p></div>
          </div>
          {selected.items.map((item, index) => (
            <article key={`${selected.id}-${index}`} className="rounded-control border border-app-border p-2">
              <div className="flex items-start justify-between gap-2">
                <div><p className="font-medium">{item.title || item.path || item.providerId}</p><p className="flex items-center gap-2 text-xs text-app-muted-foreground"><span>{item.providerId} · {item.type || "item"}</span><TrustChip value={item.trustClass} /></p></div>
                <div className="flex flex-wrap justify-end gap-2">
                  {item.path ? <Button type="button" variant="outline" size="sm" onClick={() => void openItem(selected, index)} aria-label={`Open ${item.path}`}>Open item</Button> : null}
                  {item.suggestedCommand ? <Button type="button" variant="outline" size="sm" onClick={() => void copyItem(selected, index, item.suggestedCommand)}><Copy className="mr-1 h-3.5 w-3.5" aria-hidden />Copy command</Button> : null}
                  <Button type="button" variant="outline" size="sm" onClick={() => void rejectItem(selected, index)}>Reject</Button>
                </div>
              </div>
              {item.snippet ? <p className="mt-1 text-sm text-app-muted-foreground">{item.snippet}</p> : null}
              {item.path ? <p className="mt-1 text-xs text-app-muted-foreground">{item.path}</p> : null}
            </article>
          ))}
          <p className="text-xs text-app-muted-foreground">Searched: {selected.queriedProviders.join(", ") || "none"}. Commands are copied as suggestions only; Portal never executes them.</p>
        </div>
      ) : null}
    </section>
  );
}

export function BriefMessage({ briefId }: { briefId: string }) {
  const [brief, setBrief] = useState<Brief | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!briefId) return;
    void getPortalBrief(briefId).then(setBrief).catch((err: unknown) => setError(err instanceof Error ? err.message : "Brief unavailable"));
  }, [briefId]);

  if (error) return <p className="mt-3 text-xs text-app-danger">Context brief unavailable: {error}</p>;
  if (!brief) return null;

  return (
    <section className="mt-3 rounded-control border border-app-border bg-app-background p-2 text-sm" aria-label="Message context brief">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="font-medium">Context brief · {verdictLabel(brief)}</span>
        <span className="text-xs text-app-muted-foreground">{brief.degraded ? "degraded" : "healthy"} · {brief.latencyMs.toString()} ms</span>
      </div>
      {brief.verdict !== 1 ? <p className="mt-1 text-xs text-app-muted-foreground">{brief.reason}</p> : null}
      {brief.items.length > 0 ? (
        <div className="mt-2 space-y-1">
          {brief.items.map((item, index) => (
            <div key={`${brief.id}-${index}`} className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-app-border p-2">
              <span>{item.title || item.path || item.providerId} <TrustChip value={item.trustClass} /></span>
              {item.path ? <Button type="button" variant="outline" size="sm" onClick={() => void recordPortalBriefUse(brief.id, index, BriefUseKind.OPENED)} aria-label={`Open ${item.path}`}>Open item</Button> : null}
            </div>
          ))}
        </div>
      ) : null}
    </section>
  );
}
