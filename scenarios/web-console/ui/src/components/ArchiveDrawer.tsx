import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Archive, Download, RotateCcw, Search, Trash2 } from "lucide-react";
import { SwipeActions } from "@vrooli/react-component-library/SwipeActions";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useTranslation } from "react-i18next";

import {
  getConversationRange,
  type ArchivedConversationSearchMatch,
  type ConversationEvent,
} from "../api/conversation";
import { getContinuityReceipt, searchLocalContinuity, type ContinuityReceipt, type ContinuitySearchMatch } from "../api/continuity";
import {
  deleteSession,
  dismissRecoverableSession,
  listArchivedSessions,
  listRecoverableSessions,
  recoverSession,
  reopenSession,
  type AgentType,
  type ArchivedSession,
  type RecoverableSession,
  type RecoverResult,
} from "../api/sessions";
import { getContinuityIntegrity } from "../api/continuity";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { formatRelativeTime } from "./MessageJumpList.helpers";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import MessageExportDrawer from "./MessageExportDrawer";
import MessagesPane from "./MessagesPane";

interface ArchiveDrawerProps {
  open: boolean;
  initialSessionId?: string | null;
  onClose: () => void;
  activeSessionId: string | null;
  onSendToComposer: (text: string) => void;
  onReopened: (result: RecoverResult) => void;
  preferOrphans?: boolean;
}

const noop = () => undefined;
const noError = () => null;
const activeVersion = () => "active" as const;
const RESTORE_STATE_LABEL = {
  reopenable: strings.archiveDrawer.state_reopenable,
  read_only: strings.archiveDrawer.state_read_only,
  nothing_to_restore: strings.archiveDrawer.state_nothing_to_restore,
} as const;

type ArchiveMatch = ArchivedConversationSearchMatch | ContinuitySearchMatch;

function createdAfterFor(range: string): string | undefined {
  if (range === "any") return undefined;
  const days = range === "7d" ? 7 : range === "30d" ? 30 : 365;
  return new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString();
}

function highlightedExcerpt(text: string, query: string) {
  const terms = query.trim().split(/\s+/).filter(Boolean);
  if (terms.length === 0) return text;
  const escaped = terms.map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const matcher = new RegExp(`(${escaped.join("|")})`, "gi");
  return text.split(matcher).map((part, index) =>
    terms.some((term) => term.toLocaleLowerCase() === part.toLocaleLowerCase())
      ? <mark key={`${part}-${index}`} className="rounded bg-wc-accent/25 text-wc-text-primary">{part}</mark>
      : part,
  );
}

