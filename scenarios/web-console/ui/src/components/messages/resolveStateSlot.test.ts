import { describe, expect, it } from "vitest";
import { resolveStateSlot } from "./resolveStateSlot";
import type { SessionActivityView } from "../../api/sessionActivity";

const base: SessionActivityView = {
  state: "working",
  source: "screen",
  confidence: 0.85,
  since: "2026-09-11T08:00:00Z",
  lastOutputAt: "2026-09-11T08:01:08Z",
  harness: "claude",
};

describe("resolveStateSlot", () => {
  it("[REQ:P0-017e] working is a strip timed from since", () => {
    expect(resolveStateSlot(base)).toEqual({
      kind: "working",
      elapsedFrom: "2026-09-11T08:00:00Z",
      lastOutputAt: "2026-09-11T08:01:08Z",
    });
  });

  it("[REQ:P0-017e] waiting with no prompt text is a detected card", () => {
    const slot = resolveStateSlot({ ...base, state: "waiting", source: "hook", prompt: { kind: "unknown", text: "", options: [], answerable: false } });
    expect(slot).toEqual({ kind: "waiting-detected", harness: "claude", source: "hook", since: "2026-09-11T08:00:00Z" });
  });

  it("[REQ:P0-017e] waiting with no prompt at all is a detected card", () => {
    expect(resolveStateSlot({ ...base, state: "waiting" })?.kind).toBe("waiting-detected");
  });

  it("[REQ:P0-017e] waiting with prompt text that cannot be answered here is a rendered card", () => {
    const prompt = { kind: "permission", text: "Do you want to proceed?", options: [{ key: "1", label: "Yes", selected: true }], answerable: false };
    expect(resolveStateSlot({ ...base, state: "waiting", prompt })).toEqual({
      kind: "waiting-rendered", harness: "claude", source: "screen", since: "2026-09-11T08:00:00Z", prompt,
    });
  });

  it("[REQ:P0-017e] waiting with an answerable prompt is an answerable card", () => {
    const prompt = { kind: "permission", text: "Do you want to proceed?", options: [{ key: "1", label: "Yes", selected: true }], answerable: true };
    expect(resolveStateSlot({ ...base, state: "waiting", prompt })?.kind).toBe("waiting-answerable");
  });

  it("[REQ:P0-017e] idle, unknown, and missing show nothing", () => {
    expect(resolveStateSlot({ ...base, state: "idle" })).toBeNull();
    expect(resolveStateSlot({ ...base, state: "unknown" })).toBeNull();
    expect(resolveStateSlot(undefined)).toBeNull();
  });

  it("[REQ:P0-017e] confidence below 0.6 shows nothing", () => {
    expect(resolveStateSlot({ ...base, confidence: 0.5 })).toBeNull();
    expect(resolveStateSlot({ ...base, state: "waiting", confidence: 0.59 })).toBeNull();
  });
});
