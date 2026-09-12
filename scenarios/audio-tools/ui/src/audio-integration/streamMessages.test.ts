import { describe, expect, it, vi } from "vitest";
import { dispatchStreamMessage, type StreamMessageHandlers } from "@vrooli/audio-capture-browser";

function handlers(): StreamMessageHandlers {
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
  };
}

describe("dispatchStreamMessage", () => {
  it("[REQ:ATD-P0-004] forwards processed acknowledgements as durable bigint cursors", () => {
    const target = handlers();
    dispatchStreamMessage(JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 4 }), target, new Set());
    expect(target.onStatus).toHaveBeenCalledWith("processed_acknowledgement", "", 4n);
  });

  it("forwards provider-cell identity only when the server supplies it", () => {
    const target = handlers();
    dispatchStreamMessage(
      JSON.stringify({ type: "status", code: "ready", providerId: "kyutai", modelId: "kyutai/stt-1b-en_fr" }),
      target,
      new Set(),
    );
    expect(target.onStatus).toHaveBeenCalledWith(
      "ready",
      "",
      undefined,
      { providerId: "kyutai", modelId: "kyutai/stt-1b-en_fr" },
    );
  });

  it("[REQ:ATD-P0-001] de-duplicates a durable segment identity across replay", () => {
    const target = handlers();
    const delivered = new Set<string>();
    const frame = JSON.stringify({ type: "segment-final", text: "only once", segmentId: "segment-1", segmentIndex: 2 });
    dispatchStreamMessage(frame, target, delivered);
    dispatchStreamMessage(frame, target, delivered);
    expect(target.onSegmentFinal).toHaveBeenCalledTimes(1);
    expect(target.onSegmentFinal).toHaveBeenCalledWith("only once", 2);
  });

  it("ignores malformed frames without changing delivery state", () => {
    const target = handlers();
    const delivered = new Set<string>();
    dispatchStreamMessage("not-json", target, delivered);
    expect(delivered).toEqual(new Set());
    expect(target.onError).not.toHaveBeenCalled();
  });

  it("does not turn an invalid acknowledgement cursor into a handler exception", () => {
    const target = handlers();
    expect(() => dispatchStreamMessage(JSON.stringify({ type: "status", code: "processed_acknowledgement", processedSequence: 0.5 }), target, new Set())).not.toThrow();
    expect(target.onStatus).toHaveBeenCalledWith("processed_acknowledgement", "", undefined);
  });
});
