import { useCallback, useEffect, useState } from "react";
import { ChevronDown, Copy, ExternalLink, ShieldAlert } from "lucide-react";

import {
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

function trustLabel(value: number): string {
  return ["UNSPECIFIED", "FIRST_PARTY", "QUOTED", "EXTERNAL"][value] ?? "UNSPECIFIED";
}

export function BriefInspector({ chatId }: { chatId: string }) {
  const [briefs, setBriefs] = useState<Brief[]>([]);
  const [selected, setSelected] = useState<Brief | null>(null);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    if (!chatId) return;
    try {
      const next = await listPortalBriefs({ chatId, limit: 20 });
      setBriefs(next);
      setSelected((current) => current && next.some((brief) => brief.id === current.id) ? current : next[0] ?? null);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Brief inspector unavailable");
    }
  }, [chatId]);

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
            aria-label="Select context brief"
            className="h-9 rounded-control border border-app-border bg-app-background px-2 text-sm"
            value={selected?.id ?? ""}
            onChange={(event) => void choose(event.target.value)}
          >
            <option value="" disabled>Select brief</option>
            {briefs.map((brief) => <option key={brief.id} value={brief.id}>{verdictLabel(brief)} · {brief.createdAt}</option>)}
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
          <div><span className="text-app-muted-foreground">Latency</span><p>{selected.latencyMs.toString()} ms</p></div>
          <div><span className="text-app-muted-foreground">Trust</span><p>{trustLabel(selected.maxTrustClass)}</p></div>
          <div><span className="text-app-muted-foreground">Providers</span><p>{selected.queriedProviders.length}</p></div>
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
                <div><p className="font-medium">{item.title || item.path || item.providerId}</p><p className="text-xs text-app-muted-foreground">{item.providerId} · {trustLabel(item.trustClass)}</p></div>
                {item.suggestedCommand ? <Button type="button" variant="outline" size="sm" onClick={() => void copyItem(selected, index, item.suggestedCommand)}><Copy className="mr-1 h-3.5 w-3.5" aria-hidden />Copy command</Button> : null}
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
