import { describe, expect, it } from "vitest";

import { BRIDGE_PROTOCOL, MAX_FRAME_HEIGHT, MIN_FRAME_HEIGHT, parseBridgeMessage } from "./bridge";

const frame = {} as Window;
const message = (data: unknown, source: Window | null = frame) => parseBridgeMessage({ data, source }, frame);

describe("bridge message validation", () => {
  it("ignores anything that did not come from the explainer's own window", () => {
    expect(message({ hub: BRIDGE_PROTOCOL, type: "height", height: 400 }, {} as Window)).toBeNull();
    expect(message({ hub: BRIDGE_PROTOCOL, type: "height", height: 400 }, null)).toBeNull();
    expect(parseBridgeMessage({ data: { hub: BRIDGE_PROTOCOL, type: "height", height: 400 }, source: frame }, null)).toBeNull();
  });

  it("ignores messages outside the protocol", () => {
    for (const data of [null, "open", 5, { type: "height", height: 10 }, { hub: 2, type: "height", height: 10 }, { hub: 1, type: "navigate", url: "https://x" }]) {
      expect(message(data)).toBeNull();
    }
  });

  it("clamps a reported height and rejects one that is not a number", () => {
    expect(message({ hub: 1, type: "height", height: 5 })).toEqual({ type: "height", height: MIN_FRAME_HEIGHT });
    expect(message({ hub: 1, type: "height", height: 10 ** 7 })).toEqual({ type: "height", height: MAX_FRAME_HEIGHT });
    expect(message({ hub: 1, type: "height", height: 400.2 })).toEqual({ type: "height", height: 401 });
    expect(message({ hub: 1, type: "height", height: "400" })).toBeNull();
    expect(message({ hub: 1, type: "height", height: Number.NaN })).toBeNull();
  });

  it("accepts open only for a well-formed attachment name", () => {
    expect(message({ hub: 1, type: "open", name: "set-a.png" })).toEqual({ type: "open", name: "set-a.png", path: "" });
    expect(message({ hub: 1, type: "open", name: "renders", path: "a/b.png" })).toEqual({ type: "open", name: "renders", path: "a/b.png" });
    expect(message({ hub: 1, type: "open", name: "../etc/passwd" })).toBeNull();
    expect(message({ hub: 1, type: "open", name: "a/b" })).toBeNull();
    expect(message({ hub: 1, type: "open", name: "ok", path: 4 })).toBeNull();
    expect(message({ hub: 1, type: "open", name: "ok", path: "x".repeat(600) })).toBeNull();
  });

  it("bounds a pre-fill", () => {
    expect(message({ hub: 1, type: "prefill", questionId: "set", value: "b" })).toEqual({ type: "prefill", questionId: "set", value: "b" });
    expect(message({ hub: 1, type: "prefill", questionId: "", value: "b" })).toBeNull();
    expect(message({ hub: 1, type: "prefill", questionId: "q", value: "x".repeat(20_000) })).toBeNull();
  });

  it("accepts answer chips only within bounds and validates removals", () => {
    expect(message({ hub: 1, type: "answer", operation: "set", key: "pace", label: "Use capacity faster", value: { tier: "wednesday" } })).toEqual({
      type: "answer", operation: "set", key: "pace", label: "Use capacity faster", value: { tier: "wednesday" },
    });
    expect(message({ hub: 1, type: "answer", operation: "remove", key: "pace" })).toEqual({ type: "answer", operation: "remove", key: "pace" });
    expect(message({ hub: 1, type: "answer", operation: "set", key: "", label: "Pace", value: "fast" })).toBeNull();
    expect(message({ hub: 1, type: "answer", operation: "set", key: "pace", label: "", value: "fast" })).toBeNull();
    expect(message({ hub: 1, type: "answer", operation: "remove", key: "x".repeat(100) })).toBeNull();
    expect(message({ hub: 1, type: "answer", operation: "set", key: "pace", label: "Pace", value: "x".repeat(40_000) })).toBeNull();
  });
});
