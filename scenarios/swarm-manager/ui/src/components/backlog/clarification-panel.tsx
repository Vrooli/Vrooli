/**
 * ClarificationPanel — Slide-up panel for multi-turn clarification on
 * workshop decisions.
 *
 * Renders inside a FloatingPanel (draggable desktop / bottom-sheet mobile).
 * Contains a chat message thread, action buttons, and an input area with
 * image attachment support.
 *
 * State is driven by the Zustand `useClarificationStore`. The panel opens
 * when a user clicks ClarifyButton on any decision card and closes on
 * dismiss or action completion.
 */

import { useState, useCallback, useEffect, useRef } from "react";
import { AlertCircle, Loader2, Paperclip, SendHorizontal } from "lucide-react";
import { FloatingPanel } from "../ui/floating-panel";
import { CaptureAttachmentPreview } from "../capture/capture-attachment-preview";
import { ClarificationMessages } from "./clarification-messages";
import { ClarificationActionButtons } from "./clarification-action-buttons";
import { useClarificationStore } from "../../stores/clarification-store";
import { useAttachments } from "../../hooks/useAttachments";
import { parseImpactFromContent } from "../../lib/clarification-utils";
import { isApiError } from "../../lib/api-client";
import { backlogService } from "../../services/backlog-service";
import type { ClarificationThread } from "../../types/domain";

const POLL_INTERVAL_MS = 3000;
const MAX_VISIBLE_LINES = 4;
const LINE_HEIGHT_PX = 20;
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;
const STALENESS_THRESHOLD_MS = 90_000;
const DRAFT_KEY = "swarm-clarification-draft";
const DRAFT_DEBOUNCE_MS = 300;

const INITIAL_POSITION = {
  x: Math.max(8, window.innerWidth - 520),
  y: Math.max(8, window.innerHeight - 560),
};

interface ClarificationPanelProps {
  /** Called when a clarification action modifies workshop data (e.g. update/remove/invalidate). */
  onAction?: (action: string) => void;
}

