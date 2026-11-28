import { memo, useCallback, useEffect, useState } from "react";
import { ChevronRight } from "lucide-react";
import { CollapsibleRegion } from "@vrooli/react-component-library/CollapsibleRegion";
import { Spinner } from "@vrooli/react-component-library/Spinner";
import { getConversationWork, type ConversationEvent, type ConversationWorkSpan } from "../../api/conversation";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { useTranslate } from "./useTranslate";
import { workKindLabel, workRowLine, workSummary, type Translate } from "./workSummary";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#work-between-messages

interface WorkSpanRowProps {
  sessionId: string;
  /** The work row: its span is what the line summarises and what opens. */
  event: ConversationEvent & { work: ConversationWorkSpan };
  /** When the conversation row before and after it were written (the line's duration). */
  previousAt?: string;
  nextAt?: string;
  expanded: boolean;
  onToggle: (eventId: string) => void;
  /** Search hits inside this span, highlighted once it opens. */
  highlightSequences?: ReadonlySet<number>;
}

type WorkLoad =
  | { state: "loading" }
  | { state: "failed" }
  | { state: "loaded"; rows: ConversationEvent[] };

// A span's rows never change once its range is closed, so an opened span
// stays loaded across scroll (rows remount as the list virtualises). Bounded:
// the oldest opened span is forgotten first.
const loadedWork = new Map<string, ConversationEvent[]>();
const LOADED_WORK_LIMIT = 100;

function rememberWork(key: string, rows: ConversationEvent[]): void {
  loadedWork.delete(key);
  loadedWork.set(key, rows);
  while (loadedWork.size > LOADED_WORK_LIMIT) {
    const oldest = loadedWork.keys().next().value;
    if (oldest == null) break;
    loadedWork.delete(oldest);
  }
}

function workKey(sessionId: string, event: ConversationEvent & { work: ConversationWorkSpan }): string {
  return `${sessionId}:${event.threadId ?? ""}:${String(event.work.fromSequence)}-${String(event.work.toSequence)}`;
}

/**
 * The work between two messages, collapsed into one quiet line — "Worked for
 * 20s · 10 tool calls" — that opens in place to list the tool calls, results,
 * reasoning and injected context it stands for, each one line until tapped.
 * Rows are fetched when the span opens, as previews, so the list never
 * carries tool output it does not show.
 */
