/**
 * Local echo controller for predictive terminal input display.
 *
 * Echoes printable characters immediately before the server round-trip
 * completes, then reconciles when the server response arrives. This
 * eliminates perceived keystroke latency, especially on mobile.
 */
export class LocalEchoController {
  private predicted: string[] = [];
  private _enabled = true;

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
    this.predicted.push(data);
    return data;
  }

  /**
   * Reconciles server output against pending predictions.
   *
   * - No predictions → return data unchanged
   * - Matching chars → consume predictions, suppress echoed chars
   * - Mismatch → erase remaining predictions with backspace sequences,
   *   then return the full server data
   */
  processOutput(data: string): string {
    if (this.predicted.length === 0) return data;

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