export function ClarificationPanel({ onAction }: ClarificationPanelProps) {
  const { isOpen, target, thread, isCreating, isLoading, close, setThread, setCreating, setLoading } =
    useClarificationStore();

  const [text, setText] = useState(() => {
    try {
      return localStorage.getItem(DRAFT_KEY) ?? "";
    } catch {
      return "";
    }
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isActing, setIsActing] = useState(false);
  const [isStale, setIsStale] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const stalenessTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const draftTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const { attachments, addFile, removeFile, clearAll, getFiles } = useAttachments();

  // Persist text draft to localStorage (debounced).
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
        // Storage full or unavailable — ignore.
      }
    }, DRAFT_DEBOUNCE_MS);
    return () => {
      if (draftTimerRef.current !== null) clearTimeout(draftTimerRef.current);
    };
  }, [text]);

  // Auto-resize textarea.
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`;
  }, [text]);

  // Focus textarea and re-measure size when panel opens (textarea may have
  // hydrated text from localStorage that wasn't measured before mounting).
  useEffect(() => {
    if (isOpen) {
      const timer = setTimeout(() => {
        const el = textareaRef.current;
        if (el) {
          el.style.height = "auto";
          el.style.height = `${Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`;
          el.focus();
        }
      }, 250);
      return () => clearTimeout(timer);
    }
  }, [isOpen]);

  // ---------------------------------------------------------------------------
  // Fetch existing thread on panel reopen
  // ---------------------------------------------------------------------------

  useEffect(() => {
    const threadId = target?.clarificationId;
    if (!isOpen || !threadId) return;

    let cancelled = false;

    (async () => {
      try {
        const resp = await backlogService.getClarification(
          target.backlogKind,
          target.backlogName,
          threadId,
        );
        if (!cancelled) {
          setThread(resp.thread);
        }
      } catch (err) {
        if (!cancelled) {
          if (isApiError(err) && err.status === 404) {
            // Thread was deleted (e.g. round invalidation) — show empty state.
          } else {
            setError(isApiError(err) ? err.userMessage : "Failed to load clarification thread.");
          }
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();

    return () => { cancelled = true; };
  }, [isOpen, target?.clarificationId, target?.backlogKind, target?.backlogName, setThread, setLoading]);

  // ---------------------------------------------------------------------------
  // Polling for agent responses
  // ---------------------------------------------------------------------------

  const lastMsg = thread?.messages[thread.messages.length - 1];
  const isWaitingForAgent =
    thread?.status === "active" && lastMsg?.role === "user";

  const pollForResponse = useCallback(async () => {
    if (!target || !thread) return;
    try {
      const resp = await backlogService.getClarification(
        target.backlogKind,
        target.backlogName,
        thread.id,
      );
      if (
        resp.thread &&
        (resp.thread.messages.length > thread.messages.length ||
          resp.thread.status !== "active")
      ) {
        setThread(resp.thread);
      }
    } catch {
      // Polling failure is non-fatal.
    }
  }, [target, thread, setThread]);

  useEffect(() => {
    if (isWaitingForAgent) {
      pollingRef.current = setInterval(pollForResponse, POLL_INTERVAL_MS);
      return () => {
        if (pollingRef.current) clearInterval(pollingRef.current);
      };
    }
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, [isWaitingForAgent, pollForResponse]);

  // Clean up polling on unmount / panel close.
  useEffect(() => {
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, []);

  // Staleness timeout — warn user when agent takes too long.
  useEffect(() => {
    if (isWaitingForAgent) {
      setIsStale(false);
      stalenessTimerRef.current = setTimeout(() => setIsStale(true), STALENESS_THRESHOLD_MS);
      return () => {
        if (stalenessTimerRef.current) clearTimeout(stalenessTimerRef.current);
      };
    }
    setIsStale(false);
    if (stalenessTimerRef.current) {
      clearTimeout(stalenessTimerRef.current);
      stalenessTimerRef.current = null;
    }
  }, [isWaitingForAgent]);

  // ---------------------------------------------------------------------------
  // Submission handlers
  // ---------------------------------------------------------------------------

  const handleSubmit = useCallback(async () => {
    const trimmed = text.trim();
    if ((!trimmed && attachments.length === 0) || !target) return;
    if (isSubmitting || isCreating) return;

    const files = getFiles();

    setError(null);

    if (!thread) {
      // First message — create a new clarification thread.
      setCreating(true);
      setText("");
      clearAll();
      try {
        const resp = await backlogService.createClarification(
          target.backlogKind,
          target.backlogName,
          target.roundNumber,
          target.itemId,
          trimmed || undefined,
          files.length > 0 ? files : undefined,
        );
        setThread(resp.thread);
      } catch (err) {
        console.error("Create clarification failed:", err);
        setError(isApiError(err) ? err.userMessage : "Failed to start clarification. Please try again.");
        setText(trimmed);
      } finally {
        setCreating(false);
      }
    } else {
      // Continue existing thread.
      setIsSubmitting(true);
      setText("");
      clearAll();
      try {
        const resp = await backlogService.continueClarification(
          target.backlogKind,
          target.backlogName,
          thread.id,
          trimmed,
          files.length > 0 ? files : undefined,
        );
        setThread(resp.thread);
      } catch (err) {
        console.error("Continue clarification failed:", err);
        setError(isApiError(err) ? err.userMessage : "Failed to send message. Please try again.");
        setText(trimmed);
      } finally {
        setIsSubmitting(false);
      }
    }
  }, [
    text,
    attachments.length,
    target,
    thread,
    isSubmitting,
    isCreating,
    getFiles,
    clearAll,
    setCreating,
    setThread,
  ]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const fileList = e.target.files;
    if (!fileList) return;
    Array.from(fileList).forEach(addFile);
    e.target.value = "";
  };

  // ---------------------------------------------------------------------------
  // Action handler
  // ---------------------------------------------------------------------------

  const handleAction = useCallback(
    async (action: string) => {
      if (!target || !thread) return;
      setIsActing(true);
      setError(null);
      try {
        const suggestedUpdate = thread.latest_impact?.suggested_update;
        await backlogService.clarificationAction(
          target.backlogKind,
          target.backlogName,
          thread.id,
          action,
          action === "update_decision" && suggestedUpdate
            ? suggestedUpdate
            : undefined,
        );
        close();
        onAction?.(action);
      } catch (err) {
        console.error("Clarification action failed:", err);
        setError(isApiError(err) ? err.userMessage : "Action failed. Please try again.");
      } finally {
        setIsActing(false);
      }
    },
    [target, thread, close, onAction],
  );

  // ---------------------------------------------------------------------------
  // Close handler — reset local state when panel closes.
  // ---------------------------------------------------------------------------

  const handleClose = useCallback(() => {
    setText("");
    setError(null);
    setIsStale(false);
    clearAll();
    try { localStorage.removeItem(DRAFT_KEY); } catch { /* ignore */ }
    close();
  }, [close, clearAll]);

  // ---------------------------------------------------------------------------
  // Derived state
  // ---------------------------------------------------------------------------

  const latestAssistantMsg = thread?.messages
    ? [...thread.messages].reverse().find((m) => m.role === "assistant")
    : null;
  const impact =
    thread?.latest_impact ??
    (latestAssistantMsg
      ? parseImpactFromContent(latestAssistantMsg.content)
      : null);
  const hasAssistantResponse = thread?.messages.some(
    (m) => m.role === "assistant",
  );

  const canSubmit =
    (text.trim().length > 0 || attachments.length > 0) &&
    !isSubmitting &&
    !isCreating &&
    !isLoading;
  const isTerminal =
    thread?.status === "resolved" || thread?.status === "dismissed";

  const panelTitle = target?.itemTopic
    ? `Clarify: ${target.itemTopic.length > 40 ? target.itemTopic.slice(0, 40) + "…" : target.itemTopic}`
    : "Clarification";

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={handleClose}
      title={panelTitle}
      initialPosition={INITIAL_POSITION}
      className="max-w-lg"
      testId="clarification-panel"
    >
      <div className="flex h-[50vh] max-h-[500px] flex-col">
        {/* Chat messages */}
        {isLoading ? (
          <div className="flex flex-1 items-center justify-center px-4" data-testid="clarification-loading">
            <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
          </div>
        ) : thread && thread.messages.length > 0 ? (
          <ClarificationMessages
            messages={thread.messages}
            isWaitingForAgent={isWaitingForAgent ?? false}
          />
        ) : (
          <div className="flex flex-1 items-center justify-center px-4">
            <p className="text-center text-sm text-slate-500">
              {isCreating
                ? "Starting clarification..."
                : "Ask a question about this decision. You can include images."}
            </p>
          </div>
        )}

        {/* Impact context note preview */}
        {impact?.context_note && (
          <div className="mx-1 mb-2 rounded-md border border-cyan-500/20 bg-cyan-500/5 px-3 py-2">
            <span className="text-[10px] font-medium text-cyan-400">
              Context note
            </span>
            <p className="mt-0.5 text-xs text-slate-400">
              {impact.context_note}
            </p>
          </div>
        )}

        {/* Action buttons */}
        {hasAssistantResponse && thread?.status === "active" && (
          <div className="mx-1 mb-2">
            <ClarificationActionButtons
              impact={impact ?? undefined}
              disabled={isActing}
              onAction={handleAction}
            />
          </div>
        )}

        {/* Terminal status */}
        {isTerminal && (
          <div className="mx-1 mb-2 text-xs text-slate-500">
            Thread {thread?.status}.
          </div>
        )}

        {/* Error banner */}
        {error && (
          <div className="mx-1 mb-1 flex items-start gap-2 rounded-md border border-red-500/20 bg-red-500/10 px-3 py-2">
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-red-400" />
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
          <div className="mx-1 mb-1 rounded-md border border-amber-500/20 bg-amber-500/10 px-3 py-2" data-testid="staleness-warning">
            <p className="text-xs text-amber-300">
              Agent is taking longer than expected. You can close this panel and come back later.
            </p>
          </div>
        )}

        {/* Input area */}
        {!isTerminal && (
          <div className="border-t border-slate-700 px-1 pt-2 pb-1">
            <CaptureAttachmentPreview
              attachments={attachments}
              onRemove={removeFile}
            />
            <div className="flex items-end gap-2 rounded-xl border border-slate-700 bg-slate-800/50 px-3 py-2 transition-colors focus-within:border-cyan-500/50 focus-within:bg-slate-800">
              <textarea
                ref={textareaRef}
                value={text}
                onChange={(e) => setText(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={
                  thread
                    ? "Ask a follow-up..."
                    : "What would you like to know about this decision?"
                }
                disabled={isSubmitting || isCreating || isWaitingForAgent || isLoading}
                rows={1}
                style={{ maxHeight: MAX_TEXTAREA_HEIGHT }}
                className="w-full resize-none bg-transparent text-base text-slate-200 placeholder-slate-500 outline-none disabled:opacity-50"
              />

              {/* Attach image */}
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={isSubmitting || isCreating || isWaitingForAgent || isLoading}
                className="mb-0.5 shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300 disabled:opacity-50"
                title="Attach image"
              >
                <Paperclip className="h-4 w-4" />
              </button>

              {/* Send */}
              <button
                type="button"
                onClick={handleSubmit}
                disabled={!canSubmit || isWaitingForAgent}
                className="mb-0.5 shrink-0 rounded p-1 text-cyan-500 transition-colors hover:bg-cyan-500/10 hover:text-cyan-400 disabled:text-slate-600 disabled:hover:bg-transparent"
                title="Send (Ctrl+Enter)"
              >
                {isSubmitting || isCreating ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <SendHorizontal className="h-4 w-4" />
                )}
              </button>

              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                multiple
                className="hidden"
                onChange={handleFileSelect}
              />
            </div>
            <p className="mt-1 px-2 text-xs text-slate-600">
              Ctrl+Enter to send
            </p>
          </div>
        )}
      </div>
    </FloatingPanel>
  );
}
