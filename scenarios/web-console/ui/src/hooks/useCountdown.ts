import { useState, useEffect } from "react";
import type { PolicyMode } from "../lib/api";
import { parseDurationMs, formatCountdown } from "../lib/format";

// [REQ:P1-001b] Policy Configuration UI - countdown display

/**
 * Computes a live countdown string for a session's expiration policy.
 * Returns null if the policy is "never" or has no valid duration.
 */
export function useCountdown(createdAt: string, mode: PolicyMode, duration?: string): string | null {
  const [remaining, setRemaining] = useState<string | null>(null);

  useEffect(() => {
    if (mode === "never") {
      setRemaining(null);
      return;
    }

    const durationMs = parseDurationMs(duration);
    if (durationMs <= 0) {
      setRemaining(null);
      return;
    }

    const update = () => {
      const created = new Date(createdAt).getTime();
      const expiresAt = created + durationMs;
      const secs = (expiresAt - Date.now()) / 1000;
      setRemaining(formatCountdown(secs));
    };

    update();
    const id = setInterval(update, 1000);
    return () => clearInterval(id);
  }, [createdAt, mode, duration]);

  return remaining;
}
