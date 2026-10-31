import { useCallback, useEffect, useRef, useState } from "react";

export type ClipboardFeedbackStatus = "idle" | "copied" | "error";

export type ClipboardCopyResult =
  | { ok: true }
  | { ok: false; error: unknown };

export function useClipboardFeedback(resetAfterMs = 2000) {
  const [status, setStatus] = useState<ClipboardFeedbackStatus>("idle");
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(false);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (timerRef.current !== null) clearTimeout(timerRef.current);
      timerRef.current = null;
    };
  }, []);

  const showStatus = useCallback((nextStatus: Exclude<ClipboardFeedbackStatus, "idle">) => {
    if (!mountedRef.current) return;
    if (timerRef.current !== null) clearTimeout(timerRef.current);
    setStatus(nextStatus);
    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      if (mountedRef.current) setStatus("idle");
    }, resetAfterMs);
  }, [resetAfterMs]);

  const copyText = useCallback(async (text: string): Promise<ClipboardCopyResult> => {
    try {
      await navigator.clipboard.writeText(text);
      showStatus("copied");
      return { ok: true };
    } catch (error) {
      showStatus("error");
      return { ok: false, error };
    }
  }, [showStatus]);

  return { status, copyText };
}
