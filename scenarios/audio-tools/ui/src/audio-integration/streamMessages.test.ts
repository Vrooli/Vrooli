import { describe, expect, it, vi } from "vitest";
import { dispatchStreamMessage, type StreamMessageHandlers } from "@vrooli/audio-capture-browser";

function handlers(overrides: Partial<StreamMessageHandlers> = {}): StreamMessageHandlers {
  return {
    onStatus: vi.fn(),
    onPartial: vi.fn(),
    onSegmentFinal: vi.fn(),
    onSegmentAccepted: vi.fn(),
    onSegmentRejected: vi.fn(),
    onSpeakerStatus: vi.fn(),
    onVadState: vi.fn(),
    onFinal: vi.fn(),
    onError: vi.fn(),
    ...overrides,
  };
}

describe("dispatchStreamMessage", () => {
  it("[REQ:ATD-P0-004] forwards processed acknowledgements as durable bigint cursors", () => {
    const onStatus = vi.fn();
    const target = handlers({ onStatus });
    dispatchStreamMessage(JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 4 }), target, new Set());
    expect(onStatus).toHaveBeenCalledWith("processed_acknowledgement", "", 4n);
  });

  it("forwards provider-cell identity only when the server supplies it", () => {
    const onStatus = vi.fn();
    const target = handlers({ onStatus });
    dispatchStreamMessage(
      JSON.stringify({ type: "status", code: "ready", providerId: "kyutai", modelId: "kyutai/stt-1b-en_fr" }),
      target,
      new Set(),
    );
    expect(onStatus).toHaveBeenCalledWith(
      "ready",
      "",
      undefined,
      { providerId: "kyutai", modelId: "kyutai/stt-1b-en_fr" },
    );
  });

  it("forwards explicit native speaker policy on durable segments", () => {
    const onSegmentFinal = vi.fn();
    const target = handlers({ onSegmentFinal });
    dispatchStreamMessage(
      JSON.stringify({ type: "segment-final", text: "degraded text", segmentIndex: 1, speakerPolicy: "degraded" }),
      target,
      new Set(),
    );
    expect(onSegmentFinal).toHaveBeenCalledWith("degraded text", 1, "degraded", undefined);
  });

  it("[REQ:ATD-P0-001] de-duplicates a durable segment identity across replay", () => {
    const onSegmentFinal = vi.fn();
    const target = handlers({ onSegmentFinal });
    const delivered = new Set<string>();
    const frame = JSON.stringify({ type: "segment-final", text: "only once", segmentId: "segment-1", segmentIndex: 2 });
    dispatchStreamMessage(frame, target, delivered);
    dispatchStreamMessage(frame, target, delivered);
    expect(onSegmentFinal).toHaveBeenCalledTimes(1);
    expect(onSegmentFinal).toHaveBeenCalledWith("only once", 2, undefined, undefined);
  });

  it("ignores malformed frames without changing delivery state", () => {
    const onError = vi.fn();
    const target = handlers({ onError });
    const delivered = new Set<string>();
    dispatchStreamMessage("not-json", target, delivered);
    expect(delivered).toEqual(new Set());
    expect(onError).not.toHaveBeenCalled();
  });

  it("does not turn an invalid acknowledgement cursor into a handler exception", () => {
    const onStatus = vi.fn();
    const target = handlers({ onStatus });
    expect(() => dispatchStreamMessage(JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0.5 }), target, new Set())).not.toThrow();
    expect(onStatus).toHaveBeenCalledWith("processed_acknowledgement", "", undefined);
  });
});
