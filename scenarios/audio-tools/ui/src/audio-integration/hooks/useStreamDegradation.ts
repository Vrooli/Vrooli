import { useCallback, useState } from "react";

export const BUFFERED_MODE_NOTICE = "Streaming degraded — buffered mode is active for this transcription.";

/** Per-stream degradation state. This is deliberately independent from
 * provider health: a buffered completion is usable but is not real-time. */
export function useStreamDegradation() {
  const [notice, setNotice] = useState<string | null>(null);
  const observeStatus = useCallback((code: string, message?: string) => {
    if (code === "backend_degraded") {
      setNotice(message?.trim() || BUFFERED_MODE_NOTICE);
    }
  }, []);
  const observeCompletion = useCallback((fellBackToUnary: boolean) => {
    setNotice(fellBackToUnary ? BUFFERED_MODE_NOTICE : null);
  }, []);
  return { notice, observeStatus, observeCompletion };
}
