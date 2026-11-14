import { uncommittedRemainder } from "./voice/trailingPartial";

export interface TranscriptBufferSnapshot {
  /** Text that has crossed a segment-final or turn-end boundary. */
  readonly committedText: string;
  /** The latest replaceable hypothesis after the committed prefix. */
  readonly interimText: string;
  /** False when a duplicate segment-final was ignored. */
  readonly accepted: boolean;
}

function normalize(text: string): string {
  return text.trim();
}

/**
 * The one spacing rule every dictation surface must agree on: punctuation
 * attaches to the word before it, anything else gets a single separating space.
 *
 * Returns the separator rather than the joined string because the composer
 * overlays draw settled and unsettled text as two separate elements and need
 * the gap between them, while the buffer needs the concatenation. Both call
 * this so a surface can never drift from the transcript it is previewing.
 */
export function transcriptSeparator(left: string, right: string): string {
  if (!left || !right) return "";
  if (/\s$/.test(left)) return "";
  return /^[,.!?;:]/.test(right) ? "" : " ";
}

/** Join settled text to the text that follows it. See `transcriptSeparator`. */
export function joinTranscriptText(left: string, right: string): string {
  const a = normalize(left);
  const b = normalize(right);
  if (!a) return b;
  if (!b) return a;
  return `${a}${transcriptSeparator(a, b)}${b}`;
}

const joinText = joinTranscriptText;

/**
 * Stateful transcript boundary handling for streaming dictation.
 *
 * Partials are replaceable. Segment-finals are durable. A segment-final may
 * cover only the prefix of the current partial, so the uncovered suffix stays
 * visible as interim text instead of being cleared or appended a second time.
 */
export class TranscriptBuffer {
  private committed = "";
  private interim = "";
  private readonly finalizedSegments = new Set<number>();

  reset(): TranscriptBufferSnapshot {
    this.committed = "";
    this.interim = "";
    this.finalizedSegments.clear();
    return this.snapshot();
  }

  snapshot(): TranscriptBufferSnapshot {
    return { committedText: this.committed, interimText: this.interim, accepted: true };
  }

  /** Replace the current interim hypothesis; never append it. */
  partial(text: string): TranscriptBufferSnapshot {
    const remainder = uncommittedRemainder({
      committedText: this.committed,
      latestPartial: text,
    });
    this.interim = normalize(remainder ?? "");
    return this.snapshot();
  }

  /**
   * Commit one server segment and retain any uncovered portion of the latest
   * partial. Duplicate segment indexes are harmless and produce no delivery.
   */
  segmentFinal(text: string, segmentIndex: number): TranscriptBufferSnapshot {
    if (this.finalizedSegments.has(segmentIndex)) return { ...this.snapshot(), accepted: false };
    this.finalizedSegments.add(segmentIndex);
    const finalText = normalize(text);
    if (!finalText) return this.snapshot();

    const previousCommitted = this.committed;
    const previousInterim = this.interim;
    this.committed = joinText(this.committed, finalText);
    this.interim = this.remainderAfterCommit(previousCommitted, previousInterim, finalText, this.committed);
    return this.snapshot();
  }

  /**
   * Accept the provider's turn-end text. Providers may send either the pure
   * tail or a full running hypothesis, so committed-prefix subtraction is
   * applied before delivery.
   */
  result(text: string): string | null {
    const finalText = normalize(text);
    if (!finalText) return null;

    const previousCommitted = this.committed;
    const previousInterim = this.interim;
    const delta = uncommittedRemainder({
      committedText: this.committed,
      latestPartial: finalText,
    });
    const delivered = normalize(delta ?? "");
    this.committed = joinText(this.committed, delivered);
    this.interim = this.remainderAfterCommit(previousCommitted, previousInterim, delivered || finalText, this.committed);
    return delivered || null;
  }

  /** Promote an uncommitted tail exactly once at turn end. */
  promoteTurnEnd(): string | null {
    const promoted = uncommittedRemainder({
      committedText: this.committed,
      latestPartial: this.interim,
    });
    const text = normalize(promoted ?? "");
    if (!text) {
      this.interim = "";
      return null;
    }
    this.committed = joinText(this.committed, text);
    this.interim = "";
    return text;
  }

  private remainderAfterCommit(
    previousCommitted: string,
    previousInterim: string,
    finalText: string,
    committedText: string,
  ): string {
    const interim = normalize(previousInterim);
    const final = normalize(finalText);
    const committed = normalize(committedText);
    if (!interim || !final || !committed) return "";

    // A longer final supersedes a shortened interim. Otherwise the helper
    // below preserves the exact committed-prefix arithmetic for both a pure
    // tail and a full running hypothesis.
    if (final.startsWith(interim)) return "";
    const fullHypothesis = joinText(previousCommitted, interim);
    const remainder = uncommittedRemainder({
      committedText: committed,
      latestPartial: fullHypothesis,
    });
    return normalize(remainder ?? "");
  }

}
