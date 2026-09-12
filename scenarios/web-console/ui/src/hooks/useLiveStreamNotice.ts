import { useEffect, useState } from "react";
import { isLiveStreamInterrupted, LIVE_INTERRUPTION_GRACE_MS, useLiveStreamStore } from "../stores/useLiveStreamStore";

/**
 * Reports whether the live-update stream has been interrupted long enough to
 * be worth showing.
 *
 * The raw store status flips on the first failed frame. Surfacing that directly
 * made the pane announce "reconnecting" for drops that resolved before the user
 * could read the sentence — including every routine API restart.
 */
export function useLiveStreamNotice(graceMs: number = LIVE_INTERRUPTION_GRACE_MS): boolean {
  const interrupted = useLiveStreamStore((state) => isLiveStreamInterrupted(state.status));
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    if (!interrupted) {
      setSettled(false);
      return;
    }
    const timer = setTimeout(() => { setSettled(true); }, graceMs);
    return () => { clearTimeout(timer); };
  }, [interrupted, graceMs]);

  return interrupted && settled;
}
