import { describe, expect, it } from "vitest";
import { TranscriptBuffer } from "./transcriptBuffer";

describe("TranscriptBuffer", () => {
  it("replaces partial revisions instead of appending them", () => {
    const buffer = new TranscriptBuffer();

    expect(buffer.partial("hello wor").interimText).toBe("hello wor");
    expect(buffer.partial("hello world").interimText).toBe("hello world");
    expect(buffer.partial("hello").interimText).toBe("hello");
    expect(buffer.snapshot().committedText).toBe("");
  });

  it("absorbs a segment-final prefix and preserves the uncovered tail", () => {
    const buffer = new TranscriptBuffer();
    buffer.partial("hello world");

    const state = buffer.segmentFinal("hello", 0);

    expect(state.committedText).toBe("hello");
    expect(state.interimText).toBe("world");
  });

  it("clears a shortened partial when the final contains it", () => {
    const buffer = new TranscriptBuffer();
    buffer.partial("hello wor");

    const state = buffer.segmentFinal("hello world", 0);

    expect(state.committedText).toBe("hello world");
    expect(state.interimText).toBe("");
  });

  it("keeps a disagreeing partial replaceable rather than duplicating it", () => {
    const buffer = new TranscriptBuffer();
    buffer.partial("weather");

    const state = buffer.segmentFinal("whether", 0);

    expect(state.committedText).toBe("whether");
    expect(state.interimText).toBe("weather");
  });

  it("promotes a turn-end tail once and never delivers it twice", () => {
    const buffer = new TranscriptBuffer();
    buffer.segmentFinal("committed", 0);
    buffer.partial("committed final tail");

    expect(buffer.promoteTurnEnd()).toBe("final tail");
    expect(buffer.promoteTurnEnd()).toBeNull();
    expect(buffer.snapshot().committedText).toBe("committed final tail");
    expect(buffer.snapshot().interimText).toBe("");
  });

  it("starts and remains empty when a strategy emits no partials", () => {
    const buffer = new TranscriptBuffer();

    expect(buffer.snapshot().interimText).toBe("");
    expect(buffer.reset().interimText).toBe("");
    expect(buffer.snapshot().committedText).toBe("");
  });

  it("ignores a duplicate segment-final without a second delivery", () => {
    const buffer = new TranscriptBuffer();

    expect(buffer.segmentFinal("hello", 3).accepted).toBe(true);
    expect(buffer.segmentFinal("hello", 3).accepted).toBe(false);
    expect(buffer.snapshot().committedText).toBe("hello");
  });

  it("accepts a provider final as either a full hypothesis or a pure tail", () => {
    const full = new TranscriptBuffer();
    full.segmentFinal("hello", 0);
    expect(full.result("hello world")).toBe("world");

    const tail = new TranscriptBuffer();
    tail.segmentFinal("hello", 0);
    expect(tail.result("world")).toBe("world");
  });
});
