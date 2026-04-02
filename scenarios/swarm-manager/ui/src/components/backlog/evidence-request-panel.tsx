/**
 * EvidenceRequestPanel — Drawer-based chat panel for requesting more
 * evidence from the review agent.
 *
 * Opens when a user clicks "Request More" on an evidence item or round
 * header. Uses the Drawer component (slide-in from right). State is
 * driven by `useReviewStore`.
 *
 * The panel receives `reviewRounds` from the parent page's React Query
 * cache and extracts thread data from there. Polling for agent responses
 * is handled by the parent refetching rounds — the panel reactively
 * updates when the prop changes.
 */

import { useState, useCallback, useEffect, useRef } from "react";
import { Loader2, SendHorizontal, XCircle } from "lucide-react";
import { Drawer } from "../ui/drawer";
import { EvidenceRequestMessages } from "./evidence-request-messages";
import { useReviewStore } from "../../stores/review-store";
import { useAutoResizeTextarea } from "../../hooks/useAutoResizeTextarea";
import { reviewService } from "../../services/review-service";
import { selectors } from "../../consts/selectors";
import type { ReviewRound, RequestThread } from "../../services/review-service";

const MAX_VISIBLE_LINES = 4;
const LINE_HEIGHT_PX = 20;
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;
const STALENESS_THRESHOLD_MS = 90_000;
const DRAFT_KEY = "swarm-evidence-request-draft";
const DRAFT_DEBOUNCE_MS = 300;

export interface EvidenceRequestPanelProps {
  backlogKind: string;
  backlogName: string;
  reviewRounds: ReviewRound[];
  /** Called after a successful request/dismiss to refresh review data. */
  onAction?: () => void;
}

