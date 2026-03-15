/** Maximum time (ms) predictions can sit unmatched before auto-reset. */
const MAX_PREDICTION_AGE_MS = 2000;

/** Maximum pending predictions before auto-reset. */
const MAX_PENDING_PREDICTIONS = 32;

/**
 * Local echo controller for predictive terminal input display.
 *
 * Echoes printable characters immediately before the server round-trip
 * completes, then reconciles when the server response arrives. This
 * eliminates perceived keystroke latency, especially on mobile.
 *
 * Predictions auto-reset if they sit unmatched longer than
 * MAX_PREDICTION_AGE_MS or exceed MAX_PENDING_PREDICTIONS, preventing
 * stale predictions from suppressing legitimate server output.
 */
export class LocalEchoController {
  private predicted: string[] = [];
  private _enabled = true;
  private lastPredictionTime = 0;
  private clock: () => number;

  constructor(clock: () => number = Date.now) {
    this.clock = clock;
  }

  get enabled(): boolean {
    return this._enabled;
  }

  set enabled(value: boolean) {
    this._enabled = value;
    if (!value) this.predicted = [];
  }

  get pendingCount(): number {
    return this.predicted.length;
  }

  /**
   * Decides whether to locally echo `data` before sending to the server.
   * Returns the character to write to the terminal, or null if it should
   * not be locally echoed (control chars, multi-char paste, disabled, etc.).
   */
  handleInput(data: string): string | null {
    if (!this._enabled) return null;
    // Multi-char input (paste, surrogate pairs) — skip local echo
    if (data.length !== 1) return null;
    const code = data.charCodeAt(0);
    // Only echo printable ASCII (space through tilde)
    if (code < 0x20 || code === 0x7f) return null;

    // Auto-reset stale predictions that were never matched
    if (this.predicted.length > 0 &&
        this.clock() - this.lastPredictionTime > MAX_PREDICTION_AGE_MS) {
      this.predicted = [];
    }

    // Cap pending predictions to avoid unbounded growth
    if (this.predicted.length >= MAX_PENDING_PREDICTIONS) {
      this.predicted = [];
      return null;
    }

    this.predicted.push(data);
    this.lastPredictionTime = this.clock();
    return data;
  }

  /**
   * Reconciles server output against pending predictions.
   *
   * - No predictions → return data unchanged
   * - Stale predictions → discard and return data unchanged
   * - Matching chars → consume predictions, suppress echoed chars
   * - Mismatch → erase remaining predictions with backspace sequences,
   *   then return the full server data
   */
  processOutput(data: string): string {
    if (this.predicted.length === 0) return data;

    // Discard stale predictions — they are too old to trust
    if (this.clock() - this.lastPredictionTime > MAX_PREDICTION_AGE_MS) {
      this.predicted = [];
      return data;
    }

    // If server output starts with an ANSI escape sequence, skip
    // reconciliation. Colored prompts and readline sequences make
    // character-by-character matching unreliable — clear predictions
    // and pass through unchanged to avoid visual flicker.
    if (data.charCodeAt(0) === 0x1b) {
      this.predicted = [];
      return data;
    }

    let i = 0;
    // Walk through server data, consuming matching predictions
    while (i < data.length && this.predicted.length > 0) {
      if (data[i] === this.predicted[0]) {
        this.predicted.shift();
        i++;
      } else {
        // Mismatch — erase all remaining predictions and return unmatched server data
        const eraseCount = this.predicted.length;
        this.predicted = [];
        const erase = "\b \b".repeat(eraseCount);
        return erase + data.slice(i);
      }
    }

    // All data chars matched predictions — they were already echoed
    if (i === data.length) return "";

    // Matched some predictions but server sent extra data beyond them
    return data.slice(i);
  }

  /** Clears all pending predictions. Call on connect/disconnect. */
  reset(): void {
    this.predicted = [];
  }
}
