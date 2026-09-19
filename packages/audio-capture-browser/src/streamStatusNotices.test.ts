import { describe, expect, it, vi } from "vitest";
import { dispatchStreamMessage, type StreamMessageHandlers } from "./streamMessages";

/**
 * Regression guard for a warning banner that appeared and vanished several
 * times a second while the operator was dictating.
 *
 * Two defects compounded. The wire decoder substituted a human-readable
 * sentence for any status frame that carried no text, and the host treated
 * every status code as user-facing unless it appeared on a short deny-list.
 * `processed_acknowledgement` is emitted per acknowledged wire batch and
 * carries no text, so each one produced a notice; each partial transcript
 * cleared it again.
 *
 * The contract now: a status frame reaches the operator only when its emitter
 * supplied copy meant for a person.
 */

function handlers(): StreamMessageHandlers {
  return {
    onPartial: vi.fn(),
    onSegmentFinal: vi.fn(),
    onSegmentAccepted: vi.fn(),
    onSegmentRejected: vi.fn(),
    onSpeakerStatus: vi.fn(),
    onVadState: vi.fn(),
    onStatus: vi.fn(),
    onFinal: vi.fn(),
    onError: vi.fn(),
  };
}

describe("stream status copy", () => {
  it("does not invent copy for a status frame that carries none", () => {
    const target = handlers();
    dispatchStreamMessage(
      JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 4 }),
      target,
      new Set(),
    );
    expect(target.onStatus).toHaveBeenCalledWith("processed_acknowledgement", "", 4n);
  });

  it("passes through copy the emitter actually wrote", () => {
    const target = handlers();
    dispatchStreamMessage(
      JSON.stringify({
        type: "status",
        code: "backend_degraded",
        text: "Streaming degraded — buffered mode is active.",
      }),
      target,
      new Set(),
    );
    expect(target.onStatus).toHaveBeenCalledWith(
      "backend_degraded",
      "Streaming degraded — buffered mode is active.",
      undefined,
    );
  });

  it("keeps the sequence number of an acknowledgement it stays silent about", () => {
    // Silence for the operator must not mean silence for the durability
    // ledger — the acknowledgement still has to compact the journal.
    const target = handlers();
    dispatchStreamMessage(
      JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 91 }),
      target,
      new Set(),
    );
    const [, text, processed] = vi.mocked(target.onStatus).mock.calls[0]!;
    expect(text).toBe("");
    expect(processed).toBe(91n);
  });

  it("stays silent across a burst of acknowledgements", () => {
    // What a second of dictation looks like on the wire.
    const target = handlers();
    for (let sequence = 0; sequence < 12; sequence += 1) {
      dispatchStreamMessage(
        JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: sequence }),
        target,
        new Set(),
      );
    }
    const spokenTo = vi
      .mocked(target.onStatus)
      .mock.calls.filter(([, text]) => text.length > 0);
    expect(spokenTo).toHaveLength(0);
  });
});
