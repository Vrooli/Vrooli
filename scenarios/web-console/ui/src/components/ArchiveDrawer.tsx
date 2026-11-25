import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Archive, RotateCcw, Search } from "lucide-react";
import { SwipeActions } from "@vrooli/react-component-library/SwipeActions";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useTranslation } from "react-i18next";

import {
  getConversationRange,
  searchNodeArchive,
  searchSemanticConversations,
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
import type { OverlayActionList, OverlayPrimaryAction, OverlaySecondaryAction } from "@vrooli/react-component-library/OverlayActionBar/1";
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
const NATIVE_ARCHIVE_HARNESSES: Record<string, string> = {
  claude: "Claude",
  codex: "Codex",
  grok: "Grok",
  opencode: "OpenCode",
  "gemini-cli": "Gemini CLI",
  "qwen-code": "Qwen Code",
  cursor: "Cursor",
};

function nativeArchiveHarness(sessionId: string): string | undefined {
  const match = /^wc-archive-([a-z0-9-]+)-/.exec(sessionId);
  if (!match) return undefined;
  // Match the longest known slug because Gemini CLI contains a hyphen.
  return Object.entries(NATIVE_ARCHIVE_HARNESSES)
    .sort(([left], [right]) => right.length - left.length)
    .find(([slug]) => sessionId.startsWith(`wc-archive-${slug}-`))?.[1];
}

type RemoteArchiveMatch = {
  eventId: string;
  sessionId: string;
  role: string;
  createdAt: string;
  occurredAt?: string;
  excerpt: string;
  source: "semantic" | "node";
  harness?: string;
  nodeId?: string;
  weak?: boolean;
};

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
  const [searchMode, setSearchMode] = useState<"text" | "semantic" | "node">("text");
  const [nodeId, setNodeId] = useState("");
  const [remoteMatches, setRemoteMatches] = useState<RemoteArchiveMatch[]>([]);
  const [selectedRemote, setSelectedRemote] = useState<RemoteArchiveMatch | null>(null);
  const [searchNotice, setSearchNotice] = useState<string | null>(null);
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
  const selectedNativeHarness = selectedSession && !selectedIsArchived ? nativeArchiveHarness(selectedSession.id) : undefined;

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
    if (searchMode !== "text") return;
    if (!trimmed) {
      setMatches([]);
      setRemoteMatches([]);
      setSelectedRemote(null);
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
  }, [agentType, allStates, archiveById, myMessages, open, orphansOnly, query, searchMode, timeRange]);

  useEffect(() => {
    const trimmed = query.trim();
    if (!open || searchMode === "text" || !trimmed) {
      setRemoteMatches([]);
      setSelectedRemote(null);
      setSearchNotice(null);
      if (searchMode !== "text" && !trimmed) {
        setMatches([]);
        setSelected(null);
        setTotalMatches(0);
        setDistinctSessions(0);
      }
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      setLoading(true);
      setError(null);
      setSearchNotice(null);
      const role = myMessages ? "user" : undefined;
      const filters = { agentType: agentType === "all" ? undefined : agentType, role };
      const search = searchMode === "semantic"
        ? searchSemanticConversations(trimmed, filters, 100).then((result) => {
          if (cancelled) return;
          setMatches(result.local.matches.filter((match) => !myMessages || match.role === "user"));
          setTotalMatches(result.local.totalMatches);
          setDistinctSessions(result.local.distinctSessions);
          const semantic = result.semanticHits.filter((hit) => !myMessages || hit.role === "user").map((hit) => ({
            eventId: hit.stableHitId || hit.eventId,
            sessionId: hit.sourceSessionId,
            role: hit.role,
            createdAt: hit.occurredAt,
            excerpt: hit.snippet,
            source: "semantic" as const,
            harness: hit.harness,
            weak: hit.weak,
          }));
          setRemoteMatches(semantic);
          setSelected((current) => result.local.matches.find((match) => match.eventId === current?.eventId) ?? result.local.matches[0] ?? null);
          setSearchNotice(result.semanticAvailable
            ? t(strings.archiveDrawer.semanticCoverage, { ratio: result.coverage?.semanticRatio ?? 0, count: semantic.length, took: result.semanticTookMs })
            : result.semanticError || t(strings.archiveDrawer.semanticUnavailable));
        })
        : (!nodeId.trim()
          ? Promise.resolve().then(() => {
            if (!cancelled) {
              setMatches([]);
              setSelected(null);
              setSelectedArchiveId(null);
              setTotalMatches(0);
              setDistinctSessions(0);
              setSearchNotice(t(strings.archiveDrawer.nodeIdRequired));
            }
          })
          : searchNodeArchive(nodeId.trim(), trimmed, filters, 100).then((result) => {
            if (cancelled) return;
            setMatches([]);
            setSelected(null);
            setTotalMatches(result.search.totalMatches);
            setDistinctSessions(result.search.distinctSessions);
            setRemoteMatches(result.search.matches.map((match) => ({ ...match, source: "node" as const, nodeId: result.nodeId })));
            setSearchNotice(result.available ? t(strings.archiveDrawer.nodeResults, { node: result.nodeId, count: result.search.totalMatches }) : result.degradation || t(strings.archiveDrawer.nodeUnavailable));
          }));
      void search.catch((cause) => {
        if (!cancelled) {
          setRemoteMatches([]);
          setSearchNotice(cause instanceof Error ? cause.message : String(cause));
        }
      }).finally(() => { if (!cancelled) setLoading(false); });
    }, 250);
    return () => { cancelled = true; window.clearTimeout(timer); };
  }, [agentType, myMessages, nodeId, open, query, searchMode, t]);

  const displayedMessageCount = query.trim()
    ? totalMatches + (searchMode === "semantic" ? remoteMatches.length : 0)
    : browseRows.reduce((total, row) => total + row.message_count, 0);
  const displayedSessionCount = query.trim() ? distinctSessions + (searchMode === "node" ? remoteMatches.length : 0) : browseRows.length;

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

  const GermanContractShowcase = import.meta.env.DEV && new URLSearchParams(window.location.search).get("rcl-contract-showcase") === "1";
  const archiveActions: OverlayActionList | undefined = selectedSession ? (() : OverlayActionList => {
    const reopenAction: OverlayPrimaryAction = {
      id: "reopen",
      testId: selectedSession.awaiting_recovery ? "archive-reattach" : "archive-reopen",
      label: GermanContractShowcase ? "Sitzung wiederherstellen" : (selectedSession.awaiting_recovery ? t(strings.recoverableSessions.reattach) : t(strings.archiveDrawer.reopen)),
      primary: true,
      disabled: selectedSession.awaiting_recovery
        ? !recoverableById.get(selectedSession.id)?.recoverable || reopeningId !== null
        : selectedSession.restore_state !== "reopenable" || reopeningId !== null,
      onSelect: () => selectedSession.awaiting_recovery ? void recoverOne(selectedSession.id) : void reopenSelected(),
    };
    const exportAction: OverlaySecondaryAction = {
      id: "export",
      label: GermanContractShowcase ? "Verlauf sicher exportieren" : t(strings.archiveDrawer.export),
      disabled: !selectedIsArchived,
      onSelect: () => void openExport(),
    };
    const receiptAction: OverlaySecondaryAction = {
      id: "receipt",
      label: GermanContractShowcase ? "Beleg anzeigen" : t(strings.archiveDrawer.receipt),
      disabled: receiptLoading,
      onSelect: () => void inspectReceipt(),
    };
    const deleteAction: OverlaySecondaryAction = {
      id: "delete",
      label: GermanContractShowcase ? "Sitzung dauerhaft löschen" : t(strings.archiveDrawer.delete),
      disabled: !selectedIsArchived,
      onSelect: () => setDeleteTarget(selectedSession),
    };
    // Closing is not a bar action: the drawer header already carries the
    // close control (`closeLabel`), and the contract caps the bar at four.
    return selectedIsArchived
      ? [reopenAction, exportAction, receiptAction, deleteAction]
      : [reopenAction, exportAction, deleteAction];
  })() : undefined;

  return (
    <>
      <FullPageDrawer
        avoidKeyboard
        open={open}
        onClose={onClose}
        title={t(strings.archiveDrawer.title)}
        closeLabel={t(strings.archiveDrawer.close)}
        testId="archive-drawer"
        actions={archiveActions}
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
              {(["text", "semantic", "node"] as const).map((mode) => (
                <button key={mode} type="button" data-testid={`archive-search-mode-${mode}`} aria-pressed={searchMode === mode}
                  onClick={() => { setSearchMode(mode); setSelectedRemote(null); if (mode !== "text") setOrphansOnly(false); }}
                  className={cn("rounded-full border px-3 py-1", searchMode === mode ? "border-wc-accent/50 bg-wc-accent/20 font-medium text-wc-text-primary" : "border-wc-default text-wc-text-secondary")}>
                  {t(mode === "text" ? strings.archiveDrawer.textSearch : mode === "semantic" ? strings.archiveDrawer.semanticSearch : strings.archiveDrawer.nodeSearch)}
                </button>
              ))}
              {searchMode === "node" && <input data-testid="archive-node-id" aria-label={t(strings.archiveDrawer.nodeIdLabel)} value={nodeId} onChange={(event) => setNodeId(event.target.value)} placeholder={t(strings.archiveDrawer.nodeIdPlaceholder)} className="min-w-32 max-w-48 rounded-full border border-wc-default bg-wc-surface-input px-3 py-1 text-wc-text-secondary" />}
              <select data-testid="archive-agent-filter" value={agentType} onChange={(event) => setAgentType(event.target.value as "all" | AgentType)} className="rounded-full border border-wc-default bg-wc-surface-input px-3 py-1 text-wc-text-secondary">
                <option value="all">{t(strings.archiveDrawer.allAgents)}</option>
                <option value="claude">Claude</option><option value="codex">Codex</option><option value="opencode">OpenCode</option><option value="grok">Grok</option><option value="none">{t(strings.archiveDrawer.shell)}</option>
              </select>
              <select data-testid="archive-time-filter" value={timeRange} disabled={searchMode !== "text"} onChange={(event) => setTimeRange(event.target.value)} className="rounded-full border border-wc-default bg-wc-surface-input px-3 py-1 text-wc-text-secondary disabled:opacity-50">
                <option value="any">{t(strings.archiveDrawer.anyTime)}</option><option value="7d">{t(strings.archiveDrawer.last7Days)}</option><option value="30d">{t(strings.archiveDrawer.last30Days)}</option><option value="365d">{t(strings.archiveDrawer.lastYear)}</option>
              </select>
              <label className="inline-flex items-center gap-1.5 rounded-full border border-wc-default px-3 py-1 text-wc-text-secondary"><input data-testid="archive-my-messages" type="checkbox" checked={myMessages} disabled={allStates} onChange={(event) => setMyMessages(event.target.checked)} />{t(strings.archiveDrawer.myMessages)}</label>
              <label className="inline-flex items-center gap-1.5 rounded-full border border-wc-default px-3 py-1 text-wc-text-secondary"><input data-testid="archive-all-states" type="checkbox" checked={allStates} onChange={(event) => { setAllStates(event.target.checked); if (event.target.checked) setMyMessages(false); }} />{t(strings.archiveDrawer.allStates)}</label>
              {orphanRows.length > 0 && <button type="button" data-testid="archive-orphans-filter" aria-pressed={orphansOnly} onClick={() => { setOrphansOnly((value) => !value); setSelected(null); setSelectedArchiveId(orphanRows[0]?.id ?? null); }} className={cn("rounded-full border px-3 py-1", orphansOnly ? "border-amber-400/60 bg-amber-500/15 text-amber-100" : "border-wc-default text-wc-text-secondary")}>{t(strings.archiveDrawer.crashOrphans, { count: orphanRows.length })}</button>}
            </div>
            {searchNotice && <div role="status" data-testid="archive-search-notice" className="rounded bg-wc-surface-input px-2 py-1.5 text-xs text-wc-text-muted">{searchNotice}</div>}
            {searchMode === "text" && timeRange !== "any" && <div role="note" data-testid="archive-source-time-note" className="rounded bg-wc-surface-input px-2 py-1.5 text-xs text-wc-text-muted">{t(strings.archiveDrawer.nativeTimeFilterNotice)}</div>}
            {error && <div role="alert" className="rounded bg-red-500/10 px-3 py-1.5 text-xs text-red-300">{error}</div>}
            {continuityWarning && <div role="status" data-testid="archive-continuity-warning" className="rounded bg-amber-500/10 px-3 py-1.5 text-xs text-amber-200">{continuityWarning}</div>}
          </div>
        }
      >
        <div className="flex h-full min-h-0 flex-col md:flex-row">
          <section aria-label={t(strings.archiveDrawer.resultsHeading)} className="max-h-[42%] shrink-0 overflow-y-auto border-b border-wc-default md:max-h-none md:w-[22rem] md:border-b-0 md:border-e">
            {orphansOnly && !query.trim() ? (
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
                      {row.archive_note && <span data-testid={`archive-note-${row.id}`} className="mt-1 block line-clamp-2 text-[11px] text-wc-text-secondary">{row.archive_note}</span>}
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
            ) : matches.length === 0 && remoteMatches.length === 0 ? (
              <div className="p-5 text-sm text-wc-text-muted">{t(strings.archiveDrawer.noResults)}</div>
            ) : <>{matches.map((match) => {
              const row = archiveById.get(match.sessionId);
              return (
                <button
                  key={match.eventId}
                  type="button"
                  data-testid={`archive-hit-${match.eventId}`}
                  onClick={() => { setSelected(match); setSelectedRemote(null); setSelectedArchiveId(match.sessionId); }}
                  className={cn("block w-full border-b border-wc-default/60 p-3 text-start transition", selected?.eventId === match.eventId ? "bg-wc-accent/10" : "hover:bg-wc-surface-raised")}
                >
                  <div className="truncate text-xs font-semibold text-wc-text-primary">{row?.agent_type ?? "agent"} · {row?.pane_name ?? match.sessionId.slice(0, 8)}</div>
                  <div className="mt-1 line-clamp-3 text-xs leading-relaxed text-wc-text-secondary">{highlightedExcerpt(match.excerpt, query)}</div>
                  <div className="mt-1 text-[10px] text-wc-text-faint">{match.occurredAt ? `Message · ${formatRelativeTime(match.occurredAt)}` : nativeArchiveHarness(match.sessionId) ? `Indexed · ${formatRelativeTime(match.createdAt)}` : formatRelativeTime(match.createdAt)} · {match.role === "user" ? t(strings.archiveDrawer.you) : row?.agent_type ?? match.role}</div>
                </button>
              );
            })}{remoteMatches.map((match) => (
              <button key={`${match.source}:${match.eventId}`} type="button" data-testid={`archive-remote-hit-${match.source}-${match.eventId}`}
                onClick={() => { setSelected(null); setSelectedArchiveId(null); setSelectedRemote(match); }}
                className={cn("block w-full border-b border-wc-default/60 p-3 text-start transition", selectedRemote?.eventId === match.eventId ? "bg-wc-accent/10" : "hover:bg-wc-surface-raised")}>
                <div className="truncate text-xs font-semibold text-wc-text-primary">{match.source === "node" ? `${match.nodeId} · ${match.harness || "agent"}` : `${match.harness || "agent"} · ${t(strings.archiveDrawer.semanticSearch)}`}</div>
                <div className="mt-1 line-clamp-3 text-xs leading-relaxed text-wc-text-secondary">{highlightedExcerpt(match.excerpt, query)}</div>
                <div className="mt-1 text-[10px] text-wc-text-faint">{match.occurredAt ? `Message · ${formatRelativeTime(match.occurredAt)}` : match.source === "node" ? `Indexed · ${formatRelativeTime(match.createdAt)}` : formatRelativeTime(match.createdAt)} · {match.role === "user" ? t(strings.archiveDrawer.you) : match.role} · {match.sessionId.slice(0, 8)}</div>
              </button>
            ))}</>}
          </section>

          <section aria-label={t(strings.archiveDrawer.readerHeading)} className="flex min-h-0 min-w-0 flex-1 flex-col">
            {selectedSession ? (
              <>
                <header className="shrink-0 border-b border-wc-default px-3 py-2">
                  <div className="flex flex-wrap items-start gap-2">
                    <span className="mt-1 h-3 w-3 rounded-full border border-white/20" style={{ backgroundColor: selectedSession.header_color || "transparent" }} aria-hidden="true" />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-semibold text-wc-text-primary" data-testid={selectedNativeHarness ? "native-archive-heading" : undefined}>
                        {selectedNativeHarness ? `${selectedNativeHarness} · Native archive` : `${selectedSession.agent_type} · ${selectedSession.pane_name}`}
                      </div>
                      <div className="truncate text-xs text-wc-text-muted">
                        {selectedIsArchived
                          ? <>{t(strings.archiveDrawer.archivedAt, { time: formatRelativeTime(selectedSession.archived_at) })} · {t(strings.archiveDrawer.messageCount, { count: selectedSession.message_count })}{selectedSession.cwd ? ` · ${selectedSession.cwd}` : ""}</>
                          : selectedNativeHarness ? `${selected?.occurredAt ? "Message" : "Indexed"} · ${formatRelativeTime(selected?.occurredAt || selected?.createdAt || selectedSession.created_at)}` : null}
                      </div>
                      <div className="mt-1 text-xs text-wc-text-secondary">{t(RESTORE_STATE_LABEL[selectedSession.restore_state])}{selectedSession.restore_state_reason ? ` — ${selectedSession.restore_state_reason}` : ""}</div>
                      {selectedSession.archive_note && <div data-testid="archive-note" className="mt-1 whitespace-pre-wrap text-xs text-wc-text-secondary">{selectedSession.archive_note}</div>}
                    </div>
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
            ) : selectedRemote ? (
              <article data-testid="archive-remote-preview" data-source={selectedRemote.source} data-node-id={selectedRemote.nodeId} className="space-y-3 p-5">
                <div className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">{selectedRemote.source === "node" ? t(strings.archiveDrawer.nodePreview, { node: selectedRemote.nodeId ?? "" }) : t(strings.archiveDrawer.semanticPreview)}</div>
                <div className="text-xs text-wc-text-faint">{selectedRemote.harness ?? "agent"} · {selectedRemote.sessionId} · {selectedRemote.eventId}</div>
                <p className="whitespace-pre-wrap text-sm leading-relaxed text-wc-text-primary">{selectedRemote.excerpt}</p>
                <p className="text-xs text-wc-text-muted">{t(strings.archiveDrawer.remotePreviewOnly)}</p>
              </article>
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