export default function ArchiveDrawer({ open, initialSessionId = null, onClose, activeSessionId, onSendToComposer, onReopened, preferOrphans = false }: ArchiveDrawerProps) {
  const { t } = useTranslation();
  const searchRef = useRef<HTMLInputElement>(null);
  const reopenKeys = useRef(new Map<string, string>());
  const recoveryKeys = useRef(new Map<string, string>());
  const orphanFilterInitialized = useRef(false);
  const isMobile = useMediaQuery("(max-width: 767px)");
  // Only one row rests open at a time; opening another closes the last.
  const [swipeOpenArchiveId, setSwipeOpenArchiveId] = useState<string | null>(null);
  const [archive, setArchive] = useState<ArchivedSession[]>([]);
  const [recoverable, setRecoverable] = useState<RecoverableSession[]>([]);
  const [orphansOnly, setOrphansOnly] = useState(false);
  const [query, setQuery] = useState("");
  const [agentType, setAgentType] = useState<"all" | AgentType>("all");
  const [timeRange, setTimeRange] = useState("any");
  const [myMessages, setMyMessages] = useState(false);
  const [allStates, setAllStates] = useState(false);
  const [matches, setMatches] = useState<ArchiveMatch[]>([]);
  const [totalMatches, setTotalMatches] = useState(0);
  const [distinctSessions, setDistinctSessions] = useState(0);
  const [selected, setSelected] = useState<ArchivedConversationSearchMatch | null>(null);
  const [selectedArchiveId, setSelectedArchiveId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [exportEvents, setExportEvents] = useState<ConversationEvent[]>([]);
  const [exportOpen, setExportOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ArchivedSession | null>(null);
  const [reopeningId, setReopeningId] = useState<string | null>(null);
  const [continuityWarning, setContinuityWarning] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<ContinuityReceipt | null>(null);
  const [receiptLoading, setReceiptLoading] = useState(false);

  const archiveById = useMemo(() => new Map(archive.map((row) => [row.id, row])), [archive]);
  const recoverableById = useMemo(() => new Map(recoverable.map((row) => [row.id, row])), [recoverable]);
  const orphanRows = useMemo(() => archive.filter((row) => row.awaiting_recovery), [archive]);
  const browseRows = useMemo(() => {
    const after = createdAfterFor(timeRange);
    const afterTimestamp = after ? Date.parse(after) : Number.NEGATIVE_INFINITY;
    return archive.filter((row) => !row.awaiting_recovery)
      .filter((row) => agentType === "all" || row.agent_type === agentType)
      .filter((row) => Date.parse(row.archived_at) >= afterTimestamp);
  }, [agentType, archive, timeRange]);
  const selectedSession = archiveById.get(selected?.sessionId ?? selectedArchiveId ?? "") ?? (selected ? {
    id: selected.sessionId,
    archived_at: selected.createdAt,
    created_at: selected.createdAt,
    agent_type: "none" as AgentType,
    pane_name: selected.sessionId.slice(0, 8),
    message_count: 1,
    restore_state: "read_only" as const,
    awaiting_recovery: false,
  } : undefined);
  const selectedIsArchived = selectedSession ? archiveById.has(selectedSession.id) : false;

  const refreshArchive = useCallback(async () => {
    const [result, recoverableRows] = await Promise.all([listArchivedSessions(), listRecoverableSessions()]);
    setArchive(result.sessions);
    setRecoverable(recoverableRows);
    const awaiting = result.sessions.filter((row) => row.awaiting_recovery);
    setSelectedArchiveId((current) => {
      const preferred = current ?? initialSessionId;
      if (preferred && result.sessions.some((row) => row.id === preferred)) return preferred;
      return awaiting[0]?.id ?? null;
    });
    if (awaiting.length === 0) setOrphansOnly(false);
    if (!orphanFilterInitialized.current) {
      const nextOrphansOnly = preferOrphans || (!initialSessionId && awaiting.length > 0);
      setOrphansOnly(nextOrphansOnly);
      if (nextOrphansOnly) setSelectedArchiveId(awaiting[0]?.id ?? null);
      orphanFilterInitialized.current = true;
    }
  }, [initialSessionId, preferOrphans]);

  useEffect(() => {
    if (!open) {
      orphanFilterInitialized.current = false;
      return;
    }
    setError(null);
    setContinuityWarning(null);
    setReceipt(null);
    void refreshArchive().catch((cause) => setError(cause instanceof Error ? cause.message : String(cause)));
    let cancelled = false;
    void getContinuityIntegrity().then((report) => {
      if (cancelled) return;
      // Catalog-only conversations are a distinct drift class. Include them
      // without double-counting rows already represented by the legacy orphan
      // conversation count.
      const orphanCount = Math.max(report.orphanConversations, report.uncatalogedConversations) + report.orphanCheckpoints + report.orphanWorkspacePanes;
      if (orphanCount > 0) setContinuityWarning(t(strings.archiveDrawer.continuityWarning, { count: orphanCount }));
    }).catch(() => undefined);
    const frame = requestAnimationFrame(() => searchRef.current?.focus());
    return () => { cancelled = true; cancelAnimationFrame(frame); };
  }, [open, refreshArchive, t]);

  useEffect(() => {
    if (!open) return;
    const trimmed = query.trim();
    if (!trimmed) {
      setMatches([]);
      setTotalMatches(0);
      setDistinctSessions(0);
      setSelected(null);
      setLoading(false);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      setLoading(true);
      setError(null);
      // Local continuity search is the archive authority too. The legacy
      // archive endpoint joins through the live session projection and can
      // therefore hide catalog-only orphan evidence. Keep the role filter in
      // the presentation layer until it is part of the shared continuity
      // contract.
      const search = searchLocalContinuity(trimmed, {
        agentType: agentType === "all" ? undefined : agentType,
        after: createdAfterFor(timeRange),
        state: allStates ? "any" : "archived",
      });
      void search.then((result) => {
        if (cancelled) return;
        const visible = result.matches.filter((match) => !myMessages || match.role === "user");
        setMatches(visible);
        setTotalMatches(result.totalMatches);
        setDistinctSessions(result.distinctSessions);
        setSelected((current) => visible.find((match) => match.eventId === current?.eventId) ?? visible[0] ?? null);
      }).catch((cause) => {
        if (!cancelled) {
          setMatches([]);
          setSelected(null);
          setError(cause instanceof Error ? cause.message : String(cause));
        }
      }).finally(() => {
        if (!cancelled) setLoading(false);
      });
    }, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [agentType, allStates, archiveById, myMessages, open, orphansOnly, query, timeRange]);

  const displayedMessageCount = query.trim()
    ? totalMatches
    : browseRows.reduce((total, row) => total + row.message_count, 0);
  const displayedSessionCount = query.trim() ? distinctSessions : browseRows.length;

  const openExport = useCallback(async () => {
    if (!selectedSession) return;
    setError(null);
    try {
      const response = await getConversationRange(selectedSession.id, 1, Math.max(1, selectedSession.message_count));
      setExportEvents(response.events);
      setExportOpen(true);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    }
  }, [selectedSession]);

  const confirmDelete = useCallback(async () => {
    if (!deleteTarget) return;
    const id = deleteTarget.id;
    setDeleteTarget(null);
    setError(null);
    try {
      await deleteSession(id);
      await refreshArchive();
      setMatches((current) => current.filter((match) => match.sessionId !== id));
      setSelected((current) => current?.sessionId === id ? null : current);
      setSelectedArchiveId((current) => current === id ? null : current);
      window.dispatchEvent(new CustomEvent("web-console:archive-changed"));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    }
  }, [deleteTarget, refreshArchive]);

  const reopenById = useCallback(async (id: string) => {
    const key = reopenKeys.current.get(id) ?? crypto.randomUUID();
    reopenKeys.current.set(id, key);
    setReopeningId(id);
    setError(null);
    try {
      const result = await reopenSession(id, key);
      onReopened(result);
      window.dispatchEvent(new CustomEvent("web-console:archive-changed"));
      onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setReopeningId(null);
    }
  }, [onClose, onReopened]);

  const reopenSelected = useCallback(async () => {
    if (!selectedSession || selectedSession.restore_state !== "reopenable") return;
    await reopenById(selectedSession.id);
  }, [reopenById, selectedSession]);

  const recoverOne = useCallback(async (id: string) => {
    const key = recoveryKeys.current.get(id) ?? crypto.randomUUID();
    recoveryKeys.current.set(id, key);
    setReopeningId(id);
    setError(null);
    try {
      const result = await recoverSession(id, key);
      onReopened(result);
      await refreshArchive();
      window.dispatchEvent(new CustomEvent("web-console:archive-changed"));
      window.dispatchEvent(new CustomEvent("web-console:recoverable-changed"));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setReopeningId(null);
    }
  }, [onReopened, refreshArchive]);

  const dismissOne = useCallback(async (id: string) => {
    setReopeningId(id);
    setError(null);
    try {
      await dismissRecoverableSession(id);
      await refreshArchive();
      setSelectedArchiveId((current) => current === id ? null : current);
      window.dispatchEvent(new CustomEvent("web-console:archive-changed"));
      window.dispatchEvent(new CustomEvent("web-console:recoverable-changed"));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setReopeningId(null);
    }
  }, [refreshArchive]);

  const recoverAll = useCallback(async () => {
    for (const row of recoverable) if (row.recoverable) await recoverOne(row.id);
  }, [recoverOne, recoverable]);

  const dismissAll = useCallback(async () => {
    for (const row of recoverable) await dismissOne(row.id);
  }, [dismissOne, recoverable]);

  const inspectReceipt = useCallback(async () => {
    if (!selectedSession) return;
    setReceiptLoading(true);
    setError(null);
    try {
      const command = selectedSession.awaiting_recovery ? "dismiss" : "archive";
      setReceipt(await getContinuityReceipt(`${command}:${selectedSession.id}`));
    } catch (cause) {
      setReceipt(null);
      setError(cause instanceof Error ? cause.message : t(strings.archiveDrawer.receiptNotFound));
    } finally {
      setReceiptLoading(false);
    }
  }, [selectedSession, t]);

  return (
    <>
      <FullPageDrawer
        avoidKeyboard
        open={open}
        onClose={onClose}
        title={t(strings.archiveDrawer.title)}
        closeLabel={t(strings.archiveDrawer.close)}
        testId="archive-drawer"
        headerExtra={
          <div className="mt-3 space-y-2">
            <div className="flex flex-col gap-2 md:flex-row md:items-center">
              <label className="flex min-w-0 flex-1 items-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2">
                <Search className="h-4 w-4 shrink-0 text-wc-text-muted" aria-hidden="true" />
                <input
                  ref={searchRef}
                  data-testid="archive-search-input"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  className="min-w-0 flex-1 bg-transparent text-sm text-wc-text-primary outline-none"
                  placeholder={t(strings.archiveDrawer.searchPlaceholder)}
                />
              </label>
              <div data-testid="archive-search-counts" className="shrink-0 text-xs text-wc-text-muted">
                {t(strings.archiveDrawer.resultCounts, { messages: displayedMessageCount, sessions: displayedSessionCount })}
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2 text-xs">
              <span className="rounded-full bg-wc-accent/20 px-3 py-1 font-medium text-wc-text-primary">{t(strings.archiveDrawer.textSearch)}</span>
              <button type="button" disabled title={t(strings.archiveDrawer.semanticUnavailable)} className="rounded-full border border-wc-default px-3 py-1 text-wc-text-faint opacity-60">{t(strings.archiveDrawer.semanticSearch)}</button>
              <select data-testid="archive-agent-filter" value={agentType} onChange={(event) => setAgentType(event.target.value as "all" | AgentType)} className="rounded-full border border-wc-default bg-wc-surface-input px-3 py-1 text-wc-text-secondary">
                <option value="all">{t(strings.archiveDrawer.allAgents)}</option>
                <option value="claude">Claude</option><option value="codex">Codex</option><option value="opencode">OpenCode</option><option value="grok">Grok</option><option value="none">{t(strings.archiveDrawer.shell)}</option>
              </select>
              <select data-testid="archive-time-filter" value={timeRange} onChange={(event) => setTimeRange(event.target.value)} className="rounded-full border border-wc-default bg-wc-surface-input px-3 py-1 text-wc-text-secondary">
                <option value="any">{t(strings.archiveDrawer.anyTime)}</option><option value="7d">{t(strings.archiveDrawer.last7Days)}</option><option value="30d">{t(strings.archiveDrawer.last30Days)}</option><option value="365d">{t(strings.archiveDrawer.lastYear)}</option>
              </select>
              <label className="inline-flex items-center gap-1.5 rounded-full border border-wc-default px-3 py-1 text-wc-text-secondary"><input data-testid="archive-my-messages" type="checkbox" checked={myMessages} disabled={allStates} onChange={(event) => setMyMessages(event.target.checked)} />{t(strings.archiveDrawer.myMessages)}</label>
              <label className="inline-flex items-center gap-1.5 rounded-full border border-wc-default px-3 py-1 text-wc-text-secondary"><input data-testid="archive-all-states" type="checkbox" checked={allStates} onChange={(event) => { setAllStates(event.target.checked); if (event.target.checked) setMyMessages(false); }} />{t(strings.archiveDrawer.allStates)}</label>
              {orphanRows.length > 0 && <button type="button" data-testid="archive-orphans-filter" aria-pressed={orphansOnly} onClick={() => { setOrphansOnly((value) => !value); setSelected(null); setSelectedArchiveId(orphanRows[0]?.id ?? null); }} className={cn("rounded-full border px-3 py-1", orphansOnly ? "border-amber-400/60 bg-amber-500/15 text-amber-100" : "border-wc-default text-wc-text-secondary")}>{t(strings.archiveDrawer.crashOrphans, { count: orphanRows.length })}</button>}
            </div>
            {error && <div role="alert" className="rounded bg-red-500/10 px-3 py-1.5 text-xs text-red-300">{error}</div>}
            {continuityWarning && <div role="status" data-testid="archive-continuity-warning" className="rounded bg-amber-500/10 px-3 py-1.5 text-xs text-amber-200">{continuityWarning}</div>}
          </div>
        }
      >
        <div className="flex h-full min-h-0 flex-col md:flex-row">
          <section aria-label={t(strings.archiveDrawer.resultsHeading)} className="max-h-[42%] shrink-0 overflow-y-auto border-b border-wc-default md:max-h-none md:w-[22rem] md:border-b-0 md:border-e">
            {orphansOnly ? (
              <div data-testid="archive-orphan-results">
                <div className="flex flex-wrap gap-2 border-b border-wc-default p-3">
                  <button type="button" onClick={() => void recoverAll()} disabled={reopeningId !== null || !recoverable.some((row) => row.recoverable)} className="rounded border border-amber-400/50 px-2 py-1 text-xs text-amber-100 disabled:opacity-50">{t(strings.recoverableSessions.reattachAll)}</button>
                  <button type="button" onClick={() => void dismissAll()} disabled={reopeningId !== null} className="rounded border border-wc-default px-2 py-1 text-xs text-wc-text-secondary disabled:opacity-50">{t(strings.recoverableSessions.dismissAll)}</button>
                </div>
                {orphanRows.map((row) => {
                  const recovery = recoverableById.get(row.id);
                  return <div key={row.id} data-testid={`recoverable-row-${row.id}`} className={cn("border-b border-wc-default/60 p-3", selectedArchiveId === row.id && "bg-wc-accent/10")}>
                    <button type="button" onClick={() => { setSelected(null); setSelectedArchiveId(row.id); }} className="block w-full text-start">
                      <div className="truncate text-xs font-semibold text-wc-text-primary">{row.agent_type} · {row.pane_name}</div>
                      <div className="mt-1 truncate text-xs text-wc-text-muted">{row.cwd}</div>
                    </button>
                    <div className="mt-2 flex gap-2">
                      <button type="button" data-testid={`recoverable-row-${row.id}-recover`} disabled={!recovery?.recoverable || reopeningId !== null} title={recovery?.recoverable ? t(strings.recoverableSessions.reattachTitle) : recovery?.not_recoverable_reason} onClick={() => void recoverOne(row.id)} className="rounded border px-2 py-1 text-xs disabled:opacity-50">{t(strings.recoverableSessions.reattach)}</button>
                      <button type="button" data-testid={`recoverable-row-${row.id}-dismiss`} disabled={reopeningId !== null} onClick={() => void dismissOne(row.id)} className="rounded border px-2 py-1 text-xs disabled:opacity-50">{t(strings.recoverableSessions.dismiss)}</button>
                    </div>
                  </div>;
                })}
              </div>
            ) : !query.trim() ? (
              browseRows.length === 0 ? (
                <div className="p-5 text-sm text-wc-text-muted"><Archive className="mb-3 h-7 w-7" />{t(strings.archiveDrawer.noResults)}</div>
              ) : browseRows.map((row) => {
                const rowButton = (
                <button
                  type="button"
                  data-testid={`archive-session-${row.id}`}
                  onClick={() => { setSelected(null); setSelectedArchiveId(row.id); }}
                  className={cn("group block w-full border-b border-wc-default/60 p-3 text-start transition", selectedArchiveId === row.id ? "bg-wc-accent/10" : "hover:bg-wc-surface-raised")}
                >
                  <div className="flex items-start gap-2.5">
                    <span className="mt-0.5 h-3 w-3 shrink-0 rounded-full border border-white/20" style={{ backgroundColor: row.header_color || "transparent" }} aria-hidden="true" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-xs font-semibold text-wc-text-primary">{row.agent_type} · {row.pane_name}</span>
                      {row.cwd && <span className="mt-1 block truncate text-[11px] text-wc-text-muted">{row.cwd}</span>}
                      <span className="mt-1.5 flex flex-wrap items-center gap-x-1.5 text-[10px] text-wc-text-faint">
                        <span>{formatRelativeTime(row.archived_at)}</span><span aria-hidden="true">·</span>
                        <span>{t(strings.archiveDrawer.messageCount, { count: row.message_count })}</span><span aria-hidden="true">·</span>
                        <span>{t(RESTORE_STATE_LABEL[row.restore_state])}</span>
                      </span>
                    </span>
                  </div>
                </button>
                );

                // Reopen is the only action worth accelerating here: it is the
                // reason the drawer is open, and the detail pane already offers
                // export and permanent deletion behind their own confirmations.
                // A row that cannot be reopened gets no gesture rather than a
                // disabled one, so the affordance never lies.
                if (!isMobile || row.restore_state !== "reopenable") {
                  return <div key={row.id}>{rowButton}</div>;
                }
                return (
                  <SwipeActions
                    key={row.id}
                    testId={`archive-swipe-${row.id}`}
                    label={t(strings.archiveDrawer.swipeActionsAria, { name: row.pane_name })}
                    open={swipeOpenArchiveId === row.id}
                    onOpenChange={(next) => { setSwipeOpenArchiveId(next ? row.id : null); }}
                    actions={[
                      {
                        id: "reopen",
                        label: t(strings.archiveDrawer.reopen),
                        icon: <RotateCcw className="h-4 w-4" aria-hidden />,
                        tone: "primary",
                        onSelect: () => { void reopenById(row.id); },
                      },
                    ]}
                  >
                    {rowButton}
                  </SwipeActions>
                );
              })
            ) : loading ? (
              <div className="p-5 text-sm text-wc-text-muted">{t(strings.archiveDrawer.searching)}</div>
            ) : matches.length === 0 ? (
              <div className="p-5 text-sm text-wc-text-muted">{t(strings.archiveDrawer.noResults)}</div>
            ) : matches.map((match) => {
              const row = archiveById.get(match.sessionId);
              return (
                <button
                  key={match.eventId}
                  type="button"
                  data-testid={`archive-hit-${match.eventId}`}
                  onClick={() => { setSelected(match); setSelectedArchiveId(match.sessionId); }}
                  className={cn("block w-full border-b border-wc-default/60 p-3 text-start transition", selected?.eventId === match.eventId ? "bg-wc-accent/10" : "hover:bg-wc-surface-raised")}
                >
                  <div className="truncate text-xs font-semibold text-wc-text-primary">{row?.agent_type ?? "agent"} · {row?.pane_name ?? match.sessionId.slice(0, 8)}</div>
                  <div className="mt-1 line-clamp-3 text-xs leading-relaxed text-wc-text-secondary">{highlightedExcerpt(match.excerpt, query)}</div>
                  <div className="mt-1 text-[10px] text-wc-text-faint">{formatRelativeTime(match.createdAt)} · {match.role === "user" ? t(strings.archiveDrawer.you) : row?.agent_type ?? match.role}</div>
                </button>
              );
            })}
          </section>

          <section aria-label={t(strings.archiveDrawer.readerHeading)} className="flex min-h-0 min-w-0 flex-1 flex-col">
            {selectedSession ? (
              <>
                <header className="shrink-0 border-b border-wc-default px-3 py-2">
                  <div className="flex flex-wrap items-start gap-2">
                    <span className="mt-1 h-3 w-3 rounded-full border border-white/20" style={{ backgroundColor: selectedSession.header_color || "transparent" }} aria-hidden="true" />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-semibold text-wc-text-primary">{selectedSession.agent_type} · {selectedSession.pane_name}</div>
                      <div className="truncate text-xs text-wc-text-muted">{t(strings.archiveDrawer.archivedAt, { time: formatRelativeTime(selectedSession.archived_at) })} · {t(strings.archiveDrawer.messageCount, { count: selectedSession.message_count })}{selectedSession.cwd ? ` · ${selectedSession.cwd}` : ""}</div>
                      <div className="mt-1 text-xs text-wc-text-secondary">{t(RESTORE_STATE_LABEL[selectedSession.restore_state])}{selectedSession.restore_state_reason ? ` — ${selectedSession.restore_state_reason}` : ""}</div>
                    </div>
                    {selectedSession.awaiting_recovery ? <button
                      type="button"
                      data-testid="archive-reattach"
                      disabled={!recoverableById.get(selectedSession.id)?.recoverable || reopeningId !== null}
                      title={recoverableById.get(selectedSession.id)?.recoverable ? t(strings.recoverableSessions.reattachTitle) : recoverableById.get(selectedSession.id)?.not_recoverable_reason}
                      onClick={() => void recoverOne(selectedSession.id)}
                      className="rounded border border-amber-400/50 px-2 py-1 text-xs text-amber-100 disabled:opacity-50"
                    >{reopeningId === selectedSession.id ? t(strings.archiveDrawer.reopening) : t(strings.recoverableSessions.reattach)}</button> : <button
                      type="button"
                      data-testid="archive-reopen"
                      disabled={selectedSession.restore_state !== "reopenable" || reopeningId !== null}
                      title={selectedSession.restore_state === "reopenable" ? t(strings.archiveDrawer.reopen) : selectedSession.restore_state_reason}
                      onClick={() => void reopenSelected()}
                      className="rounded border border-wc-default px-2 py-1 text-xs text-wc-text-secondary disabled:text-wc-text-faint disabled:opacity-60"
                    >
                      {reopeningId === selectedSession.id ? t(strings.archiveDrawer.reopening) : t(strings.archiveDrawer.reopen)}
                    </button>}
                    {selectedIsArchived && <button type="button" onClick={() => void openExport()} className="inline-flex items-center gap-1 rounded border border-wc-default px-2 py-1 text-xs text-wc-text-secondary"><Download className="h-3 w-3" />{t(strings.archiveDrawer.export)}</button>}
                    {selectedIsArchived && <button type="button" onClick={() => void inspectReceipt()} disabled={receiptLoading} className="rounded border border-wc-default px-2 py-1 text-xs text-wc-text-secondary disabled:opacity-60">{receiptLoading ? t(strings.archiveDrawer.receiptLoading) : t(strings.archiveDrawer.receipt)}</button>}
                    {selectedIsArchived && <button type="button" onClick={() => setDeleteTarget(selectedSession)} className="inline-flex items-center gap-1 rounded border border-red-500/40 px-2 py-1 text-xs text-red-300"><Trash2 className="h-3 w-3" />{t(strings.archiveDrawer.delete)}</button>}
                  </div>
                  {receipt && <div data-testid="archive-receipt" className="mt-2 rounded border border-wc-default/70 bg-wc-surface-input px-2 py-1.5 text-[11px] text-wc-text-muted">
                    <span className="font-medium text-wc-text-secondary">{receipt.status}</span> · {receipt.command} · {receipt.fromState} → {receipt.toState}
                    {receipt.errorCode && ` · ${receipt.errorCode}`}
                  </div>}
                </header>
                <div className="min-h-0 flex-1">
                  <MessagesPane
                    key={`${selectedSession.id}:${selected?.eventId ?? "archive"}`}
                    sessionId={selectedSession.id}
                    readOnly
                    focusEventId={selected?.eventId}
                    focusSequence={selected?.sequence}
                    onSendToComposer={activeSessionId ? onSendToComposer : undefined}
                    onPlayFromHere={noop}
                    onPlayEvent={noop}
                    activeSpeakingEventId={null}
                    isTtsSpeaking={false}
                    summarizeLevel="moderate"
                    selectedVersionForEvent={activeVersion}
                    summarizingEventId={null}
                    getSummarizeError={noError}
                    onClearSummarizeError={noop}
                    onToggleSummarized={noop}
                    onChangeLevel={noop}
                    playbackFocusRequest={null}
                  />
                </div>
              </>
            ) : (
              <div className="flex h-full items-center justify-center p-6 text-sm text-wc-text-muted">{t(strings.archiveDrawer.selectResult)}</div>
            )}
          </section>
        </div>
      </FullPageDrawer>

      <MessageExportDrawer open={exportOpen} events={exportEvents} onClose={() => setExportOpen(false)} />
      <AlertDialog
        open={deleteTarget !== null}
        title={t(strings.archiveDrawer.deleteTitle)}
        description={t(strings.archiveDrawer.deleteBody, { name: deleteTarget?.pane_name ?? "" })}
        cancelLabel={t(strings.confirmDelete.cancel)}
        confirmLabel={t(strings.archiveDrawer.deleteConfirm)}
        destructive
        onCancel={() => setDeleteTarget(null)}
        onConfirm={() => void confirmDelete()}
        testIdPrefix="archive-delete"
      />
    </>
  );
}
