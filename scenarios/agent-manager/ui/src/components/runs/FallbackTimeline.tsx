// Run-scoped timeline of typed runner+model fallback events.
//
// Sources every typed `*.fallback.*` event from /api/v1/events?run=<id>
// and renders an ordered list with timestamps, reasons, and chain
// position. Collapsible by default to keep the run page short.

import { ChevronDown, ChevronRight } from "lucide-react";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "../ui/button";
import { formatStandardDateTime } from "../../lib/dateTime";
import {
  fetchEventsForRun,
  eventsQueryKeys,
  type TypedEventRow,
} from "../../features/health/api/eventsClient";

type FallbackKind = "runner.fallback.attempted" | "runner.fallback.exhausted" | "model.fallback.attempted" | "model.fallback.exhausted";

const FALLBACK_TYPES = new Set<FallbackKind>([
  "runner.fallback.attempted",
  "runner.fallback.exhausted",
  "model.fallback.attempted",
  "model.fallback.exhausted",
]);

interface FallbackTimelineProps {
  runId: string;
  defaultOpen?: boolean;
}

export function FallbackTimeline({ runId, defaultOpen = false }: FallbackTimelineProps) {
  const [open, setOpen] = useState(defaultOpen);
  const { data, isLoading, error } = useQuery({
    queryKey: eventsQueryKeys.forRun(runId),
    queryFn: () => fetchEventsForRun(runId),
    enabled: open,
    staleTime: 10_000,
  });

  const fallbackEvents = useMemo(() => {
    if (!data) return [] as TypedEventRow[];
    return data.events.filter((e) => FALLBACK_TYPES.has(e.event_type as FallbackKind));
  }, [data]);

  return (
    <section className="rounded-lg border border-border bg-card/40" data-testid="fallback-timeline">
      <Button
        variant="ghost"
        className="w-full justify-between px-3 py-2 text-sm font-medium"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <span className="flex items-center gap-2">
          {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          Fallback timeline
        </span>
        {!isLoading && data ? (
          <span className="text-xs text-muted-foreground">{fallbackEvents.length} events</span>
        ) : null}
      </Button>
      {open ? (
        <div className="border-t border-border/60 px-3 py-2">
          {isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : error ? (
            <p className="text-sm text-destructive">Failed to load fallback events: {(error as Error).message}</p>
          ) : fallbackEvents.length === 0 ? (
            <p className="text-sm text-muted-foreground" data-testid="fallback-timeline-empty">
              No runner or model fallback events for this run.
            </p>
          ) : (
            <ol className="space-y-2" data-testid="fallback-timeline-list">
              {fallbackEvents.map((ev) => (
                <li
                  key={ev.id}
                  className="rounded-md border border-border/60 bg-background/40 p-2 text-sm"
                  data-testid={`fallback-event-${ev.id}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-mono text-xs">{ev.event_type}</span>
                    <span className="text-xs text-muted-foreground">
                      {formatStandardDateTime(ev.timestamp)}
                    </span>
                  </div>
                  <FallbackPayloadSummary type={ev.event_type as FallbackKind} payload={ev.payload} />
                </li>
              ))}
            </ol>
          )}
        </div>
      ) : null}
    </section>
  );
}

function FallbackPayloadSummary({ type, payload }: { type: FallbackKind; payload: unknown }) {
  if (typeof payload !== "object" || payload === null) {
    return null;
  }
  const p = payload as Record<string, unknown>;
  switch (type) {
    case "runner.fallback.attempted":
    case "model.fallback.attempted": {
      const from = String(p.from ?? "");
      const to = String(p.to ?? "");
      const reason = String(p.reason ?? "");
      const attempt = typeof p.attempt_no === "number" ? p.attempt_no : undefined;
      const chainPos = typeof p.chain_position === "number" ? p.chain_position : undefined;
      const chainLen = typeof p.chain_length === "number" ? p.chain_length : undefined;
      return (
        <div className="mt-1 text-xs text-muted-foreground">
          <span className="font-mono">{from}</span> → <span className="font-mono">{to}</span>
          {reason ? <span> · reason: <span className="font-mono">{reason}</span></span> : null}
          {attempt !== undefined ? <span> · attempt {attempt}</span> : null}
          {chainPos !== undefined && chainLen !== undefined ? (
            <span> · {chainPos}/{chainLen}</span>
          ) : null}
        </div>
      );
    }
    case "runner.fallback.exhausted": {
      const tried = Array.isArray(p.candidates_tried) ? (p.candidates_tried as unknown[]).join(", ") : "";
      return (
        <div className="mt-1 text-xs text-muted-foreground">
          primary: <span className="font-mono">{String(p.primary ?? "")}</span>
          {tried ? <span> · tried: <span className="font-mono">{tried}</span></span> : null}
          {p.last_reason ? <span> · last_reason: <span className="font-mono">{String(p.last_reason)}</span></span> : null}
        </div>
      );
    }
    case "model.fallback.exhausted": {
      const chain = Array.isArray(p.chain) ? (p.chain as unknown[]).join(", ") : "";
      return (
        <div className="mt-1 text-xs text-muted-foreground">
          preset: <span className="font-mono">{String(p.preset ?? "")}</span>
          {chain ? <span> · chain: <span className="font-mono">{chain}</span></span> : null}
          {p.last_reason ? <span> · last_reason: <span className="font-mono">{String(p.last_reason)}</span></span> : null}
        </div>
      );
    }
  }
}
