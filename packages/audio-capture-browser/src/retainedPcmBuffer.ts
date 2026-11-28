import { concatInt16 } from "./pcm";

/** Bounded whole-turn copy used for retry UX; the journal owns long turns. */
export class RetainedPcmBuffer {
  private chunks: Int16Array[] = [];
  private bytes = 0;
  private overflowed = false;

  constructor(private readonly maxBytes: number) {
    if (!Number.isSafeInteger(maxBytes) || maxBytes <= 0) {
      throw new RangeError("maxBytes must be a positive safe integer");
    }
  }

  append(samples: Int16Array): void {
    if (this.overflowed) return;
    if (this.bytes + samples.byteLength > this.maxBytes) {
      // A partial copy is not a safe retry artifact, so release it immediately.
      this.chunks = [];
      this.bytes = 0;
      this.overflowed = true;
      return;
    }
    this.chunks.push(samples);
    this.bytes += samples.byteLength;
  }

  get isOverflowed(): boolean { return this.overflowed; }
  get byteLength(): number { return this.bytes; }
  get hasCompleteAudio(): boolean { return !this.overflowed && this.chunks.length > 0; }

  snapshot(): Int16Array | null {
    return this.hasCompleteAudio ? concatInt16(this.chunks) : null;
  }

  clear(): void {
    this.chunks = [];
    this.bytes = 0;
    this.overflowed = false;
  }
}
