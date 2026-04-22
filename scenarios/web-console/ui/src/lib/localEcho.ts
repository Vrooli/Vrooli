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
 * Reconciliation behaviour on mismatch: predictions are dropped and
 * the server output is passed through unchanged. The previous
 * implementation wrote \b \b sequences to "undo" predictions; that
 * produced visible flicker on high-latency links and on shells that
 * emit cursor motion in their echo (readline, zsh ZLE). The current
 * implementation trusts xterm.js to repaint from scrollback when the
 * server-authoritative bytes arrive.
 *
 * The controller can be disabled externally (useTerminalSession
 * disables it whenever the PTY is in the alternate screen buffer;
 * alt-buffer TUIs do their own rendering and predictions are always
 * wrong).
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
   * Decides whether to locally echo `data` before sending to the
   * server. Returns the character to write to the terminal, or null
   * if it should not be locally echoed (control chars, multi-char
   * paste, disabled, etc.).
   */
  handleInput(data: string): string | null {
    if (!this._enabled) return null;
    if (data.length !== 1) return null;
    const code = data.charCodeAt(0);
    if (code < 0x20 || code === 0x7f) return null;

    if (
      this.predicted.length > 0 &&
      this.clock() - this.lastPredictionTime > MAX_PREDICTION_AGE_MS
    ) {
      this.predicted = [];
    }

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
   *  - No predictions → return data unchanged.
   *  - Stale predictions → discard; return data unchanged.
   *  - Server output starts with ESC → discard predictions; return
   *    data unchanged. ANSI sequences cannot be matched
   *    character-by-character against single-char predictions, and
   *    readline often moves the cursor before echoing.
   *  - Matching prefix → consume predictions and suppress the echoed
   *    prefix. Any trailing unmatched server bytes pass through.
   *  - Mismatch → drop all remaining predictions and return the
   *    unmatched server data unchanged. No \b \b is written; xterm
   *    repaints from the server-authoritative bytes.
   */
  processOutput(data: string): string {
    if (this.predicted.length === 0) return data;

    if (this.clock() - this.lastPredictionTime > MAX_PREDICTION_AGE_MS) {
      this.predicted = [];
      return data;
    }

    if (data.charCodeAt(0) === 0x1b) {
      this.predicted = [];
      return data;
    }

    let i = 0;
    while (i < data.length && this.predicted.length > 0) {
      if (data[i] === this.predicted[0]) {
        this.predicted.shift();
        i++;
      } else {
        // Mismatch: drop all remaining predictions. Return the
        // unmatched server bytes verbatim — no backspace erasure.
        this.predicted = [];
        return data.slice(i);
      }
    }

    if (i === data.length) return "";
    return data.slice(i);
  }

  /** Clears all pending predictions. Call on connect/disconnect. */
  reset(): void {
    this.predicted = [];
  }
}
