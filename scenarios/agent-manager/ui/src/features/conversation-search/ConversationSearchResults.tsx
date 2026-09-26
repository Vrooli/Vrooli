import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, ChevronDown, ChevronRight, Loader2, RotateCw } from "lucide-react";
import type {
  ConversationContextEvent,
  ConversationHighlight,
  ConversationSearchHit,
  GetConversationContextResponse,
} from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import {
  ConversationContentClass,
  ConversationIndexState,
  ConversationSearchDegradationReason,
  ConversationSearchMode,
  ConversationSearchSort,
} from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import { Button } from "../../components/ui/button";
import { Checkbox } from "../../components/ui/checkbox";
import { cn } from "../../lib/utils";
import {
  DEFAULT_CONVERSATION_FILTERS,
  getConversationContext,
  type ConversationSearchFiltersState,
  useConversationSearch,
} from "./useConversationSearch";

interface ConversationSearchResultsProps {
  query: string;
  filters: ConversationSearchFiltersState;
  onFiltersChange: (filters: ConversationSearchFiltersState) => void;
  onOpenHit: (hit: ConversationSearchHit) => void;
  onResultCount?: (count: number) => void;
}

function enumLabel(value: string): string {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (character) => character.toUpperCase());
}

function degradationSummary(reason: ConversationSearchDegradationReason): string {
  switch (reason) {
    case ConversationSearchDegradationReason.SEMANTIC_UNAVAILABLE:
    case ConversationSearchDegradationReason.EMBEDDING_UNAVAILABLE:
    case ConversationSearchDegradationReason.VECTOR_STORE_UNAVAILABLE:
      return "Semantic ranking is temporarily unavailable.";
    case ConversationSearchDegradationReason.INDEX_STALE:
      return "The newest retained messages may still be indexing.";
    case ConversationSearchDegradationReason.INDEX_LAYOUT_MISMATCH:
      return "Semantic index maintenance is required.";
    case ConversationSearchDegradationReason.CANDIDATE_LIMIT:
      return "The search candidate limit was reached; refine the query for more precise results.";
    case ConversationSearchDegradationReason.DEADLINE:
      return "One search path exceeded its time budget.";
    case ConversationSearchDegradationReason.AUTHORIZATION_FILTERED:
      return "Some results were excluded by access policy.";
    case ConversationSearchDegradationReason.RERANK_UNAVAILABLE:
      return "Advanced result ordering is temporarily unavailable.";
    default:
      return "One search path is temporarily unavailable.";
  }
}

export function safeHighlightParts(text: string, highlights: ConversationHighlight[]): Array<{ text: string; highlighted: boolean }> {
  const graphemes = typeof Intl.Segmenter === "function"
    ? [...new Intl.Segmenter(undefined, { granularity: "grapheme" }).segment(text)].map((part) => part.segment)
    : Array.from(text);
  const ranges = highlights
    .filter((range) => !range.field || range.field === "snippet")
    .map((range) => ({ start: Math.min(range.startGrapheme, graphemes.length), end: Math.min(range.endGrapheme, graphemes.length) }))
    .filter((range) => range.end > range.start)
    .sort((left, right) => left.start - right.start || left.end - right.end);
  const merged: Array<{ start: number; end: number }> = [];
  for (const range of ranges) {
    const previous = merged.at(-1);
    if (previous && range.start <= previous.end) previous.end = Math.max(previous.end, range.end);
    else merged.push({ ...range });
  }
  const parts: Array<{ text: string; highlighted: boolean }> = [];
  let cursor = 0;
  for (const range of merged) {
    if (range.start > cursor) parts.push({ text: graphemes.slice(cursor, range.start).join(""), highlighted: false });
    parts.push({ text: graphemes.slice(range.start, range.end).join(""), highlighted: true });
    cursor = range.end;
  }
  if (cursor < graphemes.length || parts.length === 0) parts.push({ text: graphemes.slice(cursor).join(""), highlighted: false });
  return parts;
}

export function SafeHighlight({ hit }: { hit: ConversationSearchHit }) {
  return (
    <span className="whitespace-pre-wrap break-words">
      {safeHighlightParts(hit.snippet, hit.highlights).map((part, index) =>
        part.highlighted
          ? <mark key={index} className="rounded bg-amber-300/40 px-0.5 text-inherit">{part.text}</mark>
          : <span key={index}>{part.text}</span>
      )}
    </span>
  );
}

