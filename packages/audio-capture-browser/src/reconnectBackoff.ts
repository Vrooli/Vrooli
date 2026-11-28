/** Consecutive stream reconnect budget and delays, before bounded-audio fallback. */
export const MAX_CONSECUTIVE_RECONNECTS = 5;
export const RECONNECT_BACKOFF_MS = [1_000, 2_000, 4_000, 8_000, 8_000] as const;

/**
 * Tracks only the retry budget. The provider owns timer cancellation and
 * capture-generation fencing because those depend on its active turn.
 */
export class ReconnectBackoff {
  private attempts = 0;

  reset(): void {
    this.attempts = 0;
  }

  nextDelay(): number | null {
    if (this.attempts >= MAX_CONSECUTIVE_RECONNECTS) return null;
    const delay = RECONNECT_BACKOFF_MS[this.attempts] ?? RECONNECT_BACKOFF_MS.at(-1)!;
    this.attempts += 1;
    return delay;
  }

  get attemptCount(): number {
    return this.attempts;
  }
}
