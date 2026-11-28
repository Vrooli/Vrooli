import { describe, expect, it, vi } from "vitest";
import type { Dispatch, SetStateAction } from "react";
import type { VoiceInputState } from "./types";
import { StreamNoticeController } from "./streamNoticeController";

function harness() {
  let state = { fallbackNotice: null, streamingDegradationNotice: null } as VoiceInputState;
  const setState: Dispatch<SetStateAction<VoiceInputState>> = vi.fn((update) => {
    state = typeof update === "function" ? update(state) : update;
  });
  return { controller: new StreamNoticeController(setState), setState, get state() { return state; } };
}

describe("StreamNoticeController", () => {
  it("ignores protocol bookkeeping and only shows statuses with human copy", () => {
    const h = harness();
    h.controller.handleStatus("processed_acknowledgement", "internal ack");
    h.controller.handleStatus("unknown_status");
    expect(h.setState).not.toHaveBeenCalled();
    h.controller.handleStatus("unknown_status", "Connection recovered");
    expect(h.state.fallbackNotice).toBe("Connection recovered");
  });

  it("records degradation and clears recoverable notices on recovery", () => {
    const h = harness();
    h.controller.handleStatus("backend_degraded");
    expect(h.state.streamingDegradationNotice).toContain("buffered mode");
    h.controller.handleStatus("stream_connected");
    expect(h.state.fallbackNotice).toBeNull();
    h.controller.handleStatus("reconnect_exhausted", "Reconnect failed");
    expect(h.state.fallbackNotice).toBe("Reconnect failed");
  });

  it("keeps dismissed copy hidden until a partial or recovery clears dismissal", () => {
    const h = harness();
    h.controller.handleStatus("notice", "Mic disconnected");
    h.controller.dismissFallback();
    h.controller.handleStatus("notice", "Mic disconnected");
    expect(h.state.fallbackNotice).toBeNull();
    h.controller.clearFallbackOnPartial();
    h.controller.handleStatus("notice", "Mic disconnected");
    expect(h.state.fallbackNotice).toBe("Mic disconnected");
  });
});