function ContextPanel({ hit }: { hit: ConversationSearchHit }) {
  const [context, setContext] = useState<GetConversationContextResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const load = async () => {
    if (context) {
      setContext(null);
      return;
    }
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    setLoading(true);
    setError(null);
    try {
      setContext(await getConversationContext(hit, controller.signal));
    } catch (caught) {
      if (!controller.signal.aborted) setError(caught instanceof Error ? caught.message : "Context is unavailable.");
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  };

  return (
    <div>
      <Button type="button" variant="ghost" size="sm" className="h-7 gap-1 px-2 text-xs" onClick={() => void load()} aria-expanded={Boolean(context)}>
        {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : context ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        {context ? "Hide context" : "Show nearby context"}
      </Button>
      {error ? <p role="alert" className="mt-1 text-xs text-destructive">{error}</p> : null}
      {context ? (
        <div className="mt-2 space-y-2 rounded-md border border-border/70 bg-background/60 p-2" aria-label="Nearby conversation context">
          {context.events.map((event) => <ContextEvent key={event.eventId} event={event} />)}
          {context.truncated ? <p className="text-xs text-muted-foreground">Context is bounded; open the run for the full transcript.</p> : null}
        </div>
      ) : null}
    </div>
  );
}

function ContextEvent({ event }: { event: ConversationContextEvent }) {
  return (
    <div className={cn("border-l-2 pl-2 text-xs", event.matched ? "border-primary" : "border-border")}>
      <div className="font-medium">{event.role || "event"} · sequence {event.eventSequence.toString()}</div>
      <div className="mt-0.5 whitespace-pre-wrap break-words text-muted-foreground">{event.boundedContent}</div>
    </div>
  );
}

function AdvancedFilters({ filters, onChange }: { filters: ConversationSearchFiltersState; onChange: (value: ConversationSearchFiltersState) => void }) {
  const [open, setOpen] = useState(false);
  const update = <K extends keyof ConversationSearchFiltersState>(key: K, value: ConversationSearchFiltersState[K]) => onChange({ ...filters, [key]: value });
  const inputClass = "min-w-0 rounded-md border border-input bg-background px-2 py-1.5 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";
  return (
    <div className="rounded-md border border-border/70 bg-muted/20 p-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <button type="button" className="flex items-center gap-1 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={() => setOpen((value) => !value)} aria-expanded={open}>
          {open ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />} Advanced conversation filters
        </button>
        <Button type="button" variant="ghost" size="sm" onClick={() => onChange(DEFAULT_CONVERSATION_FILTERS)}>Reset</Button>
      </div>
      {open ? (
        <fieldset className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2" aria-label="Advanced conversation filters">
          <label className="grid gap-1 text-xs">Mode<select className={inputClass} value={filters.mode} onChange={(event) => update("mode", Number(event.target.value) as ConversationSearchMode)}>{[ConversationSearchMode.HYBRID, ConversationSearchMode.TEXT, ConversationSearchMode.REGEX, ConversationSearchMode.SEMANTIC].map((value) => <option key={value} value={value}>{enumLabel(ConversationSearchMode[value])}</option>)}</select></label>
          <label className="grid gap-1 text-xs">Sort<select className={inputClass} value={filters.sort} onChange={(event) => update("sort", Number(event.target.value) as ConversationSearchSort)}>{[ConversationSearchSort.RELEVANCE, ConversationSearchSort.NEWEST, ConversationSearchSort.OLDEST].map((value) => <option key={value} value={value}>{enumLabel(ConversationSearchSort[value])}</option>)}</select></label>
          {(["role", "harness", "project", "model", "profile", "runStatus"] as const).map((key) => <label key={key} className="grid gap-1 text-xs">{enumLabel(key)}<input className={inputClass} value={filters[key]} onChange={(event) => update(key, event.target.value)} /></label>)}
          <label className="grid gap-1 text-xs">After<input className={inputClass} type="date" value={filters.after} onChange={(event) => update("after", event.target.value)} /></label>
          <label className="grid gap-1 text-xs">Before<input className={inputClass} type="date" value={filters.before} onChange={(event) => update("before", event.target.value)} /></label>
          <label className="grid gap-1 text-xs">Content class<select className={inputClass} value={filters.contentClass} onChange={(event) => update("contentClass", Number(event.target.value) as ConversationContentClass)}>{[ConversationContentClass.UNSPECIFIED, ConversationContentClass.PROSE, ConversationContentClass.QUOTED_PROSE, ConversationContentClass.INJECTED_CONTEXT, ConversationContentClass.TOOL_CALL, ConversationContentClass.TOOL_RESULT].map((value) => <option key={value} value={value}>{value === ConversationContentClass.UNSPECIFIED ? "Any" : enumLabel(ConversationContentClass[value])}</option>)}</select></label>
          <label className="flex items-center gap-2 self-end py-1.5 text-sm"><Checkbox checked={filters.includeToolEvents} onCheckedChange={(value) => update("includeToolEvents", value === true)} />Include tool events</label>
        </fieldset>
      ) : null}
    </div>
  );
}

export function ConversationSearchResults({ query, filters, onFiltersChange, onOpenHit, onResultCount }: ConversationSearchResultsProps) {
  const search = useConversationSearch(query, filters);
  const groups = useMemo(() => {
    const grouped = new Map<string, ConversationSearchHit[]>();
    for (const hit of search.hits) grouped.set(hit.runId, [...(grouped.get(hit.runId) ?? []), hit]);
    return [...grouped.entries()];
  }, [search.hits]);
  const allWeak = search.hits.length > 0 && search.hits.every((hit) => hit.weak);
  const coverage = search.response?.coverage ?? search.status?.coverage;
  const degradations = search.response?.degradations ?? search.status?.degradations ?? [];
  const stale = search.status?.state === ConversationIndexState.STALE || Number(coverage?.freshnessLagMs ?? 0n) > 60_000;

  useEffect(() => onResultCount?.(search.hits.length), [onResultCount, search.hits.length]);

  return (
    <div className="min-w-0 space-y-3" data-testid="conversation-search-results">
      <AdvancedFilters filters={filters} onChange={onFiltersChange} />
      <div className="sr-only" role="status" aria-live="polite">{search.loading ? "Searching conversation history" : `${search.hits.length} conversation matches loaded`}</div>
      {degradations.length > 0 || stale ? (
        <div role="status" className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-xs">
          <div className="flex items-center gap-2 font-medium"><AlertTriangle className="h-4 w-4" />Partial search coverage</div>
          <p className="mt-1">Lexical matches remain available. {stale ? "The index may not include the newest messages. " : ""}{[...new Set(degradations.map((item) => degradationSummary(item.reason)))].join(" ")}</p>
          {degradations.some((item) => item.retryable) ? <Button type="button" variant="link" size="sm" className="mt-1 h-auto p-0 text-xs" onClick={search.retry}><RotateCw className="mr-1 h-3 w-3" />Retry search</Button> : null}
        </div>
      ) : null}
      {coverage ? <p className="text-xs text-muted-foreground">Coverage: {Math.round(coverage.lexicalRatio * 100)}% lexical · {Math.round(coverage.semanticRatio * 100)}% semantic · {coverage.pendingDocuments.toString()} pending</p> : null}
      {search.loading ? <div className="flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground"><Loader2 className="h-4 w-4 animate-spin" />Searching all retained conversations…</div> : null}
      {search.error ? <div role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm"><p className="font-medium">{search.error.kind === "invalid" ? "Search query is invalid" : search.error.kind === "permission" ? "Conversation search is restricted" : search.error.kind === "admission" ? "Search is temporarily busy" : "Conversation search failed"}</p><p className="mt-1 text-muted-foreground">{search.error.message}</p><Button type="button" variant="outline" size="sm" className="mt-2" onClick={search.retry}>Try again</Button></div> : null}
      {!search.loading && !search.error && search.hits.length === 0 ? <div className="py-10 text-center"><p className="font-medium">No conversation matches</p><p className="mt-1 text-sm text-muted-foreground">Try broader wording, text mode, or fewer filters.</p></div> : null}
      {allWeak ? <div role="status" className="rounded-md border border-border bg-muted/40 p-2 text-xs">Only weak matches were found. Review the match explanation or refine the query.</div> : null}
      {groups.map(([runId, hits]) => {
        const run = hits[0]?.run;
        return <section key={runId} aria-labelledby={`conversation-run-${runId}`} className="overflow-hidden rounded-lg border border-border bg-card"><header className="border-b border-border bg-muted/30 px-3 py-2"><h3 id={`conversation-run-${runId}`} className="truncate text-sm font-semibold">{run?.label || `Run ${runId.slice(0, 8)}`}</h3><p className="truncate text-xs text-muted-foreground">{run?.status || "unknown status"} · {run?.runner || "unknown runner"} · {run?.model || "unknown model"}</p></header><div className="divide-y divide-border/70">{hits.map((hit) => <article key={hit.stableHitId} className="p-3"><button type="button" className="block w-full rounded text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" onClick={() => { search.recordSelection(hit, search.hits.findIndex((candidate) => candidate.stableHitId === hit.stableHitId) + 1); onOpenHit(hit); }}><div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs"><span className="font-semibold">{hit.role || "event"}</span><span className="text-muted-foreground">{hit.occurredAt ? new Date(timestampMs(hit.occurredAt)).toLocaleString() : "time unavailable"}</span>{hit.weak ? <span className="rounded border border-border px-1.5">Weak match</span> : null}</div><p className="mt-2 text-sm leading-relaxed"><SafeHighlight hit={hit} /></p><p className="mt-2 text-xs text-muted-foreground">{hit.rankEvidence.map((evidence) => evidence.explanation).filter(Boolean).join(" · ") || "Matched indexed conversation content"}</p><p className="mt-1 text-xs text-muted-foreground">Source: {hit.provenance?.harness || "unknown harness"}{hit.provenance?.sourceSessionId ? ` · session ${hit.provenance.sourceSessionId}` : ""}</p><span className="mt-2 inline-block text-xs font-medium text-primary">Open matched event →</span></button><ContextPanel hit={hit} /></article>)}</div></section>;
      })}
      {search.hasMore ? <Button type="button" variant="outline" className="w-full" disabled={search.loadingMore} onClick={search.loadMore}>{search.loadingMore ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Loading more</> : "Load more results"}</Button> : null}
    </div>
  );
}
