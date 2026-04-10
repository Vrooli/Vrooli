/**
 * useClarificationThread — Thread loading, polling, submit, and action dispatch
 * for the clarification panel.
 *
 * Extracted from clarification-panel.tsx.
 */

import { useState, useCallback, useEffect, useRef } from "react";
import { useClarificationStore } from "../stores/clarification-store";
import { useStorePolling } from "./useStorePolling";
import { isApiError } from "../lib/api-client";
import { backlogService } from "../services/backlog-service";
import type { ClarificationThread } from "../types/domain";

const POLL_INTERVAL_MS = 3000;
const STALENESS_THRESHOLD_MS = 90_000;

export interface UseClarificationThreadResult {
  thread: ClarificationThread | null;
  target: ReturnType<typeof useClarificationStore.getState>["target"];
  isOpen: boolean;
  isCreating: boolean;
  isLoading: boolean;
  isSubmitting: boolean;
  isActing: boolean;
  isStale: boolean;
  isWaitingForAgent: boolean;
  error: string | null;
  setError: (error: string | null) => void;
  handleSubmit: (text: string, files: File[]) => Promise<void>;
  handleAction: (action: string) => Promise<void>;
  handleConfirmUpdate: () => Promise<void>;
  handleClose: () => void;
}

export function useClarificationThread({
  onAction,
  onClose: externalOnClose,
}: {
  onAction?: (action: string) => void;
  onClose?: () => void;
}): UseClarificationThreadResult {
  const { isOpen, target, thread, isCreating, isLoading, close, setThread, setCreating, setLoading } =
    useClarificationStore();

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isActing, setIsActing] = useState(false);
  const [isStale, setIsStale] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const stalenessTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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

  useStorePolling({
    enabled: isWaitingForAgent ?? false,
    intervalMs: POLL_INTERVAL_MS,
    pollFn: pollForResponse,
  });

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
  // Submission handler
  // ---------------------------------------------------------------------------

  const handleSubmit = useCallback(async (text: string, files: File[]) => {
    const trimmed = text.trim();
    if ((!trimmed && files.length === 0) || !target) return;
    if (isSubmitting || isCreating) return;

    setError(null);

    if (!thread) {
      // First message — create a new clarification thread.
      setCreating(true);
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
        throw err; // Re-throw so caller can restore text
      } finally {
        setCreating(false);
      }
    } else {
      // Continue existing thread.
      setIsSubmitting(true);
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
        throw err; // Re-throw so caller can restore text
      } finally {
        setIsSubmitting(false);
      }
    }
  }, [target, thread, isSubmitting, isCreating, setCreating, setThread]);

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

  const handleConfirmUpdate = useCallback(async () => {
    if (!target || !thread) return;
    setIsActing(true);
    setError(null);
    try {
      const suggestedUpdate = thread.latest_impact?.suggested_update;
      await backlogService.clarificationAction(
        target.backlogKind,
        target.backlogName,
        thread.id,
        "update_decision",
        suggestedUpdate,
      );
      close();
      onAction?.("update_decision");
    } catch (err) {
      console.error("Clarification action failed:", err);
      setError(isApiError(err) ? err.userMessage : "Action failed. Please try again.");
      throw err; // Re-throw so caller can handle UI state
    } finally {
      setIsActing(false);
    }
  }, [target, thread, close, onAction]);

  // ---------------------------------------------------------------------------
  // Close handler
  // ---------------------------------------------------------------------------

  const handleClose = useCallback(() => {
    setError(null);
    setIsStale(false);
    externalOnClose?.();
    close();
  }, [close, externalOnClose]);

  return {
    thread,
    target,
    isOpen,
    isCreating,
    isLoading,
    isSubmitting,
    isActing,
    isStale,
    isWaitingForAgent: isWaitingForAgent ?? false,
    error,
    setError,
    handleSubmit,
    handleAction,
    handleConfirmUpdate,
    handleClose,
  };
}