export function EvidenceRequestPanel({
  backlogKind,
  backlogName,
  reviewRounds,
  onAction,
}: EvidenceRequestPanelProps) {
  const {
    requestPanelOpen,
    requestTarget,
    activeThreadId,
    activeThread,
    isCreating,
    isSending,
    closeRequestPanel,
    setActiveThread,
    setCreating,
    setSending,
  } = useReviewStore();

  const [text, setText] = useState(() => {
    try {
      return localStorage.getItem(DRAFT_KEY) ?? "";
    } catch {
      return "";
    }
  });
  const [error, setError] = useState<string | null>(null);
  const [isStale, setIsStale] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const stalenessTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const draftTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // -------------------------------------------------------------------------
  // Draft persistence (debounced localStorage)
  // -------------------------------------------------------------------------
  useEffect(() => {
    if (draftTimerRef.current !== null) clearTimeout(draftTimerRef.current);
    draftTimerRef.current = setTimeout(() => {
      try {
        if (text) {
          localStorage.setItem(DRAFT_KEY, text);
        } else {
          localStorage.removeItem(DRAFT_KEY);
        }
      } catch {
        // Storage full — ignore.
      }
    }, DRAFT_DEBOUNCE_MS);
    return () => {
      if (draftTimerRef.current !== null) clearTimeout(draftTimerRef.current);
    };
  }, [text]);

  // Auto-resize textarea.
  useAutoResizeTextarea(textareaRef, text, { maxHeight: MAX_TEXTAREA_HEIGHT });

  // Focus textarea when panel opens.
  useEffect(() => {
    if (requestPanelOpen) {
      const timer = setTimeout(() => textareaRef.current?.focus(), 250);
      return () => clearTimeout(timer);
    }
  }, [requestPanelOpen]);

  // -------------------------------------------------------------------------
  // Sync thread data from reviewRounds prop
  // -------------------------------------------------------------------------
  useEffect(() => {
    if (!requestPanelOpen || !activeThreadId || !requestTarget) return;
    const round = reviewRounds.find((r) => r.round === requestTarget.round);
    const thread = round?.request_threads?.find((t) => t.id === activeThreadId);
    if (thread) {
      setActiveThread(thread);
    }
  }, [requestPanelOpen, activeThreadId, requestTarget, reviewRounds, setActiveThread]);

  // -------------------------------------------------------------------------
  // Staleness warning
  // -------------------------------------------------------------------------
  useEffect(() => {
    if (stalenessTimerRef.current) clearTimeout(stalenessTimerRef.current);
    setIsStale(false);

    const messages = activeThread?.messages;
    if (!messages || messages.length === 0) return;
    const lastMsg = messages[messages.length - 1] as typeof messages[number] | undefined;
    if (!lastMsg || lastMsg.role !== "user") return;

    stalenessTimerRef.current = setTimeout(() => setIsStale(true), STALENESS_THRESHOLD_MS);
    return () => {
      if (stalenessTimerRef.current) clearTimeout(stalenessTimerRef.current);
    };
  }, [activeThread?.messages]);

  // -------------------------------------------------------------------------
  // Computed values
  // -------------------------------------------------------------------------
  const messages = activeThread?.messages ?? [];
  const lastMessage = messages.length > 0 ? messages[messages.length - 1] : undefined;
  const isWaitingForAgent =
    messages.length > 0 &&
    lastMessage?.role === "user" &&
    activeThread?.status === "pending";
  const isTerminal = activeThread?.status === "fulfilled" || activeThread?.status === "dismissed";
  const canSubmit = text.trim().length > 0 && !isCreating && !isSending && !isWaitingForAgent && !isTerminal;

  // -------------------------------------------------------------------------
  // Handlers
  // -------------------------------------------------------------------------
  const handleSubmit = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || !requestTarget) return;

    setError(null);

    try {
      if (!activeThreadId) {
        // Create new request thread.
        setCreating(true);
        const { thread_id } = await reviewService.requestMoreEvidence(
          backlogKind,
          backlogName,
          requestTarget.round,
          trimmed,
          requestTarget.evidenceId,
        );
        // Set the thread ID — the next rounds refresh will populate thread data.
        setActiveThread({ id: thread_id, status: "pending", messages: [{ role: "user", content: trimmed, timestamp: new Date().toISOString() }], created_at: new Date().toISOString(), evidence_id: requestTarget.evidenceId } as RequestThread);
        setCreating(false);
      } else {
        // Continue existing thread.
        setSending(true);
        await reviewService.continueRequest(
          backlogKind,
          backlogName,
          requestTarget.round,
          activeThreadId,
          trimmed,
        );
        // Optimistically add the user message.
        if (activeThread) {
          setActiveThread({
            ...activeThread,
            messages: [
              ...activeThread.messages,
              { role: "user", content: trimmed, timestamp: new Date().toISOString() },
            ],
          });
        }
        setSending(false);
      }

      setText("");
      try {
        localStorage.removeItem(DRAFT_KEY);
      } catch { /* ignore */ }
      onAction?.();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to send request";
      setError(msg);
      setCreating(false);
      setSending(false);
    }
  }, [text, requestTarget, activeThreadId, activeThread, backlogKind, backlogName, setCreating, setSending, setActiveThread, onAction]);

  const handleDismiss = useCallback(async () => {
    if (!requestTarget || !activeThreadId) return;

    try {
      await reviewService.dismissRequest(
        backlogKind,
        backlogName,
        requestTarget.round,
        activeThreadId,
      );
      onAction?.();
      closeRequestPanel();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to dismiss";
      setError(msg);
    }
  }, [requestTarget, activeThreadId, backlogKind, backlogName, onAction, closeRequestPanel]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        if (canSubmit) handleSubmit();
      }
    },
    [canSubmit, handleSubmit],
  );

  // -------------------------------------------------------------------------
  // Build drawer title/description
  // -------------------------------------------------------------------------
  const targetLabel = requestTarget
    ? `Round ${requestTarget.round}${requestTarget.evidenceId ? ` — Item ${requestTarget.evidenceId}` : ""}`
    : "";

  // -------------------------------------------------------------------------
  // Footer with input area
  // -------------------------------------------------------------------------
  const footer = !isTerminal ? (
    <div>
      {/* Error banner */}
      {error && (
        <div className="mb-2 flex items-start gap-2 rounded-md border border-red-500/20 bg-red-500/10 px-3 py-2" data-testid={selectors.evidenceRequest.error}>
          <p className="flex-1 text-xs text-red-300">{error}</p>
          <button
            type="button"
            onClick={() => setError(null)}
            className="shrink-0 text-xs text-red-400 hover:text-red-300"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Staleness warning */}
      {isStale && (
        <div className="mb-2 rounded-md border border-amber-500/20 bg-amber-500/10 px-3 py-2" data-testid={selectors.evidenceRequest.staleWarning}>
          <p className="text-xs text-amber-300">
            Agent is taking longer than expected. You can close this panel and come back later.
          </p>
        </div>
      )}

      {/* Input area */}
      <div className="flex items-end gap-2 rounded-xl border border-slate-700 bg-slate-800/50 px-3 py-2 transition-colors focus-within:border-violet-500/50 focus-within:bg-slate-800">
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={
            activeThread
              ? "Ask a follow-up..."
              : "What additional evidence do you need?"
          }
          disabled={isCreating || isSending || isWaitingForAgent}
          rows={1}
          style={{ maxHeight: MAX_TEXTAREA_HEIGHT }}
          className="w-full resize-none bg-transparent text-base text-slate-200 placeholder-slate-500 outline-none disabled:opacity-50"
          data-testid={selectors.evidenceRequest.textInput}
        />
        <button
          type="button"
          onClick={handleSubmit}
          disabled={!canSubmit}
          className="mb-0.5 shrink-0 rounded p-1 text-violet-500 transition-colors hover:bg-violet-500/10 hover:text-violet-400 disabled:text-slate-600 disabled:hover:bg-transparent"
          title="Send (Ctrl+Enter)"
          data-testid={selectors.evidenceRequest.sendButton}
        >
          {isCreating || isSending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <SendHorizontal className="h-4 w-4" />
          )}
        </button>
      </div>
      <p className="mt-1 px-2 text-xs text-slate-600">Ctrl+Enter to send</p>
    </div>
  ) : undefined;

  return (
    <Drawer
      isOpen={requestPanelOpen}
      onClose={closeRequestPanel}
      title="Request Evidence"
      description={targetLabel}
      footer={footer}
      testId={selectors.evidenceRequest.panel}
    >
      <div className="flex h-full flex-col">
        {/* Target context */}
        {requestTarget?.evidenceId && (
          <div className="border-b border-white/10 px-4 py-2 text-xs text-slate-400" data-testid={selectors.evidenceRequest.targetContext}>
            Requesting evidence for item: <span className="text-slate-300">{requestTarget.evidenceId}</span>
          </div>
        )}

        {/* Messages */}
        {messages.length > 0 || isWaitingForAgent || isCreating ? (
          <EvidenceRequestMessages
            messages={messages}
            isWaitingForAgent={isWaitingForAgent || isCreating}
          />
        ) : (
          <div className="flex flex-1 items-center justify-center px-4">
            <p className="text-center text-sm text-slate-500">
              Describe what additional evidence you need from the review agent.
            </p>
          </div>
        )}

        {/* Dismiss button (when thread exists and not terminal) */}
        {activeThreadId && !isTerminal && (
          <div className="border-t border-white/10 px-4 py-2">
            <button
              type="button"
              onClick={handleDismiss}
              className="flex items-center gap-1.5 rounded px-2 py-1 text-xs text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-300"
              data-testid={selectors.evidenceRequest.dismissButton}
            >
              <XCircle className="h-3.5 w-3.5" />
              Dismiss request
            </button>
          </div>
        )}

        {/* Terminal state */}
        {isTerminal && (
          <div className="border-t border-white/10 px-4 py-3 text-center text-xs text-slate-500">
            This request has been {activeThread?.status === "fulfilled" ? "fulfilled" : "dismissed"}.
          </div>
        )}
      </div>
    </Drawer>
  );
}
