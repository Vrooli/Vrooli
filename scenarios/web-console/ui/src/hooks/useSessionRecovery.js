import { useEffect, useRef, useState } from "react";
import { listSessionsWithRecovery } from "../api/sessions";
const POLL_MS = 1500;
const INITIAL = {
    inProgress: false,
    total: 0,
    recovered: 0,
    awaitingRecovery: 0,
    adopted: 0,
    justCompleted: false,
};
export function useSessionRecovery() {
    const [state, setState] = useState(INITIAL);
    const sawInProgress = useRef(false);
    useEffect(() => {
        let cancelled = false;
        let timer = null;
        const poll = async () => {
            try {
                const { recovery } = await listSessionsWithRecovery();
                if (cancelled)
                    return;
                if (recovery.in_progress)
                    sawInProgress.current = true;
                const justCompleted = sawInProgress.current &&
                    !recovery.in_progress &&
                    recovery.recovered + recovery.adopted > 0;
                setState({
                    inProgress: recovery.in_progress,
                    total: recovery.total,
                    recovered: recovery.recovered,
                    awaitingRecovery: recovery.awaiting_recovery,
                    adopted: recovery.adopted,
                    justCompleted,
                });
                if (recovery.in_progress && !cancelled) {
                    timer = setTimeout(() => void poll(), POLL_MS);
                }
            }
            catch {
                // Best-effort — a transient list failure must not wedge the banner.
                // Keep polling only while we still believe recovery is running.
                if (!cancelled && sawInProgress.current) {
                    timer = setTimeout(() => void poll(), POLL_MS);
                }
            }
        };
        void poll();
        return () => {
            cancelled = true;
            if (timer)
                clearTimeout(timer);
        };
    }, []);
    return state;
}