function WorkSpanRowImpl({ sessionId, event, previousAt, nextAt, expanded, onToggle, highlightSequences }: WorkSpanRowProps) {
  const t = useTranslate();
  const summary = workSummary(event.work, t, previousAt, nextAt);
  const key = workKey(sessionId, event);
  const [load, setLoad] = useState<WorkLoad>(() => {
    const cached = loadedWork.get(key);
    return cached ? { state: "loaded", rows: cached } : { state: "loading" };
  });
  const fetchRows = useCallback(() => {
    setLoad({ state: "loading" });
    let cancelled = false;
    void getConversationWork(sessionId, event.work, event.threadId ?? "").then((rows) => {
      rememberWork(key, rows);
      if (!cancelled) setLoad({ state: "loaded", rows });
    }).catch(() => {
      if (!cancelled) setLoad({ state: "failed" });
    });
    return () => { cancelled = true; };
  }, [event.threadId, event.work, key, sessionId]);
  useEffect(() => {
    if (!expanded) return;
    const cached = loadedWork.get(key);
    if (cached) {
      setLoad({ state: "loaded", rows: cached });
      return;
    }
    return fetchRows();
  }, [expanded, fetchRows, key]);

  if (summary.length === 0) return null;
  const regionId = `work-rows-${event.id}`;
  return (
    <div data-testid={`work-row-${event.id}`} data-expanded={expanded || undefined} className="border-b border-wc-default/60 px-3">
      <button
        type="button"
        data-testid={`work-toggle-${event.id}`}
        aria-expanded={expanded}
        aria-controls={regionId}
        aria-label={`${summary.join(" · ")}. ${t(expanded ? strings.messagesPane.work.hideAria : strings.messagesPane.work.showAria)}`}
        onClick={() => { onToggle(event.id); }}
        className="flex min-h-9 w-full min-w-0 items-center gap-1.5 py-1 text-start text-[12px] text-wc-text-faint transition-colors hover:text-wc-text-secondary"
      >
        <ChevronRight aria-hidden="true" className={cn("h-3.5 w-3.5 shrink-0 transition-transform rtl:-scale-x-100", expanded && "rotate-90 rtl:rotate-90")} />
        <span data-testid={`work-summary-${event.id}`} className="truncate">{summary.join(" · ")}</span>
      </button>
      <CollapsibleRegion open={expanded}>
        <div id={regionId} data-testid={`work-rows-${event.id}`} className="pb-2 ps-5">
          {load.state === "loading" && (
            <div className="flex items-center gap-2 py-1 text-[12px] text-wc-text-faint">
              <Spinner label={t(strings.messagesPane.work.loading)} className="h-3.5 w-3.5" />
              {t(strings.messagesPane.work.loading)}
            </div>
          )}
          {load.state === "failed" && (
            <div role="alert" className="flex items-center gap-2 py-1 text-[12px] text-red-400">
              {t(strings.messagesPane.work.loadFailed)}
              <button type="button" onClick={fetchRows} className="min-h-9 px-2 font-medium text-wc-text-muted hover:text-wc-text-primary">
                {t(strings.messagesPane.work.retry)}
              </button>
            </div>
          )}
          {load.state === "loaded" && load.rows.length === 0 && (
            <p className="py-1 text-[12px] text-wc-text-faint">{t(strings.messagesPane.work.empty)}</p>
          )}
          {load.state === "loaded" && load.rows.length > 0 && (
            <ol className="space-y-0.5">
              {load.rows.map((row) => (
                <WorkItem key={row.id} row={row} t={t} highlighted={highlightSequences?.has(row.sequence) ?? false} />
              ))}
            </ol>
          )}
        </div>
      </CollapsibleRegion>
    </div>
  );
}

function WorkItem({ row, t, highlighted }: { row: ConversationEvent; t: Translate; highlighted: boolean }) {
  const [open, setOpen] = useState(highlighted);
  const kindLabel = workKindLabel(row.kind);
  const line = workRowLine(row);
  const title = row.kind === "tool_call" ? line.title ?? (kindLabel ? t(kindLabel) : "") : kindLabel ? t(kindLabel) : "";
  return (
    <li data-testid={`work-item-${row.id}`} data-kind={row.kind} data-highlighted={highlighted || undefined}>
      <button
        type="button"
        aria-expanded={open}
        onClick={() => { setOpen((value) => !value); }}
        className={cn(
          "flex min-h-8 w-full min-w-0 items-baseline gap-2 rounded px-1 text-start text-[12px] text-wc-text-muted hover:bg-wc-surface-input/50",
          highlighted && "ring-1 ring-wc-accent/50",
        )}
      >
        <span className={cn("shrink-0 font-medium", row.kind === "tool_call" ? "font-mono text-wc-text-secondary" : "text-wc-text-faint")}>{title}</span>
        <span className="min-w-0 truncate font-mono text-[11px] text-wc-text-faint">{line.detail}</span>
      </button>
      {open && (
        <div className="mb-1 ms-1">
          <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-wc-surface-input/40 p-2 font-mono text-[11px] text-wc-text-secondary [overflow-wrap:anywhere]">{row.text}</pre>
          {row.textTruncated && <p className="mt-0.5 text-[10px] text-wc-text-faint">{t(strings.messagesPane.work.truncated)}</p>}
        </div>
      )}
    </li>
  );
}

export const WorkSpanRow = memo(WorkSpanRowImpl);
