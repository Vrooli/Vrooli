import { useCallback, useEffect, useState } from "react";
import { endUsageMode, getUsageMode, startUsageMode, type UsageModeStart, type UsageModeState } from "../../api/usageMode";

const EMPTY: UsageModeState = { modes: [], agents: [], canControl: false };

/**
 * The active usage mode, read from Web Console's own API. A failed read means
 * "no mode": the launcher must never wait on, or be blocked by, this.
 */
export function useUsageMode(enabled: boolean) {
  const [state, setState] = useState<UsageModeState>(EMPTY);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    if (!enabled) { setState(EMPTY); return; }
    const abort = new AbortController();
    getUsageMode(abort.signal).then((next) => { if (!abort.signal.aborted) setState(next); })
      .catch(() => { if (!abort.signal.aborted) setState(EMPTY); });
    // An expiring mode must disappear on its own while the launcher is open.
    const timer = window.setInterval(() => { setRevision((value) => value + 1); }, 60_000);
    return () => { abort.abort(); window.clearInterval(timer); };
  }, [enabled, revision]);
  const run = useCallback(async (action: () => Promise<UsageModeState>) => {
    setBusy(true); setError(false);
    try { setState(await action()); return true; } catch { setError(true); return false; } finally { setBusy(false); }
  }, []);
  return {
    state, busy, error,
    start: useCallback((body: UsageModeStart) => run(() => startUsageMode(body)), [run]),
    end: useCallback((id: string) => run(() => endUsageMode(id)), [run]),
  };
}

export type UsageModeController = ReturnType<typeof useUsageMode>;
